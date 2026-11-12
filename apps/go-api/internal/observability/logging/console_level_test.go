package logging

import (
	"log/slog"
	"testing"
)

func TestConsoleLevelFromEnv(t *testing.T) {
	cases := []struct {
		raw  string
		want slog.Level
	}{
		{"", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		{" DEBUG ", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"verbeux", slog.LevelInfo},
	}
	for _, c := range cases {
		t.Setenv("LEVELUP_LOG_LEVEL", c.raw)
		if got := ConsoleLevelFromEnv(); got != c.want {
			t.Errorf("LEVELUP_LOG_LEVEL=%q : %v, attendu %v", c.raw, got, c.want)
		}
	}
}
