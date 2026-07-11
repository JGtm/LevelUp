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
//	go run ./cmd/cleanup_media_index                      # vide tout (comportement historique)
//	go run ./cmd/cleanup_media_index --foreign-only       # purge UNIQUEMENT les fichiers
//	    revendiqués par un AUTRE titre (DEC-8, plan résidus H5 : les 84 clips
//	    Halo_5_Guardians-* indexés à tort dans le shared_social halo_infinite).
//	    Motifs = title.toml media_filename_prefixes des autres titres.
//	Flags communs : --dry-run (aucune écriture), --title <slug> (défaut halo_infinite).
//
// ATTENTION : un seul writer par DB — arrêter le serveur avant (le CLI ouvre en RW).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "affiche les counts sans supprimer")
	foreignOnly := flag.Bool("foreign-only", false, "purge uniquement les médias revendiqués par un autre titre (préfixes title.toml)")
	titleSlug := flag.String("title", title.DefaultSlug, "slug du titre dont on nettoie le shared_social")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fatalf("config: %v", err)
	}
	// Registre chargé depuis config/titles/ (comme au boot serveur) : nécessaire pour
	// connaître les media_filename_prefixes des autres titres (--foreign-only).
	title.SetDefaultRegistry(title.NewRegistryFromConfig(cfg.RepoRoot, slog.Default()))
	pr := title.NewPathResolver(cfg.RepoRoot)
	path := pr.SharedSocialDBPath(*titleSlug)

	if _, err := os.Stat(path); err != nil {
		fatalf("shared_social.duckdb introuvable : %v", err)
	}

	fmt.Printf("Cible : %s\n", path)

	if *foreignOnly {
		purgeForeignMedia(path, *titleSlug, *dryRun)
		return
	}

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
		filepath.Join(cfg.RepoRoot, "data", "titles", *titleSlug, "players"))
}

// purgeForeignMedia supprime du shared_social du titre `slug` les media_files dont le
// nom de fichier matche un préfixe revendiqué par un AUTRE titre (DEC-8) + leurs
// associations (_history) et likes. Idempotente (0 ligne au 2e run). Écritures suivies
// d'un CHECKPOINT (ADR 0022 — sans lui le WAL peut être perdu).
func purgeForeignMedia(path, slug string, dryRun bool) {
	prefixes := title.DefaultRegistry().ForeignMediaFilenamePrefixes(slug)
	if len(prefixes) == 0 {
		fmt.Printf("Aucun préfixe étranger déclaré (autres titres sans media_filename_prefixes) — rien à purger.\n")
		return
	}
	fmt.Printf("Préfixes étrangers à purger de %s : %v\n", slug, prefixes)

	db, err := sql.Open("duckdb", path)
	if err != nil {
		fatalf("ouverture: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	// Prédicat : basename(file_path) commence par un préfixe étranger (insensible à
	// la casse). file_name porte déjà le basename.
	var conds []string
	var args []any
	for _, p := range prefixes {
		conds = append(conds, "lower(file_name) LIKE lower(?) || '%'")
		args = append(args, p)
	}
	where := strings.Join(conds, " OR ")

	var nFiles, nAssoc, nLikes int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM media_files WHERE `+where, args...).Scan(&nFiles); err != nil {
		fatalf("count media_files (foreign): %v", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM media_match_associations_history
		 WHERE media_file_id IN (SELECT id FROM media_files WHERE `+where+`)`, args...).Scan(&nAssoc); err != nil {
		fatalf("count associations (foreign): %v", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM media_likes
		 WHERE media_path IN (SELECT file_path FROM media_files WHERE `+where+`)`, args...).Scan(&nLikes); err != nil {
		fatalf("count likes (foreign): %v", err)
	}
	fmt.Printf("À purger : media_files=%d, associations_history=%d, likes=%d\n", nFiles, nAssoc, nLikes)

	if dryRun {
		fmt.Println("[dry-run] aucune modification.")
		return
	}
	if nFiles == 0 && nAssoc == 0 && nLikes == 0 {
		fmt.Println("Rien à purger (déjà propre).")
		return
	}

	if _, err := db.ExecContext(ctx,
		`DELETE FROM media_likes
		 WHERE media_path IN (SELECT file_path FROM media_files WHERE `+where+`)`, args...); err != nil {
		fatalf("DELETE media_likes (foreign): %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM media_match_associations_history
		 WHERE media_file_id IN (SELECT id FROM media_files WHERE `+where+`)`, args...); err != nil {
		fatalf("DELETE associations_history (foreign): %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM media_files WHERE `+where, args...); err != nil {
		fatalf("DELETE media_files (foreign): %v", err)
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		fatalf("CHECKPOINT: %v", err)
	}
	fmt.Printf("Purge OK : %d media_files étrangers supprimés (+%d assoc, +%d likes) + CHECKPOINT.\n",
		nFiles, nAssoc, nLikes)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "cleanup_media_index: "+format+"\n", args...)
	os.Exit(1)
}
