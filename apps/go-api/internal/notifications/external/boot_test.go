package external

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// captureSlog remplace temporairement le logger par défaut par un handler JSON
// écrivant dans un buffer, et restaure l'original en fin de test.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func TestLogBootState_Active(t *testing.T) {
	buf := captureSlog(t)
	LogBootState(context.Background(), activeSettings(t))
	out := buf.String()
	if !strings.Contains(out, "relais coach Discord") {
		t.Errorf("ligne de boot absente : %s", out)
	}
	if !strings.Contains(out, `"actif":true`) {
		t.Errorf("état actif non loggué : %s", out)
	}
}

func TestLogBootState_Inactive(t *testing.T) {
	buf := captureSlog(t)
	settings := writeSettings(t, map[string]any{
		"discord_notifications_enabled": false,
		"discord_notify_coach":          false,
	})
	LogBootState(context.Background(), settings)
	out := buf.String()
	if !strings.Contains(out, `"actif":false`) {
		t.Errorf("état inactif non loggué : %s", out)
	}
	// Une seule ligne INFO (pas de bruit).
	if n := strings.Count(strings.TrimSpace(out), "\n"); n != 0 {
		t.Errorf("plus d'une ligne loggée au boot : %s", out)
	}
}
