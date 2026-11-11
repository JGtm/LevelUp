package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestConsoleLogLevel(t *testing.T) {
	cases := []struct {
		raw  string
		want slog.Level
	}{
		{"", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		{" DEBUG ", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"verbeux", slog.LevelInfo},
	}
	for _, c := range cases {
		t.Setenv(envLogLevel, c.raw)
		if got := ConsoleLogLevel(); got != c.want {
			t.Errorf("LEVELUP_LOG_LEVEL=%q : %v, attendu %v", c.raw, got, c.want)
		}
	}
}

func TestOAuthClientIDFromEnv(t *testing.T) {
	t.Setenv(envOAuthClientID, "")
	if got := OAuthClientIDFromEnv(); got != "" {
		t.Errorf("variable absente : %q, attendu vide", got)
	}
	t.Setenv(envOAuthClientID, "abc-123")
	if got := OAuthClientIDFromEnv(); got != "abc-123" {
		t.Errorf("variable posée : %q, attendu abc-123", got)
	}
}

func TestLegacyFilmCacheDirFromEnv(t *testing.T) {
	t.Setenv(envLegacyFilmCacheDir, "  /cache/films  ")
	if got := LegacyFilmCacheDirFromEnv(); got != "/cache/films" {
		t.Errorf("espaces de bord non retirés : %q", got)
	}
	t.Setenv(envLegacyFilmCacheDir, "")
	if got := LegacyFilmCacheDirFromEnv(); got != "" {
		t.Errorf("variable absente : %q, attendu vide", got)
	}
}

func TestEventsConvergenceTuning(t *testing.T) {
	cases := []struct {
		chunk, yield string
		wantChunk    int
		wantYield    time.Duration
	}{
		{"", "", 0, 0},
		{"25", "500", 25, 500 * time.Millisecond},
		{"0", "-3", 0, 0},
		{"abc", "2000", 0, 2 * time.Second},
	}
	for _, c := range cases {
		t.Setenv(envEventsConvChunk, c.chunk)
		t.Setenv(envEventsConvYieldMS, c.yield)
		chunk, yield := EventsConvergenceTuning()
		if chunk != c.wantChunk || yield != c.wantYield {
			t.Errorf("chunk=%q yield=%q : (%d, %v), attendu (%d, %v)", c.chunk, c.yield, chunk, yield, c.wantChunk, c.wantYield)
		}
	}
}

func TestNotifyVersionsEnabled(t *testing.T) {
	for raw, want := range map[string]bool{"": false, "0": false, "true": false, "1": true} {
		t.Setenv(envNotifyVersions, raw)
		if got := NotifyVersionsEnabled(); got != want {
			t.Errorf("LEVELUP_NOTIFY_VERSIONS=%q : %v, attendu %v", raw, got, want)
		}
	}
}
