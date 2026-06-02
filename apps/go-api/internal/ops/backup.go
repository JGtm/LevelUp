// Package ops — backup.go : sauvegarde des bases DuckDB joueur en Parquet Zstd.
//
// Portage de scripts/backup_player.py (Python).
//
// Usage :
//
//	result, err := BackupPlayer(BackupOptions{
//	    Gamertag:         "SpartanB",
//	    PlayerDBPath:     "data/players/SpartanB/stats.duckdb",
//	    OutputDir:        "data/backups/SpartanB",
//	    CompressionLevel: 9,
//	})
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// BackupOptions configure une opération de sauvegarde.
type BackupOptions struct {
	Gamertag         string
	PlayerDBPath     string
	OutputDir        string
	CompressionLevel int  // Zstd 1-22, défaut 9
	IncludeMetadata  bool // true par défaut
}

// BackupResult résume le résultat d'un backup.
type BackupResult struct {
	Success   bool
	Message   string
	Tables    map[string]TableBackupInfo
	Timestamp string
	OutputDir string
}

// TableBackupInfo contient les stats de backup d'une table.
type TableBackupInfo struct {
	Rows          int64  `json:"rows"`
	FileSizeBytes int64  `json:"file_size_bytes"`
	ParquetPath   string `json:"parquet_path"`
}

// BackupPlayer sauvegarde toutes les tables d'une DB joueur en Parquet Zstd.
// Portage de backup_player() Python.
func BackupPlayer(ctx context.Context, opts BackupOptions) (BackupResult, error) {
	if opts.CompressionLevel == 0 {
		opts.CompressionLevel = 9
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return BackupResult{}, fmt.Errorf("création répertoire backup: %w", err)
	}

	db, err := sql.Open("duckdb", opts.PlayerDBPath+"?access_mode=read_only")
	if err != nil {
		return BackupResult{}, fmt.Errorf("ouverture DB: %w", err)
	}
	defer db.Close()

	tables, err := listBaseTables(ctx, db)
	if err != nil {
		return BackupResult{}, fmt.Errorf("liste tables: %w", err)
	}

	ts := time.Now().UTC().Format("20060102_150405")
	tableInfos := make(map[string]TableBackupInfo, len(tables))

	for _, table := range tables {
		outPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s_%s.parquet", table, ts))
		rows, err := exportTableToParquet(ctx, db, table, outPath, opts.CompressionLevel)
		if err != nil {
			return BackupResult{}, fmt.Errorf("export table %s: %w", table, err)
		}
		fi, _ := os.Stat(outPath)
		var sz int64
		if fi != nil {
			sz = fi.Size()
		}
		tableInfos[table] = TableBackupInfo{
			Rows:          rows,
			FileSizeBytes: sz,
			ParquetPath:   outPath,
		}
	}

	result := BackupResult{
		Success:   true,
		Message:   fmt.Sprintf("Backup terminé : %d tables exportées", len(tables)),
		Tables:    tableInfos,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		OutputDir: opts.OutputDir,
	}

	if opts.IncludeMetadata {
		metaPath := filepath.Join(opts.OutputDir, fmt.Sprintf("backup_metadata_%s.json", ts))
		if err := writeBackupMetadata(metaPath, opts.Gamertag, opts.CompressionLevel, result); err != nil {
			return result, fmt.Errorf("écriture métadonnées: %w", err)
		}
	}
	return result, nil
}

// listBaseTables retourne les noms des tables BASE TABLE (pas les vues).
func listBaseTables(ctx context.Context, db *sql.DB) ([]string, error) {
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

// exportTableToParquet exporte une table en Parquet Zstd et retourne le nb de lignes.
func exportTableToParquet(ctx context.Context, db *sql.DB, table, outPath string, compressionLevel int) (int64, error) {
	// Compter d'abord
	var count int64
	if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %q", table)).Scan(&count); err != nil {
		return 0, err
	}
	// COPY TO avec Zstd
	q := fmt.Sprintf(
		`COPY %q TO '%s' (FORMAT PARQUET, COMPRESSION 'zstd', COMPRESSION_LEVEL %d)`,
		table, outPath, compressionLevel,
	)
	if _, err := db.ExecContext(ctx, q); err != nil {
		return 0, err
	}
	return count, nil
}

// backupMetadata est la structure JSON de backup_metadata_*.json.
type backupMetadata struct {
	Gamertag    string                     `json:"gamertag"`
	BackupAt    string                     `json:"backup_datetime"`
	Compression string                     `json:"compression"`
	Level       int                        `json:"compression_level"`
	Tables      map[string]TableBackupInfo `json:"tables"`
}

// writeBackupMetadata écrit le JSON de métadonnées de backup.
func writeBackupMetadata(path, gamertag string, level int, result BackupResult) error {
	meta := backupMetadata{
		Gamertag:    gamertag,
		BackupAt:    result.Timestamp,
		Compression: "zstd",
		Level:       level,
		Tables:      result.Tables,
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
