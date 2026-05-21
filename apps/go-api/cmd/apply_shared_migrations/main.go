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
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/migration"
	ddb "levelup/go-api/internal/platform/duckdb"
)

func main() {
	sharedPath := flag.String("shared", "", "Path vers shared_matches_v2.duckdb (obligatoire si pas de --shared-only-skip)")
	playersDir := flag.String("players", "", "Optionnel : dossier data/titles/<title>/players/ pour appliquer aussi TargetPlayer sur chaque stats.duckdb")
	flag.Parse()

	if *sharedPath == "" && *playersDir == "" {
		fmt.Fprintln(os.Stderr, "usage: apply_shared_migrations -shared <path> [-players <dir>]")
		os.Exit(2)
	}

	// Force registration init() side-effects.
	_ = migration.All()

	if *sharedPath != "" {
		if err := applyOne(*sharedPath, migration.TargetShared); err != nil {
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
			if err := applyOne(path, migration.TargetPlayer); err != nil {
				fmt.Fprintf(os.Stderr, "FAIL player %s: %v\n", e.Name(), err)
				continue
			}
			fmt.Printf("OK player: %s\n", e.Name())
		}
	}
}

func applyOne(path string, target migration.TargetDB) error {
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw %s: %w", path, err)
	}
	defer db.Close()
	return migration.RunForDB(db.SQLDb(), target)
}
