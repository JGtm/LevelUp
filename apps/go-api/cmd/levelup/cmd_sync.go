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
	"levelup/go-api/internal/platform/auth/capturecli"
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
	tokens, err := haloTokensForPlayer(ctx, cfg.RepoRoot, player.Gamertag)
	if err != nil {
		return err
	}

	provider := auth_platform.NewSISUProvider()
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, tokens, provider).
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

	provider := auth_platform.NewSISUProvider()
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	opts := buildSyncOptions(maxMatches, matchType, rps)

	// ─── Création du pool de tokens (Discovery + Resolver + Pool) ───
	// Note : depuis la migration AutoSyncScheduler→Pool, le mode "sans pool"
	// n'est plus supporté. tokenPoolSize devient simplement le MaxSize du pool
	// (0 = tous les sources découverts).
	pool, poolErr := buildCLITokenPool(ctx, cfg, provider, players, tokenPoolSize, rps)
	if poolErr != nil {
		return poolErr
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
	tokens, err := haloTokensForPlayer(ctx, cfg.RepoRoot, player.Gamertag)
	if err != nil {
		return err
	}

	provider := auth_platform.NewSISUProvider()
	engine := go_sync.NewSyncEngine(cfg.RepoRoot, player.Gamertag, player.XUID, tokens, provider).
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

	provider := auth_platform.NewSISUProvider()
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	opts := buildSyncOptions(maxMatches, matchType, rps)

	pool, poolErr := buildCLITokenPool(ctx, cfg, provider, players, tokenPoolSize, rps)
	if poolErr != nil {
		return poolErr
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

// buildCLITokenPool construit le pool de tokens des commandes `--all`
// (sync-delta / sync-full) depuis le MultiUserTokenStore — source unique
// ADR 0023 Phase 5. Avant la Phase 5 ce chemin passait par `NewDiscovery` sans
// store, ce qui, une fois les fallbacks retirés, ne découvrait plus AUCUNE
// source (pool vide → « pool creation » en erreur).
//
// La rotation du refresh token est PERSISTÉE dans le store (callback onRotated).
// C'est non négociable depuis que le CLI consomme le RT canonique du serveur :
// Microsoft rotate le RT à chaque usage, et ne pas réécrire la rotation
// brûlerait le token du store — invalid_grant au refresh suivant, côté serveur
// comme côté CLI (classe d'incident Madina, ADR 0023).
func buildCLITokenPool(
	ctx context.Context,
	cfg *config.AppConfig,
	provider auth_platform.TokenProvider,
	players []domain.PlayerSummary,
	tokenPoolSize, rps int,
) (auth_pool.Pool, error) {
	pathResolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	store := auth_platform.NewMultiUserTokenStore(pathResolver.WatcherTokensDir())

	discovery := auth_pool.NewDiscoveryWithStore(cfg, pathResolver, titlePkg.DefaultSlug, store)
	sources, err := discovery.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("pool discovery: %w", err)
	}

	onRotated := func(rctx context.Context, gamertag, newRT string) error {
		xuid := capturecli.ResolveXUIDForRotation(rctx, store, players, gamertag)
		if xuid == "" {
			return fmt.Errorf("rotation RT non persistée : xuid introuvable pour %s", gamertag)
		}
		return store.UpdateOAuthRefreshToken(xuid, newRT)
	}

	// 0 = TTL par défaut du resolver (~3h30, durée de vie Spartan ~4h).
	poolResolver := auth_pool.NewResolver(provider, 0, onRotated)
	pool, err := auth_pool.NewPool(ctx, poolResolver, sources, auth_pool.PoolOptions{
		MaxSize:     tokenPoolSize, // 0 = utiliser toutes les sources découvertes
		PerTokenRPS: rps,
	})
	if err != nil {
		return nil, fmt.Errorf("pool creation: %w", err)
	}
	return pool, nil
}

// haloTokensForPlayer obtient des tokens Halo (Spartan + Clearance) frais pour
// un joueur depuis le MultiUserTokenStore — source unique ADR 0023 Phase 5
// (2026-08-25 : le repli par variable d'environnement n'est plus lu).
// La rotation du refresh token est persistée par le helper canonique.
func haloTokensForPlayer(ctx context.Context, repoRoot, gamertag string) (*domain.HaloTokens, error) {
	store := auth_platform.NewMultiUserTokenStore(titlePkg.NewPathResolver(repoRoot).WatcherTokensDir())
	user, err := store.LoadByGamertag(gamertag)
	if err != nil || user == nil || user.OAuthRefreshToken == "" {
		return nil, fmt.Errorf("aucun refresh token pour %s dans data/auth/watcher_tokens "+
			"(se connecter via le SSO Xbox ou lancer `go run ./cmd/token-capture/ %s`)", gamertag, gamertag)
	}
	result, err := auth_platform.RefreshHaloTokensViaStoreFirst(
		ctx, store, auth_platform.NewSISUProvider(), user.XUID, gamertag)
	if err != nil {
		return nil, fmt.Errorf("refresh store pour %s: %w", gamertag, err)
	}
	tokens := auth_platform.HaloTokensFromExchange(result)
	if tokens == nil {
		return nil, fmt.Errorf("aucun token Halo obtenu pour %s", gamertag)
	}
	return tokens, nil
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
