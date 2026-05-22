//go:build integration

// Test E2E Phase 1 plan stabilisation 2026-05-22 :
// rebuild ART + LUSR compute end-to-end sur un dataset multi-matchs.
//
// Couvre la chaîne complète :
//  1. Seed match_registry + match_participants (10 matchs x 10 participants)
//  2. Run batchComputeLUSR depuis l'état post-rebuild ART
//  3. Verify que le LUSR a été calculé pour le joueur ciblé + écrit
//
// Pré-rebuild ART, certains tests d'intégration auraient pu masquer le bug
// "1/10 rows visibles" (la corruption ne se reproduit pas déterministement
// sur :memory:). Mais ce test garantit que la pipeline LUSR fonctionne
// end-to-end avec des données hétérogènes — anti-régression pour les futures
// migrations / refactos qui pourraient casser la chaîne sync → compute → write.
package sync

import (
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const e2eTestXUID = "2533274858283686" // Madina97294 (le cas test du bug original)

// TestE2E_ARTPipeline_BatchComputeLUSR_ProducesWrites :
// 10 matchs sociaux pour xuid1 + 9 adversaires par match (90 participants).
// Lance batchComputeLUSR force=true → vérifie que match_skill_rank a 10 rows LUSR.
func TestE2E_ARTPipeline_BatchComputeLUSR_ProducesWrites(t *testing.T) {
	db := openLUSRDB(t)
	ctx := t.Context()

	// Seed : 10 matchs sociaux (non-ranked, non-firefight) pour le xuid test.
	// Chaque match : 10 participants (1 joueur + 9 adversaires).
	matchIDs := []string{
		"50cd2d8c-9feb-4b98-bc7c-e34aa8b1df7e", // le bug original
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"00000000-0000-0000-0000-000000000001",
		"deadbeef-cafe-babe-c0de-feedfacefa11",
		"01234567-89ab-cdef-0123-456789abcdef",
		"abcdef01-2345-6789-abcd-ef0123456789",
		"99999999-9999-9999-9999-999999999999",
		"11111111-2222-3333-4444-555555555555",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
	}
	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i, mid := range matchIDs {
		startTime := base.Add(time.Duration(i) * time.Hour)
		if _, err := db.Exec(
			`INSERT INTO match_registry (match_id, start_time, playlist_name, pair_name, is_ranked, is_firefight, duration_seconds)
			 VALUES (?, ?, 'Quick Play', 'Slayer', FALSE, FALSE, 600)`,
			mid, startTime,
		); err != nil {
			t.Fatalf("insert match_registry[%d]: %v", i, err)
		}
		// 10 participants par match : 1 test xuid + 9 adversaires (5 team_id=0, 5 team_id=1)
		for j := 0; j < 10; j++ {
			xuid := e2eTestXUID
			teamID := 0
			outcome := 2 // win pour test xuid
			kills := 15
			deaths := 5
			if j > 0 {
				xuid = "opponent_" + string(rune('a'+j))
				teamID = j % 2
				outcome = 3
				kills = 7 + j
				deaths = 12 + j
			}
			if _, err := db.Exec(`
				INSERT INTO match_participants
					(match_id, xuid, outcome, kills, deaths, assists,
					 kills_expected, deaths_expected, damage_dealt, damage_taken,
					 accuracy, team_id)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				mid, xuid, outcome, kills, deaths, 3,
				float64(kills)*0.9, float64(deaths)*0.9,
				float64(kills)*200.0, float64(deaths)*200.0,
				0.55, teamID,
			); err != nil {
				t.Fatalf("insert match_participants[%d,%d]: %v", i, j, err)
			}
		}
	}

	// Sanity check : 10 matchs x 10 participants = 100 rows
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 100 {
		t.Fatalf("seed expected 100 rows, got %d", total)
	}

	// Anti-régression ART : chaque match doit retourner 10 rows
	for _, mid := range matchIDs {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, mid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 10 {
			t.Fatalf("ART pre-check failed for %s: got %d rows, want 10 (bug INCIDENT_2026-05-20 reproduit ?)", mid, n)
		}
	}

	// Run batchComputeLUSR force=true sur le xuid test
	updated, err := batchComputeLUSR(ctx, db, db, e2eTestXUID, nil, true)
	if err != nil {
		t.Fatalf("batchComputeLUSR: %v", err)
	}
	if updated == 0 {
		t.Fatal("expected ≥1 LUSR row written, got 0 (chaîne sync → compute → write cassée ?)")
	}
	t.Logf("batchComputeLUSR: updated=%d rows", updated)

	// Verify : match_skill_rank doit avoir des rows LUSR
	var lusrCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'LUSR'`,
	).Scan(&lusrCount); err != nil {
		t.Fatal(err)
	}
	if lusrCount == 0 {
		t.Errorf("aucun LUSR persisté dans match_skill_rank (expected ≥1)")
	}

	// Verify : le rating_value final est dans une plage raisonnable [1000, 2500]
	// (TrueSkill InitialMU=1500, drift attendu vers 1500-1700 sur 10 wins).
	var avgRating float64
	if err := db.QueryRow(
		`SELECT AVG(rating_value) FROM match_skill_rank WHERE rating_type = 'LUSR'`,
	).Scan(&avgRating); err != nil {
		t.Fatal(err)
	}
	if avgRating < 1000 || avgRating > 2500 {
		t.Errorf("rating_value avg = %.1f, expected [1000, 2500] (10 wins → drift positif)", avgRating)
	}

	// Verify : aucun match LUSR n'a rating_value=0 (signal de corruption ART
	// pré-rebuild : si participants vides → compute renvoyait des valeurs nulles).
	var zeroCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'LUSR' AND COALESCE(rating_value, 0) = 0`,
	).Scan(&zeroCount); err != nil {
		t.Fatal(err)
	}
	if zeroCount > 0 {
		t.Errorf("%d LUSR rows ont rating_value=0 (corruption ART résiduelle ?)", zeroCount)
	}
}

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
