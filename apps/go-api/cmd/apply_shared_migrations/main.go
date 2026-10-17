// cmd/apply_shared_migrations — one-shot tool qui applique toutes les
// migrations TargetShared sur shared_matches_v2.duckdb (et toutes les player
// stats.duckdb si --players).
//
// Utilisé pour appliquer manuellement la migration
// rebuild_match_participants_defeat_art_corruption (Phase 1 plan
// stabilisation 2026-05-22) sans avoir à booter le serveur.
//
// Usage :
//
//	go run ./cmd/apply_shared_migrations -shared data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb
//	go run ./cmd/apply_shared_migrations -shared <path> -players data/titles/halo_infinite/players
//	go run ./cmd/apply_shared_migrations -title halo_5 -shared <path>
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	halo5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	ddb "levelup/go-api/internal/platform/duckdb"
)

func main() {
	// Phase 1.5.1 B : steps Halo title-owned fournis au runner via le provider.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	// Le jeu de migrations d'un titre non défaut doit être enregistré AVANT RunForTitleDB.
	halo5migrations.Register()
	sharedPath := flag.String("shared", "", "Path vers shared_matches_v2.duckdb (obligatoire si pas de --shared-only-skip)")
	playersDir := flag.String("players", "", "Optionnel : dossier data/titles/<title>/players/ pour appliquer aussi TargetPlayer sur chaque stats.duckdb")
	title := flag.String("title", migration.DefaultSlug, "slug du titre dont les migrations sont jouées (défaut : titre par défaut)")
	flag.Parse()

	if *sharedPath == "" && *playersDir == "" {
		fmt.Fprintln(os.Stderr, "usage: apply_shared_migrations -shared <path> [-players <dir>] [-title <slug>]")
		os.Exit(2)
	}

	// Force registration init() side-effects.
	_ = migration.All()

	if *sharedPath != "" {
		if err := applyOne(*sharedPath, *title, migration.TargetShared); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL shared: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("OK shared: %s\n", *sharedPath)
	}

	if *playersDir != "" {
		entries, err := os.ReadDir(*playersDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL read players dir: %v\n", err)
			os.Exit(1)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			path := filepath.Join(*playersDir, e.Name(), "stats.duckdb")
			if _, err := os.Stat(path); err != nil {
				fmt.Printf("SKIP %s (no stats.duckdb)\n", e.Name())
				continue
			}
			if err := applyOne(path, *title, migration.TargetPlayer); err != nil {
				fmt.Fprintf(os.Stderr, "FAIL player %s: %v\n", e.Name(), err)
				continue
			}
			fmt.Printf("OK player: %s\n", e.Name())
		}
	}
}

func applyOne(path, title string, target migration.TargetDB) error {
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw %s: %w", path, err)
	}
	defer db.Close()
	return migration.RunForTitleDB(db.SQLDb(), title, target)
}
