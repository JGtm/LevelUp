//go:build integration

package main

// cmd_backfill_killsource_selection_placement_integration_test.go — LA FRAICHEUR DU PLACEMENT DES
// VIES DANS LA SELECTION DU RATTRAPAGE (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, decision V12).
//
// `matchsAJour` exige, pour tout match qui a des vies, une passe de `match_life_placement_latest`
// a `killcollector.PlacementRev`. La requete tourne ici sur une vraie base migree : les quatre
// cas ci-dessous ne peuvent pas mentir sur la syntaxe ni sur l'ensemble selectionne.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync/killcollector"
)

// baseDeSelection ouvre un shared migre par les VRAIES migrations.
func baseDeSelection(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "selection.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

// executer : un ordre de fixture, qui echoue le test s'il echoue.
func executer(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("%s : %v", q, err)
	}
}

// matchAJourDeSesVies inscrit un match dont le journal ET les faits d'isolement portent leurs
// revisions courantes : positions presentes, equipes en base, vies a IsolationDecoderRev.
func matchAJourDeSesVies(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	executer(t, db, `INSERT INTO match_kill_events (match_id, decode_pass, decoder_rev, time_ms,
		victim_gamertag, feed_present, assist_known, publishable, read_path, read_origin)
		VALUES (?, 'k1', ?, 1000, 'V', TRUE, TRUE, TRUE, ?, 'concordant-test')`,
		id, decfilm.Rev, killscope.ReadPathFilmWalk)
	executer(t, db, `INSERT INTO kill_positions (match_id, decode_pass, killer_xuid, time_ms)
		VALUES (?, 'p1', '111', 1000)`, id)
	executer(t, db, `INSERT INTO match_participants (match_id, xuid, team_id) VALUES (?, '111', 0)`, id)
	executer(t, db, `INSERT INTO match_lives (match_id, decode_pass, decoder_rev, xuid, start_ms,
		end_ms, end_cause, named_by) VALUES (?, 'v1', ?, '111', 0, 10000, 'death', 'death')`,
		id, killcollector.IsolationDecoderRev)
}

// placementA ecrit une passe de placement d'une ligne, a la revision `rev`.
func placementA(t *testing.T, db *sql.DB, id, rev string) {
	t.Helper()
	executer(t, db, `INSERT INTO match_life_placement (match_id, decode_pass, decoder_rev, xuid,
		start_ms, end_ms, duration_ms, measured_ms, carrier_ms, team_down_ms, unplaced_ms,
		teammate_unplaced_ms, kills) VALUES (?, 'pl1', ?, '111', 0, 10000, 10000, 10100, 0, 0, 0, 0, 0)`,
		id, rev)
}

// TestMatchsAJour_ExigeLePlacementDesVies — decision V12.
//
//	sans-placement     des vies, aucun placement           -> a redecoder
//	placement-perime   des vies, placement a une autre rev -> a redecoder
//	placement-courant  des vies, placement a PlacementRev  -> a jour
//	sans-vies          journal a jour, ni positions ni vies -> a jour (le placement ne peut naitre)
func TestMatchsAJour_ExigeLePlacementDesVies(t *testing.T) {
	db := baseDeSelection(t)
	for _, id := range []string{"sans-placement", "placement-perime", "placement-courant"} {
		matchAJourDeSesVies(t, db, id)
	}
	placementA(t, db, "placement-perime", "placement-2020-01-01")
	placementA(t, db, "placement-courant", killcollector.PlacementRev)
	executer(t, db, `INSERT INTO match_kill_events (match_id, decode_pass, decoder_rev, time_ms,
		victim_gamertag, feed_present, assist_known, publishable, read_path, read_origin)
		VALUES ('sans-vies', 'k1', ?, 1000, 'V', TRUE, TRUE, TRUE, ?, 'concordant-test')`,
		decfilm.Rev, killscope.ReadPathFilmWalk)

	aJour, err := matchsAJour(context.Background(), db)
	if err != nil {
		t.Fatalf("matchsAJour: %v", err)
	}
	attendu := map[string]bool{"placement-courant": true, "sans-vies": true}
	for _, id := range []string{"sans-placement", "placement-perime", "placement-courant", "sans-vies"} {
		if aJour[id] != attendu[id] {
			t.Errorf("%s : a jour = %v, attendu %v", id, aJour[id], attendu[id])
		}
	}
}
