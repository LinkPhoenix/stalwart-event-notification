package events

import "strings"

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityAlert   Severity = "alert"
)

type Definition struct {
	Type     string
	HasIP    bool
	Severity Severity
}

var Registry = map[string]Definition{
	"auth.error":                  {Type: "auth.error", HasIP: true, Severity: SeverityWarning},
	"auth.failed":                 {Type: "auth.failed", HasIP: true, Severity: SeverityWarning},
	"auth.success":                {Type: "auth.success", HasIP: true, Severity: SeverityInfo},
	"delivery.completed":          {Type: "delivery.completed", HasIP: true, Severity: SeverityInfo},
	"delivery.delivered":          {Type: "delivery.delivered", HasIP: true, Severity: SeverityInfo},
	"delivery.failed":             {Type: "delivery.failed", HasIP: true, Severity: SeverityAlert},
	"security.abuse-ban":          {Type: "security.abuse-ban", HasIP: true, Severity: SeverityAlert},
	"security.authentication-ban": {Type: "security.authentication-ban", HasIP: true, Severity: SeverityAlert},
	"security.ip-blocked":         {Type: "security.ip-blocked", HasIP: true, Severity: SeverityAlert},
	"server.startup":              {Type: "server.startup", HasIP: false, Severity: SeverityInfo},
	"server.startup-error":        {Type: "server.startup-error", HasIP: false, Severity: SeverityAlert},
}

func SupportedTypes() []string {
	types := make([]string, 0, len(Registry))
	for eventType := range Registry {
		types = append(types, eventType)
	}
	return types
}

func IsSupported(eventType string) bool {
	_, ok := Registry[strings.TrimSpace(eventType)]
	return ok
}

func EventSeverity(eventType string) Severity {
	if definition, ok := Registry[strings.TrimSpace(eventType)]; ok {
		return definition.Severity
	}
	return SeverityInfo
}

func SeverityRank(severity Severity) int {
	switch severity {
	case SeverityAlert:
		return 2
	case SeverityWarning:
		return 1
	default:
		return 0
	}
}
