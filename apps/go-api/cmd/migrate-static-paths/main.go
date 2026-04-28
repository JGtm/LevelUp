// Command migrate-static-paths — script jetable Phase 6.5 du plan finition
// multi-titres (BACKLOG.md §[Multi-titre] Migration static/).
//
// Met à jour les colonnes DB qui stockent des chemins /static/... pour
// refléter la nouvelle arborescence title-scopée :
//
//   - metadata.duckdb.map_images_registry.local_path (E1) :
//     /static/maps/X.png → /static/maps/halo_infinite/X.png
//
//   - metadata.duckdb.citation_mappings.image_path (E2 + G) :
//     static/commendations/h5g/X → static/commendations/halo_5_guardians/X
//     static/commendations/hi/X  → static/commendations/halo_infinite/X
//
// Usage :
//
//	go run ./cmd/migrate-static-paths --dry-run            # afficher ce qui serait fait
//	go run ./cmd/migrate-static-paths                      # appliquer
//
// Le script est idempotent — relancer après une exécution complète ne touche
// plus aucune ligne (les paths déjà au format title-scoped sont skippés via
// LIKE pattern).
//
// À supprimer après run prod (Phase 6.6 cleanup).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

func main() {
	var dryRun bool
	flag.BoolVar(&dryRun, "dry-run", false, "Afficher les UPDATE sans les exécuter")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := run(dryRun); err != nil {
		slog.Error("migrate-static-paths failed", "err", err)
		os.Exit(1)
	}
}

func run(dryRun bool) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("open metadata DB: %w", err)
	}
	defer db.Close()

	updates := []struct {
		name string
		sql  string
	}{
		{
			name: "E1: map_images_registry flat → halo_infinite",
			sql: `
				UPDATE map_images_registry
				SET local_path = REPLACE(local_path, '/static/maps/', '/static/maps/halo_infinite/')
				WHERE title_id = 'halo_infinite'
				  AND local_path LIKE '/static/maps/%'
				  AND local_path NOT LIKE '/static/maps/halo_infinite/%'
				  AND local_path NOT LIKE '/static/maps/synthetic_title_b/%'
			`,
		},
		{
			name: "G: citation_mappings h5g → halo_5_guardians",
			sql: `
				UPDATE citation_mappings
				SET image_path = REPLACE(image_path, 'static/commendations/h5g/', 'static/commendations/halo_5_guardians/')
				WHERE image_path LIKE 'static/commendations/h5g/%'
			`,
		},
		{
			name: "G: citation_mappings hi → halo_infinite",
			sql: `
				UPDATE citation_mappings
				SET image_path = REPLACE(image_path, 'static/commendations/hi/', 'static/commendations/halo_infinite/')
				WHERE image_path LIKE 'static/commendations/hi/%'
			`,
		},
	}

	for _, u := range updates {
		if dryRun {
			slog.Info("[dry-run] would execute", "name", u.name)
			continue
		}
		res, err := db.Exec(ctx, u.sql)
		if err != nil {
			return fmt.Errorf("update %q: %w", u.name, err)
		}
		rows, _ := res.RowsAffected()
		slog.Info("update applied", "name", u.name, "rows_affected", rows)
	}

	if dryRun {
		slog.Info("dry-run complete — no rows touched")
	} else {
		slog.Info("migration complete")
	}

	// Vérification post-UPDATE (sanity check) — relancer ces queries à la main
	// pour vérifier qu'aucun row ne reste au format flat.
	if !dryRun {
		var staleCount int64
		if err := db.QueryRow(ctx,
			`SELECT COUNT(*) FROM map_images_registry
			 WHERE title_id = 'halo_infinite'
			   AND local_path LIKE '/static/maps/%'
			   AND local_path NOT LIKE '/static/maps/halo_infinite/%'`,
		).Scan(&staleCount); err != nil {
			slog.Warn("post-check query failed", "err", err)
		} else if staleCount > 0 {
			slog.Warn("post-check: stale flat paths remain", "count", staleCount)
		} else {
			slog.Info("post-check OK: no stale flat paths in map_images_registry")
		}
	}

	return nil
}

// stub pour s'assurer que sql est bien importé (utilisé indirectement via duckdb.DB).
var _ = sql.ErrNoRows
