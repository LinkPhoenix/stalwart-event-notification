package notifications

import (
	"strconv"
	"strings"
	"time"

	"stalwart-event-notification/internal/events"
)

func PassesSeverity(eventType string, minSeverity events.Severity) bool {
	return events.SeverityRank(events.EventSeverity(eventType)) >= events.SeverityRank(minSeverity)
}

func IsInQuietHours(now time.Time, start string, end string) bool {
	startMinutes, okStart := parseHourMinute(start)
	endMinutes, okEnd := parseHourMinute(end)
	if !okStart || !okEnd {
		return false
	}

	current := now.Hour()*60 + now.Minute()
	if startMinutes <= endMinutes {
		return current >= startMinutes && current < endMinutes
	}
	return current >= startMinutes || current < endMinutes
}

func parseHourMinute(input string) (int, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, false
	}
	hourRaw, minuteRaw, ok := strings.Cut(input, ":")
	if !ok {
		return 0, false
	}
	hour, errHour := strconv.Atoi(hourRaw)
	minute, errMinute := strconv.Atoi(minuteRaw)
	if errHour != nil || errMinute != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}
