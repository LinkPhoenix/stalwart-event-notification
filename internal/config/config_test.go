package config

import "testing"

func TestLoadRequiresTokenAndDatabaseURL(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected missing token error")
	}

	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing database url error")
	}
}

func TestLoadDefaultsAndIgnoredIPs(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("AUTH_SUCCESS_IGNORED_IPS", "1.1.1.1, 2.2.2.2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Port != 3000 {
		t.Fatalf("port = %d", cfg.Port)
	}
	if cfg.DefaultLocale != "en" {
		t.Fatalf("default locale = %s", cfg.DefaultLocale)
	}
	if got := cfg.IgnoredIPsByEvent["auth.success"]; len(got) != 2 || got[1] != "2.2.2.2" {
		t.Fatalf("ignored ips = %#v", got)
	}
}
