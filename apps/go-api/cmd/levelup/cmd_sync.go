package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
	go_sync "levelup/go-api/internal/sync"
)

func runSyncDelta(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("sync-delta", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	allPlayers := fs.Bool("all", false, "Synchronise tous les joueurs configurés")
	maxMatches := fs.Int("max-matches", 25, "Nombre max de nouveaux matchs à insérer")
	matchType := fs.String("match-type", "matchmaking", "Type de match: all|matchmaking|custom|local")
	rps := fs.Int("rps", 1, "Nombre max de requêtes API par seconde")
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
		return runSyncDeltaAll(ctx, cfg, *maxMatches, *matchType, *rps)
	}

	player, err := loadPlayerSummary(cfg, *gamertag)
	if err != nil {
		return err
	}
	refreshToken := oauthRefreshTokenForPlayer(player.Gamertag)
	if refreshToken == "" {
		return fmt.Errorf("aucun refresh token OAuth trouvé pour %s (%s)", player.Gamertag, oauthRefreshEnvKey(player.Gamertag))
	}

	provider := auth_platform.NewMSALProvider()
	accessToken, err := provider.TryOAuthRefresh(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("oauth refresh: %w", err)
	}
	if accessToken == "" {
		return fmt.Errorf("oauth refresh n'a pas retourné d'access_token pour %s", player.Gamertag)
	}
	result, err := provider.Exchange(ctx, accessToken)
	if err != nil {
		return fmt.Errorf("exchange Halo: %w", err)
	}

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, result.Tokens)
	opts := domain.DefaultSyncOptions()
	opts.MatchType = *matchType
	opts.MaxMatches = *maxMatches
	opts.RequestsPerSecond = *rps

	syncResult, err := engine.RunDelta(ctx, opts)
	if err != nil {
		return fmt.Errorf("run delta: %w", err)
	}

	postSync := false
	careerSynced := false
	if syncResult.PostSync != nil {
		postSync = true
		careerSynced = syncResult.PostSync.CareerSynced
	}
	fmt.Printf(
		"sync delta OK: gamertag=%s inserted=%d skipped=%d status=%s post_sync=%t career_synced=%t duration=%.2fs\n",
		player.Gamertag,
		syncResult.MatchesInserted,
		syncResult.MatchesSkipped,
		syncResult.Status(),
		postSync,
		careerSynced,
		syncResult.DurationSeconds,
	)
	if len(syncResult.Warnings) > 0 {
		fmt.Printf("warnings=%d first=%s\n", len(syncResult.Warnings), syncResult.Warnings[0])
	}
	return nil
}

func runSyncDeltaAll(
	ctx context.Context,
	cfg *config.AppConfig,
	maxMatches int,
	matchType string,
	rps int,
) error {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	if len(players) == 0 {
		return fmt.Errorf("aucun joueur configuré")
	}

	provider := auth_platform.NewMSALProvider()
	settingsStore := settings_platform.NewStore(cfg.AppSettingsPath)
	autoScheduler := scheduler.New(cfg, settingsStore, provider)
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	opts := buildSyncOptions(maxMatches, matchType, rps)

	total := len(players)
	synced := 0
	skipped := 0
	failed := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("sync delta SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}

		accessToken, tokenErr := autoScheduler.TokenReader(ctx, dbPath, player.Gamertag, provider)
		if tokenErr != nil {
			failed++
			fmt.Printf("sync delta FAIL: gamertag=%s stage=token_read err=%v\n", player.Gamertag, tokenErr)
			continue
		}
		if strings.TrimSpace(accessToken) == "" {
			skipped++
			fmt.Printf("sync delta SKIP: gamertag=%s reason=no_token\n", player.Gamertag)
			continue
		}

		result, exchangeErr := provider.Exchange(ctx, accessToken)
		if exchangeErr != nil {
			failed++
			fmt.Printf("sync delta FAIL: gamertag=%s stage=exchange err=%v\n", player.Gamertag, exchangeErr)
			continue
		}

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, result.Tokens)
		syncResult, syncErr := engine.RunDelta(ctx, opts)
		if syncErr != nil {
			failed++
			fmt.Printf("sync delta FAIL: gamertag=%s stage=run_delta err=%v\n", player.Gamertag, syncErr)
			continue
		}

		synced++
		careerSynced := syncResult.PostSync != nil && syncResult.PostSync.CareerSynced
		fmt.Printf(
			"sync delta OK: gamertag=%s inserted=%d skipped=%d status=%s career_synced=%t duration=%.2fs\n",
			player.Gamertag,
			syncResult.MatchesInserted,
			syncResult.MatchesSkipped,
			syncResult.Status(),
			careerSynced,
			syncResult.DurationSeconds,
		)
		if len(syncResult.Warnings) > 0 {
			fmt.Printf("warnings=%d first=%s\n", len(syncResult.Warnings), syncResult.Warnings[0])
		}
	}

	fmt.Printf("sync delta batch: total=%d synced=%d skipped=%d failed=%d\n", total, synced, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("sync delta batch: %d joueur(s) en échec", failed)
	}
	return nil
}

func buildSyncOptions(maxMatches int, matchType string, rps int) domain.SyncOptions {
	opts := domain.DefaultSyncOptions()
	opts.MatchType = matchType
	opts.MaxMatches = maxMatches
	opts.RequestsPerSecond = rps
	return opts
}

func loadPlayerSummary(cfg *config.AppConfig, gamertag string) (*domain.PlayerSummary, error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil, fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	for _, player := range players {
		if strings.EqualFold(player.Gamertag, gamertag) || strings.EqualFold(player.PlayerSlug, gamertag) {
			playerCopy := player
			return &playerCopy, nil
		}
	}
	return nil, fmt.Errorf("joueur introuvable dans db_profiles.json: %s", gamertag)
}

func oauthRefreshTokenForPlayer(gamertag string) string {
	return os.Getenv(oauthRefreshEnvKey(gamertag))
}

func oauthRefreshEnvKey(gamertag string) string {
	key := strings.ToUpper(strings.TrimSpace(gamertag))
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	return "SPNKR_OAUTH_REFRESH_TOKEN_" + key
}
