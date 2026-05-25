package duckdbbackup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2" // DuckDB driver
)

// ExportTarget exports all BASE TABLE tables from t to outputDir as Parquet+zstd.
//
// The connection is opened with ?access_mode=read_only, making it safe to call
// while another connection (e.g. the API server) holds the same file open in
// read-write mode within the same process.
//
// Returns the number of tables exported.
func ExportTarget(ctx context.Context, t Target, outputDir string, compressionLevel int) (int, error) {
	if compressionLevel <= 0 {
		compressionLevel = 9
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", outputDir, err)
	}

	db, err := sql.Open("duckdb", t.Path+"?access_mode=read_only")
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", t.Path, err)
	}
	defer db.Close()

	tables, err := listTables(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("list tables %s: %w", t.Key, err)
	}

	ts := time.Now().UTC().Format("20060102_150405")
	for _, table := range tables {
		outPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.parquet", table, ts))
		if err := exportTable(ctx, db, table, outPath, compressionLevel); err != nil {
			return 0, fmt.Errorf("export %s.%s: %w", t.Key, table, err)
		}
	}
	return len(tables), nil
}

func listTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func exportTable(ctx context.Context, db *sql.DB, table, outPath string, compressionLevel int) error {
	q := fmt.Sprintf(
		`COPY %q TO '%s' (FORMAT PARQUET, COMPRESSION 'zstd', COMPRESSION_LEVEL %d)`,
		table, outPath, compressionLevel,
	)
	_, err := db.ExecContext(ctx, q)
	return err
}
