package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	telegrammodels "github.com/go-telegram/bot/models"

	"stalwart-event-notification/internal/config"
	"stalwart-event-notification/internal/storage"
	"stalwart-event-notification/internal/webhook"
)

func TestHandlerRejectsUnauthorizedUser(t *testing.T) {
	store := newFakeBotStore()
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{AllowedUserID: "1", DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "2", "", "/start", "fr-FR"); err != nil {
		t.Fatalf("handle message: %v", err)
	}
	if !strings.Contains(messenger.lastText, "access_denied") {
		t.Fatalf("message = %q", messenger.lastText)
	}
}

func TestHandlerSubscribeAll(t *testing.T) {
	store := newFakeBotStore()
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "123", "", "/subscribe all", "en"); err != nil {
		t.Fatalf("handle subscribe all: %v", err)
	}
	if store.subscribeAllCount == 0 {
		t.Fatal("expected subscribe all to be called")
	}
	if !strings.Contains(messenger.lastText, "subscribe.all_success") {
		t.Fatalf("message = %q", messenger.lastText)
	}
	if !messenger.lastSilent {
		t.Fatal("expected handler response to be silent")
	}
}

func TestMainMenuButtonStyles(t *testing.T) {
	keyboard := mainMenuKeyboard()
	var foundLanguage bool
	styles := make(map[string]string)
	for _, row := range keyboard.Keyboard {
		for _, button := range row {
			styles[button.Text] = button.Style
			if button.Text == "🌐 Language" {
				foundLanguage = true
			}
		}
	}

	if foundLanguage {
		t.Fatal("language button should not be in main menu")
	}
	if styles[menuSubscribe] != buttonStyleSuccess {
		t.Fatalf("subscribe style = %q", styles[menuSubscribe])
	}
	if styles[menuSubscribeAll] != buttonStyleSuccess {
		t.Fatalf("subscribe all style = %q", styles[menuSubscribeAll])
	}
	if styles[menuUnsubscribe] != buttonStyleDanger {
		t.Fatalf("unsubscribe style = %q", styles[menuUnsubscribe])
	}
	if styles[menuUnsubscribeAll] != buttonStyleDanger {
		t.Fatalf("unsubscribe all style = %q", styles[menuUnsubscribeAll])
	}
	if styles[menuPrefs] != buttonStylePrimary {
		t.Fatalf("preferences style = %q", styles[menuPrefs])
	}
}

func TestHandlerShortPreferenceCanTurnOff(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC", ShortNotifications: true}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 10, "123", "prefs:short:off", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if store.prefs["123"].ShortNotifications {
		t.Fatal("expected short notifications to be off")
	}
}

func TestEmojiLanguageButtonShowsFlagOnlyKeyboard(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "fr", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "123", "", "/lang", "fr-FR"); err != nil {
		t.Fatalf("handle language menu: %v", err)
	}

	keyboard, ok := messenger.lastKeyboard.(*telegrammodels.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("keyboard = %T", messenger.lastKeyboard)
	}
	if len(keyboard.InlineKeyboard) != 5 {
		t.Fatalf("rows = %d, want 5", len(keyboard.InlineKeyboard))
	}
	wantFlags := []string{"🇩🇪", "🇬🇧", "🇪🇸", "🇫🇷", "🇮🇹", "🇵🇹", "🇷🇺", "🇺🇦"}
	var gotFlags []string
	for _, row := range keyboard.InlineKeyboard[:4] {
		if len(row) > 2 {
			t.Fatalf("language row has %d buttons", len(row))
		}
		for _, button := range row {
			gotFlags = append(gotFlags, button.Text)
			if button.Text == "🇫🇷" && button.Style != "success" {
				t.Fatalf("selected button style = %q", button.Style)
			}
		}
	}
	if strings.Join(gotFlags, ",") != strings.Join(wantFlags, ",") {
		t.Fatalf("flags = %v", gotFlags)
	}
	nav := keyboard.InlineKeyboard[4]
	if len(nav) != 2 {
		t.Fatalf("navigation row has %d buttons", len(nav))
	}
	back := nav[0]
	if back.Text != "↩️" || back.CallbackData != "prefs:back" || back.Style != buttonStylePrimary {
		t.Fatalf("back button = %#v", back)
	}
	exit := nav[1]
	if exit.Text != "✖️" || exit.CallbackData != "prefs:exit" || exit.Style != buttonStylePrimary {
		t.Fatalf("exit button = %#v", exit)
	}
}

func TestTelegramLanguageInitializesPrefsOnce(t *testing.T) {
	store := newFakeBotStore()
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "123", "", "/start", "pt-BR"); err != nil {
		t.Fatalf("handle start: %v", err)
	}
	if got := store.prefs["123"].Locale; got != "pt" {
		t.Fatalf("locale = %q, want pt", got)
	}
	if err := handler.HandleMessage(context.Background(), 99, "123", "", "/start", "ru-RU"); err != nil {
		t.Fatalf("handle second start: %v", err)
	}
	if got := store.prefs["123"].Locale; got != "pt" {
		t.Fatalf("locale was overwritten: %q", got)
	}
}

func TestUnsupportedTelegramLanguageFallsBackToDefault(t *testing.T) {
	store := newFakeBotStore()
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "123", "", "/start", "ja-JP"); err != nil {
		t.Fatalf("handle start: %v", err)
	}
	if got := store.prefs["123"].Locale; got != "en" {
		t.Fatalf("locale = %q, want en", got)
	}
}

func TestLanguageCallbackSavesAndEditsPreferencesSummary(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 77, "123", "prefs:lang:uk", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if got := store.prefs["123"].Locale; got != "uk" {
		t.Fatalf("locale = %q, want uk", got)
	}
	if messenger.editedMessageID != 77 {
		t.Fatalf("edited message = %d, want 77", messenger.editedMessageID)
	}
	if !strings.Contains(messenger.editedText, "prefs.summary") {
		t.Fatalf("edited text = %q", messenger.editedText)
	}
}

func TestPrefsMainButtonsHaveNoStyle(t *testing.T) {
	keyboard := prefsInlineKeyboard("en", fakeTranslator{})
	for i, row := range keyboard.InlineKeyboard[:3] {
		button := row[0]
		if button.Style != "" {
			t.Fatalf("button %d style = %q", i, button.Style)
		}
	}
	exit := keyboard.InlineKeyboard[3][0]
	if exit.Style != buttonStylePrimary {
		t.Fatalf("exit style = %q", exit.Style)
	}
}

func TestPrefsSubmenuUsesEditedMessage(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 55, "123", "prefs:short", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if messenger.editedMessageID != 55 {
		t.Fatalf("edited message = %d, want 55", messenger.editedMessageID)
	}
	if !strings.Contains(messenger.editedText, "prefs.short_prompt") {
		t.Fatalf("edited text = %q", messenger.editedText)
	}
	keyboard, ok := messenger.editedKeyboard.(*telegrammodels.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("keyboard = %T", messenger.editedKeyboard)
	}
	if keyboard.InlineKeyboard[0][1].Style != buttonStyleSuccess {
		t.Fatalf("off button style = %q", keyboard.InlineKeyboard[0][1].Style)
	}
	if strings.Contains(keyboard.InlineKeyboard[0][1].Text, "✓") {
		t.Fatalf("off button text = %q", keyboard.InlineKeyboard[0][1].Text)
	}
	nav := keyboard.InlineKeyboard[1]
	if len(nav) != 2 {
		t.Fatalf("navigation row has %d buttons", len(nav))
	}
	if nav[0].Text != "↩️" || nav[1].Text != "✖️" {
		t.Fatalf("navigation row = %#v", nav)
	}
}

func TestBackCallbackEditsPreferencesSummary(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 55, "123", "prefs:back", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if messenger.editedMessageID != 55 {
		t.Fatalf("edited message = %d, want 55", messenger.editedMessageID)
	}
	if !strings.Contains(messenger.editedText, "prefs.summary") {
		t.Fatalf("edited text = %q", messenger.editedText)
	}
}

func TestTimezoneCallbackSavesValidTimezone(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 55, "123", "prefs:tz:Europe/Paris", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if got := store.prefs["123"].Timezone; got != "Europe/Paris" {
		t.Fatalf("timezone = %q, want Europe/Paris", got)
	}
	if !strings.Contains(messenger.editedText, "prefs.summary") {
		t.Fatalf("edited text = %q", messenger.editedText)
	}
}

func TestTimezoneSubmenuMarksCurrentTimezone(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "Europe/Paris"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 55, "123", "prefs:timezone", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	keyboard, ok := messenger.editedKeyboard.(*telegrammodels.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("keyboard = %T", messenger.editedKeyboard)
	}
	var found bool
	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			if button.Text == "Europe/Paris" {
				found = true
				if button.Style != buttonStyleSuccess {
					t.Fatalf("timezone style = %q", button.Style)
				}
			}
		}
	}
	if !found {
		t.Fatal("Europe/Paris button not found")
	}
}

func TestTimezoneCommandRejectsInvalidTimezone(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "123", "", "/timezone Mars/Base", "en"); err != nil {
		t.Fatalf("handle message: %v", err)
	}
	if store.prefs["123"].Timezone != "UTC" {
		t.Fatalf("timezone changed to %q", store.prefs["123"].Timezone)
	}
	if !strings.Contains(messenger.lastText, "prefs.timezone_invalid") {
		t.Fatalf("message = %q", messenger.lastText)
	}
	if !messenger.lastSilent {
		t.Fatal("expected invalid timezone response to be silent")
	}
}

func TestTimezoneCallbackRejectsInvalidTimezone(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 55, "123", "prefs:tz:Mars/Base", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if store.prefs["123"].Timezone != "UTC" {
		t.Fatalf("timezone changed to %q", store.prefs["123"].Timezone)
	}
	if !strings.Contains(messenger.lastText, "prefs.timezone_invalid") {
		t.Fatalf("message = %q", messenger.lastText)
	}
}

func TestHelpMenuSendsHelpMessage(t *testing.T) {
	store := newFakeBotStore()
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "123", "", menuHelp, "en"); err != nil {
		t.Fatalf("handle help: %v", err)
	}
	if !strings.Contains(messenger.lastText, "help") {
		t.Fatalf("message = %q", messenger.lastText)
	}
	if !messenger.lastSilent {
		t.Fatal("expected help response to be silent")
	}
}

func TestExitCallbackDeletesMenuAndSendsHint(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC"}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, 55, "123", "prefs:exit", "en"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if messenger.deletedMessageID != 55 {
		t.Fatalf("deleted message = %d, want 55", messenger.deletedMessageID)
	}
	if !strings.Contains(messenger.lastText, "exit.keyboard_hint") {
		t.Fatalf("hint = %q", messenger.lastText)
	}
	if !messenger.lastSilent {
		t.Fatal("expected exit hint to be silent")
	}
}

type fakeTranslator struct{}

func (fakeTranslator) T(locale string, key string, params map[string]string) string {
	text := key
	for name, value := range params {
		text += " " + name + "=" + value
	}
	return text
}

func (fakeTranslator) Resolve(locale string) string {
	if locale == "" {
		return "en"
	}
	locale = normalizeLocale(locale)
	if !isSupportedLocale(locale) {
		return "en"
	}
	return locale
}

type fakeMessenger struct {
	lastText         string
	lastKeyboard     interface{}
	lastSilent       bool
	editedText       string
	editedKeyboard   interface{}
	editedMessageID  int
	deletedMessageID int
}

func (f *fakeMessenger) SendMessage(ctx context.Context, chatID int64, text string, keyboard interface{}) (int, error) {
	f.lastText = text
	f.lastKeyboard = keyboard
	f.lastSilent = false
	return 1, nil
}

func (f *fakeMessenger) SendSilentMessage(ctx context.Context, chatID int64, text string, keyboard interface{}) (int, error) {
	f.lastText = text
	f.lastKeyboard = keyboard
	f.lastSilent = true
	return 1, nil
}

func (f *fakeMessenger) EditMessageText(ctx context.Context, chatID int64, messageID int, text string, keyboard interface{}) error {
	f.editedMessageID = messageID
	f.editedText = text
	f.editedKeyboard = keyboard
	return nil
}

func (f *fakeMessenger) AnswerCallback(ctx context.Context, callbackID string) error {
	return nil
}

func (f *fakeMessenger) DeleteMessage(ctx context.Context, chatID int64, messageID int) error {
	f.deletedMessageID = messageID
	return nil
}

func (f *fakeMessenger) ClearInlineKeyboard(ctx context.Context, chatID int64, messageID int) error {
	return nil
}

type fakeBotStore struct {
	prefs             map[string]storage.UserPrefs
	subscribeAllCount int
}

func newFakeBotStore() *fakeBotStore {
	return &fakeBotStore{prefs: make(map[string]storage.UserPrefs)}
}

func (f *fakeBotStore) Ping(ctx context.Context) error         { return nil }
func (f *fakeBotStore) EnsureSchema(ctx context.Context) error { return nil }
func (f *fakeBotStore) Close() error                           { return nil }
func (f *fakeBotStore) Subscribe(ctx context.Context, userID string, eventType string) (bool, error) {
	return true, nil
}
func (f *fakeBotStore) SubscribeAll(ctx context.Context, userID string, eventTypes []string) (int64, error) {
	f.subscribeAllCount++
	return int64(len(eventTypes)), nil
}
func (f *fakeBotStore) Unsubscribe(ctx context.Context, userID string, eventType string) (bool, error) {
	return true, nil
}
func (f *fakeBotStore) UnsubscribeAll(ctx context.Context, userID string) (int64, error) {
	return 1, nil
}
func (f *fakeBotStore) GetSubscriptions(ctx context.Context, userID string) ([]string, error) {
	return []string{"auth.failed"}, nil
}
func (f *fakeBotStore) GetSubscribers(ctx context.Context, eventType string) ([]string, error) {
	return []string{"123"}, nil
}
func (f *fakeBotStore) GetPrefs(ctx context.Context, userID string) (storage.UserPrefs, error) {
	return f.prefs[userID], nil
}
func (f *fakeBotStore) SetPrefs(ctx context.Context, userID string, prefs storage.UserPrefs) error {
	f.prefs[userID] = prefs
	return nil
}
func (f *fakeBotStore) StoreEvent(ctx context.Context, event webhook.Event) error { return nil }
func (f *fakeBotStore) StoreBlockedIP(ctx context.Context, ip string, eventID string) error {
	return nil
}
func (f *fakeBotStore) SyncIgnoredIPs(ctx context.Context, ignored map[string][]string) error {
	return nil
}
func (f *fakeBotStore) PurgeOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	return 0, nil
}
func (f *fakeBotStore) SubscribersCount(ctx context.Context) (int64, error)      { return 1, nil }
func (f *fakeBotStore) EventsCount(ctx context.Context, days int) (int64, error) { return 1, nil }
func (f *fakeBotStore) EventsCountByType(ctx context.Context, days int) (map[string]int64, error) {
	return map[string]int64{"auth.failed": 1}, nil
}
func (f *fakeBotStore) UsersWithSubscriptionCount(ctx context.Context) ([]storage.UserSubscriptionCount, error) {
	return []storage.UserSubscriptionCount{{UserID: "123", Count: 1}}, nil
}
func (f *fakeBotStore) BlockedIPs(ctx context.Context, limit int) ([]storage.BlockedIP, error) {
	return []storage.BlockedIP{{IP: "1.1.1.1", EventID: "1", CreatedAt: time.Now()}}, nil
}
