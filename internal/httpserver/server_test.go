package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stalwart-event-notification/internal/config"
	"stalwart-event-notification/internal/notifications"
	"stalwart-event-notification/internal/storage"
	"stalwart-event-notification/internal/webhook"
)

func TestEndpoints(t *testing.T) {
	store := &fakeStore{}
	notifier := &fakeNotifier{}
	metrics := notifications.NewMetrics()
	server := New(config.Config{}, store, notifier, metrics, slog.Default(), func(context.Context) bool { return true })
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("GET / = %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"database":true`) {
		t.Fatalf("GET /health = %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "webhook_requests_total") {
		t.Fatalf("GET /metrics = %d %s", res.Code, res.Body.String())
	}
}

func TestWebhookResponses(t *testing.T) {
	store := &fakeStore{}
	notifier := &fakeNotifier{}
	metrics := notifications.NewMetrics()
	cfg := config.Config{WebhookUsername: "u", WebhookPassword: "p"}
	server := New(cfg, store, notifier, metrics, slog.Default(), nil)
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"events":[]}`))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`bad`))
	req.SetBasicAuth("u", "p")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"events":[{"id":"1","createdAt":"2026-05-13T12:00:00Z","type":"auth.failed","data":{"remoteIp":"1.1.1.1"}}]}`))
	req.SetBasicAuth("u", "p")
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if notifier.count != 1 {
		t.Fatalf("notifier count = %d", notifier.count)
	}
}

type fakeNotifier struct {
	count int
}

func (f *fakeNotifier) ProcessEvent(ctx context.Context, event webhook.Event) error {
	f.count++
	return nil
}

type fakeStore struct{}

func (f *fakeStore) Ping(ctx context.Context) error         { return nil }
func (f *fakeStore) EnsureSchema(ctx context.Context) error { return nil }
func (f *fakeStore) Close() error                           { return nil }
func (f *fakeStore) Subscribe(ctx context.Context, userID string, eventType string) (bool, error) {
	return true, nil
}
func (f *fakeStore) SubscribeAll(ctx context.Context, userID string, eventTypes []string) (int64, error) {
	return int64(len(eventTypes)), nil
}
func (f *fakeStore) Unsubscribe(ctx context.Context, userID string, eventType string) (bool, error) {
	return true, nil
}
func (f *fakeStore) UnsubscribeAll(ctx context.Context, userID string) (int64, error) { return 1, nil }
func (f *fakeStore) GetSubscriptions(ctx context.Context, userID string) ([]string, error) {
	return []string{"auth.failed"}, nil
}
func (f *fakeStore) GetSubscribers(ctx context.Context, eventType string) ([]string, error) {
	return []string{"123"}, nil
}
func (f *fakeStore) GetPrefs(ctx context.Context, userID string) (storage.UserPrefs, error) {
	return storage.UserPrefs{}, nil
}
func (f *fakeStore) SetPrefs(ctx context.Context, userID string, prefs storage.UserPrefs) error {
	return nil
}
func (f *fakeStore) StoreEvent(ctx context.Context, event webhook.Event) error           { return nil }
func (f *fakeStore) StoreBlockedIP(ctx context.Context, ip string, eventID string) error { return nil }
func (f *fakeStore) SyncIgnoredIPs(ctx context.Context, ignored map[string][]string) error {
	return nil
}
func (f *fakeStore) PurgeOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	return 0, nil
}
func (f *fakeStore) SubscribersCount(ctx context.Context) (int64, error)      { return 1, nil }
func (f *fakeStore) EventsCount(ctx context.Context, days int) (int64, error) { return 1, nil }
func (f *fakeStore) EventsCountByType(ctx context.Context, days int) (map[string]int64, error) {
	return map[string]int64{"auth.failed": 1}, nil
}
func (f *fakeStore) UsersWithSubscriptionCount(ctx context.Context) ([]storage.UserSubscriptionCount, error) {
	return []storage.UserSubscriptionCount{{UserID: "123", Count: 1}}, nil
}
func (f *fakeStore) BlockedIPs(ctx context.Context, limit int) ([]storage.BlockedIP, error) {
	return []storage.BlockedIP{{IP: "1.1.1.1", EventID: "1", CreatedAt: time.Now()}}, nil
}
