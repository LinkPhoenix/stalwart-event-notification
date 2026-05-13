package notifications

import (
	"sync"
	"time"

	"stalwart-event-notification/internal/webhook"
)

type Deduplicator struct {
	enabled bool
	window  time.Duration
	now     func() time.Time
	mu      sync.Mutex
	seen    map[string]time.Time
}

func NewDeduplicator(enabled bool, windowSeconds int) *Deduplicator {
	if windowSeconds <= 0 {
		windowSeconds = 60
	}
	return &Deduplicator{
		enabled: enabled,
		window:  time.Duration(windowSeconds) * time.Second,
		now:     time.Now,
		seen:    make(map[string]time.Time),
	}
}

func (d *Deduplicator) ShouldNotify(event webhook.Event) bool {
	if !d.enabled {
		return true
	}
	key := event.Type + "|" + webhook.SourceIP(event)
	if key == event.Type+"|" {
		key = event.Type + "|no-ip"
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	now := d.now()
	if lastSeen, ok := d.seen[key]; ok && now.Sub(lastSeen) < d.window {
		return false
	}
	d.seen[key] = now

	cutoff := now.Add(-d.window)
	for currentKey, timestamp := range d.seen {
		if timestamp.Before(cutoff) {
			delete(d.seen, currentKey)
		}
	}
	return true
}
