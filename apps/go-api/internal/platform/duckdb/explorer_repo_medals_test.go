//go:build integration

// Package duckdb — explorer_repo_medals_test.go : agrégat « top médailles » de
// l'Explorer sur une liste de match_id (DuckDB in-memory).
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"testing"
)

// TestExplorerRepo_GetTopMedalsForMatches valide l'agrégat « top médailles » sur
// une liste de match_id explicite : SUM(count) par identifiant, tri décroissant,
// cap `limit`, et exclusion stricte des matchs hors liste.
func TestExplorerRepo_GetTopMedalsForMatches(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	const targetXUID = "xuid_target_medals"

	seed := []struct {
		medalNameID uint64
		matchID     string
		count       int
	}{
		{100, "m1", 3},  // medal 100 : 3 sur m1 ...
		{100, "m2", 4},  // ... + 4 sur m2 = 7 (top)
		{200, "m1", 5},  // medal 200 : 5
		{300, "m2", 5},  // medal 300 : 5 (égalité avec 200 → départage par id croissant)
		{999, "m3", 50}, // hors de la liste demandée → doit être exclu
	}
	for _, m := range seed {
		_, err := pdb.Player.Exec(ctx,
			`INSERT INTO shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count) VALUES (?,?,?,?,?)`,
			m.medalNameID, m.medalNameID, targetXUID, m.matchID, m.count)
		if err != nil {
			t.Fatalf("insert medal: %v", err)
		}
	}
	// Bruit : un autre joueur sur les mêmes matchs ne doit pas fuiter.
	if _, err := pdb.Player.Exec(ctx,
		`INSERT INTO shared.medals_earned (medal_id, medal_name_id, xuid, match_id, count) VALUES (?,?,?,?,?)`,
		uint64(100), uint64(100), "xuid_autre", "m1", 99); err != nil {
		t.Fatalf("insert medal autre joueur: %v", err)
	}

	repo := NewExplorerRepo(pdb, pTestXUID)

	t.Run("agrégat trié, m3 et autre joueur exclus", func(t *testing.T) {
		got, err := repo.GetTopMedalsForMatches(ctx, targetXUID, []string{"m1", "m2"}, 10)
		if err != nil {
			t.Fatalf("GetTopMedalsForMatches: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("attendu 3 médailles distinctes (100/200/300), got %d : %+v", len(got), got)
		}
		if got[0].NameID != 100 || got[0].Count != 7 {
			t.Errorf("tête attendue medal 100 count 7 (3+4), got %+v", got[0])
		}
		// Égalité 5/5 → départage déterministe par identifiant croissant.
		if got[1].NameID != 200 || got[2].NameID != 300 {
			t.Errorf("départage à égalité attendu 200 puis 300, got %d puis %d", got[1].NameID, got[2].NameID)
		}
	})

	t.Run("limit borne le résultat", func(t *testing.T) {
		got, err := repo.GetTopMedalsForMatches(ctx, targetXUID, []string{"m1", "m2"}, 1)
		if err != nil {
			t.Fatalf("GetTopMedalsForMatches: %v", err)
		}
		if len(got) != 1 || got[0].NameID != 100 {
			t.Errorf("limit=1 attendu [medal 100], got %+v", got)
		}
	})

	t.Run("entrées vides → nil", func(t *testing.T) {
		cases := []struct {
			name     string
			xuid     string
			matchIDs []string
			limit    int
		}{
			{"matchIDs vide", targetXUID, nil, 10},
			{"xuid vide", "  ", []string{"m1"}, 10},
			{"limit nul", targetXUID, []string{"m1"}, 0},
		}
		for _, c := range cases {
			got, err := repo.GetTopMedalsForMatches(ctx, c.xuid, c.matchIDs, c.limit)
			if err != nil {
				t.Fatalf("%s: err inattendue: %v", c.name, err)
			}
			if got != nil {
				t.Errorf("%s: attendu nil, got %+v", c.name, got)
			}
		}
	})

	t.Run("xuid sans médaille → vide", func(t *testing.T) {
		got, err := repo.GetTopMedalsForMatches(ctx, "xuid_inconnu", []string{"m1", "m2"}, 10)
		if err != nil {
			t.Fatalf("GetTopMedalsForMatches: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("attendu aucune médaille, got %+v", got)
		}
	})
}
