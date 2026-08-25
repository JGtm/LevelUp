// cmd_replay_events.go — Sous-commande `levelup replay-events`.
//
// Re-parse les highlight events des matchs cassés silencieusement (cf.
// PLAN_HIGHLIGHT_EVENTS_BACKFILL.md, mai 2026) :
//   - events_loaded=TRUE mais 0 row dans highlight_events (parser cassé)
//   - kills présents mais 0 row dans killer_victim_pairs (InsertKVP buggé)
//
// Usage :
//
//	levelup replay-events --gamertag JGtm [--limit 200] [--dry-run] [--rps 1]
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

func runReplayEvents(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("replay-events", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	limit := fs.Int("limit", 200, "Nombre maximum de matchs à rejouer (les plus récents en premier)")
	dryRun := fs.Bool("dry-run", false, "Lister les matchs cassés sans les rejouer")
	rps := fs.Int("rps", 1, "Requêtes API par seconde (rate limiting)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*gamertag) == "" {
		return fmt.Errorf("--gamertag obligatoire")
	}

	ctx := context.Background()

	// 1. Résoudre le joueur.
	player, err := loadPlayerSummary(cfg, *gamertag)
	if err != nil {
		return err
	}
	// 2. Ouvrir la shared DB en RW (échoue si serveur tient le lock).
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := resolver.SharedDBPath(titlePkg.DefaultSlug)
	sharedHandle, err := duckdbpkg.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (server LevelUp tourne ?): %w", err)
	}
	defer sharedHandle.Close()
	sharedDB := sharedHandle.SQLDb()

	// 3. Détecter les matchs cassés.
	broken, err := go_sync.FindBrokenHighlightEventMatches(ctx, sharedDB, *limit)
	if err != nil {
		return fmt.Errorf("FindBrokenHighlightEventMatches: %w", err)
	}
	fmt.Printf("Matchs cassés détectés : %d (limit=%d)\n", len(broken), *limit)
	if len(broken) == 0 {
		fmt.Println("Rien à rejouer.")
		return nil
	}

	if *dryRun {
		fmt.Println("--- dry-run, liste seulement ---")
		for i, id := range broken {
			if i < 20 {
				fmt.Printf("  %s\n", id)
			}
		}
		if len(broken) > 20 {
			fmt.Printf("  ... (+ %d autres)\n", len(broken)-20)
		}
		return nil
	}

	// 4. Tokens Halo depuis le MultiUserTokenStore (source unique ADR 0023).
	tokens, err := haloTokensForPlayer(ctx, cfg.RepoRoot, player.Gamertag)
	if err != nil {
		return err
	}

	// 5. Client + replay.
	client := go_sync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, *rps)
	beforeAnomaly := observability.LoadCounter("highlight_events_parse_anomaly_total")

	progress := func(done, total int, matchID, status string) {
		fmt.Printf("[%d/%d] %s %s\n", done, total, matchID, status)
	}

	res, err := go_sync.ReplayHighlightEventsForMatches(ctx, client, sharedDB, broken, progress)
	if err != nil {
		fmt.Printf("replay interrompu: %v\n", err)
	}

	afterAnomaly := observability.LoadCounter("highlight_events_parse_anomaly_total")
	fmt.Println("\n=== Rapport replay highlight events ===")
	fmt.Printf("  total processed   : %d\n", res.Total)
	fmt.Printf("  healed (events>0) : %d  (events insérés : %d)\n", res.Healed, res.EventsInserted)
	fmt.Printf("  no_film (404)     : %d\n", res.NoFilm)
	fmt.Printf("  parse_anomaly     : %d  (compteur expvar : %d → %d, delta=%d)\n",
		res.ParseAnomaly, beforeAnomaly, afterAnomaly, afterAnomaly-beforeAnomaly)
	fmt.Printf("  errors            : %d\n", res.Errors)

	if res.Errors > 0 {
		return fmt.Errorf("%d erreur(s) lors du replay", res.Errors)
	}
	return nil
}

// _ = domain.SyncResult permettra une future éventuelle extension du résumé.
var _ = domain.SyncResult{}
