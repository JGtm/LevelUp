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
	auth_pool "levelup/go-api/internal/platform/auth/pool"
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
	tokenPoolSize := fs.Int("token-pool-size", 0, "Taille du pool de tokens (0=auto-detect, 1=désactiver)")
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
		return runSyncDeltaAll(ctx, cfg, *maxMatches, *matchType, *rps, *tokenPoolSize)
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

	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, result.Tokens, provider).
		WithCSRSeasonID(cfg.CurrentCSRSeasonID)
	if cache := loadLocalFilmCache(); cache != nil {
		engine.SetLocalFilmCache(cache)
	}
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
	tokenPoolSize int,
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

	// ─── Création du pool de tokens (si tokenPoolSize != 1) ───
	var pool auth_pool.Pool
	if tokenPoolSize == 1 {
		// Pool désactivé
		fmt.Println("pool: désactivé (--token-pool-size 1)")
	} else {
		// Créer le pool avec Discovery + Resolver
		discovery := auth_pool.NewDiscovery(cfg, resolver, titlePkg.DefaultSlug)
		sources, discoveryErr := discovery.Scan(ctx)
		if discoveryErr != nil {
			return fmt.Errorf("pool discovery: %w", discoveryErr)
		}

		poolResolver := auth_pool.NewResolver(provider, 0) // 0 = default TTL ~3h30
		poolOpts := auth_pool.PoolOptions{
			MaxSize:     tokenPoolSize, // 0 = utiliser tous les sources
			PerTokenRPS: rps,
		}

		p, poolErr := auth_pool.NewPool(ctx, poolResolver, sources, poolOpts)
		if poolErr != nil {
			return fmt.Errorf("pool creation: %w", poolErr)
		}
		pool = p
		defer pool.Close()

		fmt.Printf("pool: créé avec %d token(s) découverts\n", pool.Size())
	}

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

		// ─── Client setup (pool ou standard) ───
		var syncErr error

		if pool != nil {
			// Utiliser le pool : créer un PooledHaloClient pinné à ce joueur
			// Pas besoin de TokenReader — les tokens sont déjà dans le pool
			engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, &domain.HaloTokens{}, provider).
				WithCSRSeasonID(cfg.CurrentCSRSeasonID)
			cache := loadLocalFilmCache()
			if cache != nil {
				engine.SetLocalFilmCache(cache)
			}

			// Créer un client poolé pinné (avec cache film si dispo)
			pooledClient := go_sync.NewPooledHaloClient(pool, player.Gamertag, player.XUID)
			if cache != nil {
				pooledClient.WithLocalFilmCache(cache)
			}
			engine.SetCustomClient(pooledClient)

			syncResult, syncErr := engine.RunDelta(ctx, opts)
			if syncErr != nil {
				failed++
				fmt.Printf("sync delta FAIL: gamertag=%s stage=run_delta err=%v\n", player.Gamertag, syncErr)
				continue
			}

			synced++
			careerSynced := syncResult.PostSync != nil && syncResult.PostSync.CareerSynced
			fmt.Printf(
				"sync delta OK: gamertag=%s inserted=%d skipped=%d status=%s career_synced=%t duration=%.2fs (pool)\n",
				player.Gamertag,
				syncResult.MatchesInserted,
				syncResult.MatchesSkipped,
				syncResult.Status(),
				careerSynced,
				syncResult.DurationSeconds,
			)
			if len(syncResult.Warnings) > 0 {
				fmt.Printf("  warnings=%d first=%s\n", len(syncResult.Warnings), syncResult.Warnings[0])
			}
			continue
		}

		// ─── Fallback standard (sans pool) ───
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

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, result.Tokens, provider).
			WithCSRSeasonID(cfg.CurrentCSRSeasonID)
		if cache := loadLocalFilmCache(); cache != nil {
			engine.SetLocalFilmCache(cache)
		}
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

// loadLocalFilmCache resout LEVELUP_LEGACY_FILM_CACHE_DIR (ex.
// `C:\...\LevelUp\data\cache`) et instancie un LocalFilmCache si le
// repertoire existe. Retourne nil sinon (cache desactive).
func loadLocalFilmCache() *go_sync.LocalFilmCache {
	dir := strings.TrimSpace(os.Getenv("LEVELUP_LEGACY_FILM_CACHE_DIR"))
	if dir == "" {
		return nil
	}
	return go_sync.NewLocalFilmCache(dir)
}
