package sync

// schema_sequences_test.go — EnsurePlayerSchema (rejoué à chaque OpenPlayerDB) aligne une
// séquence en retard sur le max de sa colonne : l'écriture LUSR append-only, qui laisse l'id
// à son DEFAULT nextval('msr_seq'), cesse de collisionner sur la clé primaire. Forme reproduite :
// base joueur dont msr_seq a été recréée à START 1 alors que des ids existaient déjà.
// La mécanique (technique, idempotence, base saine) est tenue par
// internal/migration/sequence_alignment_test.go ; ce test tient le BRANCHEMENT.

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestEnsurePlayerSchema_AligneUneSequenceEnRetard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()

	if err := EnsurePlayerSchema(ctx, db); err != nil {
		t.Fatalf("EnsurePlayerSchema (création): %v", err)
	}
	// Ids 1..3 posés hors séquence : msr_seq rend encore 1.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO match_skill_rank (id, match_id, rating_type)
		SELECT range + 1, 'legacy-' || range, 'LUSR' FROM range(3)`); err != nil {
		t.Fatalf("pose des ids legacy: %v", err)
	}
	const insertLUSR = `INSERT INTO match_skill_rank (match_id, rating_type) VALUES ('m-neuf', 'LUSR')`
	if _, err := db.ExecContext(ctx, insertLUSR); err == nil || !strings.Contains(err.Error(), "Duplicate key") {
		t.Fatalf("prémisse : l'insertion LUSR doit collisionner avant le soin, err=%v", err)
	}

	if err := EnsurePlayerSchema(ctx, db); err != nil {
		t.Fatalf("EnsurePlayerSchema (soin): %v", err)
	}
	if _, err := db.ExecContext(ctx, insertLUSR); err != nil {
		t.Fatalf("insertion LUSR après EnsurePlayerSchema : %v", err)
	}
}
