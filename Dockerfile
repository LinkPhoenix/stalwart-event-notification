# syntax=docker/dockerfile:1.7

FROM golang:1.26.2-alpine AS builder
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/stalwart-event-notification ./cmd/stalwart-event-notification

FROM alpine:3.22 AS runtime
WORKDIR /app

RUN addgroup -S app && adduser -S app -G app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/stalwart-event-notification /usr/local/bin/stalwart-event-notification
COPY --from=builder /app/locales /app/locales
COPY --from=builder /app/database /app/database

USER app
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/stalwart-event-notification"]

