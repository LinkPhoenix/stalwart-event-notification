package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"stalwart-event-notification/internal/webhook"
)

type PostgresStore struct {
	db *sql.DB
}

func OpenPostgres(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return s.db.PingContext(pingCtx)
}

func (s *PostgresStore) EnsureSchema(ctx context.Context) error {
	migrationPath := filepath.Join("database", "migrations", "001_init.sql")
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", migrationPath, err)
	}
	if _, err := s.db.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func (s *PostgresStore) Subscribe(ctx context.Context, userID string, eventType string) (bool, error) {
	const query = `
INSERT INTO subscriptions (user_id, event_type)
VALUES ($1, $2)
ON CONFLICT (user_id, event_type) DO NOTHING`
	result, err := s.db.ExecContext(ctx, query, strings.TrimSpace(userID), strings.TrimSpace(eventType))
	if err != nil {
		return false, fmt.Errorf("subscribe: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

func (s *PostgresStore) SubscribeAll(ctx context.Context, userID string, eventTypes []string) (int64, error) {
	var added int64
	for _, eventType := range eventTypes {
		ok, err := s.Subscribe(ctx, userID, eventType)
		if err != nil {
			return added, err
		}
		if ok {
			added++
		}
	}
	return added, nil
}

func (s *PostgresStore) Unsubscribe(ctx context.Context, userID string, eventType string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE user_id = $1 AND event_type = $2`, userID, eventType)
	if err != nil {
		return false, fmt.Errorf("unsubscribe: %w", err)
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (s *PostgresStore) UnsubscribeAll(ctx context.Context, userID string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("unsubscribe all: %w", err)
	}
	return result.RowsAffected()
}

func (s *PostgresStore) GetSubscriptions(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_type FROM subscriptions WHERE user_id = $1 ORDER BY event_type`, userID)
	if err != nil {
		return nil, fmt.Errorf("get subscriptions: %w", err)
	}
	defer rows.Close()

	var subscriptions []string
	for rows.Next() {
		var eventType string
		if err := rows.Scan(&eventType); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, eventType)
	}
	return subscriptions, rows.Err()
}

func (s *PostgresStore) GetSubscribers(ctx context.Context, eventType string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id FROM subscriptions WHERE event_type = $1 ORDER BY user_id`, eventType)
	if err != nil {
		return nil, fmt.Errorf("get subscribers: %w", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	return users, rows.Err()
}

func (s *PostgresStore) GetPrefs(ctx context.Context, userID string) (UserPrefs, error) {
	const query = `SELECT COALESCE(locale, ''), COALESCE(timezone, ''), short_notifications FROM user_preferences WHERE user_id = $1`
	var prefs UserPrefs
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&prefs.Locale, &prefs.Timezone, &prefs.ShortNotifications)
	if err == sql.ErrNoRows {
		return UserPrefs{}, nil
	}
	if err != nil {
		return UserPrefs{}, fmt.Errorf("get prefs: %w", err)
	}
	return prefs, nil
}

func (s *PostgresStore) SetPrefs(ctx context.Context, userID string, prefs UserPrefs) error {
	current, err := s.GetPrefs(ctx, userID)
	if err != nil {
		return err
	}
	if prefs.Locale == "" {
		prefs.Locale = current.Locale
	}
	if prefs.Timezone == "" {
		prefs.Timezone = current.Timezone
	}

	const query = `
INSERT INTO user_preferences (user_id, locale, timezone, short_notifications, updated_at)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, NOW())
ON CONFLICT (user_id)
DO UPDATE SET
    locale = EXCLUDED.locale,
    timezone = EXCLUDED.timezone,
    short_notifications = EXCLUDED.short_notifications,
    updated_at = NOW()`
	if _, err := s.db.ExecContext(ctx, query, userID, prefs.Locale, prefs.Timezone, prefs.ShortNotifications); err != nil {
		return fmt.Errorf("set prefs: %w", err)
	}
	return nil
}

func (s *PostgresStore) StoreEvent(ctx context.Context, event webhook.Event) error {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	sourceIP := webhook.SourceIP(event)
	if sourceIP == "" {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO events (id, type, created_at, data, source_ip)
VALUES ($1, $2, $3, $4, NULL)
ON CONFLICT (id) DO NOTHING`, event.ID, event.Type, event.CreatedAt, data)
	} else {
		_, err = s.db.ExecContext(ctx, `
INSERT INTO events (id, type, created_at, data, source_ip)
VALUES ($1, $2, $3, $4, $5::inet)
ON CONFLICT (id) DO NOTHING`, event.ID, event.Type, event.CreatedAt, data, sourceIP)
	}
	if err != nil {
		return fmt.Errorf("store event: %w", err)
	}
	return nil
}

func (s *PostgresStore) StoreBlockedIP(ctx context.Context, ip string, eventID string) error {
	if strings.TrimSpace(ip) == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO blocked_ips (ip, event_id)
VALUES ($1::inet, NULLIF($2, ''))
ON CONFLICT (ip, event_id) DO NOTHING`, ip, eventID)
	if err != nil {
		return fmt.Errorf("store blocked ip: %w", err)
	}
	return nil
}

func (s *PostgresStore) SyncIgnoredIPs(ctx context.Context, ignored map[string][]string) error {
	for eventType, ips := range ignored {
		for _, ip := range ips {
			if strings.TrimSpace(ip) == "" {
				continue
			}
			if _, err := s.db.ExecContext(ctx, `
INSERT INTO ignored_ips (event_type, ip)
VALUES ($1, $2::inet)
ON CONFLICT (event_type, ip) DO NOTHING`, eventType, ip); err != nil {
				return fmt.Errorf("sync ignored ip: %w", err)
			}
		}
	}
	return nil
}

func (s *PostgresStore) PurgeOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE created_at < NOW() - ($1::int * INTERVAL '1 day')`, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("purge old events: %w", err)
	}
	return result.RowsAffected()
}

func (s *PostgresStore) SubscribersCount(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT user_id) FROM subscriptions`).Scan(&count); err != nil {
		return 0, fmt.Errorf("subscribers count: %w", err)
	}
	return count, nil
}

func (s *PostgresStore) EventsCount(ctx context.Context, days int) (int64, error) {
	query := `SELECT COUNT(*) FROM events`
	args := []interface{}{}
	if days > 0 {
		query += ` WHERE created_at >= NOW() - ($1::int * INTERVAL '1 day')`
		args = append(args, days)
	}
	var count int64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("events count: %w", err)
	}
	return count, nil
}

func (s *PostgresStore) EventsCountByType(ctx context.Context, days int) (map[string]int64, error) {
	query := `SELECT type, COUNT(*) FROM events`
	args := []interface{}{}
	if days > 0 {
		query += ` WHERE created_at >= NOW() - ($1::int * INTERVAL '1 day')`
		args = append(args, days)
	}
	query += ` GROUP BY type ORDER BY COUNT(*) DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("events by type: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		result[eventType] = count
	}
	return result, rows.Err()
}

func (s *PostgresStore) UsersWithSubscriptionCount(ctx context.Context) ([]UserSubscriptionCount, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT user_id, COUNT(*)
FROM subscriptions
GROUP BY user_id
ORDER BY COUNT(*) DESC, user_id`)
	if err != nil {
		return nil, fmt.Errorf("users with subscription count: %w", err)
	}
	defer rows.Close()

	var result []UserSubscriptionCount
	for rows.Next() {
		var item UserSubscriptionCount
		if err := rows.Scan(&item.UserID, &item.Count); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *PostgresStore) BlockedIPs(ctx context.Context, limit int) ([]BlockedIP, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT ip::text, COALESCE(event_id, ''), created_at
FROM blocked_ips
ORDER BY created_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("blocked ips: %w", err)
	}
	defer rows.Close()

	var result []BlockedIP
	for rows.Next() {
		var item BlockedIP
		if err := rows.Scan(&item.IP, &item.EventID, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
