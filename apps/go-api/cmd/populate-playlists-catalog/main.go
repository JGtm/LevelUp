// cmd/populate-playlists-catalog — CLI bootstrap du catalogue Playlists/Pairs/Maps.
//
// Phase G du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Workflow :
//  1. SELECT DISTINCT playlist_id, pair_id, map_id, game_variant_id depuis
//     shared.match_registry → INSERT OR IGNORE dans catalog_fetch_queue.
//  2. Lance CatalogFetcherService.Drain() pour vider la queue via DiscoveryUGC.
//  3. Affiche les compteurs finaux (X playlists, Y pairs, Z maps, W variants, E erreurs).
//
// Usage :
//
//	./populate-playlists-catalog --title halo_infinite \
//	    --metadata-db data/warehouse/metadata.duckdb \
//	    --shared-db data/warehouse/shared_matches_v2.duckdb \
//	    --experience-rules config/titles/halo_infinite/catalog/experience_rules.toml
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/service"
)

func main() {
	titleSlug := flag.String("title", "halo_infinite", "title slug à bootstrapper")
	metadataDBPath := flag.String("metadata-db", "data/warehouse/metadata.duckdb", "chemin metadata.duckdb")
	sharedDBPath := flag.String("shared-db", "data/warehouse/shared_matches_v2.duckdb", "chemin shared_matches_v2.duckdb")
	rulesPath := flag.String("experience-rules", "config/titles/halo_infinite/catalog/experience_rules.toml", "chemin experience_rules.toml")
	maxRetries := flag.Int("max-retries", 5, "nombre max de tentatives par asset avant skip")
	dryRun := flag.Bool("dry-run", false, "ne fait que le seed de la queue, pas de fetch DiscoveryUGC")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()

	metadataDB, err := sql.Open("duckdb", *metadataDBPath)
	if err != nil {
		fatal("open metadata DB: %v", err)
	}
	defer metadataDB.Close()

	sharedDB, err := sql.Open("duckdb", *sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		fatal("open shared DB: %v", err)
	}
	defer sharedDB.Close()

	// Migrations metadata pour s'assurer que les tables catalogue existent.
	if err := migration.RunForDB(metadataDB, migration.TargetMetadata); err != nil {
		fatal("migrate metadata: %v", err)
	}

	// 1. Seed la queue depuis match_registry.
	seeded, err := seedQueueFromMatchRegistry(ctx, metadataDB, sharedDB, *titleSlug)
	if err != nil {
		fatal("seed queue: %v", err)
	}
	slog.InfoContext(ctx, "queue seeded from match_registry",
		"playlists", seeded.Playlists,
		"pairs", seeded.Pairs,
		"maps", seeded.Maps,
		"game_variants", seeded.GameVariants)

	if *dryRun {
		slog.InfoContext(ctx, "dry-run: skip drain")
		return
	}

	// 2. Construire le resolver + adapter Halo.
	provider := halo.NewHaloProvider() // pas de tokens nécessaires pour DiscoveryUGC
	fetcher := halo.NewCatalogFetcher(provider)
	adapter, err := halo_infinite.NewCatalogAdapter(fetcher, *rulesPath)
	if err != nil {
		fatal("init catalog adapter: %v", err)
	}
	resolver := games.NewStaticResolver(*titleSlug)
	resolver.RegisterCatalog(adapter)

	// 3. Lance le drain (loop jusqu'à queue vide ou tous skipped).
	svc := service.NewCatalogFetcherService(metadataDB, resolver, *maxRetries)
	t0 := time.Now()
	totalPlaylists, totalPairs, totalMaps, totalGV, totalErrors := 0, 0, 0, 0, 0
	for pass := 1; ; pass++ {
		res, err := svc.Drain(ctx, *titleSlug)
		if err != nil {
			fatal("drain pass %d: %v", pass, err)
		}
		if res.Playlists+res.Pairs+res.Maps+res.GameVariants+res.Errors == 0 {
			break // queue vide ou tous en max-retries
		}
		totalPlaylists += res.Playlists
		totalPairs += res.Pairs
		totalMaps += res.Maps
		totalGV += res.GameVariants
		totalErrors += res.Errors
		slog.InfoContext(ctx, "drain pass complete",
			"pass", pass,
			"playlists", res.Playlists, "pairs", res.Pairs,
			"maps", res.Maps, "game_variants", res.GameVariants,
			"errors", res.Errors)
		if pass > 10 {
			slog.WarnContext(ctx, "drain: too many passes, stopping")
			break
		}
	}

	slog.InfoContext(ctx, "bootstrap complete",
		"duration", time.Since(t0).String(),
		"playlists", totalPlaylists, "pairs", totalPairs,
		"maps", totalMaps, "game_variants", totalGV,
		"errors", totalErrors)
}

type seedCounters struct {
	Playlists, Pairs, Maps, GameVariants int
}

// seedQueueFromMatchRegistry insère dans catalog_fetch_queue tous les asset IDs
// distincts vus dans shared.match_registry, sans appel HTTP.
func seedQueueFromMatchRegistry(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (seedCounters, error) {
	var counters seedCounters
	type seedSpec struct {
		assetType  string
		col        string
		versionCol string
		counter    *int
	}
	specs := []seedSpec{
		{"playlist", "playlist_id", "playlist_version_id", &counters.Playlists},
		{"pair", "pair_id", "pair_version_id", &counters.Pairs},
		{"map", "map_id", "map_version_id", &counters.Maps},
		{"game_variant", "game_variant_id", "game_variant_version_id", &counters.GameVariants},
	}
	for _, s := range specs {
		query := fmt.Sprintf(
			`SELECT DISTINCT %s, COALESCE(%s, '') FROM match_registry WHERE %s IS NOT NULL AND %s != ''`,
			s.col, s.versionCol, s.col, s.col,
		)
		rows, err := sharedDB.QueryContext(ctx, query)
		if err != nil {
			return counters, fmt.Errorf("select %s: %w", s.assetType, err)
		}
		var inserted int
		for rows.Next() {
			var id, ver string
			if err := rows.Scan(&id, &ver); err != nil {
				rows.Close()
				return counters, err
			}
			res, err := metadataDB.ExecContext(ctx,
				`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id) VALUES (?, ?, ?, ?)`,
				titleSlug, s.assetType, id, ver)
			if err != nil {
				slog.WarnContext(ctx, "seed: insert queue failed", "err", err, "asset_type", s.assetType, "asset_id", id)
				continue
			}
			n, _ := res.RowsAffected()
			if n > 0 {
				inserted++
			}
		}
		rows.Close()
		*s.counter = inserted
	}
	return counters, nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
