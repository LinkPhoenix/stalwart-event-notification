package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	if got := bundle.Resolve("fr-FR"); got != "fr" {
		t.Fatalf("resolve regional = %q", got)
	}
	if got := bundle.Resolve("fr_FR"); got != "fr" {
		t.Fatalf("resolve underscored = %q", got)
	}
}

func TestProjectLocaleFilesHaveSameKeysAndSupportedList(t *testing.T) {
	dir := filepath.Join("..", "..", "locales")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read locales: %v", err)
	}

	var locales []string
	catalogs := make(map[string]map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		locale := entry.Name()[:len(entry.Name())-len(".json")]
		locales = append(locales, locale)
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(content, &catalog); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		catalogs[locale] = catalog
	}
	sort.Strings(locales)
	wantLocales := []string{"de", "en", "es", "fr", "it", "pt", "ru", "uk"}
	if join(locales) != join(wantLocales) {
		t.Fatalf("locales = %v, want %v", locales, wantLocales)
	}

	enKeys := sortedKeys(catalogs["en"])
	for _, locale := range wantLocales {
		if got := sortedKeys(catalogs[locale]); join(got) != join(enKeys) {
			t.Fatalf("%s keys differ from en", locale)
		}
		timezonePrompt := catalogs[locale]["prefs.timezone_prompt"]
		if strings.Contains(timezonePrompt, "<zone>") {
			t.Fatalf("%s timezone prompt contains raw HTML-like placeholder: %q", locale, timezonePrompt)
		}
		if !strings.Contains(timezonePrompt, "&lt;zone&gt;") {
			t.Fatalf("%s timezone prompt does not escape zone placeholder: %q", locale, timezonePrompt)
		}
	}
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func join(values []string) string {
	return strings.Join(values, "\x00")
}
