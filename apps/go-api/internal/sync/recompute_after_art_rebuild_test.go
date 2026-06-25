//go:build integration

// Package sync — recompute_after_art_rebuild_test.go : tests TDD pour le
// wrapper RecomputeAfterARTRebuild (plan stabilisation 2026-05-22 §4.4).
//
// Contexte : pendant la période où l'ART était corrompue, les batchs computed
// sur les rows partiellement visibles ont produit des résultats FAUX (ex.
// LUSR Madina figé à Argent IV au lieu de Platine). Phase 4.1 répare l'ART
// au boot, mais les valeurs dérivées restent figées. Phase 4.4 fournit le
// wrapper orchestrateur qui recalcule force=true pour un joueur :
//   - batchComputeLUSR (LUSR cascade)
//   - BatchComputePerformanceScores
//   - BackfillDominanceFlags
//   - RecomputeIsWithFriendsCore (skip si friend list vide)
//
// **TDD strict** : ces tests définissent le contrat AVANT impl. Ils doivent
// échouer baseline (undefined) puis passer post-impl.

package sync

import (
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openRecomputeDB seed un schéma complet pour tester RecomputeAfterARTRebuild :
// match_registry + match_participants + match_skill_rank + player_match_enrichment
// + medals_earned (pour BackfillDominanceFlags) + xuid_aliases (pour future
// extension friends recompute).
func openRecomputeDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			start_time_utc TIMESTAMPTZ,
			playlist_name VARCHAR,
			pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			outcome INTEGER,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			kda DOUBLE, accuracy DOUBLE,
			time_played_seconds INTEGER,
			personal_score INTEGER, damage_dealt DOUBLE, damage_taken DOUBLE,
			rank INTEGER,
			team_mmr DOUBLE, enemy_mmr DOUBLE,
			kills_expected DOUBLE, deaths_expected DOUBLE,
			team_id INTEGER
		);
		CREATE TABLE match_skill_rank (
			match_id         VARCHAR PRIMARY KEY,
			rating_type      VARCHAR NOT NULL,
			rating_value     DOUBLE,
			rating_deviation DOUBLE,
			tier             VARCHAR,
			tier_fr          VARCHAR,
			sub_tier         SMALLINT DEFAULT 0,
			tier_label       VARCHAR,
			rating_delta     DOUBLE,
			playlist_group   VARCHAR,
			expected_win_prob FLOAT,
			start_time       TIMESTAMPTZ,
			created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			performance_chain VARCHAR,
			dominance_flag INTEGER,
			updated_at TIMESTAMPTZ
		);
		CREATE TABLE medals_earned (
			match_id VARCHAR,
			xuid VARCHAR,
			medal_name_id BIGINT,
			count INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR
		);
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	return db
}

// seedRecomputeMatches insère N matchs sociaux (non-ranked, non-firefight)
// pour le xuid test, avec 10 participants chacun (1 cible + 9 adversaires).
// Win pour le xuid cible (outcome=2) → drift LUSR positif attendu.
func seedRecomputeMatches(t *testing.T, db *sql.DB, n int, xuid string) []string {
	t.Helper()
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		mid := fmt.Sprintf("rec-%04d", i)
		ids = append(ids, mid)
		ts := fmt.Sprintf("2025-02-%02dT%02d:00:00Z", (i/24)+1, i%24)
		if _, err := db.Exec(
			`INSERT INTO match_registry (match_id, start_time, pair_name, playlist_name, is_ranked, is_firefight, duration_seconds)
			 VALUES (?, ?::TIMESTAMPTZ, ?, 'Quick Play', FALSE, FALSE, 600)`,
			mid, ts, "Arena:Slayer"); err != nil {
			t.Fatalf("insert match_registry[%d]: %v", i, err)
		}
		for j := 0; j < 10; j++ {
			thisXUID := xuid
			teamID := 0
			outcome := 2
			kills := 15
			deaths := 5
			if j > 0 {
				thisXUID = fmt.Sprintf("opp_%d_%d", i, j)
				teamID = j % 2
				outcome = 3
				kills = 7 + j
				deaths = 12 + j
			}
			if _, err := db.Exec(`
				INSERT INTO match_participants
					(match_id, xuid, outcome, kills, deaths, assists,
					 kda, accuracy, time_played_seconds, personal_score,
					 damage_dealt, damage_taken, rank, team_mmr, enemy_mmr,
					 kills_expected, deaths_expected, team_id)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				mid, thisXUID, outcome, kills, deaths, 3,
				1.5, 0.55, 600, 1000+i,
				float64(kills)*200.0, float64(deaths)*200.0,
				1, 1500.0, 1500.0, 10.0, 5.0, teamID,
			); err != nil {
				t.Fatalf("insert match_participants[%d,%d]: %v", i, j, err)
			}
		}
		// Seed player_match_enrichment row (en prod, posée à l'ingestion
		// initiale du match). Sans cette row, les UPDATE de la cascade
		// (performance, dominance) sont des no-ops silencieux.
		if _, err := db.Exec(
			`INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid,
		); err != nil {
			t.Fatalf("insert player_match_enrichment[%d]: %v", i, err)
		}
	}
	return ids
}

const recomputeTestXUID = "2533274858283686" // même cas test que ART E2E

// TestRecomputeAfterARTRebuild_ProducesAllCascadeOutputs :
// 10 matchs seedés → après recompute, on doit avoir :
//   - 10 lignes match_skill_rank (LUSR)
//   - 10 lignes player_match_enrichment avec performance_score non-null
//   - 10 lignes player_match_enrichment avec dominance_flag (peut être 0 = aucun)
//
// Garantit que les 4 étapes du wrapper sont bien exécutées sur les données.
func TestRecomputeAfterARTRebuild_ProducesAllCascadeOutputs(t *testing.T) {
	db := openRecomputeDB(t)
	// 15 matchs > MinMatchesPerChainForRelative (10) pour que la cascade
	// performance produise des updates (sinon "skipped_below_threshold").
	const nMatches = 15
	seedRecomputeMatches(t, db, nMatches, recomputeTestXUID)

	report, err := RecomputeAfterARTRebuild(t.Context(), db, db, recomputeTestXUID, nil)
	if err != nil {
		t.Fatalf("RecomputeAfterARTRebuild: %v", err)
	}

	// Sanity report
	if report.XUID != recomputeTestXUID {
		t.Errorf("report.XUID = %q, want %q", report.XUID, recomputeTestXUID)
	}
	if report.LUSRUpdated == 0 {
		t.Errorf("LUSRUpdated = 0, attendu >0 (%d matchs sociaux seeded)", nMatches)
	}
	if report.PerformanceUpdated == 0 {
		t.Errorf("PerformanceUpdated = 0, attendu >0 (%d matchs seeded, threshold=10)", nMatches)
	}
	if report.DominanceMatches != nMatches {
		t.Errorf("DominanceMatches = %d, attendu %d (un par match)",
			report.DominanceMatches, nMatches)
	}

	// Vérifs DB
	var lusrCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'LUSR'`,
	).Scan(&lusrCount); err != nil {
		t.Fatal(err)
	}
	if lusrCount == 0 {
		t.Error("aucun LUSR persisté dans match_skill_rank")
	}

	var perfCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`,
	).Scan(&perfCount); err != nil {
		t.Fatal(err)
	}
	if perfCount == 0 {
		t.Error("aucun performance_score persisté dans player_match_enrichment")
	}

	var domCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE dominance_flag IS NOT NULL`,
	).Scan(&domCount); err != nil {
		t.Fatal(err)
	}
	if domCount == 0 {
		t.Error("aucun dominance_flag persisté dans player_match_enrichment")
	}
}

// TestRecomputeAfterARTRebuild_EmptyData_NoOp : pas d'erreur sur DB vide,
// rapport avec compteurs à 0. Important pour le cas où l'auto-heal cible
// un joueur dont la DB existe mais n'a pas (encore) de matchs.
func TestRecomputeAfterARTRebuild_EmptyData_NoOp(t *testing.T) {
	db := openRecomputeDB(t)

	report, err := RecomputeAfterARTRebuild(t.Context(), db, db, "xuid_inexistant", nil)
	if err != nil {
		t.Fatalf("recompute sur DB vide: %v", err)
	}

	if report.LUSRUpdated != 0 {
		t.Errorf("LUSRUpdated = %d sur DB vide, attendu 0", report.LUSRUpdated)
	}
	if report.PerformanceUpdated != 0 {
		t.Errorf("PerformanceUpdated = %d sur DB vide, attendu 0", report.PerformanceUpdated)
	}
	if report.DominanceMatches != 0 {
		t.Errorf("DominanceMatches = %d sur DB vide, attendu 0", report.DominanceMatches)
	}
}

// TestRecomputeAfterARTRebuild_Idempotent : 2e passe sur les mêmes données
// produit les mêmes résultats (force=true override la cache LUSR/perf).
// Critère essentiel : on pourra rappeler le wrapper plusieurs fois sans
// corruption ni accumulation.
func TestRecomputeAfterARTRebuild_Idempotent(t *testing.T) {
	db := openRecomputeDB(t)
	seedRecomputeMatches(t, db, 5, recomputeTestXUID)

	r1, err := RecomputeAfterARTRebuild(t.Context(), db, db, recomputeTestXUID, nil)
	if err != nil {
		t.Fatalf("recompute 1: %v", err)
	}

	// Snapshot du state DB après 1ère passe.
	var lusr1, perf1 int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'LUSR'`,
	).Scan(&lusr1); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`,
	).Scan(&perf1); err != nil {
		t.Fatal(err)
	}

	// 2e passe.
	r2, err := RecomputeAfterARTRebuild(t.Context(), db, db, recomputeTestXUID, nil)
	if err != nil {
		t.Fatalf("recompute 2: %v", err)
	}

	if r1.DominanceMatches != r2.DominanceMatches {
		t.Errorf("DominanceMatches non idempotent : r1=%d r2=%d",
			r1.DominanceMatches, r2.DominanceMatches)
	}

	var lusr2, perf2 int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type = 'LUSR'`,
	).Scan(&lusr2); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE performance_score IS NOT NULL`,
	).Scan(&perf2); err != nil {
		t.Fatal(err)
	}

	if lusr1 != lusr2 {
		t.Errorf("LUSR count non idempotent : r1=%d r2=%d", lusr1, lusr2)
	}
	if perf1 != perf2 {
		t.Errorf("performance count non idempotent : r1=%d r2=%d", perf1, perf2)
	}
}

// TestRecomputeAfterARTRebuild_SkipsFriendsWhenEmpty : friend list vide →
// FriendsResult.FriendXUIDsCount = 0, pas d'erreur. Cas typique au boot
// quand on n'a pas encore chargé la liste d'amis.
func TestRecomputeAfterARTRebuild_SkipsFriendsWhenEmpty(t *testing.T) {
	db := openRecomputeDB(t)
	seedRecomputeMatches(t, db, 3, recomputeTestXUID)

	report, err := RecomputeAfterARTRebuild(t.Context(), db, db, recomputeTestXUID, nil)
	if err != nil {
		t.Fatalf("recompute: %v", err)
	}
	if report.FriendsResult.FriendXUIDsCount != 0 {
		t.Errorf("FriendsResult.FriendXUIDsCount = %d sur friends nil, attendu 0",
			report.FriendsResult.FriendXUIDsCount)
	}
}
