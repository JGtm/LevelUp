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
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	opts := buildSyncOptions(maxMatches, matchType, rps)

	// ─── Création du pool de tokens (Discovery + Resolver + Pool) ───
	// Note : depuis la migration AutoSyncScheduler→Pool, le mode "sans pool"
	// n'est plus supporté. tokenPoolSize devient simplement le MaxSize du pool
	// (0 = tous les sources découverts).
	discovery := auth_pool.NewDiscovery(cfg, resolver, titlePkg.DefaultSlug)
	sources, discoveryErr := discovery.Scan(ctx)
	if discoveryErr != nil {
		return fmt.Errorf("pool discovery: %w", discoveryErr)
	}

	poolResolver := auth_pool.NewResolver(provider, 0, nil) // 0 = default TTL ~3h30 ; pas de persistance RT en mode CLI
	poolOpts := auth_pool.PoolOptions{
		MaxSize:     tokenPoolSize, // 0 = utiliser tous les sources
		PerTokenRPS: rps,
	}

	pool, poolErr := auth_pool.NewPool(ctx, poolResolver, sources, poolOpts)
	if poolErr != nil {
		return fmt.Errorf("pool creation: %w", poolErr)
	}
	defer pool.Close()

	fmt.Printf("pool: créé avec %d token(s) découverts\n", pool.Size())

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

		// ─── Client setup (toujours via le pool) ───
		// Skip si ce joueur n'a pas de token dans le pool (cas où Discovery
		// n'a rien trouvé pour lui : pas d'env var, pas de sync_meta).
		if !pool.HasPlayer(player.Gamertag) {
			skipped++
			fmt.Printf("sync delta SKIP: gamertag=%s reason=not_in_pool\n", player.Gamertag)
			continue
		}

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, &domain.HaloTokens{}, provider).
			WithCSRSeasonID(cfg.CurrentCSRSeasonID)
		cache := loadLocalFilmCache()
		if cache != nil {
			engine.SetLocalFilmCache(cache)
		}

		pooledClient := go_sync.NewPooledHaloClient(pool, player.Gamertag, player.XUID, 0) // 0 = defaultPooledRPS
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
	}

	fmt.Printf("sync delta batch: total=%d synced=%d skipped=%d failed=%d\n", total, synced, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("sync delta batch: %d joueur(s) en échec", failed)
	}
	return nil
}

func runSyncFull(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("sync-full", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire sauf avec --all)")
	allPlayers := fs.Bool("all", false, "Comble les trous pour tous les joueurs configurés")
	maxMatches := fs.Int("max-matches", 150, "Nombre de matchs API à parcourir (défaut 150 = 6 pages)")
	matchType := fs.String("match-type", "matchmaking", "Type de match: all|matchmaking|custom|local")
	rps := fs.Int("rps", 1, "Nombre max de requêtes API par seconde")
	tokenPoolSize := fs.Int("token-pool-size", 0, "Taille du pool de tokens (0=auto-detect)")
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
		return runSyncFullAll(ctx, cfg, *maxMatches, *matchType, *rps, *tokenPoolSize)
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

	syncResult, err := engine.RunFull(ctx, opts)
	if err != nil {
		return fmt.Errorf("run full: %w", err)
	}

	postSync := false
	careerSynced := false
	if syncResult.PostSync != nil {
		postSync = true
		careerSynced = syncResult.PostSync.CareerSynced
	}
	fmt.Printf(
		"sync full OK: gamertag=%s inserted=%d skipped=%d status=%s post_sync=%t career_synced=%t duration=%.2fs\n",
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

func runSyncFullAll(
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
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	opts := buildSyncOptions(maxMatches, matchType, rps)

	discovery := auth_pool.NewDiscovery(cfg, resolver, titlePkg.DefaultSlug)
	sources, discoveryErr := discovery.Scan(ctx)
	if discoveryErr != nil {
		return fmt.Errorf("pool discovery: %w", discoveryErr)
	}

	poolResolver := auth_pool.NewResolver(provider, 0, nil)
	poolOpts := auth_pool.PoolOptions{
		MaxSize:     tokenPoolSize,
		PerTokenRPS: rps,
	}

	pool, poolErr := auth_pool.NewPool(ctx, poolResolver, sources, poolOpts)
	if poolErr != nil {
		return fmt.Errorf("pool creation: %w", poolErr)
	}
	defer pool.Close()

	fmt.Printf("pool: créé avec %d token(s) découverts\n", pool.Size())

	total := len(players)
	synced := 0
	skipped := 0
	failed := 0

	for _, player := range players {
		dbPath := resolver.PlayerDBPath(titlePkg.DefaultSlug, player.Gamertag)
		if _, statErr := os.Stat(dbPath); os.IsNotExist(statErr) {
			skipped++
			fmt.Printf("sync full SKIP: gamertag=%s reason=no_player_db\n", player.Gamertag)
			continue
		}
		if !pool.HasPlayer(player.Gamertag) {
			skipped++
			fmt.Printf("sync full SKIP: gamertag=%s reason=not_in_pool\n", player.Gamertag)
			continue
		}

		engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, &domain.HaloTokens{}, provider).
			WithCSRSeasonID(cfg.CurrentCSRSeasonID)
		cache := loadLocalFilmCache()
		if cache != nil {
			engine.SetLocalFilmCache(cache)
		}

		pooledClient := go_sync.NewPooledHaloClient(pool, player.Gamertag, player.XUID, 0)
		if cache != nil {
			pooledClient.WithLocalFilmCache(cache)
		}
		engine.SetCustomClient(pooledClient)

		syncResult, syncErr := engine.RunFull(ctx, opts)
		if syncErr != nil {
			failed++
			fmt.Printf("sync full FAIL: gamertag=%s stage=run_full err=%v\n", player.Gamertag, syncErr)
			continue
		}

		synced++
		careerSynced := syncResult.PostSync != nil && syncResult.PostSync.CareerSynced
		fmt.Printf(
			"sync full OK: gamertag=%s inserted=%d skipped=%d status=%s career_synced=%t duration=%.2fs (pool)\n",
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
	}

	fmt.Printf("sync full batch: total=%d synced=%d skipped=%d failed=%d\n", total, synced, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("sync full batch: %d joueur(s) en échec", failed)
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
