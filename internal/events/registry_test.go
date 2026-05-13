package events

import "testing"

func TestRegistry(t *testing.T) {
	if !IsSupported("security.ip-blocked") {
		t.Fatal("security.ip-blocked should be supported")
	}
	if IsSupported("unknown.event") {
		t.Fatal("unknown event should not be supported")
	}
	if SeverityRank(EventSeverity("delivery.failed")) <= SeverityRank(SeverityWarning) {
		t.Fatal("delivery.failed should be alert severity")
	}
}
