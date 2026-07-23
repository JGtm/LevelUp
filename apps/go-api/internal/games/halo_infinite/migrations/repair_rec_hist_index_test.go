//go:build integration

// repair_rec_hist_index_test.go — F4 : le step correctif
// repair_rec_hist_achieved_index_canonical_v1 réaligne idx_rec_hist_achieved_desc
// sur (user_id, achieved_at DESC) pour les DBs passées par l'ancienne dédup qui
// le recréait sur (achieved_at DESC) sans user_id. Résolu via StepsFor + appliqué
// directement (pas de RunForDB → provider non requis).
package migrations

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func repairRecHistIdxMig(t *testing.T) *migration.Migration {
	t.Helper()
	steps := StepsFor(migration.TargetPlayer)
	for i := range steps {
		if steps[i].Name == "repair_rec_hist_achieved_index_canonical_v1" {
			return &steps[i]
		}
	}
	t.Fatal("migration repair_rec_hist_achieved_index_canonical_v1 introuvable dans StepsFor(TargetPlayer)")
	return nil
}

// achievedIndexSQL retourne la définition SQL de idx_rec_hist_achieved_desc (ou "").
func achievedIndexSQL(t *testing.T, db *sql.DB) string {
	t.Helper()
	var sqlDef sql.NullString
	err := db.QueryRow(
		`SELECT sql FROM duckdb_indexes() WHERE index_name = 'idx_rec_hist_achieved_desc'`,
	).Scan(&sqlDef)
	if err == sql.ErrNoRows {
		return ""
	}
	if err != nil {
		t.Fatalf("query index sql: %v", err)
	}
	return sqlDef.String
}

func seedRecordHistory(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
		CREATE TABLE record_history (
			id           VARCHAR PRIMARY KEY,
			user_id      VARCHAR NOT NULL,
			title_slug   VARCHAR NOT NULL,
			metric       VARCHAR NOT NULL,
			period       VARCHAR NOT NULL,
			value        DOUBLE NOT NULL,
			achieved_at  TIMESTAMP NOT NULL
		);
	`); err != nil {
		t.Fatalf("create record_history: %v", err)
	}
}

func TestMigration_RepairRecHistAchievedIndex_OnDivergentDB(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	seedRecordHistory(t, db)
	// Simule une DB passée par l'ANCIENNE dédup : index divergent sans user_id.
	if _, err := db.Exec(`CREATE INDEX idx_rec_hist_achieved_desc ON record_history(achieved_at DESC)`); err != nil {
		t.Fatalf("seed divergent index: %v", err)
	}
	if got := achievedIndexSQL(t, db); !strings.Contains(got, "achieved_at") || strings.Contains(got, "user_id") {
		t.Fatalf("précondition : index attendu divergent (achieved_at sans user_id), got %q", got)
	}

	mig := repairRecHistIdxMig(t)
	if err := mig.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema repair: %v", err)
	}

	got := achievedIndexSQL(t, db)
	if !strings.Contains(got, "user_id") || !strings.Contains(got, "achieved_at") {
		t.Errorf("après réparation, index doit être canonique (user_id, achieved_at DESC), got %q", got)
	}

	// Idempotent : une 2e passe ne casse rien et conserve la forme canonique.
	if err := mig.ApplySchema(db); err != nil {
		t.Errorf("ApplySchema (2e passe) doit être idempotent: %v", err)
	}
	if got := achievedIndexSQL(t, db); !strings.Contains(got, "user_id") {
		t.Errorf("2e passe : index doit rester canonique, got %q", got)
	}
}

func TestMigration_RepairRecHistAchievedIndex_NoTableIsNoOp(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Aucune table record_history → le step doit être silencieux (garde TableExists).
	mig := repairRecHistIdxMig(t)
	if err := mig.ApplySchema(db); err != nil {
		t.Errorf("ApplySchema sans table record_history doit être un no-op: %v", err)
	}
}
