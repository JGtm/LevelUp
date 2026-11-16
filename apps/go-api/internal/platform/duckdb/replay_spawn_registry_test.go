//go:build integration

// replay_spawn_registry_test.go — le registre lu pour la prevision de reapparition.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestReplayMapRepo_Spawn

package duckdb

import (
	"context"
	"errors"
	"testing"
)

func TestReplayMapRepo_SpawnRegistre(t *testing.T) {
	ctx := context.Background()
	shared := openMemDB(t)
	if _, err := shared.Exec(ctx, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, map_id VARCHAR, game_variant_name VARCHAR, mode_category VARCHAR, is_firefight BOOLEAN)`); err != nil {
		t.Fatal(err)
	}
	if _, err := shared.Exec(ctx, `INSERT INTO match_registry VALUES
		('b', 'carte-1', 'Slayer:Arena', 'Assassin', FALSE),
		('a', 'carte-1', 'CTF:Arena', NULL, NULL),
		('c', 'carte-1', 'Firefight', 'PvE', TRUE),
		('d', 'carte-2', 'Slayer:Arena', 'Assassin', FALSE),
		('e', NULL, 'Slayer:Arena', 'Assassin', FALSE)`); err != nil {
		t.Fatal(err)
	}
	repo := NewReplayMapRepo(LegacySharedReader(shared), nil)

	m, err := repo.SpawnMatchOf(ctx, "b")
	if err != nil || m.MapID != "carte-1" || m.Variant != "Slayer:Arena" || m.Category != "Assassin" {
		t.Errorf("SpawnMatchOf(b) = %+v, %v", m, err)
	}
	for _, id := range []string{"inconnu", "e"} {
		if _, err := repo.SpawnMatchOf(ctx, id); !errors.Is(err, ErrMatchMapUnknown) {
			t.Errorf("SpawnMatchOf(%s) : erreur %v, attendu ErrMatchMapUnknown (absent ou sans carte)", id, err)
		}
	}

	ms, err := repo.SpawnMatchesOnMap(ctx, "carte-1")
	if err != nil || len(ms) != 2 || ms[0].MatchID != "a" || ms[1].MatchID != "b" {
		t.Errorf("SpawnMatchesOnMap(carte-1) = %+v, %v ; attendu a et b (le Firefight n'y est pas), tries", ms, err)
	}
	if ms[0].Category != "" || ms[0].Variant != "CTF:Arena" {
		t.Errorf("une categorie NULL se lit vide : %+v", ms[0])
	}
	if ms, err := repo.SpawnMatchesOnMap(ctx, ""); err != nil || len(ms) != 0 {
		t.Errorf("carte vide : %+v, %v", ms, err)
	}
}
