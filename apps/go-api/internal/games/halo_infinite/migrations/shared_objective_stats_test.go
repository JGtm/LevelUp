package migrations

// shared_objective_stats_test.go — couverture P1 (PLAN_V72_OBJECTIVE_STATS) : la table
// match_objective_stats, créée DIRECTEMENT en forme append-only par le step
// shared_create_objective_stats (TargetShared), existe avec sa vue _latest, son index
// match_id, et respecte la sémantique append-only (dernière version par (match_id,xuid)).
//
// V721-02 : + le step shared_objective_stats_add_stockpile_extraction (11 colonnes
// nullable Stockpile/Extraction) et la RECRÉATION de la vue _latest qui l'accompagne.

import (
	"database/sql"
	"strings"
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
		"power_seeds_deposited", "time_as_power_seed_carrier_seconds",
		"successful_extractions",
		"vip_kills", "times_selected_as_vip", "time_as_vip_seconds",
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

// TestSharedObjectiveStatsStockpileExtractionColumns — step
// shared_objective_stats_add_stockpile_extraction (V721-02) : les 18 colonnes
// Stockpile/Extraction/VIP existent sur la TABLE **et** sont servies par la vue _latest.
//
// Le 2e point est le vrai piège : DuckDB fige la liste de colonnes d'un `SELECT *` au
// CREATE VIEW. Sans le `CREATE OR REPLACE VIEW` de la migration, la vue continuerait
// d'exposer les 25 colonnes d'origine (les lecteurs — Q12, ObjectiveStatsRepo — ne
// verraient JAMAIS les nouvelles), voire échouerait. Le SELECT ci-dessous sur la vue
// échoue si la recréation est retirée.
func TestSharedObjectiveStatsStockpileExtractionColumns(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migration.SetTitleStepsProvider(StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	// Les 18 colonnes présentes sur la table, nullable, avec la bonne FAMILLE de type.
	// Préfixe (INT… / DOUBLE…) plutôt qu'égalité stricte : le libellé exact de
	// information_schema.data_type dépend de la version DuckDB, la famille non.
	wantTypePrefix := map[string]string{
		"kills_as_power_seed_carrier":        "INT",
		"power_seed_carriers_killed":         "INT",
		"power_seeds_deposited":              "INT",
		"power_seeds_stolen":                 "INT",
		"time_as_power_seed_carrier_seconds": "DOUBLE",
		"time_as_power_seed_driver_seconds":  "DOUBLE",
		"extraction_conversions_completed":   "INT",
		"extraction_conversions_denied":      "INT",
		"extraction_initiations_completed":   "INT",
		"extraction_initiations_denied":      "INT",
		"successful_extractions":             "INT",
		"kills_as_vip":                       "INT",
		"vip_kills":                          "INT",
		"vip_assists":                        "INT",
		"times_selected_as_vip":              "INT",
		"max_killing_spree_as_vip":           "INT",
		"time_as_vip_seconds":                "DOUBLE",
		"longest_time_as_vip_seconds":        "DOUBLE",
	}
	for col, prefix := range wantTypePrefix {
		var gotType, nullable string
		err := db.QueryRow(`
			SELECT data_type, is_nullable FROM information_schema.columns
			WHERE table_name = 'match_objective_stats' AND column_name = ?`, col).Scan(&gotType, &nullable)
		if err != nil {
			t.Errorf("colonne match_objective_stats.%s absente: %v", col, err)
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(gotType), prefix) {
			t.Errorf("match_objective_stats.%s type = %s, want préfixe %s", col, gotType, prefix)
		}
		if !strings.EqualFold(nullable, "YES") {
			t.Errorf("match_objective_stats.%s is_nullable = %s, want YES (un mode absent = NULL)", col, nullable)
		}
	}

	// Round-trip par la vue _latest : preuve que le CREATE OR REPLACE VIEW a bien suivi
	// l'ALTER. Un bloc Stockpile écrit → les colonnes Extraction restent NULL.
	if _, err := db.Exec(`
		INSERT INTO match_objective_stats (
			match_id, xuid, power_seeds_deposited, power_seeds_stolen,
			time_as_power_seed_carrier_seconds
		) VALUES ('m_sp', 'x_sp', 6, 2, 59.1)`); err != nil {
		t.Fatalf("insert stockpile row: %v", err)
	}
	var deposited, stolen int
	var carrierSeconds float64
	var successfulExtractions, timesSelectedAsVip sql.NullInt64
	if err := db.QueryRow(`
		SELECT power_seeds_deposited, power_seeds_stolen, time_as_power_seed_carrier_seconds,
		       successful_extractions, times_selected_as_vip
		FROM match_objective_stats_latest WHERE match_id = 'm_sp' AND xuid = 'x_sp'`,
	).Scan(&deposited, &stolen, &carrierSeconds, &successfulExtractions, &timesSelectedAsVip); err != nil {
		t.Fatalf("SELECT sur la vue _latest (vue non recréée après l'ALTER ?): %v", err)
	}
	if deposited != 6 || stolen != 2 {
		t.Errorf("_latest power_seeds_deposited/stolen = %d/%d, want 6/2", deposited, stolen)
	}
	if carrierSeconds != 59.1 {
		t.Errorf("_latest time_as_power_seed_carrier_seconds = %v, want 59.1 (fraction préservée)", carrierSeconds)
	}
	if successfulExtractions.Valid {
		t.Errorf("_latest successful_extractions = %d, want NULL (bloc d'un autre mode)", successfulExtractions.Int64)
	}
	if timesSelectedAsVip.Valid {
		t.Errorf("_latest times_selected_as_vip = %d, want NULL (bloc d'un autre mode)", timesSelectedAsVip.Int64)
	}
}
