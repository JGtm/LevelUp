// regen-thumbnails : supprime tous les .gif/.webp existants dans
// {capturesBase}/{slug}/thumbs/ puis re-genere les .webp animes via libwebp
// pour toutes les videos. Met a jour thumbnail_path en DB.
//
// Pourquoi : apres le passage en .webp anime (commit 4c9177e9) puis le
// refactor des paths relatifs, les vieilles miniatures sont un melange de
// .gif legacy + .webp generes + paths DB pointant vers d'anciens emplacements.
// Ce cmd reset l'etat des miniatures pour partir sur du propre.
//
// Usage :
//
//	regen-thumbnails --db shared_social.duckdb [--captures-base C:\Captures] [--slug JGtm] [--dry-run]
//
// --slug optionnel : ne traite qu'un joueur. Sinon tous les sous-dossiers
//
//	de capturesBase sont scannes.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ops"
)

type appSettings struct {
	MediaCapturesBaseDir string `json:"media_captures_base_dir"`
}

func loadCapturesBase(settingsPath string) string {
	if settingsPath == "" {
		return ""
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var s appSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}
	return s.MediaCapturesBaseDir
}

// listOwnerSlugs retourne la liste des sous-dossiers directs de capturesBase
// (chaque sous-dossier = un owner_slug).
func listOwnerSlugs(capturesBase string) ([]string, error) {
	entries, err := os.ReadDir(capturesBase)
	if err != nil {
		return nil, err
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			slugs = append(slugs, e.Name())
		}
	}
	return slugs, nil
}

// deleteExistingThumbs supprime tous les .gif et .webp dans thumbsDir.
// Retourne (deletedCount, error).
func deleteExistingThumbs(thumbsDir string, dryRun bool) (int, error) {
	entries, err := os.ReadDir(thumbsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	deleted := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".gif" && ext != ".webp" {
			continue
		}
		if dryRun {
			deleted++
			continue
		}
		if err := os.Remove(filepath.Join(thumbsDir, e.Name())); err != nil {
			return deleted, fmt.Errorf("remove %s: %w", e.Name(), err)
		}
		deleted++
	}
	return deleted, nil
}

// nullifyThumbnailPaths met thumbnail_path à NULL pour tous les médias d'un
// owner — préalable au re-backfill propre. Retourne le nombre de lignes mises à jour.
func nullifyThumbnailPaths(db *sql.DB, ownerSlug string) (int, error) {
	res, err := db.Exec(`UPDATE media_files SET thumbnail_path = NULL WHERE player_slug = ?`, ownerSlug)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func main() {
	dbPath := flag.String("db", "", "path vers shared_social.duckdb (requis)")
	capturesBase := flag.String("captures-base", "", "MediaCapturesBaseDir (sinon lu depuis --settings)")
	settingsPath := flag.String("settings", "app_settings.json", "path vers app_settings.json (fallback)")
	onlySlug := flag.String("slug", "", "ne traiter qu'un seul joueur (optionnel)")
	dryRun := flag.Bool("dry-run", false, "afficher les actions sans les exécuter")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "--db requis")
		os.Exit(2)
	}
	base := *capturesBase
	if base == "" {
		base = loadCapturesBase(*settingsPath)
	}
	if base == "" {
		fmt.Fprintln(os.Stderr, "captures base introuvable")
		os.Exit(2)
	}

	fmt.Printf("DB:            %s\n", *dbPath)
	fmt.Printf("CapturesBase:  %s\n", base)
	if *onlySlug != "" {
		fmt.Printf("Slug filter:   %s\n", *onlySlug)
	}
	fmt.Printf("DryRun:        %v\n\n", *dryRun)

	var slugs []string
	if *onlySlug != "" {
		slugs = []string{*onlySlug}
	} else {
		var err error
		slugs, err = listOwnerSlugs(base)
		if err != nil {
			fmt.Fprintln(os.Stderr, "list owner slugs:", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Players a traiter: %d\n\n", len(slugs))

	// Ouvrir la DB (RW si pas dry-run, RO sinon).
	openPath := *dbPath
	if *dryRun {
		openPath = *dbPath + "?access_mode=READ_ONLY"
	}
	db, err := sql.Open("duckdb", openPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer db.Close() //nolint:errcheck

	store := ops.MediaPathStore{CapturesBase: base}
	totalDeleted := 0
	totalGenerated := 0
	totalLinked := 0

	for _, slug := range slugs {
		playerDir := filepath.Join(base, slug)
		thumbsDir := filepath.Join(playerDir, "thumbs")

		// 1. Supprimer les anciennes miniatures
		deleted, err := deleteExistingThumbs(thumbsDir, *dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] delete thumbs: %v\n", slug, err)
			continue
		}
		totalDeleted += deleted

		// 2. NULL out les thumbnail_path en DB pour ce joueur
		var nulled int
		if !*dryRun {
			nulled, err = nullifyThumbnailPaths(db, slug)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[%s] nullify thumbs: %v\n", slug, err)
				continue
			}
		}

		// 3. Regen les .webp
		generated := 0
		var genErrs []string
		if !*dryRun {
			generated, genErrs = ops.GenerateThumbnails(playerDir, thumbsDir)
		}
		totalGenerated += generated

		// 4. Relinker en DB
		linked := 0
		if !*dryRun {
			n, backfillErr := ops.BackfillThumbnailPaths(context.Background(), db, playerDir, thumbsDir, slug, store)
			if backfillErr != nil {
				fmt.Fprintf(os.Stderr, "[%s] backfill: %v\n", slug, backfillErr)
				continue
			}
			linked = n
			totalLinked += linked
		}

		fmt.Printf("[%s] deleted=%d nulled=%d generated=%d linked=%d", slug, deleted, nulled, generated, linked)
		if len(genErrs) > 0 {
			fmt.Printf(" gen_errs=%d", len(genErrs))
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Total:")
	fmt.Printf("  thumbs supprimes : %d\n", totalDeleted)
	fmt.Printf("  thumbs generes   : %d\n", totalGenerated)
	fmt.Printf("  thumbs relinkes  : %d\n", totalLinked)
	if *dryRun {
		fmt.Println("\n(dry-run : aucune ecriture)")
	}
}
