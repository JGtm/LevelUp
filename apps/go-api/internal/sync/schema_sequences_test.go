package sync

// schema_sequences_test.go — EnsurePlayerSchema (rejoué à chaque OpenPlayerDB) aligne une
// séquence en retard sur le max de sa colonne : l'écriture LUSR append-only, qui laisse l'id
// à son DEFAULT nextval('msr_seq'), cesse de collisionner sur la clé primaire. Forme reproduite :
// base joueur dont msr_seq a été recréée à START 1 alors que des ids existaient déjà.
// La mécanique (technique, idempotence, base saine) est tenue par
// internal/migration/sequence_alignment_test.go ; ce test tient le BRANCHEMENT.

import (
	"bytes"
	"database/sql"
	"log/slog"
	"path/filepath"
	"strings"
	gosync "sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/observability"
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

// lockedBuffer — sortie de log partagée sans course entre goroutines du test.
type lockedBuffer struct {
	mu  gosync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Un alignement en échec (séquence dont l'avance dépasserait MAXVALUE) ne bloque JAMAIS
// l'ouverture : EnsurePlayerSchema rend nil, l'échec est journalisé en ERROR et compté, et la
// séquence en retard voisine (msr_seq) est quand même alignée.
func TestEnsurePlayerSchema_AlignementEnEchecNeBloquePasLOuverture(t *testing.T) {
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
	for _, s := range []string{
		`CREATE SEQUENCE zz_borne MAXVALUE 100`,
		`CREATE TABLE zz_borne_t (id BIGINT DEFAULT nextval('zz_borne'))`,
		`INSERT INTO zz_borne_t (id) VALUES (100)`,
		`INSERT INTO match_skill_rank (id, match_id, rating_type)
		 SELECT range + 1, 'legacy-' || range, 'LUSR' FROM range(3)`,
	} {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("préparation %q: %v", s, err)
		}
	}

	logs := &lockedBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	const failedCounter = "duckdb_sequence_align_failed_total"
	before := observability.LoadCounter(failedCounter)

	if err := EnsurePlayerSchema(ctx, db); err != nil {
		t.Fatalf("l'échec d'alignement ne doit pas bloquer l'ouverture : %v", err)
	}
	out := logs.String()
	if !strings.Contains(out, "level=ERROR") || !strings.Contains(out, "alignement des séquences échoué") ||
		!strings.Contains(out, "zz_borne") {
		t.Fatalf("ERROR d'alignement attendu dans les logs, obtenu :\n%s", out)
	}
	if d := observability.LoadCounter(failedCounter) - before; d != 1 {
		t.Fatalf("%s : +%d, attendu +1", failedCounter, d)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO match_skill_rank (match_id, rating_type) VALUES ('m-neuf', 'LUSR')`); err != nil {
		t.Fatalf("msr_seq non alignée à côté de l'échec : %v", err)
	}
}
