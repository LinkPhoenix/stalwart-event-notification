# Stalwart Event Notification

Go Telegram bot and HTTP webhook server for Stalwart events.

This project is a complete rewrite of
[LinkPhoenix/Stalwart-Telegram-Bot-Webhook](https://github.com/LinkPhoenix/Stalwart-Telegram-Bot-Webhook),
which was originally written in TypeScript. It is now implemented in Go for
better reliability.

## Features

- Receives Stalwart events on `POST /`.
- Verifies optional HMAC `X-Signature` and Basic Auth.
- Lets Telegram users subscribe to event types.
- Supports Telegram user whitelisting for bot access and a separate admin whitelist.
- Sends localized Telegram notifications.
- Stores events, subscriptions, blocked IPs, ignored IPs, and preferences in PostgreSQL.
- Exposes `GET /`, `GET /health`, and `GET /metrics`.
- Ships with Docker Compose and GitHub Actions CI/GHCR publishing.

## Quick Start

```bash
cp .env.example .env
docker compose up -d --build
```

For local development:

```bash
go mod download
go test ./...
go run ./cmd/stalwart-event-notification
```

## Environment Variables

Copy `.env.example` to `.env` and configure the variables below.

### Required

| Variable | Description |
| --- | --- |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token from BotFather. |
| `DATABASE_URL` | PostgreSQL connection string. Example: `postgres://postgres:postgres@localhost:5432/stalwart_event_notification?sslmode=disable`. |

### Telegram Access Control

These variables are important because they control who can use the bot. Values are Telegram numeric IDs. In a private chat, the Telegram user ID and chat ID are usually the same value.

| Variable | Description |
| --- | --- |
| `ALLOWED_USER_ID` | Optional single Telegram user/chat ID allowed to use the bot. When empty, any Telegram user can interact with the bot. When set, every non-listed user receives an access denied message. |
| `ADMIN_USER_IDS` | Optional comma-separated Telegram user/chat IDs allowed to use admin commands such as `/stats`, `/users`, `/events_count`, and `/blocked`. If empty, `ALLOWED_USER_ID` is used as the admin when it is set. |

Example:

```env
ALLOWED_USER_ID=123456789
ADMIN_USER_IDS=123456789,987654321
```

To find your Telegram ID, you can send a message to an ID helper bot such as `@userinfobot`, or log incoming updates while developing.

### HTTP And Webhook Security

| Variable | Description |
| --- | --- |
| `PORT` | HTTP server port. Default: `3000`. |
| `LOG_LEVEL` | Logging level: `debug`, `info`, `warn`, or `error`. Default: `info`. |
| `WEBHOOK_KEY` | Optional Stalwart HMAC key. When set, `POST /` requires a valid `X-Signature` header. |
| `WEBHOOK_USERNAME` | Optional Basic Auth username for `POST /` and protected metrics. |
| `WEBHOOK_PASSWORD` | Optional Basic Auth password. |

### Defaults And Preferences

| Variable | Description |
| --- | --- |
| `DEFAULT_LOCALE` | Default language. Supported values: `de`, `en`, `es`, `fr`, `it`, `pt`, `ru`, `uk`. Default: `en`. |
| `DEFAULT_TIMEZONE` | Default timezone used in notifications. Default: `UTC`. |
| `LOCALES_DIR` | Directory containing locale JSON files. Default: `locales`. |

### Notification Filters

| Variable | Description |
| --- | --- |
| `SUBSCRIPTION_MIN_SEVERITY` | Minimum event severity to notify: `info`, `warning`, or `alert`. Default: `info`. |
| `QUIET_HOURS_START` / `QUIET_HOURS_END` | Optional quiet-hours window, for example `22:00` and `08:00`. Notifications are skipped during this window. |
| `DEDUP_ENABLED` | Enables deduplication of repeated `type|ip` events. Default: `true`. |
| `DEDUP_WINDOW_SECONDS` | Deduplication window in seconds. Default: `60`. |
| `NOTIFICATION_GROUP_WINDOW_SECONDS` | Groups similar notifications during this window. `0` disables grouping. |
| `EVENTS_RETENTION_DAYS` | Purges stored events older than this many days. `0` disables purge. |

### Per-Event Ignored IPs

For each supported Stalwart event, you can configure ignored source IPs. If an event source IP is listed, no Telegram notification is sent for that event.

Variable names are built from the event type by replacing `.` and `-` with `_`, uppercasing it, and adding `_IGNORED_IPS`.

| Variable | Event type |
| --- | --- |
| `AUTH_SUCCESS_IGNORED_IPS` | `auth.success` |
| `AUTH_FAILED_IGNORED_IPS` | `auth.failed` |
| `AUTH_ERROR_IGNORED_IPS` | `auth.error` |
| `DELIVERY_COMPLETED_IGNORED_IPS` | `delivery.completed` |
| `DELIVERY_DELIVERED_IGNORED_IPS` | `delivery.delivered` |
| `DELIVERY_FAILED_IGNORED_IPS` | `delivery.failed` |
| `SECURITY_ABUSE_BAN_IGNORED_IPS` | `security.abuse-ban` |
| `SECURITY_AUTHENTICATION_BAN_IGNORED_IPS` | `security.authentication-ban` |
| `SECURITY_IP_BLOCKED_IGNORED_IPS` | `security.ip-blocked` |
| `SERVER_STARTUP_IGNORED_IPS` | `server.startup` |
| `SERVER_STARTUP_ERROR_IGNORED_IPS` | `server.startup-error` |

Example:

```env
AUTH_SUCCESS_IGNORED_IPS=1.1.1.1,2.2.2.2
SECURITY_IP_BLOCKED_IGNORED_IPS=10.0.0.1
```

### Complete `.env.example` Reference

Every variable currently supported by `.env.example` is documented below.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `TELEGRAM_BOT_TOKEN` | Yes | None | Telegram bot token from BotFather. |
| `DATABASE_URL` | Yes | None | PostgreSQL connection string used by the bot and webhook server. |
| `PORT` | No | `3000` | HTTP server port. |
| `LOG_LEVEL` | No | `info` | JSON log level: `debug`, `info`, `warn`, or `error`. |
| `ALLOWED_USER_ID` | No | Empty | Single Telegram user/chat ID allowed to use the bot. Empty means no user whitelist. |
| `ADMIN_USER_IDS` | No | Empty | Comma-separated Telegram user/chat IDs allowed to use admin commands. Falls back to `ALLOWED_USER_ID` when empty. |
| `WEBHOOK_KEY` | No | Empty | Enables HMAC-SHA256 verification for Stalwart webhook requests. |
| `WEBHOOK_USERNAME` | No | Empty | Enables Basic Auth username verification for webhook requests. |
| `WEBHOOK_PASSWORD` | No | Empty | Basic Auth password paired with `WEBHOOK_USERNAME`. |
| `DEFAULT_LOCALE` | No | `en` | Default bot language. Supported: `de`, `en`, `es`, `fr`, `it`, `pt`, `ru`, `uk`. |
| `DEFAULT_TIMEZONE` | No | `UTC` | Default timezone for formatted notification timestamps. |
| `LOCALES_DIR` | No | `locales` | Directory containing locale JSON files. |
| `SUBSCRIPTION_MIN_SEVERITY` | No | `info` | Minimum event severity to send: `info`, `warning`, or `alert`. |
| `QUIET_HOURS_START` | No | Empty | Quiet-hours start time in `HH:MM`; skipped when empty. |
| `QUIET_HOURS_END` | No | Empty | Quiet-hours end time in `HH:MM`; skipped when empty. |
| `DEDUP_ENABLED` | No | `true` | Enables duplicate suppression for repeated `event type + source IP` events. |
| `DEDUP_WINDOW_SECONDS` | No | `60` | Time window used by deduplication. |
| `NOTIFICATION_GROUP_WINDOW_SECONDS` | No | `0` | Groups notifications during this many seconds. `0` disables grouping. |
| `EVENTS_RETENTION_DAYS` | No | `0` | Deletes stored events older than this number of days. `0` disables purge. |
| `*_IGNORED_IPS` | No | Empty | Comma-separated source IPs ignored for the matching event type. See the table above. |

## Telegram Commands

- `/start` - Show the welcome message.
- `/events` - List supported Stalwart events.
- `/subscribe <event|all>` - Subscribe to one or all events.
- `/unsubscribe <event|all>` - Unsubscribe.
- `/list` - Show current subscriptions.
- `/status` - Show service status.
- `/prefs` - Configure language, timezone, and short notifications.
- `/help` - Show help.
- `/stats`, `/users`, `/events_count`, `/blocked` - Admin commands.

## Stalwart Webhook

Configure Stalwart to send events to:

```text
http://your-host:3000/
```

When `WEBHOOK_KEY` is set, requests must include `X-Signature` containing the base64 HMAC-SHA256 of the raw body. When `WEBHOOK_USERNAME` is set, requests must include valid HTTP Basic Auth.
