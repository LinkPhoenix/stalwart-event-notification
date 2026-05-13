package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"stalwart-event-notification/internal/config"
	"stalwart-event-notification/internal/storage"
	"stalwart-event-notification/internal/webhook"
)

func TestHandlerRejectsUnauthorizedUser(t *testing.T) {
	store := newFakeBotStore()
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{AllowedUserID: "1", DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleMessage(context.Background(), 99, "2", "", "/start"); err != nil {
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

	if err := handler.HandleMessage(context.Background(), 99, "123", "", "/subscribe all"); err != nil {
		t.Fatalf("handle subscribe all: %v", err)
	}
	if store.subscribeAllCount == 0 {
		t.Fatal("expected subscribe all to be called")
	}
	if !strings.Contains(messenger.lastText, "subscribe.all_success") {
		t.Fatalf("message = %q", messenger.lastText)
	}
}

func TestHandlerShortPreferenceCanTurnOff(t *testing.T) {
	store := newFakeBotStore()
	store.prefs["123"] = storage.UserPrefs{Locale: "en", Timezone: "UTC", ShortNotifications: true}
	messenger := &fakeMessenger{}
	handler := NewHandler(store, messenger, fakeTranslator{}, config.Config{DefaultLocale: "en", DefaultTimezone: "UTC"})

	if err := handler.HandleCallback(context.Background(), "cb", 99, "123", "prefs:short:off"); err != nil {
		t.Fatalf("handle callback: %v", err)
	}
	if store.prefs["123"].ShortNotifications {
		t.Fatal("expected short notifications to be off")
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
	return locale
}

type fakeMessenger struct {
	lastText string
}

func (f *fakeMessenger) SendMessage(ctx context.Context, chatID int64, text string, keyboard interface{}) (int, error) {
	f.lastText = text
	return 1, nil
}

func (f *fakeMessenger) AnswerCallback(ctx context.Context, callbackID string) error {
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
