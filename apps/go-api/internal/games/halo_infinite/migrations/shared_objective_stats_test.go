package migrations

// shared_objective_stats_test.go — couverture P1 (PLAN_V72_OBJECTIVE_STATS) : la table
// match_objective_stats, créée DIRECTEMENT en forme append-only par le step
// shared_create_objective_stats (TargetShared), existe avec sa vue _latest, son index
// match_id, et respecte la sémantique append-only (dernière version par (match_id,xuid)).

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func TestSharedObjectiveStatsAppendOnlyShape(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	// Table + vue présentes.
	var nTable int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'match_objective_stats' AND table_type = 'BASE TABLE'",
	).Scan(&nTable); err != nil {
		t.Fatalf("query table: %v", err)
	}
	if nTable != 1 {
		t.Fatalf("table match_objective_stats absente après migration")
	}
	var nView int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'match_objective_stats_latest' AND table_type = 'VIEW'",
	).Scan(&nView); err != nil {
		t.Fatalf("query view: %v", err)
	}
	if nView != 1 {
		t.Errorf("vue match_objective_stats_latest absente")
	}

	// Colonnes append-only techniques + un échantillon par famille de mode.
	for _, col := range []string{
		"id", "match_id", "xuid", "written_at",
		"flag_captures", "time_as_flag_carrier_seconds",
		"zone_captures", "zone_scoring_ticks", "time_in_zones_seconds",
		"skull_grabs", "time_as_skull_carrier_seconds", "longest_time_as_skull_carrier_seconds",
	} {
		var n int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'match_objective_stats' AND column_name = ?", col,
		).Scan(&n); err != nil {
			t.Fatalf("query column %s: %v", col, err)
		}
		if n != 1 {
			t.Errorf("colonne match_objective_stats.%s absente", col)
		}
	}

	// Index match_id présent (perf JOIN scoreboard).
	var nIdx int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM duckdb_indexes() WHERE index_name = 'idx_match_objective_stats_match'`,
	).Scan(&nIdx); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if nIdx != 1 {
		t.Errorf("index idx_match_objective_stats_match absent")
	}

	// Sémantique append-only : 2 INSERT pour la même clé (match_id,xuid) → la vue _latest
	// ne garde que la version au written_at le plus récent (id le plus grand en tie-break).
	if _, err := db.Exec(`
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at)
			VALUES ('m1', 'x1', 1, TIMESTAMP '2026-01-01 00:00:00');
		INSERT INTO match_objective_stats (match_id, xuid, flag_captures, written_at)
			VALUES ('m1', 'x1', 3, TIMESTAMP '2026-01-02 00:00:00');
	`); err != nil {
		t.Fatalf("insert append-only rows: %v", err)
	}
	var rawCount, latestVal int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_objective_stats WHERE match_id='m1' AND xuid='x1'`).Scan(&rawCount); err != nil {
		t.Fatalf("count raw: %v", err)
	}
	if rawCount != 2 {
		t.Errorf("table brute = %d lignes, want 2 (append-only garde l'historique)", rawCount)
	}
	if err := db.QueryRow(`SELECT flag_captures FROM match_objective_stats_latest WHERE match_id='m1' AND xuid='x1'`).Scan(&latestVal); err != nil {
		t.Fatalf("query latest: %v", err)
	}
	if latestVal != 3 {
		t.Errorf("vue _latest flag_captures = %d, want 3 (dernière version par written_at)", latestVal)
	}
}
