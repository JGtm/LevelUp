// Package ops — archive.go : archivage des anciens matchs en Parquet temporel.
//
// Portage de scripts/archive_season.py (Python).
//
// Usage :
//
//	result, err := ArchiveMatches(ArchiveOptions{
//	    Gamertag:          "SpartanB",
//	    PlayerDBPath:      "data/players/SpartanB/stats.duckdb",
//	    SharedDBPath:      "data/warehouse/shared_matches_v2.duckdb",
//	    ArchiveDir:        "data/players/SpartanB/archive",
//	    CutoffDate:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
//	    DeleteAfter:       false,
//	    DryRun:            false,
//	    ByYear:            true,
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

// ArchiveOptions configure une opération d'archivage.
type ArchiveOptions struct {
	Gamertag     string
	PlayerDBPath string // stats.duckdb du joueur
	SharedDBPath string // shared_matches_v2.duckdb
	ArchiveDir   string // ex: data/players/{gt}/archive/
	CutoffDate   time.Time
	DeleteAfter  bool // supprimer les matchs de shared après archivage
	DryRun       bool
	ByYear       bool // partitionner par année
	XUID         string
}

// ArchiveResult résume le résultat de l'archivage.
type ArchiveResult struct {
	Success       bool
	Message       string
	MatchCount    int
	ArchivedFiles []string
	DryRun        bool
}

// ArchiveMatches exporte les matchs antérieurs à cutoffDate en Parquet.
// Portage de archive_matches() Python.
func ArchiveMatches(ctx context.Context, opts ArchiveOptions) (ArchiveResult, error) {
	if err := os.MkdirAll(opts.ArchiveDir, 0o755); err != nil {
		return ArchiveResult{}, fmt.Errorf("création archive dir: %w", err)
	}

	db, err := sql.Open("duckdb", opts.SharedDBPath+"?access_mode=read_only")
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("ouverture shared DB: %w", err)
	}
	defer db.Close()

	// Compter les matchs archivables
	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ? AND mr.start_time < ?
	`, opts.XUID, opts.CutoffDate).Scan(&count); err != nil {
		return ArchiveResult{}, fmt.Errorf("comptage matchs: %w", err)
	}

	if count == 0 {
		return ArchiveResult{
			Success: true,
			Message: "Aucun match à archiver",
			DryRun:  opts.DryRun,
		}, nil
	}

	if opts.DryRun {
		return ArchiveResult{
			Success:    true,
			Message:    fmt.Sprintf("[dry-run] %d matchs seraient archivés", count),
			MatchCount: count,
			DryRun:     true,
		}, nil
	}

	var files []string
	if opts.ByYear {
		years, err := listArchivableYears(ctx, db, opts.XUID, opts.CutoffDate)
		if err != nil {
			return ArchiveResult{}, err
		}
		for _, year := range years {
			outPath := filepath.Join(opts.ArchiveDir, fmt.Sprintf("matches_%d.parquet", year))
			if err := exportYearToParquet(ctx, db, opts.XUID, year, outPath); err != nil {
				return ArchiveResult{}, fmt.Errorf("export année %d: %w", year, err)
			}
			files = append(files, outPath)
		}
	} else {
		outPath := filepath.Join(opts.ArchiveDir, fmt.Sprintf("matches_before_%s.parquet",
			opts.CutoffDate.Format("20060102")))
		if err := exportAllToParquet(ctx, db, opts.XUID, opts.CutoffDate, outPath); err != nil {
			return ArchiveResult{}, fmt.Errorf("export: %w", err)
		}
		files = append(files, outPath)
	}

	// Écrire l'index d'archive
	if err := writeArchiveIndex(opts.ArchiveDir, opts.Gamertag, opts.CutoffDate, count, files); err != nil {
		return ArchiveResult{}, fmt.Errorf("écriture index: %w", err)
	}

	if opts.DeleteAfter && opts.XUID != "" {
		rwDB, err := sql.Open("duckdb", opts.SharedDBPath)
		if err != nil {
			return ArchiveResult{}, fmt.Errorf("ouverture shared DB rw: %w", err)
		}
		defer rwDB.Close()
		if err := deleteArchivedParticipants(ctx, rwDB, opts.XUID, opts.CutoffDate); err != nil {
			return ArchiveResult{}, fmt.Errorf("suppression: %w", err)
		}
	}

	return ArchiveResult{
		Success:       true,
		Message:       fmt.Sprintf("%d matchs archivés en %d fichier(s)", count, len(files)),
		MatchCount:    count,
		ArchivedFiles: files,
		DryRun:        false,
	}, nil
}

func listArchivableYears(ctx context.Context, db *sql.DB, xuid string, cutoff time.Time) ([]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT YEAR(mr.start_time) AS yr
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ? AND mr.start_time < ?
		ORDER BY yr
	`, xuid, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var years []int
	for rows.Next() {
		var y int
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	return years, rows.Err()
}

func exportYearToParquet(ctx context.Context, db *sql.DB, xuid string, year int, outPath string) error {
	q := fmt.Sprintf(`
		COPY (
			SELECT mp.* FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = '%s' AND YEAR(mr.start_time) = %d
		) TO '%s' (FORMAT PARQUET, COMPRESSION 'zstd', COMPRESSION_LEVEL 9)
	`, xuid, year, outPath)
	_, err := db.ExecContext(ctx, q)
	return err
}

func exportAllToParquet(ctx context.Context, db *sql.DB, xuid string, cutoff time.Time, outPath string) error {
	q := fmt.Sprintf(`
		COPY (
			SELECT mp.* FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			WHERE mp.xuid = '%s' AND mr.start_time < '%s'
		) TO '%s' (FORMAT PARQUET, COMPRESSION 'zstd', COMPRESSION_LEVEL 9)
	`, xuid, cutoff.Format(time.RFC3339), outPath)
	_, err := db.ExecContext(ctx, q)
	return err
}

func deleteArchivedParticipants(ctx context.Context, db *sql.DB, xuid string, cutoff time.Time) error {
	_, err := db.ExecContext(ctx, `
		DELETE FROM match_participants
		WHERE xuid = ?
		  AND match_id IN (
		      SELECT match_id FROM match_registry WHERE start_time < ?
		  )
	`, xuid, cutoff)
	return err
}

type archiveIndex struct {
	Gamertag   string   `json:"gamertag"`
	CutoffDate string   `json:"cutoff_date"`
	CreatedAt  string   `json:"created_at"`
	MatchCount int      `json:"match_count"`
	Files      []string `json:"files"`
}

func writeArchiveIndex(dir, gamertag string, cutoff time.Time, count int, files []string) error {
	idx := archiveIndex{
		Gamertag:   gamertag,
		CutoffDate: cutoff.Format(time.RFC3339),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		MatchCount: count,
		Files:      files,
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "archive_index.json"), data, 0o644)
}
