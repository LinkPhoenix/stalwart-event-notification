package storage

import (
	"context"
	"time"

	"stalwart-event-notification/internal/webhook"
)

type UserPrefs struct {
	Locale             string
	Timezone           string
	ShortNotifications bool
}

type UserSubscriptionCount struct {
	UserID string
	Count  int64
}

type BlockedIP struct {
	IP        string
	EventID   string
	CreatedAt time.Time
}

type Store interface {
	Ping(ctx context.Context) error
	EnsureSchema(ctx context.Context) error
	Close() error

	Subscribe(ctx context.Context, userID string, eventType string) (bool, error)
	SubscribeAll(ctx context.Context, userID string, eventTypes []string) (int64, error)
	Unsubscribe(ctx context.Context, userID string, eventType string) (bool, error)
	UnsubscribeAll(ctx context.Context, userID string) (int64, error)
	GetSubscriptions(ctx context.Context, userID string) ([]string, error)
	GetSubscribers(ctx context.Context, eventType string) ([]string, error)

	GetPrefs(ctx context.Context, userID string) (UserPrefs, error)
	SetPrefs(ctx context.Context, userID string, prefs UserPrefs) error

	StoreEvent(ctx context.Context, event webhook.Event) error
	StoreBlockedIP(ctx context.Context, ip string, eventID string) error
	SyncIgnoredIPs(ctx context.Context, ignored map[string][]string) error
	PurgeOldEvents(ctx context.Context, retentionDays int) (int64, error)

	SubscribersCount(ctx context.Context) (int64, error)
	EventsCount(ctx context.Context, days int) (int64, error)
	EventsCountByType(ctx context.Context, days int) (map[string]int64, error)
	UsersWithSubscriptionCount(ctx context.Context) ([]UserSubscriptionCount, error)
	BlockedIPs(ctx context.Context, limit int) ([]BlockedIP, error)
}
