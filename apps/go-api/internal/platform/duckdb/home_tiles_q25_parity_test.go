//go:build integration

// Package duckdb — home_tiles_q25_parity_test.go : la navigation prev/next d'un match ouvert depuis
// l'Accueil n'emporte plus de liste d'identifiants (D-9) : les voisins viennent de Q25. Ce test
// tient l'ordre de Q25 et celui de la liste de tuiles (analysis.OrderHomeTileRows, sur les lignes du
// chargeur canonique) pour UN seul ordre, ex aequo compris : même voisin, même position, même total.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/port"
)

func TestHomeTiles_OrdreIdentiqueAQ25(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedPlayerMatchesFixtures(t, pdb)
	ctx := context.Background()
	// Trois ex aequo (même instant canonique) au milieu de l'historique, insérés hors ordre.
	for _, id := range []string{"tie_a", "tie_c", "tie_b"} {
		start := "2026-03-20T12:00:00Z"
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_name, pair_name, playlist_name, is_ranked)
			VALUES (?, ?, ?, 'Aquarius', 'Arena:Slayer', 'Quick Play', false)`, id, start, start)
		execOnSharedDBs(t, pdb, ctx, `INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, 2, 0)`, id, pTestXUID, pTestGamertag)
	}

	rows, err := NewPlayerMatchesRepo(pdb).Load(ctx, port.PlayerMatchFilters{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ordered := analysis.OrderHomeTileRows(rows)
	if len(ordered) != 8 {
		t.Fatalf("liste = %d tuiles, attendu 8 (5 fixtures + 3 ex aequo)", len(ordered))
	}

	repo := NewMatchViewRepo(pdb, pTestXUID)
	for i := range ordered {
		id := ordered[i].Summary.MatchID
		got, err := repo.GetMatchNeighbors(ctx, pTestXUID, id)
		if err != nil {
			t.Fatalf("GetMatchNeighbors(%s): %v", id, err)
		}
		var wantNext, wantPrev *string
		if i > 0 {
			wantNext = &ordered[i-1].Summary.MatchID
		}
		if i < len(ordered)-1 {
			wantPrev = &ordered[i+1].Summary.MatchID
		}
		assertNeighbors(t, got, wantPrev, wantNext, i, len(ordered))
	}
}
