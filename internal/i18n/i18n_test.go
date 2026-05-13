package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundleTranslationAndFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en.json"), []byte(`{"hello":"Hello {{name}}"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fr.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	bundle, err := Load(dir, "en")
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if got := bundle.T("en", "hello", map[string]string{"name": "Sam"}); got != "Hello Sam" {
		t.Fatalf("translation = %q", got)
	}
	if got := bundle.T("fr", "hello", map[string]string{"name": "Sam"}); got != "Hello Sam" {
		t.Fatalf("fallback = %q", got)
	}
	if got := bundle.Resolve("missing"); got != "en" {
		t.Fatalf("resolve = %q", got)
	}
}
