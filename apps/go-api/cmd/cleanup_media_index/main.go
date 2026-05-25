// cleanup_media_index : one-shot CLI pour vider les tables media_files +
// media_match_associations de shared_social.duckdb (sans toucher aux autres
// tables : media_likes, match_favorites, player_notifications, player_records).
//
// Contexte : après le fix WAL/timezone du 2026-05-25, les 27 rows résiduelles
// dans media_files ont capture_start_utc=NULL (pré-fix). Un INSERT OR IGNORE
// du post-sync hook ne les touchera pas. Ce CLI les supprime pour que la
// prochaine indexation reparte propre.
//
// Usage :
//
//	go run ./cmd/cleanup_media_index
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "affiche les counts sans supprimer")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatalf("config: %v", err)
	}
	pr := title.NewPathResolver(cfg.RepoRoot)
	path := pr.SharedSocialDBPath(title.DefaultSlug)

	if _, err := os.Stat(path); err != nil {
		fatalf("shared_social.duckdb introuvable : %v", err)
	}

	fmt.Printf("Cible : %s\n", path)

	db, err := sql.Open("duckdb", path)
	if err != nil {
		fatalf("ouverture: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	var nFiles, nAssoc int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_files`).Scan(&nFiles); err != nil {
		fatalf("count media_files: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_match_associations`).Scan(&nAssoc); err != nil {
		fatalf("count media_match_associations: %v", err)
	}
	fmt.Printf("Avant : media_files=%d, media_match_associations=%d\n", nFiles, nAssoc)

	if *dryRun {
		fmt.Println("[dry-run] aucune modification.")
		return
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM media_match_associations`); err != nil {
		fatalf("DELETE media_match_associations: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM media_files`); err != nil {
		fatalf("DELETE media_files: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		fatalf("CHECKPOINT: %v", err)
	}

	fmt.Println("Index media vidé + CHECKPOINT OK.")
	fmt.Printf("Au prochain post-sync, IndexMedia repartira de zéro depuis %s\n",
		filepath.Join(cfg.RepoRoot, "data", "titles", title.DefaultSlug, "players"))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cleanup_media_index: "+format+"\n", args...)
	os.Exit(1)
}
