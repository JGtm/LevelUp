// cmd_backfill.go : sous-commande CLI pour les backfills locaux.
//
// Le backfill est principalement expose via HTTP (POST /backfill/start).
// Cette sous-commande offre une voie locale pour les backfills purement Go
// (pas d'appel API Halo requis), utile pour bootstrap ou re-run en masse.
//
// Backfills supportes :
//   - --engagement-scores [--force]
//   - --citations         [--force]
//
// Usage :
//
//	levelup backfill --gamertag X --engagement-scores [--force]
//	levelup backfill --all          --engagement-scores [--force]
//	levelup backfill --gamertag X --citations         [--force]
//	levelup backfill --all          --citations         [--force]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

func runBackfill(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (mutuellement exclusif avec --all)")
	allPlayers := fs.Bool("all", false, "Applique le backfill a tous les joueurs configures")
	engagementScores := fs.Bool("engagement-scores", false, "Backfill du score d'engagement (Phase 6 plan engagement)")
	citations := fs.Bool("citations", false, "Backfill des citations (match_citations) depuis citation_mappings + medals + stats + awards")
	force := fs.Bool("force", false, "Force le recalcul meme si deja persiste")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *allPlayers && strings.TrimSpace(*gamertag) != "" {
		return fmt.Errorf("--gamertag et --all sont mutuellement exclusifs")
	}
	if !*allPlayers && strings.TrimSpace(*gamertag) == "" {
		return fmt.Errorf("--gamertag est obligatoire sauf avec --all")
	}
	if !*engagementScores && !*citations {
		return fmt.Errorf("aucun backfill selectionne (utiliser --engagement-scores ou --citations)")
	}

	ctx := context.Background()
	if *engagementScores {
		if *allPlayers {
			if err := runBackfillAllEngagement(ctx, cfg, *force); err != nil {
				return err
			}
		} else {
			player, err := loadPlayerSummary(cfg, *gamertag)
			if err != nil {
				return err
			}
			if err := runBackfillEngagementForPlayer(ctx, cfg, player.Gamertag, player.XUID, *force); err != nil {
				return err
			}
		}
	}
	if *citations {
		if *allPlayers {
			return runBackfillAllCitations(ctx, cfg, *force)
		}
		player, err := loadPlayerSummary(cfg, *gamertag)
		if err != nil {
			return err
		}
		return runBackfillCitationsForPlayer(ctx, cfg, player.Gamertag, player.XUID, *force)
	}
	return nil
}

func runBackfillAllEngagement(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	total := len(players)
	processed := 0
	skipped := 0
	failed := 0
	totalUpdated := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill engagement SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}

		updated, runErr := runBackfillEngagementOne(ctx, cfg, player.Gamertag, player.XUID, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill engagement FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill engagement OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	}

	fmt.Printf("backfill engagement batch: total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		total, processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill engagement: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillEngagementForPlayer(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) error {
	updated, err := runBackfillEngagementOne(ctx, cfg, gamertag, xuid, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill engagement OK: gamertag=%s updated=%d force=%t\n", gamertag, updated, force)
	return nil
}

// runBackfillEngagementOne instancie un SyncEngine sans tokens (calcul local
// pur) et appelle RunBackfillEngagementScores. Aucune requete API requise.
//
// Applique les migrations Phase 2 engagement (colonnes player + match_intensity
// shared) avant le backfill, car sync.OpenPlayerDB/OpenSharedDB ne lance pas
// migration.RunForDB (contrairement au pool DuckDB / boot serveur).
func runBackfillEngagementOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) (int, error) {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)

	if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
		return 0, fmt.Errorf("migrations player %s: %w", gamertag, err)
	}
	if err := applyMigrationsOnDB(sharedDBPath, migration.TargetShared); err != nil {
		return 0, fmt.Errorf("migrations shared: %w", err)
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)
	return engine.RunBackfillEngagementScores(ctx, force)
}

// runBackfillAllCitations applique le backfill citations sur tous les joueurs
// configures dans db_profiles.json.
func runBackfillAllCitations(ctx context.Context, cfg *config.AppConfig, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	processed := 0
	skipped := 0
	failed := 0
	totalUpdated := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("backfill citations SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}

		updated, runErr := runBackfillCitationsOne(ctx, cfg, player.Gamertag, player.XUID, force)
		if runErr != nil {
			failed++
			fmt.Printf("backfill citations FAIL: gamertag=%s err=%v\n", player.Gamertag, runErr)
			continue
		}
		processed++
		totalUpdated += updated
		fmt.Printf("backfill citations OK: gamertag=%s updated=%d\n", player.Gamertag, updated)
	}

	fmt.Printf("backfill citations batch: total=%d processed=%d skipped=%d failed=%d total_updated=%d\n",
		len(players), processed, skipped, failed, totalUpdated)
	if failed > 0 {
		return fmt.Errorf("backfill citations: %d joueur(s) en echec", failed)
	}
	return nil
}

func runBackfillCitationsForPlayer(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) error {
	updated, err := runBackfillCitationsOne(ctx, cfg, gamertag, xuid, force)
	if err != nil {
		return err
	}
	fmt.Printf("backfill citations OK: gamertag=%s updated=%d force=%t\n", gamertag, updated, force)
	return nil
}

// runBackfillCitationsOne applique les migrations puis appelle
// SyncEngine.RunBackfillCitations. Aucun appel API requis.
func runBackfillCitationsOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, force bool) (int, error) {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	playerDBPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, gamertag)
	sharedDBPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
	metaDBPath := resolver.MetadataDBPath(titlePkg.DefaultSlug)

	if err := applyMigrationsOnDB(playerDBPath, migration.TargetPlayer); err != nil {
		return 0, fmt.Errorf("migrations player %s: %w", gamertag, err)
	}
	if err := applyMigrationsOnDB(sharedDBPath, migration.TargetShared); err != nil {
		return 0, fmt.Errorf("migrations shared: %w", err)
	}
	if err := applyMigrationsOnDB(metaDBPath, migration.TargetMetadata); err != nil {
		return 0, fmt.Errorf("migrations metadata: %w", err)
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)
	return engine.RunBackfillCitations(ctx, force)
}

// applyMigrationsOnDB ouvre une DB en RW et applique les migrations enregistrees
// pour la cible. Idempotent — DuckDB tolere une migration deja appliquee via
// schema_migrations.
func applyMigrationsOnDB(path string, target migration.TargetDB) error {
	_ = migration.All()
	db, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw %s: %w", path, err)
	}
	defer db.Close()
	return migration.RunForDB(db.SQLDb(), target)
}
