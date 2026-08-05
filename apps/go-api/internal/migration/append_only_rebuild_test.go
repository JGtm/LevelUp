//go:build cgo

package migration

// Tests du helper commun append_only_rebuild.go — verrouille la sûreté que les
// 5 conversions « simples » (match_skill_rank, csr_snapshots, match_csrs,
// pve_match_stats, lusr_component_history) ont GAGNÉE en passant au helper :
// swap transactionnel (rollback intégral) + recoverOrphan. Avant le helper, ces
// conversions étaient une suite de db.ExecContext qui DROPpait l'ancienne table
// AVANT toute vérification, sans rollback ni récupération de crash mid-swap.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openTmpDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "t.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// demoSpec — un spec « simple » (id + written_at, vue ROW_NUMBER) sur une table
// jouet, pour exercer la mécanique du helper sans dépendre d'un schéma métier.
func demoSpec() appendOnlyRebuild {
	return appendOnlyRebuild{
		Table:         "t_demo",
		IDSeq:         "t_demo_seq",
		SyntheticCols: synthWrittenAt,
		PostSwap: []string{
			`ALTER TABLE t_demo ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			`CREATE INDEX IF NOT EXISTS idx_t_demo_k ON t_demo(k)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW t_demo_latest AS
			SELECT * FROM t_demo
			QUALIFY ROW_NUMBER() OVER (PARTITION BY k ORDER BY written_at DESC, id DESC) = 1`,
	}
}

func seedDemoLegacy(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE t_demo (k INTEGER, v VARCHAR)`)
	mustExec(t, db, `INSERT INTO t_demo VALUES (1,'a'),(2,'b'),(3,'c')`)
}

// TestRebuildAppendOnly_HappyPath : swap nominal → id + written_at posés, vue créée,
// cardinalité préservée, double-run idempotent.
func TestRebuildAppendOnly_HappyPath(t *testing.T) {
	db := openTmpDB(t)
	seedDemoLegacy(t, db)

	if err := applyAppendOnlyRebuild(db, demoSpec()); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := countRows(t, db, "t_demo"); got != 3 {
		t.Fatalf("rows après swap = %d, attendu 3", got)
	}
	if has, _ := columnExists(db, "t_demo", "id"); !has {
		t.Fatal("colonne id absente après swap")
	}
	if got := countRows(t, db, "t_demo_latest"); got != 3 {
		t.Fatalf("vue _latest = %d, attendu 3", got)
	}

	// Idempotence : 2e passe = no-op (marqueur id présent), pas d'erreur, vue OK.
	if err := applyAppendOnlyRebuild(db, demoSpec()); err != nil {
		t.Fatalf("apply pass2: %v", err)
	}
	if got := countRows(t, db, "t_demo"); got != 3 {
		t.Fatalf("rows après pass2 = %d, attendu 3", got)
	}
}

// TestRebuildAppendOnly_LatestDedupsBySupersession : après le swap, un INSERT pur
// d'une nouvelle version (id supérieur) supersède l'ancienne dans la vue _latest,
// sans aucun DELETE/UPDATE (la table physique croît).
func TestRebuildAppendOnly_LatestDedupsBySupersession(t *testing.T) {
	db := openTmpDB(t)
	seedDemoLegacy(t, db)
	if err := applyAppendOnlyRebuild(db, demoSpec()); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// Nouvelle version de k=1 (INSERT pur, id auto via défaut).
	mustExec(t, db, `INSERT INTO t_demo (k, v) VALUES (1, 'a2')`)

	if phys := countRows(t, db, "t_demo"); phys != 4 {
		t.Fatalf("table physique = %d, attendu 4 (append pur)", phys)
	}
	if latest := countRows(t, db, "t_demo_latest"); latest != 3 {
		t.Fatalf("vue _latest = %d, attendu 3 (dédup par k)", latest)
	}
	var v string
	if err := db.QueryRow(`SELECT v FROM t_demo_latest WHERE k = 1`).Scan(&v); err != nil {
		t.Fatalf("select latest k=1: %v", err)
	}
	if v != "a2" {
		t.Fatalf("vue _latest k=1 = %q, attendu 'a2' (dernière version)", v)
	}
}

// TestRebuildAppendOnlyTx_RollbackPreservesTable : un échec DANS la TX (PostSwap
// invalide) doit ROLLBACK intégralement → table d'origine intacte, aucune perte,
// pas d'orphelin committé. C'est la garantie que les 5 conversions « simples »
// n'avaient PAS avant le helper.
func TestRebuildAppendOnlyTx_RollbackPreservesTable(t *testing.T) {
	db := openTmpDB(t)
	seedDemoLegacy(t, db)

	bad := demoSpec()
	bad.PostSwap = []string{
		`ALTER TABLE t_demo ADD PRIMARY KEY (colonne_inexistante)`, // échoue dans la TX
	}
	err := rebuildAppendOnlyTx(context.Background(), db, bad)
	if err == nil {
		t.Fatal("attendu une erreur (PostSwap invalide), got nil")
	}

	// Table d'origine PRÉSERVÉE avec ses 3 rows (rollback intégral).
	if got := countRows(t, db, "t_demo"); got != 3 {
		t.Fatalf("rows après rollback = %d, attendu 3 (zéro perte)", got)
	}
	// id NE doit PAS être présent (le swap a été annulé).
	if has, _ := columnExists(db, "t_demo", "id"); has {
		t.Fatal("colonne id présente après rollback — le swap n'a pas été annulé")
	}
	// Pas d'orphelin __appendonly committé.
	if has, _ := tableExists(db, "t_demo__appendonly"); has {
		t.Fatal("t_demo__appendonly committé malgré le rollback")
	}
}

// TestRecoverOrphanAppendOnly : un crash mid-swap laisse `<table>__appendonly`
// orpheline + table principale absente. recoverOrphanAppendOnly la restaure.
func TestRecoverOrphanAppendOnly(t *testing.T) {
	db := openTmpDB(t)
	// Simule l'état post-crash : seule l'orpheline existe.
	mustExec(t, db, `CREATE TABLE t_demo__appendonly (id BIGINT, k INTEGER, v VARCHAR)`)
	mustExec(t, db, `INSERT INTO t_demo__appendonly VALUES (1,1,'a'),(2,2,'b')`)

	if err := recoverOrphanAppendOnly(context.Background(), db, "t_demo"); err != nil {
		t.Fatalf("recover: %v", err)
	}
	if has, _ := tableExists(db, "t_demo"); !has {
		t.Fatal("t_demo non restaurée depuis l'orpheline")
	}
	if has, _ := tableExists(db, "t_demo__appendonly"); has {
		t.Fatal("orpheline __appendonly toujours présente après récupération")
	}
	if got := countRows(t, db, "t_demo"); got != 2 {
		t.Fatalf("rows restaurées = %d, attendu 2", got)
	}

	// Idempotence : 2e appel = no-op (principale présente).
	if err := recoverOrphanAppendOnly(context.Background(), db, "t_demo"); err != nil {
		t.Fatalf("recover idempotent: %v", err)
	}
}

// TestRecoverOrphanAppendOnly_NoOpWhenMainExists : si la principale existe, ne
// touche pas à une éventuelle table __appendonly (cas non concerné).
func TestRecoverOrphanAppendOnly_NoOpWhenMainExists(t *testing.T) {
	db := openTmpDB(t)
	mustExec(t, db, `CREATE TABLE t_demo (k INTEGER)`)
	if err := recoverOrphanAppendOnly(context.Background(), db, "t_demo"); err != nil {
		t.Fatalf("recover: %v", err)
	}
	if has, _ := tableExists(db, "t_demo"); !has {
		t.Fatal("t_demo supprimée à tort")
	}
}

// TestBuildAppendOnlySelectList : la clause SELECT respecte l'ordre
// [id], colonnes, [synthétiques] et l'id conditionnel.
func TestBuildAppendOnlySelectList(t *testing.T) {
	cols := []string{"k", "v"}

	// id toujours ajouté (simple).
	got := buildAppendOnlySelectList(cols, appendOnlyRebuild{IDSeq: "s", SyntheticCols: synthWrittenAt})
	want := "nextval('s') AS id, k, v, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) AS written_at"
	if got != want {
		t.Fatalf("select simple:\n got %q\nwant %q", got, want)
	}

	// Pas de synthétiques (lch).
	got = buildAppendOnlySelectList(cols, appendOnlyRebuild{IDSeq: "s"})
	if got != "nextval('s') AS id, k, v" {
		t.Fatalf("select sans synthétiques: %q", got)
	}

	// id conditionnel + déjà présent → id préservé tel quel (pas de nextval).
	got = buildAppendOnlySelectList([]string{"id", "k"}, appendOnlyRebuild{IDSeq: "s", IDConditional: true, SyntheticCols: "0::BIGINT AS generation_id"})
	if got != "id, k, 0::BIGINT AS generation_id" {
		t.Fatalf("select id conditionnel présent: %q", got)
	}

	// id conditionnel + absent → ajouté.
	got = buildAppendOnlySelectList([]string{"k"}, appendOnlyRebuild{IDSeq: "s", IDConditional: true})
	if got != "nextval('s') AS id, k" {
		t.Fatalf("select id conditionnel absent: %q", got)
	}
}
