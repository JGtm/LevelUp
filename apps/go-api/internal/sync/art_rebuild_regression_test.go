//go:build integration

// Package sync — art_rebuild_regression_test.go : tests de régression E2E
// pour la chaîne complete ART (Phase 5.5 du plan stabilisation 2026-05-22).
//
// Couverture :
//   1. ProbeARTDivergences ne reporte rien sur un dataset sain (anti-régression
//      du détecteur).
//   2. RebuildMatchParticipantsART (Phase 4.1) préserve toutes les rows +
//      la PK + les vues sur un dataset multi-match.
//   3. Re-probe POST-rebuild reste clean (l'opération n'introduit pas de
//      divergence elle-même — chunk en marche d'escalier).
//
// On ne peut PAS forcer une corruption ART déterministe en :memory: (le bug
// dépend de combinaisons de données non reproductibles cross-runs). Le test
// se concentre donc sur la **mécanique** : rebuild appliqué sur dataset sain
// reste sain. C'est suffisant pour bloquer toute régression du flow
// auto-heal (commit 20c23eda) qui pourrait casser la chaîne probe → rebuild
// → re-probe.

package sync

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// seedFullSchemaForARTRebuild crée le schéma complet requis par
// RebuildMatchParticipantsART + ses vues dépendantes (xuid_aliases,
// match_registry avec TOUTES les colonnes référencées par mv_player_matches).
// Réplique pragmatique du seed migration/steps_shared_rebuild_match_participants_test.go.
func seedFullSchemaForARTRebuild(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			gamertag VARCHAR,
			team_id INTEGER,
			outcome INTEGER,
			rank INTEGER,
			score INTEGER,
			kills INTEGER DEFAULT 0,
			deaths INTEGER DEFAULT 0,
			assists INTEGER DEFAULT 0,
			shots_fired INTEGER DEFAULT 0,
			shots_hit INTEGER DEFAULT 0,
			damage_dealt DOUBLE DEFAULT 0,
			damage_taken DOUBLE DEFAULT 0,
			kd DOUBLE DEFAULT 0,
			kda DOUBLE DEFAULT 0,
			accuracy DOUBLE DEFAULT 0,
			personal_score INTEGER DEFAULT 0,
			time_played_seconds INTEGER DEFAULT 0,
			avg_life_seconds DOUBLE DEFAULT 0,
			headshot_kills SMALLINT DEFAULT 0,
			max_killing_spree SMALLINT DEFAULT 0,
			grenade_kills SMALLINT DEFAULT 0,
			melee_kills SMALLINT DEFAULT 0,
			power_weapon_kills SMALLINT DEFAULT 0,
			kills_expected DOUBLE,
			deaths_expected DOUBLE,
			team_mmr DOUBLE,
			enemy_mmr DOUBLE,
			backfill_bits INTEGER DEFAULT 0,
			created_at TIMESTAMP,
			PRIMARY KEY (match_id, xuid)
		);
		CREATE TABLE sync_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR,
			updated_at TIMESTAMP
		);
		CREATE TABLE xuid_aliases (
			xuid VARCHAR PRIMARY KEY,
			gamertag VARCHAR,
			last_seen TIMESTAMP,
			source VARCHAR DEFAULT 'sync',
			updated_at TIMESTAMP
		);
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			start_time_utc TIMESTAMP,
			end_time_utc TIMESTAMP,
			playlist_id VARCHAR,
			playlist_name VARCHAR,
			playlist_name_fr VARCHAR,
			map_id VARCHAR,
			map_name VARCHAR,
			map_name_fr VARCHAR,
			pair_name VARCHAR,
			pair_name_fr VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			game_variant_name VARCHAR,
			mode_category VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER,
			playable_duration_seconds INTEGER,
			team_0_score SMALLINT,
			team_1_score SMALLINT,
			team_0_ps_score INTEGER,
			team_1_ps_score INTEGER,
			player_count INTEGER
		);
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
}

// seedNMatchesNParticipants insère n matchs × p participants. Format des
// match_ids "rebuild-%04d" pour traçabilité. Le test est anti-corruption :
// si l'ART corrompait quelque chose post-rebuild, le COUNT diviserait.
func seedNMatchesNParticipants(t *testing.T, db *sql.DB, nMatches, pPerMatch int) {
	t.Helper()
	for i := 0; i < nMatches; i++ {
		mid := fmt.Sprintf("rebuild-%04d", i)
		if _, err := db.Exec(`
			INSERT INTO match_registry (match_id, start_time, playlist_name, pair_name, is_ranked)
			VALUES (?, NOW(), 'Quick Play', 'Slayer', FALSE)`, mid); err != nil {
			t.Fatalf("insert match_registry[%d]: %v", i, err)
		}
		for j := 0; j < pPerMatch; j++ {
			xuid := fmt.Sprintf("xuid-%d-%d", i, j)
			if _, err := db.Exec(`
				INSERT INTO match_participants
					(match_id, xuid, gamertag, team_id, outcome, rank, score, kills, deaths, assists, created_at)
				VALUES (?, ?, ?, ?, 2, ?, 1000, 10, 5, 3, NOW())`,
				mid, xuid, fmt.Sprintf("player_%d_%d", i, j), j%2, j+1); err != nil {
				t.Fatalf("insert match_participants[%d,%d]: %v", i, j, err)
			}
		}
	}
}

// TestART_RebuildRegression_ProbeCleanBeforeAndAfter : sur dataset sain :
//   - Probe pré-rebuild → no divergence.
//   - Rebuild → préservation rows + PK + vues.
//   - Probe post-rebuild → toujours no divergence.
//
// Anti-régression du flow auto-heal complet (Phase 4.1 commit 20c23eda).
func TestART_RebuildRegression_ProbeCleanBeforeAndAfter(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	seedFullSchemaForARTRebuild(t, db)
	const nMatches = 10
	const pPerMatch = 8
	seedNMatchesNParticipants(t, db, nMatches, pPerMatch)

	ctx := context.Background()

	// 1. Probe pré-rebuild — dataset healthy, no divergence attendue.
	preReport, err := duckdb.ProbeARTDivergences(ctx, db, 5)
	if err != nil {
		t.Fatalf("probe pré-rebuild: %v", err)
	}
	if preReport.HasDivergence() {
		t.Errorf("dataset healthy seedé : probe rapporte une divergence (probe casse ?) : %+v",
			preReport.Divergences)
	}
	if preReport.TablesScanned == 0 {
		t.Error("probe a scanné 0 tables — pas de table avec PK VARCHAR détectée ?")
	}

	// 2. Compter les rows pré-rebuild pour vérif d'invariance.
	var rowsBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rowsBefore); err != nil {
		t.Fatalf("count before: %v", err)
	}
	wantRows := nMatches * pPerMatch
	if rowsBefore != wantRows {
		t.Fatalf("seed attendu %d rows, got %d", wantRows, rowsBefore)
	}

	// 3. Rebuild via le path runtime exporté (commit d2ca98ce).
	if err := migration.RebuildMatchParticipantsART(ctx, db); err != nil {
		t.Fatalf("RebuildMatchParticipantsART: %v", err)
	}

	// 4. Invariants post-rebuild :
	//    - Row count identique.
	//    - PK toujours active (INSERT duplicate doit fail).
	//    - v_gamertag_lookup existe.
	//    - Probe re-run : toujours no divergence.
	var rowsAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&rowsAfter); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if rowsAfter != rowsBefore {
		t.Errorf("row count changé post-rebuild : before=%d after=%d", rowsBefore, rowsAfter)
	}

	// PK reconstruite : INSERT dupliqué doit échouer.
	_, dupErr := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, rank, score, kills, deaths, assists, created_at)
		VALUES ('rebuild-0000', 'xuid-0-0', 'dup', 0, 2, 1, 0, 0, 0, 0, NOW())`)
	if dupErr == nil {
		t.Error("PK absente post-rebuild : INSERT dupliqué a réussi")
	}

	// v_gamertag_lookup recréée par applyResolutionViews.
	var viewCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.views
		WHERE table_schema = 'main' AND table_name = 'v_gamertag_lookup'`).Scan(&viewCount); err != nil {
		t.Fatalf("query views: %v", err)
	}
	if viewCount != 1 {
		t.Errorf("v_gamertag_lookup absent post-rebuild (count=%d)", viewCount)
	}

	// Re-probe : pas de divergence introduite par le rebuild.
	postReport, err := duckdb.ProbeARTDivergences(ctx, db, 5)
	if err != nil {
		t.Fatalf("probe post-rebuild: %v", err)
	}
	if postReport.HasDivergence() {
		t.Errorf("rebuild a introduit une divergence ART : %+v", postReport.Divergences)
	}
}

// TestART_RebuildRegression_PreservesRowsPerMatch : invariant fin grain — chaque
// match doit avoir EXACTEMENT le même nombre de participants pré et post rebuild.
// Détecte les pertes silencieuses où le total est OK mais la distribution
// par match a basculé (signal d'un swap d'index pendant le CTAS).
func TestART_RebuildRegression_PreservesRowsPerMatch(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	seedFullSchemaForARTRebuild(t, db)
	const nMatches = 20
	const pPerMatch = 6
	seedNMatchesNParticipants(t, db, nMatches, pPerMatch)

	// Snapshot pré-rebuild : map match_id → count via table-scan (`|| ''`)
	// pour court-circuiter l'ART au cas où.
	before := make(map[string]int, nMatches)
	rows, err := db.Query(`
		SELECT match_id || '', COUNT(*) FROM match_participants
		GROUP BY match_id || ''`)
	if err != nil {
		t.Fatalf("snapshot pré: %v", err)
	}
	for rows.Next() {
		var mid string
		var n int
		if err := rows.Scan(&mid, &n); err != nil {
			t.Fatal(err)
		}
		before[mid] = n
	}
	rows.Close()

	if err := migration.RebuildMatchParticipantsART(context.Background(), db); err != nil {
		t.Fatalf("RebuildMatchParticipantsART: %v", err)
	}

	// Snapshot post-rebuild.
	after := make(map[string]int, nMatches)
	rows2, err := db.Query(`
		SELECT match_id || '', COUNT(*) FROM match_participants
		GROUP BY match_id || ''`)
	if err != nil {
		t.Fatalf("snapshot post: %v", err)
	}
	for rows2.Next() {
		var mid string
		var n int
		if err := rows2.Scan(&mid, &n); err != nil {
			t.Fatal(err)
		}
		after[mid] = n
	}
	rows2.Close()

	if len(before) != len(after) {
		t.Errorf("nombre de match_ids distincts changé : before=%d after=%d",
			len(before), len(after))
	}
	for mid, nBefore := range before {
		nAfter, ok := after[mid]
		if !ok {
			t.Errorf("match %s perdu post-rebuild", mid)
			continue
		}
		if nBefore != nAfter {
			t.Errorf("match %s : count changé %d → %d", mid, nBefore, nAfter)
		}
	}
}
