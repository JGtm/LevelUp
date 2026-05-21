// Package ops — restore.go : restauration d'une DB joueur depuis des fichiers Parquet.
//
// Portage de scripts/restore_player.py (Python).
//
// Usage :
//
//	result, err := RestorePlayer(RestoreOptions{
//	    Gamertag:     "SpartanB",
//	    PlayerDBPath: "data/players/SpartanB/stats.duckdb",
//	    BackupDir:    "data/backups/SpartanB",
//	    Replace:      false,
//	    DryRun:       false,
//	})
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

// RestoreOptions configure une opération de restauration.
type RestoreOptions struct {
	Gamertag     string
	PlayerDBPath string
	BackupDir    string
	Tables       []string // nil = toutes les tables
	Replace      bool     // DROP TABLE avant restauration si true
	DryRun       bool     // lister sans modifier
}

// RestoreResult résume le résultat d'une restauration.
type RestoreResult struct {
	Success      bool
	Message      string
	TablesLoaded []string
	DryRun       bool
	BackupTS     string
}

// RestorePlayer restaure les tables d'une DB joueur depuis les fichiers Parquet.
// Portage de restore_player() Python.
func RestorePlayer(ctx context.Context, opts RestoreOptions) (RestoreResult, error) {
	parquetFiles, ts, err := findLatestParquetFiles(opts.BackupDir)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("recherche fichiers backup: %w", err)
	}
	if len(parquetFiles) == 0 {
		return RestoreResult{}, fmt.Errorf("aucun fichier Parquet trouvé dans %s", opts.BackupDir)
	}

	// Filtrer si --tables spécifié
	if len(opts.Tables) > 0 {
		want := make(map[string]bool, len(opts.Tables))
		for _, t := range opts.Tables {
			want[t] = true
		}
		filtered := make(map[string]string)
		for t, p := range parquetFiles {
			if want[t] {
				filtered[t] = p
			}
		}
		parquetFiles = filtered
	}

	result := RestoreResult{
		DryRun:   opts.DryRun,
		BackupTS: ts,
	}

	if opts.DryRun {
		for t := range parquetFiles {
			result.TablesLoaded = append(result.TablesLoaded, t)
		}
		sort.Strings(result.TablesLoaded)
		result.Success = true
		result.Message = fmt.Sprintf("[dry-run] %d tables seraient restaurées", len(parquetFiles))
		return result, nil
	}

	db, err := sql.Open("duckdb", opts.PlayerDBPath)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("ouverture DB: %w", err)
	}
	defer db.Close()

	for table, parqPath := range parquetFiles {
		if err := restoreTable(ctx, db, table, parqPath, opts.Replace); err != nil {
			return result, fmt.Errorf("restauration table %s: %w", table, err)
		}
		result.TablesLoaded = append(result.TablesLoaded, table)
	}
	sort.Strings(result.TablesLoaded)
	result.Success = true
	result.Message = fmt.Sprintf("%d tables restaurées depuis backup %s", len(result.TablesLoaded), ts)
	return result, nil
}

// findLatestParquetFiles cherche les Parquet les plus récents dans backupDir.
// Retourne map[tableName]parquetPath et le timestamp correspondant.
func findLatestParquetFiles(backupDir string) (map[string]string, string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, "", err
	}

	// Chercher le timestamp le plus récent depuis backup_metadata_*.json
	latestTS := ""
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "backup_metadata_") && strings.HasSuffix(e.Name(), ".json") {
			ts := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "backup_metadata_"), ".json")
			if ts > latestTS {
				latestTS = ts
			}
		}
	}

	// Fallback : prendre le timestamp le plus récent parmi les .parquet
	if latestTS == "" {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".parquet") {
				parts := strings.Split(strings.TrimSuffix(e.Name(), ".parquet"), "_")
				if len(parts) >= 2 {
					ts := parts[len(parts)-2] + "_" + parts[len(parts)-1]
					if ts > latestTS {
						latestTS = ts
					}
				}
			}
		}
	}

	if latestTS == "" {
		return nil, "", nil
	}

	// Collecter tous les Parquet correspondant à ce timestamp
	result := make(map[string]string)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".parquet") {
			continue
		}
		if !strings.HasSuffix(strings.TrimSuffix(name, ".parquet"), latestTS) {
			continue
		}
		table := strings.TrimSuffix(name, "_"+latestTS+".parquet")
		result[table] = filepath.Join(backupDir, name)
	}
	return result, latestTS, nil
}

// restoreTable restaure une table depuis un fichier Parquet.
func restoreTable(ctx context.Context, db *sql.DB, table, parqPath string, replace bool) error {
	if replace {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %q", table)); err != nil {
			return fmt.Errorf("DROP TABLE: %w", err)
		}
	}
	q := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %q AS SELECT * FROM read_parquet('%s')`,
		table, parqPath,
	)
	if _, err := db.ExecContext(ctx, q); err != nil {
		return err
	}
	return nil
}

// FindAvailableBackups liste les timestamps de backup disponibles dans un répertoire.
func FindAvailableBackups(backupDir string) ([]string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	seen := make(map[string]bool)
	var timestamps []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "backup_metadata_") {
			ts := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "backup_metadata_"), ".json")
			if !seen[ts] {
				seen[ts] = true
				timestamps = append(timestamps, ts)
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(timestamps)))
	return timestamps, nil
}

// ReadBackupMetadata lit le backup_metadata_*.json pour un timestamp donné.
func ReadBackupMetadata(backupDir, ts string) (map[string]any, error) {
	path := filepath.Join(backupDir, fmt.Sprintf("backup_metadata_%s.json", ts))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta map[string]any
	return meta, json.Unmarshal(data, &meta)
}
