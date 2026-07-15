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
	"levelup/go-api/internal/migration"
	auth_platform "levelup/go-api/internal/platform/auth"
	duckdbPlatform "levelup/go-api/internal/platform/duckdb"
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
	titleFlag := fs.String("title", titlePkg.DefaultSlug, "Slug du titre à synchroniser (défaut halo_infinite)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *allPlayers && strings.TrimSpace(*gamertag) != "" {
		return fmt.Errorf("--gamertag et --all sont mutuellement exclusifs")
	}
	if !*allPlayers && strings.TrimSpace(*gamertag) == "" {
		return fmt.Errorf("--gamertag est obligatoire sauf avec --all")
	}
	// Slug du titre : défaut halo_infinite → byte-identique au comportement actuel.
	titleSlug := strings.TrimSpace(*titleFlag)
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}

	ctx := context.Background()
	if *allPlayers {
		return runSyncAchievementsAll(ctx, cfg, titleSlug, *dryRun)
	}

	player, err := loadPlayerSummary(cfg, *gamertag)
	if err != nil {
		return err
	}
	return runSyncAchievementsForPlayer(ctx, cfg, player, titleSlug, *dryRun)
}

func runSyncAchievementsForPlayer(ctx context.Context, cfg *config.AppConfig, player *domain.PlayerSummary, titleSlug string, dryRun bool) error {
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	dbPath := resolver.PlayerDBPath(titleSlug, player.Gamertag)
	if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
		return fmt.Errorf("player DB introuvable pour %s (%s)", player.Gamertag, dbPath)
	}

	if dryRun {
		fmt.Printf("sync achievements DRY-RUN: title=%s gamertag=%s xuid=%s db=%s\n",
			titleSlug, player.Gamertag, player.XUID, dbPath)
		return nil
	}

	// Appliquer les migrations metadata avant le sync (title_id colonne, cleanup stale rows).
	metadataPath := resolver.MetadataDBPath(titleSlug)
	if err := applyAchievementsMigrations(metadataPath, titleSlug); err != nil {
		return fmt.Errorf("migrations metadata: %w", err)
	}

	provider := auth_platform.NewSISUProvider()
	engine := go_sync.NewSyncEngineForTitle(cfg.RepoRoot, titleSlug, player.Gamertag, player.XUID, nil, provider)

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

// applyAchievementsMigrations applique les migrations metadata en attente pour le
// TITRE ciblé. Ouvre une connexion temporaire (fermée immédiatement après) pour ne
// pas interférer avec la connexion interne du SyncEngine.
//
// RunForTitleDB (pas RunForDB) : le set metadata d'un titre non-HINF (ex. halo_5)
// possède sa propre forme de xbox_achievement_definitions (ADR 0008 — provisioning
// par titre, jamais le set HINF figé). Pour halo_infinite, RunForTitleDB retombe
// sur le set HINF → byte-identique au comportement précédent.
func applyAchievementsMigrations(metadataPath, titleSlug string) error {
	db, err := duckdbPlatform.OpenReadWrite(metadataPath)
	if err != nil {
		return fmt.Errorf("ouverture metadata DB: %w", err)
	}
	defer db.Close() //nolint:errcheck
	_ = migration.All()
	return migration.RunForTitleDB(db.SQLDb(), titleSlug, migration.TargetMetadata)
}

func runSyncAchievementsAll(ctx context.Context, cfg *config.AppConfig, titleSlug string, dryRun bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configuré")
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	// Migrations metadata une seule fois pour tout le batch.
	if !dryRun {
		metadataPath := resolver.MetadataDBPath(titleSlug)
		if err := applyAchievementsMigrations(metadataPath, titleSlug); err != nil {
			return fmt.Errorf("migrations metadata: %w", err)
		}
		fmt.Println("migrations metadata: OK")
	}

	if dryRun {
		for _, p := range players {
			dbPath := resolver.PlayerDBPath(titleSlug, p.Gamertag)
			status := "ok"
			if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
				status = "no_player_db"
			}
			fmt.Printf("sync achievements DRY-RUN: title=%s gamertag=%s xuid=%s status=%s\n",
				titleSlug, p.Gamertag, p.XUID, status)
		}
		fmt.Printf("sync achievements DRY-RUN total=%d\n", len(players))
		return nil
	}

	provider := auth_platform.NewSISUProvider()
	total := len(players)
	synced := 0
	skipped := 0
	failed := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titleSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("sync achievements SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}

		engine := go_sync.NewSyncEngineForTitle(cfg.RepoRoot, titleSlug, player.Gamertag, player.XUID, nil, provider)
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
