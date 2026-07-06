package skill

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// captureSlog redirige le logger par défaut vers un buffer le temps du test.
func captureSlog(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return buf, func() { slog.SetDefault(prev) }
}

func TestLogLUSRModeAtBoot(t *testing.T) {
	cases := []struct {
		name        string
		enabled     string
		canonical   string
		wantLevel   string
		wantContain string
	}{
		{"v1 par defaut", "", "", "INFO", "v1"},
		{"v2 shadow", "1", "", "INFO", "shadow"},
		{"v2 canonical", "1", "LUSR_V2", "INFO", "CANONICAL"},
		{"misconfig canonical sans enabled", "", "LUSR_V2", "WARN", "MISCONFIG"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(lusrV2EnvFlag, tc.enabled)
			t.Setenv(lusrCanonicalEnvFlag, tc.canonical)
			buf, restore := captureSlog(t)
			defer restore()

			LogLUSRModeAtBoot(context.Background())

			out := buf.String()
			if !strings.Contains(out, "level="+tc.wantLevel) {
				t.Errorf("niveau attendu %s, log = %q", tc.wantLevel, out)
			}
			if !strings.Contains(out, tc.wantContain) {
				t.Errorf("log doit contenir %q, got %q", tc.wantContain, out)
			}
		})
	}
}
