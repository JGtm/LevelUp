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
	go_sync "levelup/go-api/internal/sync"
)

func runSyncDelta(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("sync-delta", flag.ExitOnError)
	gamertag := fs.String("gamertag", "", "Gamertag du joueur (obligatoire)")
	allPlayers := fs.Bool("all", false, "Synchronise tous les joueurs configurés")
	maxMatches := fs.Int("max-matches", 25, "Nombre max de nouveaux matchs à insérer")
	matchType := fs.String("match-type", "matchmaking", "Type de match: all|matchmaking|custom|local")
	rps := fs.Int("rps", 1, "Nombre max de requêtes API par seconde")
	tokenPoolSize := fs.Int("token-pool-size", 0, "Nombre maximal de slots SAINS du pool de tokens (0=tous les jetons sains du parc)")
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
	// Le joueur visé n a PAS besoin de son propre token : TOUS ses endpoints sont publics et
	// servis par n importe quel token du pool (D1 du plan 2026-09-16, étendu au rang de
	// carrière par D4 du plan robustesse). Le rang de carrière n est pas synchronisé ici
	// (flux CareerLiveService) : career_synced vaut false pour tous.
	engine, closePool, err := newPooledEngineForPlayer(ctx, cfg, *player, *tokenPoolSize, *rps)
	if err != nil {
		return err
	}
	defer closePool()

	opts := buildSyncOptions(*maxMatches, *matchType, *rps)

	syncResult, err := engine.RunDelta(ctx, opts)
	if err != nil {
		return fmt.Errorf("run delta: %w", err)
	}

	return reportSyncResult(os.Stdout, "delta", player.Gamertag, &syncResult)
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
		// AUCUN skip « pas de token propre » (D1, plan 2026-09-16) : le pool sert les
		// endpoints publics de n'importe quel joueur suivi. Le garde-fou `not_in_pool` qui
		// vivait ici datait du pool-par-joueur d'avant l'ADR 0023. Le skip `no_player_db`
		// ci-dessus RESTE : `--all` ne crée pas la base d'un profil neuf, seul le sync
		// mono-joueur le fait (décision inchangée, plan 2026-09-16 §8).

		engine := newPooledEngine(cfg, provider, pool, player)

		syncResult, syncErr := engine.RunDelta(ctx, opts)
		if syncErr != nil {
			failed++
			fmt.Printf("sync delta FAIL: gamertag=%s stage=run_delta err=%v\n", player.Gamertag, syncErr)
			continue
		}

		// Un joueur dont la passe s'est arrêtée sur une erreur compte dans `failed` :
		// le compte rendu du lot doit dire la même chose que le verdict par joueur.
		if reportErr := reportSyncResult(os.Stdout, "delta", player.Gamertag, &syncResult); reportErr != nil {
			failed++
			continue
		}
		synced++
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
	tokenPoolSize := fs.Int("token-pool-size", 0, "Nombre maximal de slots SAINS du pool de tokens (0=tous les jetons sains du parc)")
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
	// Le joueur visé n a PAS besoin de son propre token : TOUS ses endpoints sont publics et
	// servis par n importe quel token du pool (D1 du plan 2026-09-16, étendu au rang de
	// carrière par D4 du plan robustesse). Le rang de carrière n est pas synchronisé ici
	// (flux CareerLiveService) : career_synced vaut false pour tous.
	engine, closePool, err := newPooledEngineForPlayer(ctx, cfg, *player, *tokenPoolSize, *rps)
	if err != nil {
		return err
	}
	defer closePool()

	opts := buildSyncOptions(*maxMatches, *matchType, *rps)

	syncResult, err := engine.RunFull(ctx, opts)
	if err != nil {
		return fmt.Errorf("run full: %w", err)
	}

	return reportSyncResult(os.Stdout, "full", player.Gamertag, &syncResult)
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
		// Idem `--all` delta : plus de skip `not_in_pool` (D1, plan 2026-09-16) ; le skip
		// `no_player_db` ci-dessus reste (décision inchangée, plan 2026-09-16 §8).

		engine := newPooledEngine(cfg, provider, pool, player)

		syncResult, syncErr := engine.RunFull(ctx, opts)
		if syncErr != nil {
			failed++
			fmt.Printf("sync full FAIL: gamertag=%s stage=run_full err=%v\n", player.Gamertag, syncErr)
			continue
		}

		// Idem `--all` delta : une passe au statut non-`success` compte dans `failed`.
		if reportErr := reportSyncResult(os.Stdout, "full", player.Gamertag, &syncResult); reportErr != nil {
			failed++
			continue
		}
		synced++
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
