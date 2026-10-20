//go:build integration

// Test E2E Phase 1 plan stabilisation 2026-05-22 :
// anti-régression ART sur un dataset multi-matchs.
//
// La chaîne LUSR v2 est couverte par match_recomputer_lusr_v2_integration_test.go (note
// servie, exclusion) et par lusr_components_v2_test.go (composantes), sur des bases migrées.
package sync

import (
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestE2E_ARTPipeline_ProbeDetectsNoCorruption :
// Anti-régression : sur dataset healthy seed, le probe ART ne doit pas
// reporter de divergences. Complémentaire à TestART_FilterPushdown_NoTruncation
// dans le package duckdb.
//
// Note : ce test n'utilise PAS le probe (interne au package duckdb), mais
// vérifie la propriété fondamentale : COUNT(pk = ?) == COUNT(pk concat empty = ?)
func TestE2E_ARTPipeline_ProbeDetectsNoCorruption(t *testing.T) {
	db := openLUSRDB(t)

	// Seed 5 matchs avec 5 participants chacun
	matchIDs := []string{"m_a", "m_b", "m_c", "m_d", "m_e"}
	for _, mid := range matchIDs {
		if _, err := db.Exec(
			`INSERT INTO match_registry (match_id, start_time, playlist_name, pair_name, is_ranked, is_firefight, duration_seconds)
			 VALUES (?, '2025-01-01 10:00:00'::TIMESTAMPTZ, 'Quick Play', 'Slayer', FALSE, FALSE, 600)`,
			mid,
		); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 5; j++ {
			if _, err := db.Exec(
				`INSERT INTO match_participants
					(match_id, xuid, outcome, kills, deaths, assists,
					 kills_expected, deaths_expected, damage_dealt, damage_taken,
					 accuracy, team_id)
				 VALUES (?, ?, 2, 10, 5, 2, 10, 5, 2000, 1000, 0.5, 0)`,
				mid, mid+"_p"+string(rune('0'+j)),
			); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Vérification ART : pk lookup vs table scan
	for _, mid := range matchIDs {
		var indexed, scan int
		if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, mid).Scan(&indexed); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id || '' = ?`, mid).Scan(&scan); err != nil {
			t.Fatal(err)
		}
		if indexed != scan {
			t.Errorf("ART divergence detected for match %s : indexed=%d, scan=%d (bug INCIDENT_2026-05-20)",
				mid, indexed, scan)
		}
		if indexed != 5 {
			t.Errorf("match %s : expected 5 rows, got indexed=%d scan=%d", mid, indexed, scan)
		}
	}
}
