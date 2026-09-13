//go:build psarepro

// Suite du harnais d_enquete jetable psa_index_repro_test.go (build tag
// `psarepro`, EXCLU des gates). Voir ce fichier pour la methode et les helpers.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── S0 : socle — version, plan d_execution, volume reel ──────────────────────

func TestPSAReproS0Baseline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()

	t.Logf("DuckDB version = %s", reproVersion(t, db))
	reproCreateFresh(t, db)

	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S0 apres 1100 matchs x4")
	reproExec(t, db, `CHECKPOINT`)
	reproReport(t, db, "S0 apres CHECKPOINT")
}

// ── S1 : vagues + generations + tombstones + reouvertures (pattern de prod) ───

func TestPSAReproS1Waves(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")

	db := reproOpen(t, path)
	reproCreateFresh(t, db)
	ids := reproIDs(1100)

	// vague 1 : sync initial
	reproWave(t, db, "2533274", ids, 4)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S1 vague1")
	reproExec(t, db, `CHECKPOINT`)
	db.Close()

	// vague 2 : reouverture + re-extraction (nouvelle generation) sur 30 % des matchs
	db = reproOpen(t, path)
	reproWave(t, db, "2533274", ids[:330], 4)
	reproReport(t, db, "S1 vague2 (regen 330)")
	reproExec(t, db, `CHECKPOINT`)
	db.Close()

	// vague 3 : tombstones (extraction vide) sur 100 matchs + regen sur 100 autres
	db = reproOpen(t, path)
	reproTombstone(t, db, "2533274", ids[900:1000])
	reproWave(t, db, "2533274", ids[1000:], 6)
	reproReport(t, db, "S1 vague3 (tombstones + regen)")
	reproExec(t, db, `CHECKPOINT`)
	db.Close()

	db = reproOpen(t, path)
	defer db.Close()
	if n := reproReport(t, db, "S1 apres reouverture finale"); n > 0 {
		t.Errorf("S1 REPRODUIT : %d cles en ecart", n)
	}
}

// ── S1b : vagues SANS aucun CHECKPOINT, reouvertures (WAL rejoue) ────────────

func TestPSAReproS1bNoCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")

	db := reproOpen(t, path)
	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S1b vague1 (sans CHECKPOINT)")
	db.Close()

	db = reproOpen(t, path)
	reproWave(t, db, "2533274", ids[:330], 4)
	reproTombstone(t, db, "2533274", ids[900:1000])
	reproReport(t, db, "S1b vague2 (sans CHECKPOINT)")
	db.Close()

	db = reproOpen(t, path)
	defer db.Close()
	if n := reproReport(t, db, "S1b apres reouverture finale"); n > 0 {
		t.Errorf("S1b REPRODUIT : %d cles en ecart", n)
	}
}

// ── S2 : CREATE INDEX APRES peuplement ───────────────────────────────────────

func TestPSAReproS2IndexAfterPopulate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()

	reproCreateNoIndex(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproCreateIndexes(t, db)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S2 index cree apres peuplement")
	reproExec(t, db, `CHECKPOINT`)
	reproWave(t, db, "2533274", ids[:200], 4)
	if n := reproReport(t, db, "S2 + vague post-index"); n > 0 {
		t.Errorf("S2 REPRODUIT : %d cles en ecart", n)
	}
}

// ── S3 : rebuild append-only sur table LEGACY deja peuplee (migration) ───────

// reproCreateLegacy pose le schema d'AVANT la conversion append-only (pas de
// generation_id / written_at / is_tombstone) avec les index ART de l'epoque.
func reproCreateLegacy(t *testing.T, db *sql.DB) {
	t.Helper()
	reproExec(t, db, `CREATE SEQUENCE IF NOT EXISTS personal_score_awards_id_seq`)
	reproExec(t, db, `CREATE TABLE personal_score_awards (
		id INTEGER PRIMARY KEY DEFAULT nextval('personal_score_awards_id_seq'),
		match_id VARCHAR NOT NULL,
		xuid VARCHAR NOT NULL,
		award_name VARCHAR NOT NULL,
		award_category VARCHAR,
		award_count INTEGER DEFAULT 1,
		award_score INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	)`)
	reproExec(t, db, `CREATE INDEX idx_psa_match ON personal_score_awards(match_id)`)
	reproExec(t, db, `CREATE INDEX idx_psa_xuid ON personal_score_awards(xuid)`)
	reproExec(t, db, `CREATE INDEX idx_psa_category ON personal_score_awards(award_category)`)
	reproExec(t, db, `CREATE INDEX idx_psa_match_xuid ON personal_score_awards(match_id, xuid)`)
}

// reproLegacyWrite rejoue l'ANCIEN pattern DELETE+INSERT (vecteur ART historique).
func reproLegacyWrite(t *testing.T, db *sql.DB, xuid string, matchIDs []string, awards int) {
	t.Helper()
	ctx := context.Background()
	for _, mid := range matchIDs {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM personal_score_awards WHERE match_id = ? AND xuid = ?`, mid, xuid); err != nil {
			_ = tx.Rollback()
			t.Fatalf("delete: %v", err)
		}
		for a := 0; a < awards; a++ {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO personal_score_awards
					(match_id, xuid, award_name, award_category, award_count, award_score)
				VALUES (?, ?, ?, ?, ?, ?)`,
				mid, xuid, fmt.Sprintf("award_%d", a), reproCategories[a%len(reproCategories)], 1, 100+a); err != nil {
				_ = tx.Rollback()
				t.Fatalf("insert: %v", err)
			}
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
}

func TestPSAReproS3MigrationOnPopulated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)

	reproCreateLegacy(t, db)
	ids := reproIDs(1100)
	reproLegacyWrite(t, db, "2533274", ids, 4)
	reproLegacyWrite(t, db, "2533274", ids[:400], 4) // re-extractions (DELETE+INSERT)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S3 legacy peuple (avant migration)")
	reproExec(t, db, `CHECKPOINT`)
	reproReport(t, db, "S3 legacy apres CHECKPOINT")

	// la migration reelle : conversion append-only + DDL + refresh vue
	if err := applyCreatePersonalScoreAwards(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	reproReport(t, db, "S3 juste apres swap CTAS + CREATE INDEX en TX")
	reproExec(t, db, `CHECKPOINT`)
	reproReport(t, db, "S3 apres CHECKPOINT")
	db.Close()

	db = reproOpen(t, path)
	defer db.Close()
	reproReport(t, db, "S3 apres reouverture")
	reproWave(t, db, "2533274", ids[:200], 4)
	if n := reproReport(t, db, "S3 + vagues append-only post-migration"); n > 0 {
		t.Errorf("S3 REPRODUIT : %d cles en ecart", n)
	}
}

// ── S4 : DELETE puis ROLLBACK (dry-run de cmd/cleanup_orphan_match) ──────────

func TestPSAReproS4DeleteRollback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()

	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproExec(t, db, `CHECKPOINT`)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S4 avant DELETE")

	// dry-run cleanup : DELETE dans une TX puis ROLLBACK
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM personal_score_awards WHERE match_id = ?`, ids[500])
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	n, _ := res.RowsAffected()
	t.Logf("  DELETE (dry-run) a touche %d lignes, ROLLBACK", n)
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	reproReport(t, db, "S4 apres DELETE+ROLLBACK")
	reproExec(t, db, `CHECKPOINT`)
	reproReport(t, db, "S4 apres CHECKPOINT")

	// masse : dry-run sur ~5 % du corpus (cleanup_post_art)
	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin2: %v", err)
	}
	for _, mid := range ids[:60] {
		if _, err := tx2.ExecContext(ctx, `DELETE FROM personal_score_awards WHERE match_id = ?`, mid); err != nil {
			t.Fatalf("delete masse: %v", err)
		}
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatalf("rollback2: %v", err)
	}
	reproReport(t, db, "S4 apres DELETE masse + ROLLBACK")
	reproExec(t, db, `CHECKPOINT`)
	reproReport(t, db, "S4 apres CHECKPOINT final")

	// re-INSERT apres le rollback (le sync repasse sur ces matchs)
	reproWave(t, db, "2533274", ids[:60], 4)
	reproExec(t, db, `CHECKPOINT`)
	if n := reproReport(t, db, "S4 apres re-INSERT post-rollback"); n > 0 {
		t.Errorf("S4 REPRODUIT : %d cles en ecart", n)
	}
}

// ── S5 : DELETE COMMIT (+ CHECKPOINT/vacuum) puis re-INSERT ──────────────────

func TestPSAReproS5DeleteCommit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()

	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproExec(t, db, `CHECKPOINT`)

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for _, mid := range ids[:60] {
		if _, err := tx.ExecContext(ctx, `DELETE FROM personal_score_awards WHERE match_id = ?`, mid); err != nil {
			t.Fatalf("delete: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S5 apres DELETE commit")
	reproExec(t, db, `CHECKPOINT`)
	reproReport(t, db, "S5 apres CHECKPOINT (vacuum)")
	// re-insertion des memes matchs (re-sync apres cleanup)
	reproWave(t, db, "2533274", ids[:60], 4)
	reproExec(t, db, `CHECKPOINT`)
	if n := reproReport(t, db, "S5 apres re-INSERT"); n > 0 {
		t.Errorf("S5 REPRODUIT : %d cles en ecart", n)
	}
}

// ── S6 : arret brutal (process tue sans Close ni CHECKPOINT) ─────────────────

// Le sous-processus est ce meme binaire de test, relance avec PSA_REPRO_CRASH.
func TestPSAReproS6HardKill(t *testing.T) {
	if os.Getenv("PSA_REPRO_CRASH") != "" {
		reproCrashChild()
		return
	}
	dir, err := os.MkdirTemp("", "psarepro")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "stats.duckdb")

	// phase 1 : socle propre
	db := reproOpen(t, path)
	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproExec(t, db, `CHECKPOINT`)
	reproAssertIndexScan(t, db, ids[7])
	reproReport(t, db, "S6 socle")
	db.Close()

	// phase 2 : N cycles « ecrit puis se fait tuer sans CHECKPOINT »
	for cycle := 0; cycle < 3; cycle++ {
		cmd := exec.Command(os.Args[0], "-test.run", "TestPSAReproS6HardKill")
		cmd.Env = append(os.Environ(), "PSA_REPRO_CRASH="+path, fmt.Sprintf("PSA_REPRO_CYCLE=%d", cycle))
		out, err := cmd.CombinedOutput()
		t.Logf("  cycle %d : enfant termine (err=%v) out=%s", cycle, err, truncate(string(out), 300))

		db = reproOpen(t, path)
		reproReport(t, db, fmt.Sprintf("S6 apres crash cycle %d", cycle))
		reproExec(t, db, `CHECKPOINT`)
		n := reproReport(t, db, fmt.Sprintf("S6 apres crash cycle %d + CHECKPOINT", cycle))
		db.Close()
		if n > 0 {
			t.Errorf("S6 REPRODUIT au cycle %d : %d cles en ecart", cycle, n)
			return
		}
	}
}

// reproCrashChild : ecrit des vagues puis se suicide SANS Close ni CHECKPOINT.
func reproCrashChild() {
	path := os.Getenv("PSA_REPRO_CRASH")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		os.Exit(3)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		fmt.Println("child ping:", err)
		os.Exit(4)
	}
	ctx := context.Background()
	ids := reproIDs(1100)
	for _, mid := range ids[:150] {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			os.Exit(5)
		}
		var gen int64
		if err := tx.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
			os.Exit(6)
		}
		for a := 0; a < 4; a++ {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO personal_score_awards
					(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				mid, "2533274", fmt.Sprintf("award_%d", a), reproCategories[a%len(reproCategories)], 1, 100+a, gen); err != nil {
				os.Exit(7)
			}
		}
		if err := tx.Commit(); err != nil {
			os.Exit(8)
		}
	}
	// arret brutal : ni Close, ni CHECKPOINT
	os.Exit(0)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
