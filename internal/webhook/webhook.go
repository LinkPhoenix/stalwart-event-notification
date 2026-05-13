package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"stalwart-event-notification/internal/events"
)

type Event struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"createdAt"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
}

type rawEvent struct {
	ID        string                 `json:"id"`
	CreatedAt string                 `json:"createdAt"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
}

type Payload struct {
	Events []Event `json:"events"`
}

func VerifySignature(rawBody []byte, key string, signature string) bool {
	signature = strings.TrimSpace(signature)
	key = strings.TrimSpace(key)
	if key == "" || signature == "" {
		return false
	}

	if verifyHMAC(rawBody, []byte(key), signature) {
		return true
	}

	if looksHex(key) {
		decoded, err := hex.DecodeString(key)
		if err == nil && verifyHMAC(rawBody, decoded, signature) {
			return true
		}
	}

	return false
}

func VerifyBasicAuth(header string, username string, password string) bool {
	if strings.TrimSpace(username) == "" {
		return true
	}
	request := &http.Request{Header: http.Header{"Authorization": []string{header}}}
	gotUser, gotPassword, ok := request.BasicAuth()
	return ok && gotUser == username && gotPassword == password
}

func ParsePayload(rawBody []byte) (Payload, error) {
	var input struct {
		Events []rawEvent `json:"events"`
	}
	if err := json.Unmarshal(rawBody, &input); err != nil {
		return Payload{}, err
	}
	if input.Events == nil {
		return Payload{}, ErrMissingEvents
	}

	parsed := Payload{Events: make([]Event, 0, len(input.Events))}
	for _, item := range input.Events {
		createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(item.CreatedAt))
		if err != nil {
			createdAt = time.Now().UTC()
		}
		data := item.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		parsed.Events = append(parsed.Events, Event{
			ID:        strings.TrimSpace(item.ID),
			CreatedAt: createdAt.UTC(),
			Type:      strings.TrimSpace(item.Type),
			Data:      data,
		})
	}

	return parsed, nil
}

func IsKnownEventType(eventType string) bool {
	return events.IsSupported(eventType)
}

func SourceIP(event Event) string {
	for _, key := range []string{"remoteIp", "ip", "source_ip"} {
		value, ok := event.Data[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func verifyHMAC(rawBody []byte, key []byte, expected string) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(rawBody)
	actual := mac.Sum(nil)

	expectedBytes, err := base64.StdEncoding.DecodeString(expected)
	if err != nil {
		return false
	}
	return hmac.Equal(actual, expectedBytes)
}

func looksHex(input string) bool {
	if len(input) == 0 || len(input)%2 != 0 {
		return false
	}
	for _, ch := range input {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
			return false
		}
	}
	return true
}
