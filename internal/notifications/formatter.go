package notifications

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"stalwart-event-notification/internal/i18n"
	"stalwart-event-notification/internal/webhook"
)

const abuseIPDBBaseURL = "https://www.abuseipdb.com/check/"

func FormatEventMessage(bundle *i18n.Bundle, event webhook.Event, locale string, timezone string, short bool) string {
	locale = bundle.Resolve(locale)
	location := resolveLocation(timezone)
	title := bundle.T(locale, "event."+event.Type+".title", nil)
	if strings.HasPrefix(title, "event.") {
		title = event.Type
	}

	lines := []string{
		fmt.Sprintf("<b>%s</b>", html.EscapeString(title)),
		"",
		fmt.Sprintf("<b>%s</b> <code>%s</code>", bundle.T(locale, "notification.event", nil), html.EscapeString(event.Type)),
		fmt.Sprintf("<b>%s</b> %s", bundle.T(locale, "notification.date", nil), html.EscapeString(event.CreatedAt.In(location).Format("2006-01-02 15:04:05 MST"))),
		fmt.Sprintf("<b>%s</b> <code>%s</code>", bundle.T(locale, "notification.ref", nil), html.EscapeString(event.ID)),
	}

	fields := eventFields(event)
	if short && len(fields) > 3 {
		fields = fields[:3]
	}
	if len(fields) > 0 {
		lines = append(lines, "", bundle.T(locale, "notification.details", nil))
		for _, field := range fields {
			lines = append(lines, fmt.Sprintf("• %s · <code>%s</code>", html.EscapeString(field.label), html.EscapeString(field.value)))
		}
	}

	if ip := webhook.SourceIP(event); ip != "" {
		lines = append(lines, "", fmt.Sprintf(`<a href="%s%s">%s</a>`, abuseIPDBBaseURL, html.EscapeString(ip), html.EscapeString(bundle.T(locale, "notification.view_on_abuseipdb", nil))))
	}

	return strings.Join(lines, "\n")
}

func FormatGroupedMessage(bundle *i18n.Bundle, events []webhook.Event, locale string, timezone string) string {
	if len(events) == 0 {
		return ""
	}
	locale = bundle.Resolve(locale)
	location := resolveLocation(timezone)
	lines := []string{
		bundle.T(locale, "notification.grouped_title", map[string]string{
			"count": fmt.Sprintf("%d", len(events)),
			"type":  events[0].Type,
		}),
		"",
	}
	for _, event := range events {
		ip := webhook.SourceIP(event)
		if ip == "" {
			ip = "no-ip"
		}
		lines = append(lines, fmt.Sprintf("• <code>%s</code> %s %s", html.EscapeString(event.ID), html.EscapeString(ip), html.EscapeString(event.CreatedAt.In(location).Format(time.TimeOnly))))
	}
	return strings.Join(lines, "\n")
}

type messageField struct {
	label string
	value string
}

func eventFields(event webhook.Event) []messageField {
	switch event.Type {
	case "auth.success":
		return pickFields(event.Data, "accountName", "accountId", "spanId", "listenerId", "localPort", "remoteIp", "remotePort")
	case "auth.failed":
		return pickFields(event.Data, "accountName", "id", "spanId", "listenerId", "localPort", "remoteIp", "remotePort")
	case "auth.error":
		return pickFields(event.Data, "details", "error", "spanId", "listenerId", "localPort", "remoteIp", "remotePort")
	case "delivery.delivered":
		return pickFields(event.Data, "from", "to", "hostname", "code", "details", "size", "elapsed", "queueName", "spanId", "total")
	case "delivery.completed":
		return pickFields(event.Data, "from", "to", "size", "elapsed", "queueName", "spanId", "total")
	case "delivery.failed":
		return pickFields(event.Data, "error", "details", "from", "to", "recipient", "remoteIp", "spanId", "messageId")
	case "security.ip-blocked":
		return pickFields(event.Data, "listenerId", "localPort", "remoteIp", "ip", "source_ip", "remotePort")
	case "security.abuse-ban", "security.authentication-ban":
		return pickFields(event.Data, "reason", "details", "accountName", "remoteIp", "ip", "source_ip")
	case "server.startup":
		return pickFields(event.Data, "version")
	case "server.startup-error":
		return pickFields(event.Data, "error", "details", "message")
	default:
		raw, _ := json.MarshalIndent(event.Data, "", "  ")
		if len(raw) == 0 || string(raw) == "{}" {
			return nil
		}
		return []messageField{{label: "data", value: string(raw)}}
	}
}

func pickFields(data map[string]interface{}, keys ...string) []messageField {
	fields := make([]messageField, 0, len(keys))
	seen := make(map[string]struct{})
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		fields = append(fields, messageField{label: key, value: valueToString(value)})
	}
	return fields
}

func valueToString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return "-"
		}
		return typed
	case []interface{}:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, valueToString(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func resolveLocation(timezone string) *time.Location {
	if strings.TrimSpace(timezone) == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}
	return location
}
