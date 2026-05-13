package storage

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"stalwart-event-notification/internal/webhook"
)

func TestSubscribeAndGetSubscriptions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO subscriptions (user_id, event_type)
VALUES ($1, $2)
ON CONFLICT (user_id, event_type) DO NOTHING`)).
		WithArgs("123", "auth.failed").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT event_type FROM subscriptions WHERE user_id = $1 ORDER BY event_type`)).
		WithArgs("123").
		WillReturnRows(sqlmock.NewRows([]string{"event_type"}).AddRow("auth.failed"))

	added, err := store.Subscribe(context.Background(), "123", "auth.failed")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if !added {
		t.Fatal("expected subscription to be added")
	}
	list, err := store.GetSubscriptions(context.Background(), "123")
	if err != nil {
		t.Fatalf("get subscriptions: %v", err)
	}
	if len(list) != 1 || list[0] != "auth.failed" {
		t.Fatalf("list = %#v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreEventAndBlockedIP(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	event := webhook.Event{
		ID:        "event-1",
		Type:      "security.ip-blocked",
		CreatedAt: time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC),
		Data:      map[string]interface{}{"remoteIp": "1.2.3.4"},
	}

	mock.ExpectExec("INSERT INTO events").
		WithArgs("event-1", "security.ip-blocked", event.CreatedAt, sqlmock.AnyArg(), "1.2.3.4").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO blocked_ips").
		WithArgs("1.2.3.4", "event-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.StoreEvent(context.Background(), event); err != nil {
		t.Fatalf("store event: %v", err)
	}
	if err := store.StoreBlockedIP(context.Background(), "1.2.3.4", "event-1"); err != nil {
		t.Fatalf("store blocked ip: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetPrefsCanDisableShortNotifications(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := &PostgresStore{db: db}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(locale, ''), COALESCE(timezone, ''), short_notifications FROM user_preferences WHERE user_id = $1`)).
		WithArgs("123").
		WillReturnRows(sqlmock.NewRows([]string{"locale", "timezone", "short_notifications"}).AddRow("en", "UTC", true))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO user_preferences (user_id, locale, timezone, short_notifications, updated_at)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, NOW())
ON CONFLICT (user_id)
DO UPDATE SET
    locale = EXCLUDED.locale,
    timezone = EXCLUDED.timezone,
    short_notifications = EXCLUDED.short_notifications,
    updated_at = NOW()`)).
		WithArgs("123", "en", "UTC", false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.SetPrefs(context.Background(), "123", UserPrefs{Locale: "en", Timezone: "UTC", ShortNotifications: false}); err != nil {
		t.Fatalf("set prefs: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
