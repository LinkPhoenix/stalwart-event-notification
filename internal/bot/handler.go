package bot

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"

	"stalwart-event-notification/internal/config"
	"stalwart-event-notification/internal/events"
	"stalwart-event-notification/internal/storage"
)

type Translator interface {
	T(locale string, key string, params map[string]string) string
	Resolve(locale string) string
}

type Handler struct {
	store      storage.Store
	messenger  Messenger
	translator Translator
	config     config.Config
}

func NewHandler(store storage.Store, messenger Messenger, translator Translator, cfg config.Config) *Handler {
	return &Handler{store: store, messenger: messenger, translator: translator, config: cfg}
}

func (h *Handler) HandleMessage(ctx context.Context, chatID int64, userID string, username string, input string) error {
	if !h.config.IsAllowedUser(userID) {
		return h.send(ctx, chatID, userID, "access_denied", nil, nil)
	}

	text := strings.TrimSpace(input)
	command := parseCommand(text)
	switch {
	case command == "start":
		return h.handleStart(ctx, chatID, userID)
	case command == "events" || text == menuEvents:
		return h.handleEvents(ctx, chatID, userID)
	case command == "list" || text == menuList:
		return h.handleList(ctx, chatID, userID)
	case command == "subscribe" || text == menuSubscribe || text == menuSubscribeAll:
		return h.handleSubscribe(ctx, chatID, userID, commandArg(text), text == menuSubscribeAll)
	case command == "unsubscribe" || text == menuUnsubscribe || text == menuUnsubscribeAll:
		return h.handleUnsubscribe(ctx, chatID, userID, commandArg(text), text == menuUnsubscribeAll)
	case command == "status" || text == menuStatus:
		return h.handleStatus(ctx, chatID, userID)
	case command == "prefs" || text == menuPrefs:
		return h.handlePrefs(ctx, chatID, userID)
	case command == "lang":
		return h.handleLangCommand(ctx, chatID, userID, commandArg(text))
	case command == "timezone":
		return h.handleTimezoneCommand(ctx, chatID, userID, commandArg(text))
	case command == "short":
		return h.handleShortCommand(ctx, chatID, userID, commandArg(text))
	case command == "help" || text == menuHelp:
		return h.handleHelp(ctx, chatID, userID)
	case command == "stats":
		return h.handleStats(ctx, chatID, userID)
	case command == "users":
		return h.handleUsers(ctx, chatID, userID)
	case command == "events_count":
		return h.handleEventsCount(ctx, chatID, userID, commandArg(text))
	case command == "blocked":
		return h.handleBlocked(ctx, chatID, userID, commandArg(text))
	default:
		return h.send(ctx, chatID, userID, "menu_hint", nil, nil)
	}
}

func (h *Handler) HandleCallback(ctx context.Context, callbackID string, chatID int64, userID string, data string) error {
	if err := h.messenger.AnswerCallback(ctx, callbackID); err != nil {
		return err
	}
	if !h.config.IsAllowedUser(userID) {
		return h.send(ctx, chatID, userID, "access_denied", nil, nil)
	}

	switch {
	case strings.HasPrefix(data, callbackSubscribePrefix):
		eventType := strings.TrimPrefix(data, callbackSubscribePrefix)
		return h.subscribeToEvent(ctx, chatID, userID, eventType)
	case strings.HasPrefix(data, callbackUnsubscribePrefix):
		eventType := strings.TrimPrefix(data, callbackUnsubscribePrefix)
		return h.unsubscribeFromEvent(ctx, chatID, userID, eventType)
	case strings.HasPrefix(data, callbackPrefsPrefix):
		return h.handlePrefsCallback(ctx, chatID, userID, strings.TrimPrefix(data, callbackPrefsPrefix))
	default:
		return nil
	}
}

func (h *Handler) handleStart(ctx context.Context, chatID int64, userID string) error {
	if err := h.ensurePrefs(ctx, userID); err != nil {
		return err
	}
	return h.send(ctx, chatID, userID, "welcome", nil, nil)
}

func (h *Handler) handleEvents(ctx context.Context, chatID int64, userID string) error {
	types := sortedEventTypes()
	lines := []string{h.text(ctx, userID, "events.title", nil), ""}
	for _, eventType := range types {
		lines = append(lines, fmt.Sprintf("• <code>%s</code> · %s", html.EscapeString(eventType), h.translator.T(h.localeForUser(ctx, userID), "event."+eventType+".description", nil)))
	}
	_, err := h.messenger.SendMessage(ctx, chatID, strings.Join(lines, "\n"), nil)
	return err
}

func (h *Handler) handleSubscribe(ctx context.Context, chatID int64, userID string, arg string, allButton bool) error {
	if allButton || arg == "all" {
		count, err := h.store.SubscribeAll(ctx, userID, sortedEventTypes())
		if err != nil {
			return err
		}
		return h.send(ctx, chatID, userID, "subscribe.all_success", map[string]string{"count": fmt.Sprintf("%d", count)}, nil)
	}
	if arg != "" {
		return h.subscribeToEvent(ctx, chatID, userID, arg)
	}
	return h.send(ctx, chatID, userID, "subscribe.prompt", nil, eventInlineKeyboard(callbackSubscribePrefix, sortedEventTypes()))
}

func (h *Handler) subscribeToEvent(ctx context.Context, chatID int64, userID string, eventType string) error {
	if !events.IsSupported(eventType) {
		return h.send(ctx, chatID, userID, "subscribe.unknown", nil, nil)
	}
	added, err := h.store.Subscribe(ctx, userID, eventType)
	if err != nil {
		return err
	}
	key := "subscribe.already"
	if added {
		key = "subscribe.success"
	}
	return h.send(ctx, chatID, userID, key, map[string]string{"event": eventType}, nil)
}

func (h *Handler) handleUnsubscribe(ctx context.Context, chatID int64, userID string, arg string, allButton bool) error {
	if allButton || arg == "all" {
		count, err := h.store.UnsubscribeAll(ctx, userID)
		if err != nil {
			return err
		}
		return h.send(ctx, chatID, userID, "unsubscribe.all_success", map[string]string{"count": fmt.Sprintf("%d", count)}, nil)
	}
	if arg != "" {
		return h.unsubscribeFromEvent(ctx, chatID, userID, arg)
	}
	current, err := h.store.GetSubscriptions(ctx, userID)
	if err != nil {
		return err
	}
	if len(current) == 0 {
		return h.send(ctx, chatID, userID, "list.empty", nil, nil)
	}
	return h.send(ctx, chatID, userID, "unsubscribe.prompt", nil, eventInlineKeyboard(callbackUnsubscribePrefix, current))
}

func (h *Handler) unsubscribeFromEvent(ctx context.Context, chatID int64, userID string, eventType string) error {
	removed, err := h.store.Unsubscribe(ctx, userID, eventType)
	if err != nil {
		return err
	}
	key := "unsubscribe.not_subscribed"
	if removed {
		key = "unsubscribe.success"
	}
	return h.send(ctx, chatID, userID, key, map[string]string{"event": eventType}, nil)
}

func (h *Handler) handleList(ctx context.Context, chatID int64, userID string) error {
	current, err := h.store.GetSubscriptions(ctx, userID)
	if err != nil {
		return err
	}
	if len(current) == 0 {
		return h.send(ctx, chatID, userID, "list.empty", nil, nil)
	}
	lines := []string{h.text(ctx, userID, "list.title", nil), ""}
	for _, eventType := range current {
		lines = append(lines, "• <code>"+html.EscapeString(eventType)+"</code>")
	}
	_, err = h.messenger.SendMessage(ctx, chatID, strings.Join(lines, "\n"), nil)
	return err
}

func (h *Handler) handleStatus(ctx context.Context, chatID int64, userID string) error {
	count, _ := h.store.SubscribersCount(ctx)
	params := map[string]string{
		"port":        fmt.Sprintf("%d", h.config.Port),
		"subscribers": fmt.Sprintf("%d", count),
	}
	return h.send(ctx, chatID, userID, "status", params, nil)
}

func (h *Handler) handlePrefs(ctx context.Context, chatID int64, userID string) error {
	prefs, err := h.prefs(ctx, userID)
	if err != nil {
		return err
	}
	locale := h.localeFromPrefs(prefs)
	timezone := prefs.Timezone
	if timezone == "" {
		timezone = h.config.DefaultTimezone
	}
	params := map[string]string{
		"locale":   locale,
		"timezone": timezone,
		"short":    boolLabel(prefs.ShortNotifications),
	}
	return h.send(ctx, chatID, userID, "prefs.summary", params, prefsInlineKeyboard(locale, h.translator))
}

func (h *Handler) handlePrefsCallback(ctx context.Context, chatID int64, userID string, action string) error {
	prefs, err := h.prefs(ctx, userID)
	if err != nil {
		return err
	}
	locale := h.localeFromPrefs(prefs)
	switch {
	case action == "lang":
		return h.send(ctx, chatID, userID, "prefs.language_prompt", nil, languageInlineKeyboard())
	case action == "timezone":
		return h.send(ctx, chatID, userID, "prefs.timezone_prompt", nil, timezoneInlineKeyboard())
	case action == "short":
		return h.send(ctx, chatID, userID, "prefs.short_prompt", nil, shortInlineKeyboard(locale, prefs.ShortNotifications, h.translator))
	case strings.HasPrefix(action, "lang:"):
		prefs.Locale = strings.TrimPrefix(action, "lang:")
		return h.savePrefsAndConfirm(ctx, chatID, userID, prefs, "prefs.language_saved", map[string]string{"locale": prefs.Locale})
	case strings.HasPrefix(action, "tz:"):
		prefs.Timezone = strings.TrimPrefix(action, "tz:")
		return h.savePrefsAndConfirm(ctx, chatID, userID, prefs, "prefs.timezone_saved", map[string]string{"timezone": prefs.Timezone})
	case action == "short:on" || action == "short:off":
		prefs.ShortNotifications = action == "short:on"
		return h.savePrefsAndConfirm(ctx, chatID, userID, prefs, "prefs.short_saved", map[string]string{"short": boolLabel(prefs.ShortNotifications)})
	default:
		return nil
	}
}

func (h *Handler) handleLangCommand(ctx context.Context, chatID int64, userID string, locale string) error {
	if locale == "" {
		return h.send(ctx, chatID, userID, "prefs.language_prompt", nil, languageInlineKeyboard())
	}
	prefs, err := h.prefs(ctx, userID)
	if err != nil {
		return err
	}
	prefs.Locale = strings.ToLower(locale)
	return h.savePrefsAndConfirm(ctx, chatID, userID, prefs, "prefs.language_saved", map[string]string{"locale": prefs.Locale})
}

func (h *Handler) handleTimezoneCommand(ctx context.Context, chatID int64, userID string, timezone string) error {
	if timezone == "" {
		return h.send(ctx, chatID, userID, "prefs.timezone_prompt", nil, timezoneInlineKeyboard())
	}
	prefs, err := h.prefs(ctx, userID)
	if err != nil {
		return err
	}
	prefs.Timezone = timezone
	return h.savePrefsAndConfirm(ctx, chatID, userID, prefs, "prefs.timezone_saved", map[string]string{"timezone": prefs.Timezone})
}

func (h *Handler) handleShortCommand(ctx context.Context, chatID int64, userID string, arg string) error {
	if arg == "" {
		prefs, err := h.prefs(ctx, userID)
		if err != nil {
			return err
		}
		return h.send(ctx, chatID, userID, "prefs.short_prompt", nil, shortInlineKeyboard(h.localeFromPrefs(prefs), prefs.ShortNotifications, h.translator))
	}
	prefs, err := h.prefs(ctx, userID)
	if err != nil {
		return err
	}
	prefs.ShortNotifications = arg == "on" || arg == "true" || arg == "1" || arg == "yes"
	return h.savePrefsAndConfirm(ctx, chatID, userID, prefs, "prefs.short_saved", map[string]string{"short": boolLabel(prefs.ShortNotifications)})
}

func (h *Handler) handleHelp(ctx context.Context, chatID int64, userID string) error {
	return h.send(ctx, chatID, userID, "help", nil, nil)
}

func (h *Handler) handleStats(ctx context.Context, chatID int64, userID string) error {
	if !h.config.IsAdminUser(userID) {
		return h.send(ctx, chatID, userID, "access_denied", nil, nil)
	}
	subscribers, _ := h.store.SubscribersCount(ctx)
	eventsCount, _ := h.store.EventsCount(ctx, 0)
	events24h, _ := h.store.EventsCount(ctx, 1)
	params := map[string]string{
		"subscribers": fmt.Sprintf("%d", subscribers),
		"events":      fmt.Sprintf("%d", eventsCount),
		"events_24h":  fmt.Sprintf("%d", events24h),
	}
	return h.send(ctx, chatID, userID, "admin.stats", params, nil)
}

func (h *Handler) handleUsers(ctx context.Context, chatID int64, userID string) error {
	if !h.config.IsAdminUser(userID) {
		return h.send(ctx, chatID, userID, "access_denied", nil, nil)
	}
	rows, err := h.store.UsersWithSubscriptionCount(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return h.send(ctx, chatID, userID, "admin.no_users", nil, nil)
	}
	lines := []string{h.text(ctx, userID, "admin.users_title", nil), ""}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("• <code>%s</code> · %d", html.EscapeString(row.UserID), row.Count))
	}
	_, err = h.messenger.SendMessage(ctx, chatID, strings.Join(lines, "\n"), nil)
	return err
}

func (h *Handler) handleEventsCount(ctx context.Context, chatID int64, userID string, arg string) error {
	if !h.config.IsAdminUser(userID) {
		return h.send(ctx, chatID, userID, "access_denied", nil, nil)
	}
	days := 7
	if arg != "" {
		if parsed, err := strconv.Atoi(arg); err == nil && parsed >= 0 {
			days = parsed
		}
	}
	counts, err := h.store.EventsCountByType(ctx, days)
	if err != nil {
		return err
	}
	lines := []string{h.text(ctx, userID, "admin.events_count_title", map[string]string{"days": fmt.Sprintf("%d", days)}), ""}
	for _, eventType := range sortedEventTypes() {
		if count, ok := counts[eventType]; ok {
			lines = append(lines, fmt.Sprintf("• <code>%s</code> · %d", html.EscapeString(eventType), count))
		}
	}
	_, err = h.messenger.SendMessage(ctx, chatID, strings.Join(lines, "\n"), nil)
	return err
}

func (h *Handler) handleBlocked(ctx context.Context, chatID int64, userID string, arg string) error {
	if !h.config.IsAdminUser(userID) {
		return h.send(ctx, chatID, userID, "access_denied", nil, nil)
	}
	limit := 50
	if arg != "" {
		if parsed, err := strconv.Atoi(arg); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	rows, err := h.store.BlockedIPs(ctx, limit)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return h.send(ctx, chatID, userID, "admin.no_blocked", nil, nil)
	}
	lines := []string{h.text(ctx, userID, "admin.blocked_title", nil), ""}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("• <a href=\"https://www.abuseipdb.com/check/%s\">%s</a> · <code>%s</code>", html.EscapeString(row.IP), html.EscapeString(row.IP), html.EscapeString(row.EventID)))
	}
	_, err = h.messenger.SendMessage(ctx, chatID, strings.Join(lines, "\n"), nil)
	return err
}

func (h *Handler) ensurePrefs(ctx context.Context, userID string) error {
	prefs, err := h.store.GetPrefs(ctx, userID)
	if err != nil {
		return err
	}
	if prefs.Locale == "" {
		prefs.Locale = h.config.DefaultLocale
	}
	if prefs.Timezone == "" {
		prefs.Timezone = h.config.DefaultTimezone
	}
	return h.store.SetPrefs(ctx, userID, prefs)
}

func (h *Handler) savePrefsAndConfirm(ctx context.Context, chatID int64, userID string, prefs storage.UserPrefs, key string, params map[string]string) error {
	if err := h.store.SetPrefs(ctx, userID, prefs); err != nil {
		return err
	}
	return h.send(ctx, chatID, userID, key, params, nil)
}

func (h *Handler) prefs(ctx context.Context, userID string) (storage.UserPrefs, error) {
	prefs, err := h.store.GetPrefs(ctx, userID)
	if err != nil {
		return storage.UserPrefs{}, err
	}
	if prefs.Locale == "" {
		prefs.Locale = h.config.DefaultLocale
	}
	if prefs.Timezone == "" {
		prefs.Timezone = h.config.DefaultTimezone
	}
	return prefs, nil
}

func (h *Handler) localeForUser(ctx context.Context, userID string) string {
	prefs, err := h.prefs(ctx, userID)
	if err != nil {
		return h.config.DefaultLocale
	}
	return h.localeFromPrefs(prefs)
}

func (h *Handler) localeFromPrefs(prefs storage.UserPrefs) string {
	if prefs.Locale == "" {
		return h.config.DefaultLocale
	}
	return h.translator.Resolve(prefs.Locale)
}

func (h *Handler) text(ctx context.Context, userID string, key string, params map[string]string) string {
	return h.translator.T(h.localeForUser(ctx, userID), key, params)
}

func (h *Handler) send(ctx context.Context, chatID int64, userID string, key string, params map[string]string, keyboard interface{}) error {
	_, err := h.messenger.SendMessage(ctx, chatID, h.text(ctx, userID, key, params), keyboard)
	return err
}

func sortedEventTypes() []string {
	types := events.SupportedTypes()
	sort.Strings(types)
	return types
}

func parseCommand(text string) string {
	if !strings.HasPrefix(text, "/") {
		return ""
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	command := strings.TrimPrefix(fields[0], "/")
	if at := strings.IndexByte(command, '@'); at >= 0 {
		command = command[:at]
	}
	return strings.ToLower(strings.TrimSpace(command))
}

func commandArg(text string) string {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return ""
	}
	return strings.TrimSpace(fields[1])
}

func boolLabel(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}
