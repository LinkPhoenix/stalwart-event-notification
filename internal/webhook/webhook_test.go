package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestVerifySignatureUTF8AndHex(t *testing.T) {
	body := []byte(`{"events":[]}`)
	key := "secret"
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !VerifySignature(body, key, signature) {
		t.Fatal("expected utf8 key signature to pass")
	}

	hexKey := hex.EncodeToString([]byte(key))
	if !VerifySignature(body, hexKey, signature) {
		t.Fatal("expected hex key fallback to pass")
	}

	if VerifySignature(body, key, "invalid") {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestVerifyBasicAuth(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	req.SetBasicAuth("user", "pass")

	if !VerifyBasicAuth(req.Header.Get("Authorization"), "user", "pass") {
		t.Fatal("expected basic auth to pass")
	}
	if VerifyBasicAuth(req.Header.Get("Authorization"), "user", "wrong") {
		t.Fatal("expected wrong password to fail")
	}
	if !VerifyBasicAuth("", "", "") {
		t.Fatal("empty username should disable basic auth")
	}
}

func TestParsePayloadAndSourceIP(t *testing.T) {
	payload, err := ParsePayload([]byte(`{"events":[{"id":"id1","createdAt":"2026-05-13T12:00:00Z","type":"auth.failed","data":{"remoteIp":"1.2.3.4"}}]}`))
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if len(payload.Events) != 1 {
		t.Fatalf("events = %d", len(payload.Events))
	}
	if SourceIP(payload.Events[0]) != "1.2.3.4" {
		t.Fatalf("source ip = %s", SourceIP(payload.Events[0]))
	}
}
