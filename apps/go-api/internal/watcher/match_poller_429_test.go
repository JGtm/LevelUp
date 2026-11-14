package watcher

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/observability"
)

type errFetcher struct{ err error }

func (e errFetcher) FetchRecentMatchIDs(context.Context, string, int) ([]string, error) {
	return nil, e.err
}

// Un 429 au poll est retenté au poll suivant : INFO et compteur, pas de WARN ; toute autre erreur de
// fetch reste un WARN.
func TestMatchPoller_429RetenteSansWarn(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	avant := observability.LoadCounter(matchPoll429Counter)

	NewMatchPoller("x", "p", errFetcher{errors.New("FetchRecentMatchIDs: HTTP 429: https://exemple")}, nil).
		poll(context.Background(), false)

	if strings.Contains(buf.String(), "level=WARN") {
		t.Errorf("WARN écrit pour un 429 retenté :\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "level=INFO") || !strings.Contains(buf.String(), "429") {
		t.Errorf("INFO attendu pour un 429 retenté :\n%s", buf.String())
	}
	if got := observability.LoadCounter(matchPoll429Counter) - avant; got != 1 {
		t.Errorf("compteur des 429 : +%d, attendu +1", got)
	}

	buf.Reset()
	NewMatchPoller("x", "p", errFetcher{errors.New("HTTP 500: https://exemple")}, nil).poll(context.Background(), false)
	if !strings.Contains(buf.String(), "level=WARN") {
		t.Errorf("une erreur 500 doit rester un WARN :\n%s", buf.String())
	}
}
