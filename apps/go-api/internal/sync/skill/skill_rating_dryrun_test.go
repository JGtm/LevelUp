//go:build integration

// Tests pour BatchComputeLUSRPreview (Phase 1.7 plan stabilisation 2026-05-22).
//
// Objectifs :
//   - Preview retourne un rapport non-vide pour des données existantes
//   - Preview N'ÉCRIT PAS dans match_skill_rank (table inchangée avant/après)
//   - Le delta MU est non-nul quand pas de seed (LUSR vierge → +N points)
//   - L'agrégation par playlist_group est correcte
package skill

import (
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestBatchComputeLUSRPreview_NoWrite : preview ne modifie pas match_skill_rank.
func TestBatchComputeLUSRPreview_NoWrite(t *testing.T) {
	db := openLUSRDB(t)
	// 2 matchs sociaux (LUSR éligibles) pour xuid1, avec participants opposés.
	if _, err := db.Exec(`INSERT INTO match_registry VALUES
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, NULL, 'Quick Play', 'Slayer', FALSE, FALSE, 600),
		('m2', '2025-01-02 10:00:00'::TIMESTAMPTZ, NULL, 'Quick Play', 'Slayer', FALSE, FALSE, 600)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants
		(match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected,
		 damage_dealt, damage_taken, accuracy, team_id)
		VALUES
		('m1', 'xuid1', 2, 15, 5, 3, 12.0, 6.0, 3000.0, 1500.0, 0.55, 0),
		('m1', 'xuid_opp', 3, 8, 12, 1, 10.0, 8.0, 1800.0, 2500.0, 0.40, 1),
		('m2', 'xuid1', 2, 18, 4, 5, 14.0, 5.0, 3500.0, 1200.0, 0.58, 0),
		('m2', 'xuid_opp', 3, 6, 13, 1, 9.0, 8.0, 1500.0, 2700.0, 0.38, 1)`); err != nil {
		t.Fatal(err)
	}

	// Snapshot table match_skill_rank avant preview.
	var beforeCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&beforeCount); err != nil {
		t.Fatal(err)
	}

	// Lancer le preview.
	report, err := BatchComputeLUSRPreview(t.Context(), db, db, "xuid1", nil)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}

	// Vérifier que match_skill_rank n'a PAS bougé.
	var afterCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&afterCount); err != nil {
		t.Fatal(err)
	}
	if afterCount != beforeCount {
		t.Fatalf("preview a modifié match_skill_rank : before=%d after=%d (devrait être 0 écriture)",
			beforeCount, afterCount)
	}

	// Le rapport doit contenir des données.
	if report.MatchesProcessed != 2 {
		t.Errorf("expected 2 matches processed, got %d", report.MatchesProcessed)
	}
	if len(report.Playlists) == 0 {
		t.Errorf("expected ≥1 playlist_group in report, got 0")
	}
}

// TestBatchComputeLUSRPreview_EmptyData : aucun match → rapport vide.
func TestBatchComputeLUSRPreview_EmptyData(t *testing.T) {
	db := openLUSRDB(t)
	report, err := BatchComputeLUSRPreview(t.Context(), db, db, "xuid_nobody", nil)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if report.MatchesProcessed != 0 {
		t.Errorf("expected 0 matches, got %d", report.MatchesProcessed)
	}
	if len(report.Playlists) != 0 {
		t.Errorf("expected 0 playlists, got %d", len(report.Playlists))
	}
	if report.HasChanges() {
		t.Errorf("expected HasChanges()=false on empty report")
	}
}

// TestBatchComputeLUSRPreview_DetectsChanges : LUSR vierge (pas de seed) →
// les nouvelles valeurs diffèrent de zéro → HasChanges()=true.
func TestBatchComputeLUSRPreview_DetectsChanges(t *testing.T) {
	db := openLUSRDB(t)
	if _, err := db.Exec(`INSERT INTO match_registry VALUES
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, NULL, 'Quick Play', 'Slayer', FALSE, FALSE, 600)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants
		(match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected,
		 damage_dealt, damage_taken, accuracy, team_id)
		VALUES
		('m1', 'xuid1', 2, 20, 3, 5, 12.0, 6.0, 4000.0, 1000.0, 0.65, 0),
		('m1', 'xuid_opp', 3, 5, 15, 0, 10.0, 9.0, 1200.0, 3000.0, 0.30, 1)`); err != nil {
		t.Fatal(err)
	}

	report, err := BatchComputeLUSRPreview(t.Context(), db, db, "xuid1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !report.HasChanges() {
		t.Errorf("expected HasChanges()=true (LUSR vierge → DeltaMU > 1.0), got false ; playlists=%v",
			report.Playlists)
	}
	for _, p := range report.Playlists {
		if p.OldMU != 0 {
			t.Errorf("playlist %s : OldMU expected 0 (pas de seed), got %v", p.PlaylistGroup, p.OldMU)
		}
		if p.NewMU == 0 {
			t.Errorf("playlist %s : NewMU should be non-zero after compute", p.PlaylistGroup)
		}
		if p.MatchCount < 1 {
			t.Errorf("playlist %s : MatchCount should be ≥1, got %d", p.PlaylistGroup, p.MatchCount)
		}
	}
}
