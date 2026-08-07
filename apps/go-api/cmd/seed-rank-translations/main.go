//go:build cgo

// cmd/seed-rank-translations — Peuple career_rank_translations offline à partir
// des libellés title-owned (halomigrations.CareerRankTranslations, posés via le
// seam migration.SetCareerRankTranslationsProvider — source unique partagée avec
// `levelup seed rank-translations`).
//
// Préférer `levelup seed rank-translations` (CLI in-image). Ce binaire reste un
// fallback direct quand refresh-career-ranks n'est pas jouable (tokens invalides,
// API GameCMS down) et que la table est vide.
//
// Usage :
//
//	go run -tags cgo ./cmd/seed-rank-translations
//	go run -tags cgo ./cmd/seed-rank-translations --dry-run
//
// IMPORTANT : stopper le serveur API avant de lancer (metadata.duckdb est
// ouvert en RW au boot — le pool détient un verrou exclusif).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

func main() {
	// Pose la source title-owned des libellés de rangs (MT-07).
	migration.SetCareerRankTranslationsProvider(halomigrations.CareerRankTranslations)

	fs := flag.NewFlagSet("seed-rank-translations", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Affiche les rangs sans écrire en base")
	titleID := fs.String("title-id", titlePkg.DefaultSlug, "Title ID (ex: halo_infinite)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if err := run(*titleID, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "erreur : %v\n", err)
		os.Exit(1)
	}
}

func run(titleID string, dryRun bool) error {
	rows := migration.CareerRankTranslationRows()

	if dryRun {
		for _, r := range rows {
			fmt.Printf("rank %3d | %s | %-30s [%s]\n", r.RankID, r.Lang, r.Title, r.Tier)
		}
		fmt.Printf("\n%d lignes (dry-run — rien écrit)\n", len(rows))
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}
	if cfg.RepoRoot == "" {
		return fmt.Errorf("LEVELUP_REPO_ROOT non défini")
	}

	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titleID)
	if _, err := os.Stat(metaPath); err != nil {
		return fmt.Errorf("metadata.duckdb introuvable (%s): %w", metaPath, err)
	}

	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("ouverture metadata.duckdb: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	if _, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS career_rank_translations (
			rank_id    INTEGER NOT NULL,
			lang       VARCHAR NOT NULL,
			title      VARCHAR,
			subtitle   VARCHAR,
			tier       VARCHAR,
			fetched_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (rank_id, lang)
		)
	`); err != nil {
		return fmt.Errorf("ensure table: %w", err)
	}

	var before int
	_ = db.QueryRow(ctx, "SELECT COUNT(*) FROM career_rank_translations").Scan(&before)
	fmt.Printf("Avant : %d lignes dans career_rank_translations\n", before)

	for _, r := range rows {
		if _, err := db.Exec(ctx, `
			INSERT OR REPLACE INTO career_rank_translations
				(rank_id, lang, title, subtitle, tier, fetched_at)
			VALUES (?, ?, ?, '', ?, CURRENT_TIMESTAMP)
		`, r.RankID, r.Lang, r.Title, r.Tier); err != nil {
			return fmt.Errorf("upsert rank %d lang %s: %w", r.RankID, r.Lang, err)
		}
	}

	var after int
	_ = db.QueryRow(ctx, "SELECT COUNT(*) FROM career_rank_translations").Scan(&after)
	fmt.Printf("Apres  : %d lignes — %d upsertées\n", after, len(rows))
	return nil
}
