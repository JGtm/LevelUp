//go:build integration

package sync

// citations_recreate_test.go — non-régression du chemin force du recompute
// citations (2026-07-24) : recreateCitationsTable doit produire le schéma
// append-only COMPLET (generation_id + vue match_citations_latest requêtable),
// pas le schéma legacy 3 colonnes qui cassait la vue au premier SELECT
// (Binder Error "generation_id not found") et bouclait conversion/recréation.

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestRecreateCitationsTable_ProducesAppendOnlySchema(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open :memory:: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	// État de départ = table déjà convertie (comme après OpenPlayerDB) avec une
	// ligne, pour vérifier que la recréation vide bien ET garde le schéma complet.
	if _, err := db.ExecContext(ctx, `CREATE TABLE match_citations (
		match_id VARCHAR, citation_name_norm VARCHAR, value INTEGER,
		id BIGINT, generation_id BIGINT, written_at TIMESTAMP)`); err != nil {
		t.Fatalf("seed table: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_citations VALUES ('m1', 'x', 1, 1, 1, now())`); err != nil {
		t.Fatalf("seed row: %v", err)
	}

	if err := recreateCitationsTable(ctx, db); err != nil {
		t.Fatalf("recreateCitationsTable: %v", err)
	}

	for _, col := range []string{"match_id", "citation_name_norm", "value", "id", "generation_id", "written_at"} {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns
			WHERE table_name='match_citations' AND column_name=?`, col).Scan(&n); err != nil {
			t.Fatalf("columns %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("colonne %q absente après recreateCitationsTable", col)
		}
	}

	// La table est vide et la vue _latest se lie SANS Binder Error (le bug d'origine).
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_citations_latest`).Scan(&count); err != nil {
		t.Fatalf("match_citations_latest doit être requêtable: %v", err)
	}
	if count != 0 {
		t.Fatalf("match_citations doit être vide après recréation, count=%d", count)
	}

	// Le cycle complet loadCumulExcluding (la requête qui échouait) passe.
	if _, err := loadCumulExcluding(ctx, db, []string{"m1", "m2"}); err != nil {
		t.Fatalf("loadCumulExcluding post-recréation: %v", err)
	}
}
