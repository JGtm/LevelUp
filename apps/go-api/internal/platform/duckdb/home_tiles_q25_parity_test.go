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
	"levelup/go-api/internal/domain"
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

// TestMatchNeighbors_LisentLeLiveJamaisLeSnapshot : Q25 est la source du prev/next et du compteur
// depuis l'Accueil ; elle lit la MÊME source que la liste de tuiles (live). Un snapshot coupé avant
// les derniers matchs ne doit ni vider les voisins de ces matchs ni fausser le total des autres.
func TestMatchNeighbors_LisentLeLiveJamaisLeSnapshot(t *testing.T) {
	pdb := newTestPlayerDBForNeighborsScenario(t) // live : n1..n8
	ctx := context.Background()

	// Snapshot immuable coupé avant les deux derniers matchs (n7, n8).
	snap := openMemDB(t)
	seedSharedDBSchema(t, snap)
	for _, q := range []string{`DELETE FROM shared.match_participants`, `DELETE FROM shared.match_registry`} {
		if _, err := snap.Exec(ctx, q); err != nil {
			t.Fatalf("wipe snapshot: %v", err)
		}
	}
	for _, m := range neighborsTestDataset {
		if m.id == "n7" || m.id == "n8" {
			continue
		}
		if _, err := snap.Exec(ctx, `INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_name, pair_name, playlist_name, is_ranked)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, m.id, m.startUTC, m.startUTC, m.mapName, m.pairName, m.playlist, m.isRanked); err != nil {
			t.Fatalf("snapshot registry %s: %v", m.id, err)
		}
		if _, err := snap.Exec(ctx, `INSERT INTO shared.match_participants
			(match_id, xuid, gamertag, outcome, team_id) VALUES (?, ?, ?, ?, 0)`, m.id, pTestXUID, pTestGamertag, m.outcome); err != nil {
			t.Fatalf("snapshot participant %s: %v", m.id, err)
		}
	}
	repo := NewMatchViewRepo(pdb, pTestXUID).WithModeTaxonomy(testModeTaxonomy()).WithSharedReader(LegacySharedReader(snap))

	// Le plus récent du live (absent du snapshot) garde ses voisins et le vrai total.
	got, err := repo.GetMatchNeighbors(ctx, pTestXUID, "n8")
	if err != nil {
		t.Fatalf("GetMatchNeighbors(n8): %v", err)
	}
	n7 := "n7"
	assertNeighbors(t, got, &n7, nil, 0, 8)

	// Un match présent dans le snapshot : le total est celui du live (8), pas du snapshot (6).
	got, err = repo.GetMatchNeighbors(ctx, pTestXUID, "n4")
	if err != nil {
		t.Fatalf("GetMatchNeighbors(n4): %v", err)
	}
	n3, n5 := "n3", "n5"
	assertNeighbors(t, got, &n3, &n5, 4, 8)

	// Même source pour la variante filtrée : la playlist « Ranked Slayer » (n8, n6, n4, n1).
	spec := &domain.MatchFilterSpec{PlaylistNames: []string{"Ranked Slayer"}}
	got, err = repo.GetMatchNeighborsFiltered(ctx, pTestXUID, "n8", spec)
	if err != nil {
		t.Fatalf("GetMatchNeighborsFiltered(n8): %v", err)
	}
	n6 := "n6"
	assertNeighbors(t, got, &n6, nil, 0, 4)
}
