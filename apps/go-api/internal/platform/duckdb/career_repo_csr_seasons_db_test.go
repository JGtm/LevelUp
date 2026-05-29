//go:build integration

package duckdb

import (
	"context"
	"testing"
)

// insertCSRSnapshot insère une ligne player_csr_snapshots (append-only) pour les tests.
func insertCSRSnapshot(t *testing.T, pdb *PlayerDB, playlistID, seasonID, tier string) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(), `INSERT INTO player_csr_snapshots
		(playlist_id, playlist_name, queue, input, season_id,
		 current_value, current_tier, current_sub_tier,
		 season_value, season_tier, season_sub_tier,
		 alltime_value, alltime_tier, alltime_sub_tier)
		VALUES (?, ?, '', '', ?, 1450.0, ?, 3, 1500.0, ?, 4, 1700.0, ?, 1)`,
		playlistID, playlistID, seasonID, tier, tier, tier); err != nil {
		t.Fatalf("seed snapshot %s/%s: %v", playlistID, seasonID, err)
	}
}

// TestAvailableCSRSeasons_DistinctPlusCurrent : 2 saisons en base + saison
// courante → liste triée récente d'abord, flag is_current correct.
func TestAvailableCSRSeasons_DistinctPlusCurrent(t *testing.T) {
	pdb := newTestPlayerDB(t)
	insertCSRSnapshot(t, pdb, "ranked-arena", "CsrSeason12-1", "Gold")
	insertCSRSnapshot(t, pdb, "ranked-arena", "CsrSeason13-1", "Platinum")
	repo := NewCareerRepo(pdb).WithCSRThresholds(NewCSRThresholdsRepo(pdb.Metadata), "CsrSeason13-1")

	seasons, err := repo.AvailableCSRSeasons(context.Background())
	if err != nil {
		t.Fatalf("AvailableCSRSeasons: %v", err)
	}
	if len(seasons) != 2 {
		t.Fatalf("attendu 2 saisons, obtenu %d : %+v", len(seasons), seasons)
	}
	// Tri récent d'abord : 13-1 avant 12-1.
	if seasons[0].SeasonID != "CsrSeason13-1" || seasons[1].SeasonID != "CsrSeason12-1" {
		t.Errorf("ordre inattendu : %s, %s", seasons[0].SeasonID, seasons[1].SeasonID)
	}
	if !seasons[0].IsCurrent || seasons[1].IsCurrent {
		t.Errorf("is_current mal positionné : %+v", seasons)
	}
	if seasons[0].Label != "Saison 13" {
		t.Errorf("label = %q, attendu 'Saison 13'", seasons[0].Label)
	}
}

// TestAvailableCSRSeasons_CurrentAlwaysIncluded : aucune donnée mais saison
// courante configurée → la saison courante apparaît quand même.
func TestAvailableCSRSeasons_CurrentAlwaysIncluded(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb).WithCSRThresholds(NewCSRThresholdsRepo(pdb.Metadata), "CsrSeason13-1")
	seasons, err := repo.AvailableCSRSeasons(context.Background())
	if err != nil {
		t.Fatalf("AvailableCSRSeasons: %v", err)
	}
	if len(seasons) != 1 || seasons[0].SeasonID != "CsrSeason13-1" || !seasons[0].IsCurrent {
		t.Fatalf("attendu [CsrSeason13-1 current], obtenu %+v", seasons)
	}
}

// TestGetCSRSnapshots_SeasonFilter : 2 snapshots même playlist, saisons
// différentes → le filtre saison ne renvoie que la saison demandée (pas de
// supplantation par une saison passée).
func TestGetCSRSnapshots_SeasonFilter(t *testing.T) {
	pdb := newTestPlayerDB(t)
	insertCSRSnapshot(t, pdb, "ranked-arena", "CsrSeason12-1", "Gold")
	insertCSRSnapshot(t, pdb, "ranked-arena", "CsrSeason13-1", "Platinum")
	repo := NewCareerRepo(pdb).WithCSRThresholds(NewCSRThresholdsRepo(pdb.Metadata), "CsrSeason13-1")

	cur, err := repo.GetCSRSnapshots(context.Background(), "CsrSeason13-1")
	if err != nil {
		t.Fatalf("GetCSRSnapshots S13: %v", err)
	}
	if len(cur) != 1 || cur[0].Current.Tier != "Platinum" {
		t.Fatalf("S13 attendu 1 snapshot Platinum, obtenu %+v", cur)
	}
	past, err := repo.GetCSRSnapshots(context.Background(), "CsrSeason12-1")
	if err != nil {
		t.Fatalf("GetCSRSnapshots S12: %v", err)
	}
	if len(past) != 1 || past[0].Current.Tier != "Gold" {
		t.Fatalf("S12 attendu 1 snapshot Gold, obtenu %+v", past)
	}
}
