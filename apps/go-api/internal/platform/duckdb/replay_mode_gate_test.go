//go:build integration

// replay_mode_gate_test.go — le registre dit quels matchs le rejeu ne gere pas.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestReplayMapRepo_ModeNonGere

package duckdb

import (
	"context"
	"testing"
)

func TestReplayMapRepo_ModeNonGere(t *testing.T) {
	ctx := context.Background()
	shared := openMemDB(t)
	if _, err := shared.Exec(ctx, `CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, is_firefight BOOLEAN)`); err != nil {
		t.Fatal(err)
	}
	if _, err := shared.Exec(ctx, `INSERT INTO match_registry VALUES
		('edc5daf6-0df8-4eb0-b578-b8eb892ce184', TRUE),
		('39910eb1-8c01-4e26-bc60-965cb2dcc789', FALSE),
		('00761d27-487c-4d7d-ac4c-bf7584de652c', NULL)`); err != nil {
		t.Fatal(err)
	}
	repo := NewReplayMapRepo(LegacySharedReader(shared), nil)

	for id, want := range map[string]bool{
		"edc5daf6-0df8-4eb0-b578-b8eb892ce184": true,
		"39910eb1-8c01-4e26-bc60-965cb2dcc789": false,
		"00761d27-487c-4d7d-ac4c-bf7584de652c": false, // drapeau NULL : jamais non gere
		"inconnu":                              false, // hors registre : on ne refuse pas
	} {
		got, err := repo.ModeNonGere(ctx, id)
		if err != nil || got != want {
			t.Errorf("ModeNonGere(%s) = %v, %v ; attendu %v", id, got, err, want)
		}
	}
	set, err := repo.ModesNonGeresShortIDs(ctx)
	if err != nil || len(set) != 1 {
		t.Fatalf("ModesNonGeresShortIDs = %v, %v ; attendu le seul Firefight", set, err)
	}
	if _, ok := set["edc5daf6"]; !ok {
		t.Errorf("la forme courte edc5daf6 manque : %v", set)
	}
}
