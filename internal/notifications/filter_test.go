package notifications

import (
	"testing"
	"time"

	"stalwart-event-notification/internal/events"
	"stalwart-event-notification/internal/webhook"
)

func TestPassesSeverity(t *testing.T) {
	if PassesSeverity("auth.success", events.SeverityWarning) {
		t.Fatal("info event should not pass warning filter")
	}
	if !PassesSeverity("delivery.failed", events.SeverityWarning) {
		t.Fatal("alert event should pass warning filter")
	}
}

func TestIsInQuietHours(t *testing.T) {
	now := time.Date(2026, 5, 13, 23, 30, 0, 0, time.UTC)
	if !IsInQuietHours(now, "22:00", "08:00") {
		t.Fatal("expected overnight quiet hours")
	}
	if IsInQuietHours(now, "08:00", "22:00") {
		t.Fatal("did not expect daytime quiet hours")
	}
}

func TestDeduplicator(t *testing.T) {
	d := NewDeduplicator(true, 60)
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	d.now = func() time.Time { return now }

	event := webhook.Event{Type: "auth.failed", Data: map[string]interface{}{"remoteIp": "1.2.3.4"}}
	if !d.ShouldNotify(event) {
		t.Fatal("first event should notify")
	}
	if d.ShouldNotify(event) {
		t.Fatal("duplicate should not notify")
	}
	now = now.Add(61 * time.Second)
	if !d.ShouldNotify(event) {
		t.Fatal("event after window should notify")
	}
}
