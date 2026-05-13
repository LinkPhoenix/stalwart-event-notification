CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_ip INET,
    stored_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at DESC);

CREATE TABLE IF NOT EXISTS blocked_ips (
    id BIGSERIAL PRIMARY KEY,
    ip INET NOT NULL,
    event_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (ip, event_id)
);

CREATE INDEX IF NOT EXISTS idx_blocked_ips_created_at ON blocked_ips(created_at DESC);

CREATE TABLE IF NOT EXISTS ignored_ips (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    ip INET NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_type, ip)
);

CREATE INDEX IF NOT EXISTS idx_ignored_ips_event_type ON ignored_ips(event_type);

CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, event_type)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_event_type ON subscriptions(event_type);

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id TEXT PRIMARY KEY,
    locale TEXT,
    timezone TEXT,
    short_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

