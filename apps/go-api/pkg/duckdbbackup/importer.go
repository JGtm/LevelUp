package duckdbbackup

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2" // DuckDB driver
)

// ImportResult summarises what was imported from a staging directory.
type ImportResult struct {
	Restored   []string // keys successfully imported
	Skipped    []string // keys with no target path mapping
	DurationMs int64
}

// tsPattern strips the timestamp suffix appended by ExportTarget to table filenames.
// e.g. "match_registry_20260525_132028.parquet" → "match_registry"
var tsPattern = regexp.MustCompile(`_\d{8}_\d{6}\.parquet$`)

// ImportFromStaging walks stagingDir and imports each key's Parquet files into DuckDB.
//
// keyToPath maps a backup key (e.g. "halo_infinite:shared_matches_v2") to the
// target DuckDB file path. Keys with no mapping (empty string) are skipped.
//
// When dryRun is true the function logs what it would do but makes no changes.
func ImportFromStaging(ctx context.Context, stagingDir string, keyToPath func(string) string, dryRun bool) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{}

	top, err := os.ReadDir(stagingDir)
	if err != nil {
		return nil, fmt.Errorf("lecture staging: %w", err)
	}

	for _, e := range top {
		if !e.IsDir() {
			continue
		}
		if walkErr := walkKeyDirs(ctx, stagingDir, e.Name(), keyToPath, dryRun, result); walkErr != nil {
			return result, walkErr
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

// walkKeyDirs recurses the staging tree and calls importKeyDB for leaf directories
// (directories that directly contain .parquet files).
func walkKeyDirs(ctx context.Context, stagingDir, rel string, keyToPath func(string) string, dryRun bool, result *ImportResult) error {
	full := filepath.Join(stagingDir, rel)
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil // skip unreadable dirs silently
	}

	hasParquet := false
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".parquet") {
			hasParquet = true
			break
		}
	}

	if hasParquet {
		// Leaf: convert filesystem path separators back to ":" to reconstruct the key.
		key := strings.ReplaceAll(filepath.ToSlash(rel), "/", ":")
		dbPath := keyToPath(key)
		if dbPath == "" {
			slog.WarnContext(ctx, "restore: clé sans mapping — ignorée", "key", key)
			result.Skipped = append(result.Skipped, key)
			return nil
		}
		if dryRun {
			slog.InfoContext(ctx, "restore: [dry-run] importerait", "key", key, "db", dbPath)
			result.Restored = append(result.Restored, key)
			return nil
		}
		if err := importKeyDB(ctx, full, dbPath); err != nil {
			return fmt.Errorf("import %s: %w", key, err)
		}
		slog.InfoContext(ctx, "restore: DB importée", "key", key, "db", dbPath)
		result.Restored = append(result.Restored, key)
		return nil
	}

	// Not a leaf — recurse.
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := walkKeyDirs(ctx, stagingDir, filepath.Join(rel, e.Name()), keyToPath, dryRun, result); err != nil {
			return err
		}
	}
	return nil
}

// importKeyDB imports all Parquet files from parquetDir into a fresh DuckDB at dbPath.
// The existing file (if any) is renamed to .bak; removed on success, restored on failure.
func importKeyDB(ctx context.Context, parquetDir, dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	bakPath := dbPath + ".bak"
	hasBak := false
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Rename(dbPath, bakPath); err != nil {
			return fmt.Errorf("sauvegarde existant: %w", err)
		}
		hasBak = true
	}

	if err := doImport(ctx, parquetDir, dbPath); err != nil {
		if hasBak {
			_ = os.Remove(dbPath)
			_ = os.Rename(bakPath, dbPath)
		}
		return err
	}

	if hasBak {
		_ = os.Remove(bakPath)
	}
	return nil
}

// doImport creates a fresh DuckDB at dbPath and imports all Parquet files as tables.
func doImport(ctx context.Context, parquetDir, dbPath string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("créer DuckDB %s: %w", dbPath, err)
	}
	defer db.Close()

	var count int
	err = filepath.WalkDir(parquetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(d.Name(), ".parquet") {
			return nil
		}
		table := tsPattern.ReplaceAllString(d.Name(), "")
		// Use forward slashes — DuckDB on Windows accepts both.
		q := fmt.Sprintf(
			`CREATE OR REPLACE TABLE %q AS SELECT * FROM read_parquet('%s')`,
			table, filepath.ToSlash(path),
		)
		if _, execErr := db.ExecContext(ctx, q); execErr != nil {
			return fmt.Errorf("table %q: %w", table, execErr)
		}
		count++
		slog.DebugContext(ctx, "restore: table importée", "table", table)
		return nil
	})
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "restore: import terminé", "tables", count, "db", dbPath)
	return nil
}
