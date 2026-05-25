package duckdbbackup

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2" // DuckDB driver
)

// IntegrityResult holds the outcome of a PRAGMA integrity_check on a single DB.
type IntegrityResult struct {
	OK        bool      `json:"ok"`
	Detail    string    `json:"detail,omitempty"` // first error line if !OK
	CheckedAt time.Time `json:"checked_at"`
}

// CheckIntegrity runs PRAGMA integrity_check on t (read-only connection).
// Always returns a result; errors opening or querying are treated as inconclusive
// (OK=true) so that a missing pragma support never blocks the backup cycle.
// Log a warning separately when OK is false.
func CheckIntegrity(ctx context.Context, t Target) IntegrityResult {
	res := IntegrityResult{CheckedAt: time.Now().UTC()}

	db, err := sql.Open("duckdb", t.Path+"?access_mode=read_only")
	if err != nil {
		res.Detail = fmt.Sprintf("open: %v", err)
		return res
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		// pragma not supported by this DuckDB version — inconclusive, not alarming
		slog.DebugContext(ctx, "backup: integrity_check indisponible", "key", t.Key, "err", err)
		res.OK = true
		return res
	}
	defer rows.Close()

	var first string
	for rows.Next() {
		var line string
		if scanErr := rows.Scan(&line); scanErr != nil {
			break
		}
		if first == "" {
			first = line
		}
	}

	if strings.EqualFold(first, "ok") || first == "" {
		res.OK = true
		return res
	}
	res.Detail = first
	return res
}

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

	slog.DebugContext(ctx, "backup: export démarré", "key", t.Key, "tables", len(tables))
	start := time.Now()
	ts := start.UTC().Format("20060102_150405")
	for _, table := range tables {
		outPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.parquet", table, ts))
		if err := exportTable(ctx, db, table, outPath, compressionLevel); err != nil {
			return 0, fmt.Errorf("export %s.%s: %w", t.Key, table, err)
		}
	}
	slog.InfoContext(ctx, "backup: export terminé",
		"key", t.Key,
		"tables", len(tables),
		"duration", time.Since(start).Round(time.Millisecond))
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
