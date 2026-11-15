package scheduler

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// Quand une panne 5xx précède un échec qui n'en est pas une, la ligne ERROR cite la cause à investiguer
// (le premier échec hors panne), pas le texte de la panne.
func TestScrapeFailures_ErreurCiteLEchecHorsPanne(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	var f scrapeFailures
	f.add("pl-a", errors.New("statut HTTP 500"))
	f.add("pl-b", errors.New("markup illisible"))
	f.add("pl-c", errors.New("statut HTTP 502"))
	f.log(context.Background(), "csrseason13-2")

	out := buf.String()
	if !strings.Contains(out, "level=ERROR") {
		t.Fatalf("une ligne ERROR était attendue :\n%s", out)
	}
	if !strings.Contains(out, "markup illisible") || strings.Contains(out, "HTTP 500") {
		t.Errorf("la ligne ERROR doit citer l'échec hors panne, pas la panne :\n%s", out)
	}
}

// Sans échec hors panne, l'exemple est le premier échec et le niveau est WARN.
func TestScrapeFailures_PanneSeuleResteUnWarn(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	var f scrapeFailures
	f.add("pl-a", errors.New("statut HTTP 500"))
	f.add("pl-b", errors.New("statut HTTP 503"))
	f.log(context.Background(), "s")

	if out := buf.String(); !strings.Contains(out, "level=WARN") || !strings.Contains(out, "HTTP 500") {
		t.Errorf("WARN avec le premier échec attendu :\n%s", out)
	}
}
