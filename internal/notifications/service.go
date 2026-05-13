package notifications

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"stalwart-event-notification/internal/config"
	"stalwart-event-notification/internal/events"
	"stalwart-event-notification/internal/i18n"
	"stalwart-event-notification/internal/storage"
	"stalwart-event-notification/internal/webhook"
)

type Messenger interface {
	SendMessage(ctx context.Context, chatID int64, text string, keyboard interface{}) (int, error)
}

type Service struct {
	store        storage.Store
	messenger    Messenger
	translations *i18n.Bundle
	config       config.Config
	dedup        *Deduplicator
	metrics      *Metrics
	logger       *slog.Logger

	mu      sync.Mutex
	buffers map[string]*groupBuffer
}

type groupBuffer struct {
	events []webhook.Event
	users  map[string]struct{}
	timer  *time.Timer
}

func NewService(
	store storage.Store,
	messenger Messenger,
	translations *i18n.Bundle,
	cfg config.Config,
	metrics *Metrics,
	logger *slog.Logger,
) *Service {
	return &Service{
		store:        store,
		messenger:    messenger,
		translations: translations,
		config:       cfg,
		dedup:        NewDeduplicator(cfg.DedupEnabled, cfg.DedupWindowSeconds),
		metrics:      metrics,
		logger:       logger,
		buffers:      make(map[string]*groupBuffer),
	}
}

func (s *Service) ProcessEvent(ctx context.Context, event webhook.Event) error {
	if !events.IsSupported(event.Type) {
		return nil
	}
	if !s.shouldNotify(event) {
		s.metrics.Inc("events_skipped", 1)
		return nil
	}

	if err := s.store.StoreEvent(ctx, event); err != nil {
		s.logger.Error("store event failed", "error", err, "event_type", event.Type)
	}
	if event.Type == "security.ip-blocked" {
		if ip := webhook.SourceIP(event); ip != "" {
			if err := s.store.StoreBlockedIP(ctx, ip, event.ID); err != nil {
				s.logger.Error("store blocked ip failed", "error", err)
			}
		}
	}

	userIDs, err := s.store.GetSubscribers(ctx, event.Type)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		s.metrics.Inc("events_skipped", 1)
		return nil
	}

	if s.config.NotificationGroupWindowSecond > 0 {
		s.enqueueGrouped(ctx, event, userIDs)
		return nil
	}

	for _, userID := range userIDs {
		if err := s.sendEvent(ctx, userID, event); err != nil {
			s.logger.Error("send notification failed", "error", err, "user_id", userID)
			s.metrics.Inc("notifications_failed", 1)
		} else {
			s.metrics.Inc("notifications_sent", 1)
		}
	}
	return nil
}

func (s *Service) shouldNotify(event webhook.Event) bool {
	ip := webhook.SourceIP(event)
	if ignored := s.config.IgnoredIPsByEvent[event.Type]; ip != "" && contains(ignored, ip) {
		return false
	}
	if !PassesSeverity(event.Type, s.config.MinSeverity) {
		return false
	}
	if IsInQuietHours(time.Now(), s.config.QuietHoursStart, s.config.QuietHoursEnd) {
		return false
	}
	return s.dedup.ShouldNotify(event)
}

func (s *Service) sendEvent(ctx context.Context, userID string, event webhook.Event) error {
	prefs, err := s.store.GetPrefs(ctx, userID)
	if err != nil {
		return err
	}
	locale := prefs.Locale
	if strings.TrimSpace(locale) == "" {
		locale = s.config.DefaultLocale
	}
	timezone := prefs.Timezone
	if strings.TrimSpace(timezone) == "" {
		timezone = s.config.DefaultTimezone
	}
	text := FormatEventMessage(s.translations, event, locale, timezone, prefs.ShortNotifications)
	chatID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	_, err = s.messenger.SendMessage(ctx, chatID, text, nil)
	return err
}

func (s *Service) enqueueGrouped(ctx context.Context, event webhook.Event, userIDs []string) {
	key := event.Type
	window := time.Duration(s.config.NotificationGroupWindowSecond) * time.Second

	s.mu.Lock()
	buffer, ok := s.buffers[key]
	if !ok {
		buffer = &groupBuffer{users: make(map[string]struct{})}
		s.buffers[key] = buffer
		buffer.timer = time.AfterFunc(window, func() {
			s.flushGrouped(context.Background(), key)
		})
	}
	buffer.events = append(buffer.events, event)
	for _, userID := range userIDs {
		buffer.users[userID] = struct{}{}
	}
	s.mu.Unlock()

	_ = ctx
}

func (s *Service) flushGrouped(ctx context.Context, key string) {
	s.mu.Lock()
	buffer := s.buffers[key]
	delete(s.buffers, key)
	s.mu.Unlock()
	if buffer == nil || len(buffer.events) == 0 {
		return
	}

	for userID := range buffer.users {
		if err := s.sendGrouped(ctx, userID, buffer.events); err != nil {
			s.logger.Error("send grouped notification failed", "error", err, "user_id", userID)
			s.metrics.Inc("notifications_failed", 1)
		} else {
			s.metrics.Inc("notifications_sent", 1)
		}
	}
}

func (s *Service) sendGrouped(ctx context.Context, userID string, eventItems []webhook.Event) error {
	prefs, err := s.store.GetPrefs(ctx, userID)
	if err != nil {
		return err
	}
	locale := prefs.Locale
	if strings.TrimSpace(locale) == "" {
		locale = s.config.DefaultLocale
	}
	timezone := prefs.Timezone
	if strings.TrimSpace(timezone) == "" {
		timezone = s.config.DefaultTimezone
	}
	text := FormatGroupedMessage(s.translations, eventItems, locale, timezone)
	chatID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return err
	}
	_, err = s.messenger.SendMessage(ctx, chatID, text, nil)
	return err
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}
