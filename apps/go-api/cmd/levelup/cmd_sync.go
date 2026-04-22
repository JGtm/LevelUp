package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	go_sync "levelup/go-api/internal/sync"
)

func runSyncDelta(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("sync-delta", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	maxMatches := fs.Int("max-matches", 25, "Nombre max de nouveaux matchs à insérer")
	matchType := fs.String("match-type", "matchmaking", "Type de match: all|matchmaking|custom|local")
	rps := fs.Int("rps", 1, "Nombre max de requêtes API par seconde")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*gamertag) == "" {
		return fmt.Errorf("--gamertag est obligatoire")
	}

	player, err := loadPlayerSummary(cfg, *gamertag)
	if err != nil {
		return err
	}
	refreshToken := oauthRefreshTokenForPlayer(player.Gamertag)
	if refreshToken == "" {
		return fmt.Errorf("aucun refresh token OAuth trouvé pour %s (%s)", player.Gamertag, oauthRefreshEnvKey(player.Gamertag))
	}

	ctx := context.Background()
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
