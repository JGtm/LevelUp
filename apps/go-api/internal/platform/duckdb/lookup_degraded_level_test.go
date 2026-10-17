package duckdb

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestDegradedLookupLevel(t *testing.T) {
	t.Parallel()
	vivante := context.Background()
	annulee, cancel := context.WithCancel(context.Background())
	cancel()
	for _, c := range []struct {
		nom  string
		ctx  context.Context
		err  error
		want slog.Level
	}{
		{"panne, requête vivante", vivante, errors.New("database is locked"), slog.LevelWarn},
		{"table absente", vivante, errors.New("Catalog Error: Table with name x does not exist"), slog.LevelDebug},
		{"requête abandonnée", annulee, errors.New("database is locked"), slog.LevelDebug},
	} {
		if got := degradedLookupLevel(c.ctx, c.err); got != c.want {
			t.Errorf("%s : %v, want %v", c.nom, got, c.want)
		}
	}
}
