//go:build integration

// arc_titles_test.go — preuve du backfill cross-titre (create_arc_titles_join,
// PLAN_CROSS_TITLE_ARCS_BACKEND Phase 1). Modèle : player_perf_chain_test.go
// (résolution du step via StepsFor + ApplySchema/ApplyBackfill directs).
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func findPlayerStep(t *testing.T, name string) *migration.Migration {
	t.Helper()
	steps := StepsFor(migration.TargetPlayer)
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}
	t.Fatalf("migration %q introuvable dans StepsFor(TargetPlayer)", name)
	return nil
}

// TestArcTitlesBackfill_OneRowPerExistingArc : sur une DB où `arc` contient déjà
// des lignes, create_arc_titles_join crée la jointure et la backfille avec
// exactement 1 ligne (arc.id, arc.title_slug) par arc — invariant de transition
// mono-titre. Le rejeu du backfill ne duplique rien (PK + ON CONFLICT).
func TestArcTitlesBackfill_OneRowPerExistingArc(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// DB « legacy » : table arc seule, peuplée, SANS arc_titles.
	if _, err := db.Exec(`
		CREATE TABLE arc (
			id VARCHAR PRIMARY KEY, user_id VARCHAR, title_slug VARCHAR,
			title VARCHAR, description VARCHAR, is_preset BOOLEAN,
			preset_id VARCHAR, created_at TIMESTAMP, completed_at TIMESTAMP
		);
		INSERT INTO arc (id, user_id, title_slug, title, is_preset) VALUES
			('arc_a', 'u1', 'halo_infinite', 'A', FALSE),
			('arc_b', 'u1', 'halo_infinite', 'B', FALSE),
			('arc_c', 'u2', 'halo_infinite', 'C', FALSE);
	`); err != nil {
		t.Fatalf("seed arc: %v", err)
	}

	mig := findPlayerStep(t, "create_arc_titles_join")
	if mig.ApplyBackfill == nil {
		t.Fatal("create_arc_titles_join doit définir ApplyBackfill")
	}
	if err := mig.ApplySchema(db); err != nil {
		t.Fatalf("ApplySchema arc_titles: %v", err)
	}
	if err := mig.ApplyBackfill(db); err != nil {
		t.Fatalf("ApplyBackfill arc_titles: %v", err)
	}

	// 1 ligne par arc.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM arc_titles`).Scan(&n); err != nil {
		t.Fatalf("count arc_titles: %v", err)
	}
	if n != 3 {
		t.Errorf("arc_titles = %d lignes, want 3 (1 par arc)", n)
	}
	// Invariant : chaque arc a sa ligne (arc_id, title_slug) primaire.
	var mismatch int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM arc a
		LEFT JOIN arc_titles act ON act.arc_id = a.id AND act.title_slug = a.title_slug
		WHERE act.arc_id IS NULL
	`).Scan(&mismatch); err != nil {
		t.Fatalf("invariant query: %v", err)
	}
	if mismatch != 0 {
		t.Errorf("%d arc(s) sans ligne arc_titles (arc_id, title_slug) correspondante", mismatch)
	}

	// Idempotence : rejouer le backfill ne duplique rien.
	if err := mig.ApplyBackfill(db); err != nil {
		t.Fatalf("ApplyBackfill (2e passe): %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM arc_titles`).Scan(&n); err != nil {
		t.Fatalf("count arc_titles (2e): %v", err)
	}
	if n != 3 {
		t.Errorf("après rejeu : arc_titles = %d, want 3 (idempotent)", n)
	}
}
