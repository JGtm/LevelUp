// purge_player_media : supprime les entrées media_files (et associations liées)
// d'un joueur dans shared_social.duckdb, afin que le prochain scan les
// réindexe avec le bon capture_start_utc (extrait du nom de fichier).
//
// Cas d'usage : un joueur dont les fichiers ont été indexés avant que la
// timezone soit configurée. capture_start_utc = mtime (faux) au lieu du
// timestamp extrait du nom de fichier (correct). INSERT OR IGNORE empêche
// toute correction sans suppression préalable.
//
// Usage :
//
//	go run ./cmd/purge_player_media --player Madina97294
//	go run ./cmd/purge_player_media --player Madina97294 --dry-run
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
)

func main() {
	playerSlug := flag.String("player", "", "slug du joueur (ex: Madina97294) — obligatoire")
	dryRun := flag.Bool("dry-run", false, "affiche les counts sans supprimer")
	flag.Parse()

	if *playerSlug == "" {
		fmt.Fprintln(os.Stderr, "purge_player_media: --player requis")
		os.Exit(1)
	}

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
	fmt.Printf("Joueur : %s\n", *playerSlug)

	db, err := sql.Open("duckdb", path)
	if err != nil {
		fatalf("ouverture: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	var nFiles, nAssoc int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM media_files WHERE player_slug = ?`, *playerSlug,
	).Scan(&nFiles); err != nil {
		fatalf("count media_files: %v", err)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM media_match_associations mma
		JOIN media_files mf ON mf.id = mma.media_file_id
		WHERE mf.player_slug = ?`, *playerSlug,
	).Scan(&nAssoc); err != nil {
		fatalf("count media_match_associations: %v", err)
	}

	fmt.Printf("Avant : media_files=%d, media_match_associations=%d\n", nFiles, nAssoc)

	if *dryRun {
		fmt.Println("[dry-run] aucune modification.")
		return
	}
	if nFiles == 0 {
		fmt.Println("Rien à supprimer.")
		return
	}

	if _, err := db.ExecContext(ctx, `
		DELETE FROM media_match_associations
		WHERE media_file_id IN (
			SELECT id FROM media_files WHERE player_slug = ?
		)`, *playerSlug,
	); err != nil {
		fatalf("DELETE media_match_associations: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM media_files WHERE player_slug = ?`, *playerSlug,
	); err != nil {
		fatalf("DELETE media_files: %v", err)
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		fatalf("CHECKPOINT: %v", err)
	}

	fmt.Printf("Supprimé : %d media_files + %d associations. CHECKPOINT OK.\n", nFiles, nAssoc)
	fmt.Println("Lance POST /api/settings/media/scan pour réindexer.")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "purge_player_media: "+format+"\n", args...)
	os.Exit(1)
}
