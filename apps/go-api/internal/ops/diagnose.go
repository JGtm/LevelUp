// Package ops — diagnose.go : diagnostic de schéma DuckDB.
//
// Portage du diagnostic de schéma Python (scripts utilitaires).
//
// Usage :
//
//	report, err := DiagnoseDB(DiagnoseOptions{DBPath: "data/warehouse/shared_matches_v2.duckdb"})
package ops

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DiagnoseOptions configure le diagnostic de schéma.
type DiagnoseOptions struct {
	DBPath  string
	Verbose bool
}

// TableSchema décrit le schéma d'une table.
type TableSchema struct {
	Name     string
	Columns  []ColumnInfo
	RowCount int64
}

// ColumnInfo décrit une colonne.
type ColumnInfo struct {
	Name     string
	DataType string
	Nullable bool
}

// DiagnoseReport résume l'état du schéma d'une DB.
type DiagnoseReport struct {
	DBPath  string
	Tables  []TableSchema
	Views   []string
	Indexes []IndexInfo
}

// IndexInfo décrit un index.
type IndexInfo struct {
	Name      string
	TableName string
}

// DiagnoseDB inspecte le schéma complet d'une DB DuckDB.
// Portage des diagnostics de schéma Python.
func DiagnoseDB(opts DiagnoseOptions) (DiagnoseReport, error) {
	db, err := sql.Open("duckdb", opts.DBPath+"?access_mode=read_only")
	if err != nil {
		return DiagnoseReport{}, fmt.Errorf("ouverture: %w", err)
	}
	defer db.Close()

	report := DiagnoseReport{DBPath: opts.DBPath}

	if report.Tables, err = describeAllTables(db, opts.Verbose); err != nil {
		return report, fmt.Errorf("tables: %w", err)
	}
	if report.Views, err = listViews(db); err != nil {
		return report, fmt.Errorf("vues: %w", err)
	}
	if report.Indexes, err = listIndexes(db); err != nil {
		return report, fmt.Errorf("index: %w", err)
	}
	return report, nil
}

// describeAllTables retourne le schéma de toutes les tables BASE TABLE.
func describeAllTables(db *sql.DB, verbose bool) ([]TableSchema, error) {
	rows, err := db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var tables []TableSchema
	for _, name := range tableNames {
		ts, err := describeTable(db, name, verbose)
		if err != nil {
			return nil, err
		}
		tables = append(tables, ts)
	}
	return tables, nil
}

// describeTable retourne le schéma d'une table.
func describeTable(db *sql.DB, name string, withCount bool) (TableSchema, error) {
	ts := TableSchema{Name: name}

	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = ?
		ORDER BY ordinal_position
	`, name)
	if err != nil {
		return ts, err
	}
	defer rows.Close()
	for rows.Next() {
		var col ColumnInfo
		var nullable string
		if err := rows.Scan(&col.Name, &col.DataType, &nullable); err != nil {
			return ts, err
		}
		col.Nullable = nullable == "YES"
		ts.Columns = append(ts.Columns, col)
	}
	if err := rows.Err(); err != nil {
		return ts, err
	}

	if withCount {
		if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %q", name)).Scan(&ts.RowCount); err != nil {
			ts.RowCount = -1
		}
	}
	return ts, nil
}

// listViews retourne les noms des vues.
func listViews(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_type = 'VIEW'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var views []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		views = append(views, name)
	}
	return views, rows.Err()
}

// listIndexes retourne les index de la DB.
func listIndexes(db *sql.DB) ([]IndexInfo, error) {
	rows, err := db.Query(`
		SELECT index_name, table_name
		FROM information_schema.statistics
		ORDER BY table_name, index_name
	`)
	if err != nil {
		// information_schema.statistics peut ne pas exister dans toutes les versions
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()
	var idxs []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		if err := rows.Scan(&idx.Name, &idx.TableName); err != nil {
			continue
		}
		idxs = append(idxs, idx)
	}
	return idxs, nil
}

// FormatDiagnoseReport retourne une représentation texte du rapport.
func FormatDiagnoseReport(r DiagnoseReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Diagnostic : %s ===\n\n", r.DBPath))
	sb.WriteString(fmt.Sprintf("Tables (%d) :\n", len(r.Tables)))
	for _, t := range r.Tables {
		sb.WriteString(fmt.Sprintf("  %-40s  %d cols", t.Name, len(t.Columns)))
		if t.RowCount >= 0 {
			sb.WriteString(fmt.Sprintf("  %d lignes", t.RowCount))
		}
		sb.WriteRune('\n')
	}
	sb.WriteString(fmt.Sprintf("\nVues (%d) : %s\n", len(r.Views), strings.Join(r.Views, ", ")))
	sb.WriteString(fmt.Sprintf("Index (%d)\n", len(r.Indexes)))
	return sb.String()
}
