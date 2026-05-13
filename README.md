# Stalwart Event Notification

Go Telegram bot and HTTP webhook server for Stalwart events.

## Features

- Receives Stalwart events on `POST /`.
- Verifies optional HMAC `X-Signature` and Basic Auth.
- Lets Telegram users subscribe to event types.
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

