// cmd_engagement_coefs.go : recompute du coef lobby global + des bins de reponse
// (modele lobby-anchored v2) sans rejouer le compute des scores.
//
// Pourquoi : `backfill --engagement-scores` rejoue tout (compute scores +
// recompute coefs) ET applique les migrations en amont, ce qui peut échouer
// sur des DBs anciennes (ex. drop_assists_expected_halo_infinite). Cette
// commande contourne le problème en n'exécutant QUE le recompute des coefs
// depuis les paces déjà persistées (~5ms par joueur, aucune migration).
//
// Pré-requis : la table `engagement_coefficients` et les colonnes
// `engagement_pace_*` doivent déjà exister sur la player DB. Sinon le
// recompute est skippé silencieusement (cf. batchRecomputeCoefficients).
//
// Usage :
//
//	levelup engagement-coefs --gamertag X
//	levelup engagement-coefs --all
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	go_sync "levelup/go-api/internal/sync"
)

func runEngagementCoefs(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("engagement-coefs", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (mutuellement exclusif avec --all)")
	allPlayers := fs.Bool("all", false, "Applique le recompute a tous les joueurs configures")
	withScores := fs.Bool("with-scores", false, "Calcule d'abord les engagement_scores depuis highlight_events (heavier, ~secondes par joueur) puis recompute les coefs. Sinon ne touche que les coefs depuis les paces existantes.")
	force := fs.Bool("force", false, "Force le recalcul des engagement_scores meme si deja persistes (no-op sans --with-scores)")
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
		return runEngagementCoefsAll(ctx, cfg, *withScores, *force)
	}
	player, err := loadPlayerSummary(cfg, *gamertag)
	if err != nil {
		return err
	}
	return runEngagementCoefsOne(ctx, cfg, player.Gamertag, player.XUID, *withScores, *force)
}

func runEngagementCoefsAll(ctx context.Context, cfg *config.AppConfig, withScores, force bool) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configure")
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	processed, skipped, failed := 0, 0, 0
	for _, p := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, p.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("engagement-coefs SKIP: gamertag=%s reason=no_player_db\n", p.Gamertag)
			continue
		}
		if err := runEngagementCoefsOne(ctx, cfg, p.Gamertag, p.XUID, withScores, force); err != nil {
			failed++
			fmt.Printf("engagement-coefs FAIL: gamertag=%s err=%v\n", p.Gamertag, err)
			continue
		}
		processed++
	}
	fmt.Printf("engagement-coefs batch: total=%d processed=%d skipped=%d failed=%d\n",
		len(players), processed, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("engagement-coefs: %d joueur(s) en echec", failed)
	}
	return nil
}

func runEngagementCoefsOne(ctx context.Context, cfg *config.AppConfig, gamertag, xuid string, withScores, force bool) error {
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, gamertag, xuid, nil, nil)
	if withScores {
		// Calcul complet : recalcule les engagement_scores depuis highlight_events
		// (peuple les paces) puis recompute les coefs. RunBackfillEngagementScores
		// appelle RunBackfillEngagementCoefficients en queue.
		n, err := engine.RunBackfillEngagementScores(ctx, force)
		if err != nil {
			return err
		}
		fmt.Printf("engagement-coefs OK (with-scores): gamertag=%s scores_updated=%d\n", gamertag, n)
		return nil
	}
	n, err := engine.RunBackfillEngagementCoefficients(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("engagement-coefs OK: gamertag=%s modes_updated=%d\n", gamertag, n)
	return nil
}
