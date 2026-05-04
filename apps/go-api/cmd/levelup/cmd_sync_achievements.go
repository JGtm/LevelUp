package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	go_sync "levelup/go-api/internal/sync"
)

// runSyncAchievements lance le backfill des achievements Xbox pour un joueur ou
// pour tous les joueurs configurés. Opération admin one-shot : à exécuter quand
// aucun sync n'est actif (lease acquis pour éviter collisions ; échec si déjà tenu).
func runSyncAchievements(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("sync-achievements", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire sauf avec --all)")
	allPlayers := fs.Bool("all", false, "Synchronise les achievements de tous les joueurs configurés")
	dryRun := fs.Bool("dry-run", false, "Liste les joueurs ciblés sans appeler l'API ni écrire")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *allPlayers && strings.TrimSpace(*gamertag) != "" {
		return fmt.Errorf("--gamertag et --all sont mutuellement exclusifs")
	}
	if !*allPlayers && strings.TrimSpace(*gamertag) == "" {
		return fmt.Errorf("--gamertag est obligatoire sauf avec --all")
	}

	ctx := context.Background()
	if *allPlayers {
		return runSyncAchievementsAll(ctx, cfg, *dryRun)
	}

	player, err := loadPlayerSummary(cfg, *gamertag)
	if err != nil {
		return err
	}
	return runSyncAchievementsForPlayer(ctx, cfg, player, *dryRun)
}

func runSyncAchievementsForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary, dryRun bool) error {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		return fmt.Errorf("player DB introuvable pour %s (%s)", player.Gamertag, dbPath)
	}

	if dryRun {
		fmt.Printf("sync achievements DRY-RUN: gamertag=%s xuid=%s db=%s\n",
			player.Gamertag, player.XUID, dbPath)
		return nil
	}

	provider := auth_platform.NewMSALProvider()
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, provider)

	start := time.Now()
	ok := engine.RunAchievementsOnly(ctx)
	duration := time.Since(start).Round(time.Millisecond)

	if !ok {
		fmt.Printf("sync achievements FAIL: gamertag=%s duration=%s\n", player.Gamertag, duration)
		return fmt.Errorf("sync achievements échouée pour %s (voir logs)", player.Gamertag)
	}

	fmt.Printf("sync achievements OK: gamertag=%s duration=%s\n", player.Gamertag, duration)
	return nil
}

func runSyncAchievementsAll(ctx context.Context, cfg *config.AppConfig, dryRun bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configuré")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	if dryRun {
		for _, p := range players {
			dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, p.Gamertag)
			status := "ok"
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				status = "no_player_db"
			}
			fmt.Printf("sync achievements DRY-RUN: gamertag=%s xuid=%s status=%s\n",
				p.Gamertag, p.XUID, status)
		}
		fmt.Printf("sync achievements DRY-RUN total=%d\n", len(players))
		return nil
	}

	provider := auth_platform.NewMSALProvider()
	total := len(players)
	synced := 0
	skipped := 0
	failed := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("sync achievements SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, nil, provider)
		start := time.Now()
		ok := engine.RunAchievementsOnly(ctx)
		duration := time.Since(start).Round(time.Millisecond)

		if ok {
			synced++
			fmt.Printf("sync achievements OK: gamertag=%s duration=%s\n", player.Gamertag, duration)
		} else {
			failed++
			fmt.Printf("sync achievements FAIL: gamertag=%s duration=%s\n", player.Gamertag, duration)
		}
	}

	fmt.Printf("sync achievements batch: total=%d synced=%d skipped=%d failed=%d\n",
		total, synced, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("sync achievements batch: %d joueur(s) en échec", failed)
	}
	return nil
}
