package notifications

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Metrics struct {
	mu       sync.Mutex
	counters map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{counters: map[string]int64{
		"webhook_requests_total":  0,
		"webhook_events_received": 0,
		"notifications_sent":      0,
		"notifications_failed":    0,
		"events_skipped":          0,
	}}
}

func (m *Metrics) Inc(name string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += delta
}

func (m *Metrics) RenderPrometheus() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	names := make([]string, 0, len(m.counters))
	for name := range m.counters {
		names = append(names, name)
	}
	sort.Strings(names)

	var builder strings.Builder
	for _, name := range names {
		builder.WriteString(fmt.Sprintf("# TYPE %s counter\n%s %d\n", name, name, m.counters[name]))
	}
	return builder.String()
}
