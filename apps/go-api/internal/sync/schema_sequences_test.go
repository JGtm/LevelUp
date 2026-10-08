package sync

// schema_sequences_test.go — une base joueur dont une séquence est en retard sur le max de sa
// colonne ne fait plus collisionner l'écriture LUSR append-only (id laissé à son DEFAULT
// nextval('msr_seq')). Deux points d'alignement, chacun tenu ici par son branchement :
//   - l'ouverture PHYSIQUE en écriture d'une base joueur (platform/duckdb/physical_open.go),
//     que traverse OpenPlayerDB comme tout autre ouvreur ;
//   - EnsurePlayerSchema, quand son soin vient de CRÉER une séquence à côté d'ids déjà posés.
// La mécanique (technique, idempotence, base saine) est tenue par
// internal/migration/sequence_alignment_test.go ; l'ouverture physique par
// internal/platform/duckdb/physical_open_test.go.

import (
	"bytes"
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

const insertLUSRSansID = `INSERT INTO match_skill_rank (match_id, rating_type) VALUES ('m-neuf', 'LUSR')`

// preparePlayerDBEnRetard pose, hors du cache du paquet duckdb, une base joueur au schéma
// complet dont msr_seq rend encore 1 alors que les ids 1..3 sont pris (forme de la base de
// Chocoboflor : séquence recréée à START 1 par une reconstruction), exécute extra, vérifie la
// collision, puis ferme le fichier. Rend le chemin, de la forme d'une base joueur.
func preparePlayerDBEnRetard(t *testing.T, extra ...string) string {
	t.Helper()
	path := titlePkg.NewPathResolver(t.TempDir()).PlayerDBPath(titlePkg.DefaultSlug, "Retard")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := EnsurePlayerSchema(ctx, db); err != nil {
		t.Fatalf("EnsurePlayerSchema (création): %v", err)
	}
	stmts := append([]string{`
		INSERT INTO match_skill_rank (id, match_id, rating_type)
		SELECT range + 1, 'legacy-' || range, 'LUSR' FROM range(3)`}, extra...)
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("préparation %q: %v", s, err)
		}
	}
	if _, err := db.ExecContext(ctx, insertLUSRSansID); err == nil || !strings.Contains(err.Error(), "Duplicate key") {
		t.Fatalf("prémisse : l'insertion LUSR doit collisionner avant l'ouverture, err=%v", err)
	}
	return path
}

func TestOpenPlayerDB_AligneUneSequenceEnRetard(t *testing.T) {
	path := preparePlayerDBEnRetard(t)
	handle, err := OpenPlayerDB(path)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	if _, err := handle.SQLDb().ExecContext(t.Context(), insertLUSRSansID); err != nil {
		t.Fatalf("insertion LUSR après OpenPlayerDB : %v", err)
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
// l'ouverture : OpenPlayerDB rend le handle, l'échec est journalisé en ERROR et compté UNE
// fois (l'ouverture physique aligne ; EnsurePlayerSchema, qui n'a créé aucune séquence, ne
// repasse pas), et la séquence en retard voisine (msr_seq) est quand même alignée.
func TestOpenPlayerDB_AlignementEnEchecNeBloquePasLOuverture(t *testing.T) {
	path := preparePlayerDBEnRetard(t,
		`CREATE SEQUENCE zz_borne MAXVALUE 100`,
		`CREATE TABLE zz_borne_t (id BIGINT DEFAULT nextval('zz_borne'))`,
		`INSERT INTO zz_borne_t (id) VALUES (100)`)

	logs := &lockedBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	const failedCounter = "duckdb_sequence_align_failed_total"
	before := observability.LoadCounter(failedCounter)

	handle, err := OpenPlayerDB(path)
	if err != nil {
		t.Fatalf("l'échec d'alignement ne doit pas bloquer l'ouverture : %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	out := logs.String()
	if !strings.Contains(out, "level=ERROR") || !strings.Contains(out, "alignement des séquences échoué") ||
		!strings.Contains(out, "zz_borne") {
		t.Fatalf("ERROR d'alignement attendu dans les logs, obtenu :\n%s", out)
	}
	if d := observability.LoadCounter(failedCounter) - before; d != 1 {
		t.Fatalf("%s : +%d, attendu +1 (une passe par ouverture physique)", failedCounter, d)
	}
	if _, err := handle.SQLDb().ExecContext(t.Context(), insertLUSRSansID); err != nil {
		t.Fatalf("msr_seq non alignée à côté de l'échec : %v", err)
	}
}

// Le soin de schéma qui CRÉE une séquence à côté d'ids déjà posés la réaligne lui-même : ici
// une table personal_score_awards legacy (ids 1..3, ni séquence ni generation_id) ; le soin
// crée personal_score_awards_id_seq (START 1), la conversion append-only garde les ids et
// branche le DEFAULT dessus. Base ouverte hors du cache (chemin quelconque) : seul
// EnsurePlayerSchema peut aligner.
func TestEnsurePlayerSchema_RealigneUneSequenceQuIlCree(t *testing.T) {
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "stats.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	for _, s := range []string{
		`CREATE TABLE personal_score_awards (
			id INTEGER PRIMARY KEY, match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
			award_name VARCHAR NOT NULL, award_category VARCHAR, award_count INTEGER DEFAULT 1,
			award_score INTEGER DEFAULT 0, created_at TIMESTAMP)`,
		`INSERT INTO personal_score_awards (id, match_id, xuid, award_name)
		 SELECT range + 1, 'legacy-' || range, 'x', 'Kill' FROM range(3)`,
	} {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("préparation %q: %v", s, err)
		}
	}
	if err := EnsurePlayerSchema(ctx, db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO personal_score_awards (match_id, xuid, award_name) VALUES ('m-neuf', 'x', 'Kill')`); err != nil {
		t.Fatalf("insertion PSA après le soin : %v", err)
	}
}
