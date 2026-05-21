// cmd/diag_player_schemas — diagnostic des schémas player DBs.
//
// Sortie : pour chaque joueur, liste des tables présentes + indique celles
// qui sont attendues mais absentes. Utilisé pour confirmer/infirmer le
// bug 2 du audit DuckDB ATTACH 2026-05-21 (media_files manquante chez JGtm).
//
// Usage : go run ./cmd/diag_player_schemas
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Tables attendues dans stats.duckdb (cf. steps_player.go).
var expectedPlayerTables = []string{
	"player_match_enrichment",
	"personal_score_awards",
	"match_citations",
	"media_files",
	"career_progression",
	"sessions",
	"sync_meta",
	"match_skill_rank",
}

func main() {
	ctx := context.Background()
	playersDir := "data/titles/halo_infinite/players"

	entries, err := os.ReadDir(playersDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read players dir: %v\n", err)
		os.Exit(1)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gt := e.Name()
		dbPath := filepath.Join(playersDir, gt, "stats.duckdb")
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		fmt.Printf("══ %s (%s) ══\n", gt, dbPath)
		if err := inspectPlayerDB(ctx, dbPath); err != nil {
			fmt.Printf("  ERR: %v\n\n", err)
			continue
		}
		fmt.Println()
	}
}

func inspectPlayerDB(ctx context.Context, path string) error {
	// Ouverture R/O sécurisée — un autre processus ne doit pas tenir le lock.
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	// Lister tables main.
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'main'
		ORDER BY table_name
	`)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	tables := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return err
		}
		tables[n] = true
	}
	rows.Close()

	// Trier pour affichage déterministe.
	names := make([]string, 0, len(tables))
	for n := range tables {
		names = append(names, n)
	}
	sort.Strings(names)

	fmt.Printf("  tables présentes (%d) : %v\n", len(names), names)

	// Tables attendues mais absentes.
	var missing []string
	for _, t := range expectedPlayerTables {
		if !tables[t] {
			missing = append(missing, t)
		}
	}
	if len(missing) > 0 {
		fmt.Printf("  ⚠ tables MANQUANTES : %v\n", missing)
	}

	// Pour media_files si présente : vérifier le nom de colonne (filename vs file_name).
	if tables["media_files"] {
		colRows, err := db.QueryContext(ctx, `
			SELECT column_name FROM information_schema.columns
			WHERE table_schema = 'main' AND table_name = 'media_files'
			ORDER BY ordinal_position
		`)
		if err != nil {
			return fmt.Errorf("media_files cols: %w", err)
		}
		var cols []string
		for colRows.Next() {
			var c string
			if err := colRows.Scan(&c); err != nil {
				colRows.Close()
				return err
			}
			cols = append(cols, c)
		}
		colRows.Close()
		fmt.Printf("  media_files cols : %v\n", cols)
	}

	// Compter rows par table importante.
	for _, t := range []string{"career_progression", "player_match_enrichment", "media_files", "match_skill_rank"} {
		if !tables[t] {
			continue
		}
		var n int
		if err := db.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, t)).Scan(&n); err == nil {
			fmt.Printf("  %s : %d rows\n", t, n)
		}
	}

	return nil
}
