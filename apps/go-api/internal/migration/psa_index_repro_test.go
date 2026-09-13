//go:build psarepro

// psa_index_repro_test.go — HARNAIS D'ENQUETE JETABLE (volet 2 « cause des index
// PSA desynchronises »). Build tag `psarepro` : EXCLU de tous les gates normaux.
//
//	go test -tags=psarepro ./internal/migration/ -run TestPSARepro -v -timeout 30m
//
// Objectif : rejouer sur DuckDB FICHIER (jamais :memory:) le pattern d'ecriture
// reel de personal_score_awards et detecter, apres chaque phase, l'ecart
// « lookup indexe < scan sequentiel » avec exactement le controle de
// cmd/repair_psa_index/diag.go.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── infrastructure ────────────────────────────────────────────────────────────

func reproOpen(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		t.Fatalf("ping %s: %v", path, err)
	}
	return db
}

func reproExec(t *testing.T, db *sql.DB, sqlText string, args ...any) {
	t.Helper()
	if _, err := db.Exec(sqlText, args...); err != nil {
		t.Fatalf("exec %.90s: %v", sqlText, err)
	}
}

// reproCreateFresh pose le schema canonique (PlayerPersonalScoreAwardsDDL) +
// la vue _latest, comme applyCreatePersonalScoreAwards sur DB vierge.
func reproCreateFresh(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := execScript(db, PlayerPersonalScoreAwardsDDL); err != nil {
		t.Fatalf("DDL fraiche: %v", err)
	}
	reproExec(t, db, psaLatestViewSQL)
}

// reproCreateNoIndex pose la table SANS index (pour tester CREATE INDEX apres peuplement).
func reproCreateNoIndex(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, s := range splitSQL(PlayerPersonalScoreAwardsDDL) {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(s)), "CREATE INDEX") {
			continue
		}
		reproExec(t, db, s)
	}
}

func reproCreateIndexes(t *testing.T, db *sql.DB) {
	t.Helper()
	reproExec(t, db, `CREATE INDEX IF NOT EXISTS idx_psa_match    ON personal_score_awards(match_id)`)
	reproExec(t, db, `CREATE INDEX IF NOT EXISTS idx_psa_category ON personal_score_awards(award_category)`)
	reproExec(t, db, `CREATE INDEX IF NOT EXISTS idx_psa_gen      ON personal_score_awards(match_id, xuid, generation_id)`)
}

func reproMatchID(i int) string {
	return fmt.Sprintf("%08x-0000-4000-8000-%012d", (i*2654435761)&0xffffffff, i)
}

var reproCategories = []string{"objective", "combat", "support", "vehicle", "misc"}

// reproWave rejoue EXACTEMENT internal/sync/writes.go::InsertPersonalScoreAwards
// (BeginTx, nextval('psa_generation_seq') partage, INSERT purs, Commit) pour un
// lot de matchs.
func reproWave(t *testing.T, db *sql.DB, xuid string, matchIDs []string, awardsPerMatch int) {
	t.Helper()
	ctx := context.Background()
	for _, mid := range matchIDs {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		var gen int64
		if err := tx.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
			_ = tx.Rollback()
			t.Fatalf("nextval: %v", err)
		}
		for a := 0; a < awardsPerMatch; a++ {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO personal_score_awards
					(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				mid, xuid, fmt.Sprintf("award_%d", a), reproCategories[a%len(reproCategories)],
				1, 100+a, gen)
			if err != nil {
				_ = tx.Rollback()
				t.Fatalf("insert: %v", err)
			}
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
}

// reproTombstone rejoue le chemin « extraction vide » (backfill_personal_scores).
func reproTombstone(t *testing.T, db *sql.DB, xuid string, matchIDs []string) {
	t.Helper()
	ctx := context.Background()
	for _, mid := range matchIDs {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		var gen int64
		if err := tx.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
			_ = tx.Rollback()
			t.Fatalf("nextval: %v", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO personal_score_awards (match_id, xuid, award_name, generation_id, is_tombstone)
			VALUES (?, ?, '', ?, TRUE)`, mid, xuid, gen); err != nil {
			_ = tx.Rollback()
			t.Fatalf("tombstone: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit tombstone: %v", err)
		}
	}
}

// ── controle (copie fidele de cmd/repair_psa_index/diag.go) ──────────────────

type reproAxis struct {
	name        string
	keyExprs    []string
	lookupWhere string
}

var reproAxes = []reproAxis{
	{"match_id", []string{"match_id || ''"}, "match_id = ?"},
	{"award_category", []string{"award_category || ''"}, "award_category = ?"},
	{"match+xuid+gen", []string{"match_id || ''", "xuid || ''", "CAST(generation_id AS VARCHAR)"},
		"match_id = ? AND xuid = ? AND generation_id = CAST(? AS BIGINT)"},
}

type reproDiverg struct {
	axis    string
	key     string
	scanned int
	indexed int
}

func reproCheck(t *testing.T, db *sql.DB, phase string) []reproDiverg {
	t.Helper()
	ctx := context.Background()
	var out []reproDiverg
	for _, a := range reproAxes {
		grp := fmt.Sprintf(`SELECT %s, COUNT(*) FROM personal_score_awards GROUP BY %s`,
			strings.Join(a.keyExprs, ", "), strings.Join(a.keyExprs, ", "))
		rows, err := db.QueryContext(ctx, grp)
		if err != nil {
			if strings.Contains(err.Error(), "not found in FROM clause") {
				continue // axe inapplicable au schema courant (ex: legacy sans generation_id)
			}
			t.Fatalf("[%s] scan de reference %s: %v", phase, a.name, err)
		}
		type kc struct {
			key []string
			n   int
		}
		var refs []kc
		for rows.Next() {
			vals := make([]sql.NullString, len(a.keyExprs))
			dest := make([]any, 0, len(vals)+1)
			for i := range vals {
				dest = append(dest, &vals[i])
			}
			var n int
			dest = append(dest, &n)
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				t.Fatalf("scan: %v", err)
			}
			key := make([]string, 0, len(vals))
			nullKey := false
			for _, v := range vals {
				if !v.Valid {
					nullKey = true
					break
				}
				key = append(key, v.String)
			}
			if nullKey {
				continue
			}
			refs = append(refs, kc{key, n})
		}
		rows.Close()

		stmt, err := db.PrepareContext(ctx, `SELECT COUNT(*) FROM personal_score_awards WHERE `+a.lookupWhere)
		if err != nil {
			t.Fatalf("prepare: %v", err)
		}
		for _, r := range refs {
			args := make([]any, 0, len(r.key))
			for _, k := range r.key {
				args = append(args, k)
			}
			var indexed int
			if err := stmt.QueryRowContext(ctx, args...).Scan(&indexed); err != nil {
				stmt.Close()
				t.Fatalf("lookup: %v", err)
			}
			if indexed != r.n {
				out = append(out, reproDiverg{a.name, strings.Join(r.key, "|"), r.n, indexed})
			}
		}
		stmt.Close()
	}
	return out
}

// reproReport logge le verdict d'une phase et retourne le nombre de cles en ecart.
func reproReport(t *testing.T, db *sql.DB, phase string) int {
	t.Helper()
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM personal_score_awards`).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	d := reproCheck(t, db, phase)
	if len(d) == 0 {
		t.Logf("  [%s] rows=%d : OK (0 ecart)", phase, total)
		return 0
	}
	t.Logf("  [%s] rows=%d : *** %d CLE(S) EN ECART ***", phase, total, len(d))
	for i, x := range d {
		if i >= 10 {
			t.Logf("     ... et %d autres", len(d)-10)
			break
		}
		t.Logf("     axe=%s cle=%s scan=%d indexe=%d", x.axis, x.key, x.scanned, x.indexed)
	}
	return len(d)
}

// reproAssertIndexScan verifie SUR PIECES que le lookup passe par l'ART.
//
// ATTENTION METHODE : `EXPLAIN` seul affiche TOUJOURS « Type: Sequential Scan »
// pour ce plan — en DuckDB 1.5.5 la strategie (sequentiel vs index) est choisie
// A L'EXECUTION. Seul `EXPLAIN ANALYZE` revele « Type: Index Scan ». Un controle
// base sur `EXPLAIN` seul conclurait a tort que l'index n'est jamais sollicite.
func reproAssertIndexScan(t *testing.T, db *sql.DB, mid string) {
	t.Helper()
	rows, err := db.Query(`EXPLAIN ANALYZE SELECT COUNT(*) FROM personal_score_awards WHERE match_id = '` + mid + `'`)
	if err != nil {
		t.Fatalf("explain analyze: %v", err)
	}
	var sb strings.Builder
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			t.Fatalf("explain analyze scan: %v", err)
		}
		sb.WriteString(b)
	}
	rows.Close()
	if !strings.Contains(sb.String(), "Index Scan") {
		t.Fatalf("CONTROLE INVALIDE : le lookup n'utilise PAS l'index (plan=%s)", truncate(sb.String(), 900))
	}
	t.Logf("  [plan] lookup match_id -> Index Scan CONFIRME")
}

// reproExplain verifie que le plan du lookup passe BIEN par un index scan.
func reproExplain(t *testing.T, db *sql.DB, mid string) string {
	t.Helper()
	rows, err := db.Query(`EXPLAIN SELECT COUNT(*) FROM personal_score_awards WHERE match_id = ?`, mid)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			t.Fatalf("explain scan: %v", err)
		}
		sb.WriteString(b)
	}
	return sb.String()
}

func reproVersion(t *testing.T, db *sql.DB) string {
	t.Helper()
	var v string
	if err := db.QueryRow(`SELECT version()`).Scan(&v); err != nil {
		t.Fatalf("version: %v", err)
	}
	return v
}

func reproIDs(n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, reproMatchID(i))
	}
	return out
}
