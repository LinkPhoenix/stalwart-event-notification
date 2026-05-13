# Project Instructions

This project is a Go rewrite of a Telegram bot that receives Stalwart webhook events and notifies subscribed Telegram users.

## Rules

- Keep all code, comments, and UI text in English.
- Keep database access inside `internal/storage`.
- Keep Telegram handling inside `internal/bot`.
- Keep Stalwart webhook parsing and verification inside `internal/webhook`.
- Use PostgreSQL as the only production storage backend.
- Use `go test ./...`, `go vet ./...`, and `gofmt` before shipping changes.
- Do not hardcode secrets; use environment variables.

