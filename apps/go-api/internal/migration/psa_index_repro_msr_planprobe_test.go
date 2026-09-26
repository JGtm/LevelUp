//go:build psarepro

// psa_index_repro_msr_planprobe_test.go — MESURE VERSIONNEE du critere D-4 (plan
// backlog 2026-09-26, lot B3.1) : les trois index secondaires de match_skill_rank
// (idx_msr_match_lookup, idx_msr_rating_type, idx_msr_playlist) peuvent-ils etre
// retires sans qu'une lecture de la table ne devienne lente ?
//
//	go test -tags=psarepro -count=1 ./internal/migration/ -run MSR -v
//
// Methode : une DB FICHIER (jamais :memory:) de 12 000 lignes realistes (chaines LUSR
// de Halo Infinite, lignes LUSR + LUSR_V2 par match social, CSR sur le classe,
// versions append-only reecrites), ecrite par petits lots avec CHECKPOINT comme la
// sync, index poses AVANT le remplissage comme en prod. Copie du fichier, puis
// DROP INDEX des trois + CHECKPOINT sur la copie (le chemin de la migration). Les deux
// fichiers sont rouverts a froid, puis chaque forme de lecture est mesuree :
//   - plan : EXPLAIN ANALYZE, msrProbeRuns fois (« IDX k/n » = Index Scan emprunte k
//     fois sur n). EXPLAIN seul affiche TOUJOURS un scan sequentiel : la strategie se
//     choisit a l'execution (RAPPORT_VOLET2_INDEX_PSA_2026-08-28 §3.1) ;
//   - temps client : mediane et p90 de msrProbeRuns executions completes (lignes
//     consommees par Scan, rien d'autre), apres msrProbeWarmup executions de chauffe ;
//   - temps moteur : mediane du « Total Time » d'EXPLAIN ANALYZE.
//
// Temoins de calibrage : C0 (PK, Index Scan attendu des deux cotes, sinon Fatal) et
// C1/C2 (forme de l'incident du 2026-09-13 sur une chaine rare).
//
// Critere D-4 (ASSERTE, a la lettre) : aucune des sept formes ne depasse
// msrProbeBudget SANS index (mediane du temps client). Les resultats avec et sans
// index doivent etre identiques (DB fraiche : l'index est coherent). Les formes
// supplementaires (invariants, vue _latest) sont mesurees et journalisees, hors
// critere.
//
// RESULTAT DU 2026-09-27 (trois passages concordants, poste charge a 87-100 % CPU par
// une autre session) : critere NON TENU — F1 (Q24LUSRHistory, 12 000 lignes lues),
// F2 (citations, IN(200)) et F3 (escouade, IN(1000)) depassent 10 ms sans index, mais
// AUTANT avec index : meme plan sequentiel des deux cotes, 30/30. Seule forme qui
// emprunte un index : F4 (rating_type = 'CSR', idx_msr_rating_type), ~10x plus LENTE
// avec index (~20 ms contre ~2 ms). Lot B3 arrete a B3.1, index conserves (plan
// backlog §5 B3, journal §7).
//
// Les textes SQL sont des INSTANTANES des lecteurs de production au 2026-09-27
// (symbole cite pour chacun) : le paquet migration ne peut importer ni sync ni
// platform/duckdb (cycle). Le DDL est celui de sync/schema.go (playerSchemaSQL).
package migration

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	msrProbeRows   = 12000
	msrProbeBudget = 10 * time.Millisecond
	msrProbeRuns   = 30
	msrProbeWarmup = 3
)

func msrProbeExplain(t *testing.T, db *sql.DB, prefix string, f msrProbeForm) string {
	t.Helper()
	r, err := db.Query(prefix+" "+f.sql, f.args...)
	if err != nil {
		t.Fatalf("[%s] %s: %v", f.id, prefix, err)
	}
	defer r.Close()
	var sb strings.Builder
	for r.Next() {
		var a, b string
		if err := r.Scan(&a, &b); err != nil {
			t.Fatalf("[%s] %s scan: %v", f.id, prefix, err)
		}
		sb.WriteString(b)
	}
	return sb.String()
}

var msrProbeTotalTime = regexp.MustCompile(`Total Time: ([0-9.]+)s`)

// msrProbeAnalyze : EXPLAIN ANALYZE. La strategie de scan (index ou sequentiel) se
// choisit A L'EXECUTION en DuckDB 1.5.5 : EXPLAIN affiche toujours « Sequential
// Scan », seul EXPLAIN ANALYZE la revele (RAPPORT_VOLET2_INDEX_PSA_2026-08-28 §3.1).
// Rend (index emprunte ?, temps MOTEUR « Total Time », sans le client Go).
func msrProbeAnalyze(t *testing.T, db *sql.DB, f msrProbeForm) (bool, time.Duration) {
	t.Helper()
	out := msrProbeExplain(t, db, "EXPLAIN ANALYZE", f)
	m := msrProbeTotalTime.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("[%s] EXPLAIN ANALYZE sans « Total Time »", f.id)
	}
	sec, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("[%s] Total Time %q: %v", f.id, m[1], err)
	}
	return strings.Contains(strings.ToUpper(out), "INDEX SCAN"), time.Duration(sec * float64(time.Second))
}

// msrProbeRun execute la forme une fois et consomme toutes les lignes, comme un
// lecteur (Scan de chaque colonne), sans autre travail : c'est ce qui est chronometre.
func msrProbeRun(t *testing.T, db *sql.DB, f msrProbeForm) int {
	t.Helper()
	r, err := db.Query(f.sql, f.args...)
	if err != nil {
		t.Fatalf("[%s] query: %v", f.id, err)
	}
	defer r.Close()
	cols, _ := r.Columns()
	vals := make([]sql.RawBytes, len(cols))
	dest := make([]any, len(cols))
	for i := range vals {
		dest[i] = &vals[i]
	}
	n := 0
	for r.Next() {
		if err := r.Scan(dest...); err != nil {
			t.Fatalf("[%s] scan: %v", f.id, err)
		}
		n++
	}
	if err := r.Err(); err != nil {
		t.Fatalf("[%s] rows: %v", f.id, err)
	}
	return n
}

// msrProbeDigest : empreinte triee du resultat (egalite avec / sans index), HORS chrono.
func msrProbeDigest(t *testing.T, db *sql.DB, f msrProbeForm) (int, string) {
	t.Helper()
	r, err := db.Query(f.sql, f.args...)
	if err != nil {
		t.Fatalf("[%s] query: %v", f.id, err)
	}
	defer r.Close()
	cols, _ := r.Columns()
	var lines []string
	for r.Next() {
		vals := make([]sql.NullString, len(cols))
		dest := make([]any, len(cols))
		for i := range vals {
			dest[i] = &vals[i]
		}
		if err := r.Scan(dest...); err != nil {
			t.Fatalf("[%s] scan: %v", f.id, err)
		}
		lines = append(lines, fmt.Sprint(vals))
	}
	if err := r.Err(); err != nil {
		t.Fatalf("[%s] rows: %v", f.id, err)
	}
	sort.Strings(lines)
	return len(lines), strings.Join(lines, "\n")
}

type msrProbeResult struct {
	plan                string
	median, p90, engine time.Duration
	n                   int
	digest              string
}

func msrProbeMedian(durs []time.Duration) (time.Duration, time.Duration) {
	sort.Slice(durs, func(a, b int) bool { return durs[a] < durs[b] })
	return durs[len(durs)/2], durs[len(durs)*9/10]
}

func msrProbeMeasure(t *testing.T, db *sql.DB, f msrProbeForm) msrProbeResult {
	t.Helper()
	var res msrProbeResult
	res.n, res.digest = msrProbeDigest(t, db, f)
	for i := 0; i < msrProbeWarmup; i++ {
		msrProbeRun(t, db, f)
	}
	durs := make([]time.Duration, msrProbeRuns)
	for i := range durs {
		t0 := time.Now()
		msrProbeRun(t, db, f)
		durs[i] = time.Since(t0)
	}
	res.median, res.p90 = msrProbeMedian(durs)
	eng := make([]time.Duration, msrProbeRuns)
	indexRuns := 0
	for i := range eng {
		var idx bool
		idx, eng[i] = msrProbeAnalyze(t, db, f)
		if idx {
			indexRuns++
		}
	}
	res.engine, _ = msrProbeMedian(eng)
	res.plan = fmt.Sprintf("SEQ %d/%d", msrProbeRuns-indexRuns, msrProbeRuns)
	if indexRuns > 0 {
		res.plan = fmt.Sprintf("IDX %d/%d", indexRuns, msrProbeRuns)
	}
	return res
}

func msrProbeCopy(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func msrProbeIndexCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duckdb_indexes()
		WHERE table_name = 'match_skill_rank' AND index_name LIKE 'idx_msr_%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestMSRIndexRemovalPlanProbe : mesure D-4 (B3.1).
func TestMSRIndexRemovalPlanProbe(t *testing.T) {
	dir := t.TempDir()
	withPath := filepath.Join(dir, "with_index.duckdb")
	withoutPath := filepath.Join(dir, "without_index.duckdb")

	db := reproOpen(t, withPath)
	if err := execScript(db, msrProbeDDL); err != nil {
		t.Fatalf("DDL: %v", err)
	}
	for _, s := range msrProbeIndexes {
		reproExec(t, db, s)
	}
	t0 := time.Now()
	data := msrProbeFill(t, db)
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != msrProbeRows {
		t.Fatalf("lignes = %d, attendu %d", total, msrProbeRows)
	}
	t.Logf("remplissage : %d lignes, %d matchs, %s", total, len(data.matchIDs), time.Since(t0).Round(time.Millisecond))
	db.Close()

	msrProbeCopy(t, withPath, withoutPath)
	db = reproOpen(t, withoutPath)
	for _, n := range []string{"idx_msr_match_lookup", "idx_msr_rating_type", "idx_msr_playlist"} {
		reproExec(t, db, `DROP INDEX IF EXISTS `+n)
	}
	reproExec(t, db, `CHECKPOINT`)
	db.Close()

	with, without := reproOpen(t, withPath), reproOpen(t, withoutPath)
	defer with.Close()
	defer without.Close()
	if got := msrProbeIndexCount(t, with); got != 3 {
		t.Fatalf("DB avec index : %d idx_msr_*, attendu 3", got)
	}
	if got := msrProbeIndexCount(t, without); got != 0 {
		t.Fatalf("DB sans index : %d idx_msr_*, attendu 0", got)
	}

	t.Logf("%-4s | %-10s %9s %9s %9s | %-10s %9s %9s %9s | %6s | forme", "id",
		"plan+idx", "med", "p90", "moteur", "plan-idx", "med", "p90", "moteur", "lignes")
	for _, f := range msrProbeForms(data) {
		a := msrProbeMeasure(t, with, f)
		b := msrProbeMeasure(t, without, f)
		t.Logf("%-4s | %-10s %9s %9s %9s | %-10s %9s %9s %9s | %6d | %s", f.id,
			a.plan, a.median.Round(time.Microsecond), a.p90.Round(time.Microsecond), a.engine.Round(time.Microsecond),
			b.plan, b.median.Round(time.Microsecond), b.p90.Round(time.Microsecond), b.engine.Round(time.Microsecond),
			b.n, f.label)
		if a.n != b.n || a.digest != b.digest {
			t.Errorf("[%s] resultats differents avec (%d lignes) et sans index (%d lignes)", f.id, a.n, b.n)
		}
		if f.id == "C0" {
			if !strings.HasPrefix(a.plan, "IDX") || !strings.HasPrefix(b.plan, "IDX") {
				t.Fatalf("CALIBRAGE : lookup PK sans Index Scan (%s / %s) — le detecteur ne voit pas l'ART", a.plan, b.plan)
			}
		} else if !strings.HasPrefix(b.plan, "SEQ") {
			t.Errorf("[%s] plan sans index = %s, attendu sequentiel", f.id, b.plan)
		}
		if f.core && b.median > msrProbeBudget {
			t.Errorf("CRITERE D-4 NON TENU : [%s] %s — mediane sans index %s > %s",
				f.id, f.label, b.median, msrProbeBudget)
		}
	}
}
