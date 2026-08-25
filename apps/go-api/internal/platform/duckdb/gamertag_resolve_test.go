//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// TestGamertagRepo_ResolveGamertags valide la résolution batch xuid → gamertag
// via le chokepoint v_gamertag_lookup (sur SharedReader, vue bare), utilisée par
// l'enrichissement des identités de la timeline d'events (MatchEventsService).
func TestGamertagRepo_ResolveGamertags(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "x1", "Alpha")
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES (?, ?)`, "x2", "Beta")

	repo := NewGamertagRepo(pdb.SharedReadDB())

	// Dédup (x1 deux fois) + vide ignoré ; xMissing absent du référentiel → exclu.
	got, err := repo.ResolveGamertags(ctx, []string{"x1", "x2", "x1", "", "xMissing"})
	if err != nil {
		t.Fatalf("ResolveGamertags: %v", err)
	}
	if got["x1"] != "Alpha" || got["x2"] != "Beta" {
		t.Errorf("résolution inattendue: %+v", got)
	}
	if _, ok := got["xMissing"]; ok {
		t.Errorf("xuid orphelin ne doit PAS apparaître (masqué au rendu front): %+v", got)
	}

	empty, err := repo.ResolveGamertags(ctx, nil)
	if err != nil || len(empty) != 0 {
		t.Errorf("xuids vide → map vide sans erreur, got %v err=%v", empty, err)
	}
}
