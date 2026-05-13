package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"stalwart-event-notification/internal/events"
)

const (
	defaultPort                       = 3000
	defaultLogLevel                   = "info"
	defaultLocale                     = "en"
	defaultTimezone                   = "UTC"
	defaultLocalesDir                 = "locales"
	defaultDedupWindowSeconds         = 60
	defaultNotificationGroupWindowSec = 0
	defaultEventsRetentionDays        = 0
)

type Config struct {
	TelegramBotToken              string
	DatabaseURL                   string
	Port                          int
	LogLevel                      string
	WebhookKey                    string
	WebhookUsername               string
	WebhookPassword               string
	AllowedUserID                 string
	AdminUserIDs                  []string
	DefaultLocale                 string
	DefaultTimezone               string
	LocalesDir                    string
	MinSeverity                   events.Severity
	QuietHoursStart               string
	QuietHoursEnd                 string
	DedupEnabled                  bool
	DedupWindowSeconds            int
	NotificationGroupWindowSecond int
	EventsRetentionDays           int
	IgnoredIPsByEvent             map[string][]string
}

func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if name == "" {
			continue
		}
		if _, exists := os.LookupEnv(name); !exists {
			_ = os.Setenv(name, value)
		}
	}
	return scanner.Err()
}

func Load() (Config, error) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	port, err := envInt("PORT", defaultPort, 1, 65535)
	if err != nil {
		return Config{}, err
	}

	minSeverityRaw := strings.ToLower(strings.TrimSpace(os.Getenv("SUBSCRIPTION_MIN_SEVERITY")))
	if minSeverityRaw == "" {
		minSeverityRaw = string(events.SeverityInfo)
	}
	minSeverity := events.Severity(minSeverityRaw)
	if minSeverity != events.SeverityInfo && minSeverity != events.SeverityWarning && minSeverity != events.SeverityAlert {
		minSeverity = events.SeverityInfo
	}

	defaultLocaleValue := strings.ToLower(strings.TrimSpace(os.Getenv("DEFAULT_LOCALE")))
	if defaultLocaleValue == "" {
		defaultLocaleValue = defaultLocale
	}

	defaultTimezoneValue := strings.TrimSpace(os.Getenv("DEFAULT_TIMEZONE"))
	if defaultTimezoneValue == "" {
		defaultTimezoneValue = defaultTimezone
	}

	localesDir := strings.TrimSpace(os.Getenv("LOCALES_DIR"))
	if localesDir == "" {
		localesDir = defaultLocalesDir
	}

	return Config{
		TelegramBotToken:              token,
		DatabaseURL:                   databaseURL,
		Port:                          port,
		LogLevel:                      envString("LOG_LEVEL", defaultLogLevel),
		WebhookKey:                    strings.TrimSpace(os.Getenv("WEBHOOK_KEY")),
		WebhookUsername:               strings.TrimSpace(os.Getenv("WEBHOOK_USERNAME")),
		WebhookPassword:               strings.TrimSpace(os.Getenv("WEBHOOK_PASSWORD")),
		AllowedUserID:                 strings.TrimSpace(os.Getenv("ALLOWED_USER_ID")),
		AdminUserIDs:                  envCSV("ADMIN_USER_IDS"),
		DefaultLocale:                 defaultLocaleValue,
		DefaultTimezone:               defaultTimezoneValue,
		LocalesDir:                    localesDir,
		MinSeverity:                   minSeverity,
		QuietHoursStart:               strings.TrimSpace(os.Getenv("QUIET_HOURS_START")),
		QuietHoursEnd:                 strings.TrimSpace(os.Getenv("QUIET_HOURS_END")),
		DedupEnabled:                  envBool("DEDUP_ENABLED", true),
		DedupWindowSeconds:            mustEnvInt("DEDUP_WINDOW_SECONDS", defaultDedupWindowSeconds, 1, 86400),
		NotificationGroupWindowSecond: mustEnvInt("NOTIFICATION_GROUP_WINDOW_SECONDS", defaultNotificationGroupWindowSec, 0, 3600),
		EventsRetentionDays:           mustEnvInt("EVENTS_RETENTION_DAYS", defaultEventsRetentionDays, 0, 36500),
		IgnoredIPsByEvent:             buildIgnoredIPsByEvent(),
	}, nil
}

func (c Config) IsAllowedUser(userID string) bool {
	if strings.TrimSpace(c.AllowedUserID) == "" {
		return true
	}
	return strings.TrimSpace(userID) == c.AllowedUserID
}

func (c Config) IsAdminUser(userID string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	if len(c.AdminUserIDs) > 0 {
		for _, adminID := range c.AdminUserIDs {
			if userID == adminID {
				return true
			}
		}
		return false
	}
	return c.AllowedUserID != "" && userID == c.AllowedUserID
}

func envString(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return strings.EqualFold(value, "true") || value == "1" || strings.EqualFold(value, "yes") || strings.EqualFold(value, "on")
}

func envCSV(name string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func envInt(name string, fallback int, min int, max int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	if value < min || value > max {
		return 0, fmt.Errorf("%s must be between %d and %d", name, min, max)
	}
	return value, nil
}

func mustEnvInt(name string, fallback int, min int, max int) int {
	value, err := envInt(name, fallback, min, max)
	if err != nil {
		return fallback
	}
	return value
}

func buildIgnoredIPsByEvent() map[string][]string {
	result := make(map[string][]string)
	for eventType := range events.Registry {
		envName := strings.ToUpper(strings.ReplaceAll(eventType, ".", "_"))
		envName = strings.ReplaceAll(envName, "-", "_") + "_IGNORED_IPS"
		values := splitCSV(os.Getenv(envName))
		if len(values) > 0 {
			result[eventType] = values
		}
	}
	return result
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
