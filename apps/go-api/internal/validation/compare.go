// Package validation — compare.go : comparaison de deux DBs DuckDB Go vs Python.
//
// Outil de parité pour Sprint 26 — Validation conditions réelles.
// Compare les DBs produites par le moteur Go et le moteur Python sur les mêmes
// données. Utilisé via `levelup compare-db`.
//
// Requiert CGO (driver DuckDB).
package validation

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// Statuts de comparaison (constantes)
// ─────────────────────────────────────────────────────────────────────────────

const (
	statusOK      = "OK"
	statusWarn    = "WARN"
	statusMissGo  = "MISS_GO"
	statusMissPy  = "MISS_PY"
	statusDiverge = "DIVERGE"
)

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

// TableComparison est le résultat de comparaison pour une table.
type TableComparison struct {
	TableName  string
	RowsGo     int64
	RowsPython int64
	Delta      int64   // RowsGo - RowsPython
	DeltaPct   float64 // Delta / RowsPython * 100 (NaN si RowsPython=0)
	Status     string  // "OK" | "WARN" | "MISS_GO" | "MISS_PY" | "DIVERGE"
	Notes      []string
}

// BitmaskStats est l'analyse des bitmasks de backfill.
type BitmaskStats struct {
	Table          string
	Column         string
	ZeroCount      int64 // Lignes avec bitmask = 0 (non backfillées)
	NonZeroCount   int64
	ZeroCountPy    int64
	ZeroCountPyPct float64 // % lignes non backfillées côté Python
	ZeroCountGoPct float64 // % lignes non backfillées côté Go
	Status         string  // "OK" | "WARN" | "ERROR"
}

// MatchOverlap est l'analyse du chevauchement des match_ids.
type MatchOverlap struct {
	OnlyInGo     int64
	OnlyInPython int64
	InBoth       int64
	JaccardScore float64 // InBoth / (InBoth + OnlyInGo + OnlyInPython)
}

// ComparisonReport est le rapport complet de parité.
type ComparisonReport struct {
	GeneratedAt  time.Time
	GoDBPath     string
	PythonDBPath string
	Tables       []TableComparison
	Bitmasks     []BitmaskStats
	MatchOverlap *MatchOverlap // nil si tables absentes
	OverallOK    bool
	Summary      string
}

// ─────────────────────────────────────────────────────────────────────────────
// Comparaison
// ─────────────────────────────────────────────────────────────────────────────

// ComparePlayerDBs compare la DB joueur Go vs Python.
//
// Les deux DBs doivent être stats.duckdb produits par le moteur Go et Python
// pour le même joueur. La comparaison est read-only.
func ComparePlayerDBs(goDBPath, pythonDBPath string) (*ComparisonReport, error) {
	goDb, err := sql.Open("duckdb", goDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open go DB: %w", err)
	}
	defer goDb.Close()

	pyDb, err := sql.Open("duckdb", pythonDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open python DB: %w", err)
	}
	defer pyDb.Close()

	report := &ComparisonReport{
		GeneratedAt:  time.Now(),
		GoDBPath:     goDBPath,
		PythonDBPath: pythonDBPath,
	}

	// 1. Comparaison row counts par table
	goTables, err := listTables(goDb)
	if err != nil {
		return nil, fmt.Errorf("list go tables: %w", err)
	}
	pyTables, err := listTables(pyDb)
	if err != nil {
		return nil, fmt.Errorf("list python tables: %w", err)
	}

	report.Tables = compareTableCounts(goDb, pyDb, goTables, pyTables)

	// 2. Bitmask analysis (player_match_enrichment)
	bitmasks, err := compareBitmasks(goDb, pyDb)
	if err == nil {
		report.Bitmasks = bitmasks
	}

	// 3. Match ID overlap (player_match_enrichment)
	overlap, err := compareMatchIDs(goDb, pyDb)
	if err == nil {
		report.MatchOverlap = overlap
	}

	report.OverallOK = isReportOK(report)
	report.Summary = buildSummary(report)
	return report, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers internes
// ─────────────────────────────────────────────────────────────────────────────

func listTables(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(
		"SELECT table_name FROM information_schema.tables WHERE table_schema='main' AND table_type='BASE TABLE'",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables[name] = true
	}
	return tables, rows.Err()
}

func countRows(db *sql.DB, table string) (int64, error) {
	var n int64
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n)
	return n, err
}

func compareTableCounts(goDb, pyDb *sql.DB, goTables, pyTables map[string]bool) []TableComparison {
	// Union des tables des deux DBs
	allTables := make(map[string]bool)
	for t := range goTables {
		allTables[t] = true
	}
	for t := range pyTables {
		allTables[t] = true
	}

	// Tables non pertinentes à comparer (schéma interne)
	skip := map[string]bool{
		"schema_migrations": true,
	}

	var result []TableComparison //nolint:prealloc
	for table := range allTables {
		if skip[table] {
			continue
		}
		tc := TableComparison{TableName: table}

		inGo := goTables[table]
		inPy := pyTables[table]

		switch {
		case !inGo && !inPy:
			continue
		case inGo && !inPy:
			tc.Status = statusMissPy
			tc.Notes = append(tc.Notes, "table absente côté Python")
			tc.RowsGo, _ = countRows(goDb, table)
		case !inGo && inPy:
			tc.Status = statusMissGo
			tc.Notes = append(tc.Notes, "table absente côté Go")
			tc.RowsPython, _ = countRows(pyDb, table)
		default:
			tc.RowsGo, _ = countRows(goDb, table)
			tc.RowsPython, _ = countRows(pyDb, table)
			tc.Delta = tc.RowsGo - tc.RowsPython
			if tc.RowsPython > 0 {
				tc.DeltaPct = float64(tc.Delta) / float64(tc.RowsPython) * 100
			}
			tc.Status = classifyDelta(tc.Delta, tc.RowsPython)
		}
		result = append(result, tc)
	}
	return result
}

func classifyDelta(delta, pyRows int64) string {
	if delta == 0 {
		return "OK"
	}
	if pyRows == 0 {
		return statusWarn
	}
	pct := math.Abs(float64(delta) / float64(pyRows) * 100)
	if pct <= 1.0 {
		return statusWarn // ≤ 1% de différence : toléré (délai d'indexation)
	}
	return statusDiverge
}

func compareBitmasks(goDb, pyDb *sql.DB) ([]BitmaskStats, error) {
	type bitmaskQuery struct {
		table  string
		column string
	}
	queries := []bitmaskQuery{
		{"player_match_enrichment", ""}, // pas de bitmask propre dans cette table v6
		{"sync_meta", ""},
	}
	_ = queries // placeholder — les vraies stats bitmask sont dans shared (match_registry)

	// Pour les DBs joueur (stats.duckdb), le champ bitmask principal est absent
	// (migré dans shared_matches_v2). On vérifie player_match_enrichment.performance_score.
	stats := make([]BitmaskStats, 0, 1)

	var goNull, pyNull int64
	errGo := goDb.QueryRow(
		"SELECT COUNT(*) FROM player_match_enrichment WHERE performance_score IS NULL",
	).Scan(&goNull)
	errPy := pyDb.QueryRow(
		"SELECT COUNT(*) FROM player_match_enrichment WHERE performance_score IS NULL",
	).Scan(&pyNull)

	if errGo != nil || errPy != nil {
		return nil, fmt.Errorf("bitmask check: go=%v py=%v", errGo, errPy)
	}

	var goTotal, pyTotal int64
	_ = goDb.QueryRow("SELECT COUNT(*) FROM player_match_enrichment").Scan(&goTotal)
	_ = pyDb.QueryRow("SELECT COUNT(*) FROM player_match_enrichment").Scan(&pyTotal)

	var goPct, pyPct float64
	if goTotal > 0 {
		goPct = float64(goNull) / float64(goTotal) * 100
	}
	if pyTotal > 0 {
		pyPct = float64(pyNull) / float64(pyTotal) * 100
	}

	status := "OK"
	if math.Abs(goPct-pyPct) > 5 {
		status = statusWarn
	}
	if math.Abs(goPct-pyPct) > 20 {
		status = "ERROR"
	}

	stats = append(stats, BitmaskStats{
		Table:          "player_match_enrichment",
		Column:         "performance_score (NULL ratio)",
		ZeroCount:      goNull,
		ZeroCountPy:    pyNull,
		ZeroCountGoPct: goPct,
		ZeroCountPyPct: pyPct,
		Status:         status,
	})
	return stats, nil
}

func compareMatchIDs(goDb, pyDb *sql.DB) (*MatchOverlap, error) {
	// Charger les match_ids de player_match_enrichment dans les deux DBs
	goIDs, err := loadMatchIDs(goDb)
	if err != nil {
		return nil, fmt.Errorf("load go match_ids: %w", err)
	}
	pyIDs, err := loadMatchIDs(pyDb)
	if err != nil {
		return nil, fmt.Errorf("load python match_ids: %w", err)
	}

	both := int64(0)
	onlyGo := int64(0)
	for id := range goIDs {
		if pyIDs[id] {
			both++
		} else {
			onlyGo++
		}
	}
	onlyPy := int64(len(pyIDs)) - both

	union := both + onlyGo + onlyPy
	jaccard := 0.0
	if union > 0 {
		jaccard = float64(both) / float64(union)
	}

	return &MatchOverlap{
		InBoth:       both,
		OnlyInGo:     onlyGo,
		OnlyInPython: onlyPy,
		JaccardScore: jaccard,
	}, nil
}

func loadMatchIDs(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT match_id FROM player_match_enrichment")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

func isReportOK(r *ComparisonReport) bool {
	for _, tc := range r.Tables {
		if tc.Status == statusDiverge || tc.Status == statusMissGo {
			return false
		}
	}
	for _, b := range r.Bitmasks {
		if b.Status == "ERROR" {
			return false
		}
	}
	if r.MatchOverlap != nil && r.MatchOverlap.JaccardScore < 0.95 {
		return false
	}
	return true
}

func buildSummary(r *ComparisonReport) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== Rapport parité Go vs Python (%s) ===\n\n",
		r.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "► Go DB    : %s\n", r.GoDBPath)
	fmt.Fprintf(&sb, "► Python DB: %s\n\n", r.PythonDBPath)

	// Tables
	sb.WriteString("── Tables ──────────────────────────────────────────\n")
	ok, warn, diverge, miss := 0, 0, 0, 0
	for _, tc := range r.Tables {
		switch tc.Status {
		case "OK":
			ok++
		case statusWarn:
			warn++
		case statusDiverge:
			diverge++
		case statusMissGo, statusMissPy:
			miss++
		}
		icon := statusIcon(tc.Status)
		fmt.Fprintf(&sb, "  %s %-35s go=%d py=%d delta=%+d",
			icon, tc.TableName, tc.RowsGo, tc.RowsPython, tc.Delta)
		if !math.IsNaN(tc.DeltaPct) && tc.RowsPython > 0 {
			fmt.Fprintf(&sb, " (%.1f%%)", tc.DeltaPct)
		}
		sb.WriteByte('\n')
	}
	fmt.Fprintf(&sb, "\n  ✅ %d OK  ⚠ %d avertissements  ❌ %d divergences  🔍 %d absentes\n\n",
		ok, warn, diverge, miss)

	// Match overlap
	if r.MatchOverlap != nil {
		o := r.MatchOverlap
		sb.WriteString("── Match ID Overlap ────────────────────────────────\n")
		fmt.Fprintf(&sb, "  Communs    : %d\n", o.InBoth)
		fmt.Fprintf(&sb, "  Go seulement: %d\n", o.OnlyInGo)
		fmt.Fprintf(&sb, "  Python seul : %d\n", o.OnlyInPython)
		fmt.Fprintf(&sb, "  Jaccard     : %.3f %s\n\n", o.JaccardScore,
			jaccardLabel(o.JaccardScore))
	}

	// Bitmasks
	if len(r.Bitmasks) > 0 {
		sb.WriteString("── Bitmasks / Enrichissement ───────────────────────\n")
		for _, b := range r.Bitmasks {
			fmt.Fprintf(&sb, "  %s %s.%s : go_null=%.1f%%  py_null=%.1f%%\n",
				statusIcon(b.Status), b.Table, b.Column, b.ZeroCountGoPct, b.ZeroCountPyPct)
		}
		sb.WriteByte('\n')
	}

	// Verdict final
	if r.OverallOK {
		sb.WriteString("✅ PARITÉ VALIDÉE — Gate Phase 4 technicalement passée.\n")
	} else {
		sb.WriteString("❌ DIVERGENCES DÉTECTÉES — analysez les tables DIVERGE/MISS_GO.\n")
	}
	return sb.String()
}

func statusIcon(status string) string {
	switch status {
	case "OK":
		return "✅"
	case statusWarn:
		return "⚠️"
	case statusDiverge:
		return "❌"
	case statusMissGo:
		return "🔍"
	case statusMissPy:
		return "📭"
	default:
		return "  "
	}
}

func jaccardLabel(j float64) string {
	switch {
	case j >= 0.99:
		return "✅ parfait"
	case j >= 0.95:
		return "✅ acceptable"
	case j >= 0.80:
		return "⚠️  attention"
	default:
		return "❌ divergence majeure"
	}
}
