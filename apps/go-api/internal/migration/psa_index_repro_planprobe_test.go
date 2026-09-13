//go:build psarepro

// Suite du harnais d_enquete jetable psa_index_repro_test.go (build tag
// `psarepro`, EXCLU des gates). Voir ce fichier pour la methode et les helpers.
package migration

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── S0b : QUAND DuckDB utilise-t-il reellement l'index ART ? ─────────────────
//
// Enjeu de methode : si le plan du lookup est un SEQ_SCAN, le controle de
// repair_psa_index ne mesure PAS l'index. On sonde le plan a chaque etape.
func TestPSAReproS0bPlanProbe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	reproCreateFresh(t, db)
	ids := reproIDs(1100)

	probe := func(label string, d *sql.DB) {
		plan := reproExplain(t, d, ids[7])
		kind := "SEQ_SCAN"
		if strings.Contains(plan, "INDEX_SCAN") || strings.Contains(strings.ToUpper(plan), "INDEX SCAN") {
			kind = "INDEX_SCAN"
		}
		t.Logf("  plan[%s] = %s", label, kind)
	}

	reproWave(t, db, "2533274", ids, 4)
	probe("apres insert, avant checkpoint", db)
	reproExec(t, db, `CHECKPOINT`)
	probe("apres CHECKPOINT", db)
	db.Close()

	db = reproOpen(t, path)
	probe("apres reouverture", db)
	// force le planner : PRAGMA / SET pour desactiver le seq scan n'existe pas ;
	// on teste l'effet du volume (plus de row groups).
	reproWave(t, db, "2533274", ids, 30)
	reproExec(t, db, `CHECKPOINT`)
	probe("apres volume x8 (~40k lignes)", db)
	db.Close()

	db = reproOpen(t, path)
	defer db.Close()
	probe("apres reouverture volume", db)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	t.Logf("  lignes finales = %d", n)
	full := reproExplain(t, db, ids[7])
	t.Logf("plan complet final:\n%s", full)
}

// ── S0c : quelles formes de requete declenchent un INDEX_SCAN en DuckDB 1.5.5 ?
func TestPSAReproS0cIndexScanShapes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()
	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproExec(t, db, `CHECKPOINT`)

	rows, err := db.Query(`SELECT index_name, is_unique, sql FROM duckdb_indexes() WHERE table_name='personal_score_awards'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n, s string
		var u bool
		if err := rows.Scan(&n, &u, &s); err != nil {
			t.Fatal(err)
		}
		t.Logf("  index present: %s unique=%v", n, u)
	}
	rows.Close()

	shapes := []struct {
		label string
		q     string
		args  []any
	}{
		{"COUNT(*) param", `EXPLAIN SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ?`, []any{ids[7]}},
		{"COUNT(*) litteral", `EXPLAIN SELECT COUNT(*) FROM personal_score_awards WHERE match_id = '` + ids[7] + `'`, nil},
		{"SELECT * param", `EXPLAIN SELECT * FROM personal_score_awards WHERE match_id = ?`, []any{ids[7]}},
		{"SELECT id litteral", `EXPLAIN SELECT id FROM personal_score_awards WHERE match_id = '` + ids[7] + `'`, nil},
		{"PK id = 5", `EXPLAIN SELECT * FROM personal_score_awards WHERE id = 5`, nil},
		{"PK id = ?", `EXPLAIN SELECT * FROM personal_score_awards WHERE id = ?`, []any{5}},
		{"triplet gen", `EXPLAIN SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ? AND xuid = ? AND generation_id = CAST(? AS BIGINT)`, []any{ids[7], "2533274", "8"}},
		{"category", `EXPLAIN SELECT COUNT(*) FROM personal_score_awards WHERE award_category = 'combat'`, nil},
	}
	for _, sh := range shapes {
		r, err := db.Query(sh.q, sh.args...)
		if err != nil {
			t.Logf("  %-22s ERREUR %v", sh.label, err)
			continue
		}
		var plan strings.Builder
		for r.Next() {
			var a, b string
			if err := r.Scan(&a, &b); err != nil {
				t.Fatal(err)
			}
			plan.WriteString(b)
		}
		r.Close()
		kind := "SEQ_SCAN"
		if strings.Contains(strings.ToUpper(plan.String()), "INDEX_SCAN") {
			kind = "INDEX_SCAN"
		}
		t.Logf("  %-22s -> %s", sh.label, kind)
	}
}

// ── S0d : reglages du planner qui gouvernent l'index scan ────────────────────
func TestPSAReproS0dIndexSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()
	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproExec(t, db, `CHECKPOINT`)

	rows, err := db.Query(`SELECT name, value, description FROM duckdb_settings() WHERE name ILIKE '%index%' OR name ILIKE '%scan%'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n, v, d string
		if err := rows.Scan(&n, &v, &d); err != nil {
			t.Fatal(err)
		}
		t.Logf("  setting %-32s = %-10s (%.70s)", n, v, d)
	}
	rows.Close()

	planOf := func(label string) {
		r, err := db.Query(`EXPLAIN SELECT COUNT(*) FROM personal_score_awards WHERE match_id = '` + ids[7] + `'`)
		if err != nil {
			t.Fatalf("explain: %v", err)
		}
		var sb strings.Builder
		for r.Next() {
			var a, b string
			_ = r.Scan(&a, &b)
			sb.WriteString(b)
		}
		r.Close()
		typ := "Sequential Scan"
		if strings.Contains(sb.String(), "Index Scan") {
			typ = "INDEX SCAN"
		}
		t.Logf("  plan[%s] = %s", label, typ)
	}
	planOf("defauts")
	for _, s := range []string{
		`SET index_scan_percentage = 1.0`,
		`SET index_scan_max_count = 1000000`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Logf("  (%s -> %v)", s, err)
			continue
		}
		planOf(s)
	}
}

// ── S0e : forcer la strategie d'execution du table scan (index vs sequentiel) ─
func TestPSAReproS0eForceStrategy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.duckdb")
	db := reproOpen(t, path)
	defer db.Close()
	reproCreateFresh(t, db)
	ids := reproIDs(1100)
	reproWave(t, db, "2533274", ids, 4)
	reproExec(t, db, `CHECKPOINT`)

	for _, v := range []string{"DEFAULT", "SEQUENTIAL", "INDEX", "INDEX_SCAN", "SCAN", "ADAPTIVE"} {
		if _, err := db.Exec(`SET debug_physical_table_scan_execution_strategy = '` + v + `'`); err != nil {
			t.Logf("  strategie %-12s : REFUSEE (%v)", v, truncate(err.Error(), 200))
			continue
		}
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ?`, ids[7]).Scan(&n)
		t.Logf("  strategie %-12s : acceptee, COUNT=%d err=%v", v, n, err)
		r, e2 := db.Query(`EXPLAIN ANALYZE SELECT COUNT(*) FROM personal_score_awards WHERE match_id = '` + ids[7] + `'`)
		if e2 == nil {
			var sb strings.Builder
			for r.Next() {
				var a, b string
				_ = r.Scan(&a, &b)
				sb.WriteString(b)
			}
			r.Close()
			typ := "Sequential Scan"
			if strings.Contains(sb.String(), "Index Scan") {
				typ = "INDEX SCAN"
			}
			t.Logf("      -> EXPLAIN ANALYZE type = %s", typ)
		}
	}
	_, _ = db.Exec(`SET debug_physical_table_scan_execution_strategy = 'DEFAULT'`)
}
