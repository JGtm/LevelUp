package main

// pool_engine.go — construction du pool de tokens et du moteur de sync poolé pour la CLI.
//
// DOCTRINE (D1, plan 2026-09-16). Un profil suivi se synchronise par le POOL, qu'il ait ou non
// son propre refresh token : l'historique, les stats, les films et les CSR sont des endpoints
// PUBLICS que n'importe quel token du parc sert (PolicyAnyPublic). Seuls les endpoints soumis
// à la vie privée exigent le token du joueur (PolicyPinnedPlayer) : la personnalisation Spartan
// (sautée par son cron) et GetCareerRank (ErrNoPinnedToken — qu aucune étape du sync n appelle :
// le rang de carrière est servi par service.CareerLiveService, hors sync).
//
// Avant le 2026-09-16, la CLI mono-joueur exigeait le refresh token du joueur visé
// (haloTokensForPlayer) et la CLI `--all` sautait en bloc tout joueur absent du pool
// (`SKIP reason=not_in_pool`) : un profil suivi sans token propre — Nuzzles — n'était jamais
// synchronisé, alors que seul son rang de carrière lui est inaccessible.

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/capturecli"
	auth_pool "levelup/go-api/internal/platform/auth/pool"
	go_sync "levelup/go-api/internal/sync"
)

// buildCLITokenPool construit le pool de tokens des commandes CLI depuis le
// MultiUserTokenStore — source unique ADR 0023 Phase 5. Avant la Phase 5 ce chemin
// passait par `NewDiscovery` sans store, ce qui, une fois les fallbacks retirés, ne
// découvrait plus AUCUNE source (pool vide → « pool creation » en erreur).
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

// newPooledEngine construit le moteur de sync d'UN joueur servi par le pool. Les tokens
// passés au moteur sont volontairement VIDES (`&domain.HaloTokens{}`) : c'est le client
// poolé posé par SetCustomClient qui fournit l'authentification, lease par lease.
//
// Source unique des trois chemins (mono-joueur, `--all` delta, `--all` full) : avant, la
// même séquence de sept lignes était recopiée par appelant, et le chemin mono-joueur en
// divergeait (tokens du joueur, pas de pool).
func newPooledEngine(
	cfg *config.AppConfig,
	provider auth_platform.TokenProvider,
	pool auth_pool.Pool,
	player domain.PlayerSummary,
) *go_sync.SyncEngine {
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
	return engine
}

// newPooledEngineForPlayer : pool + moteur poolé pour une commande MONO-JOUEUR. Rend aussi
// le fermeur du pool, que l'appelant doit `defer`.
//
// `rps` = 0 laisse le défaut du pool (1 requête/s PAR TOKEN, donc `1 × taille du parc` au
// total). C'est la posture du projet, celle de `sync-delta --all` : le débit vient du nombre
// de jetons, pas d'un seul jeton poussé plus fort.
//
// Le joueur visé n'a pas besoin d'un token propre : le pool sert ses endpoints publics avec
// n'importe quel token du parc (D1). Un parc SANS aucun token reste une erreur — il n'y a
// alors plus personne pour parler à l'API.
func newPooledEngineForPlayer(
	ctx context.Context,
	cfg *config.AppConfig,
	player domain.PlayerSummary,
	tokenPoolSize, rps int,
) (*go_sync.SyncEngine, func(), error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil, nil, fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	provider := auth_platform.NewSISUProvider()
	pool, err := buildCLITokenPool(ctx, cfg, provider, players, tokenPoolSize, rps)
	if err != nil {
		return nil, nil, err
	}
	return newPooledEngine(cfg, provider, pool, player), pool.Close, nil
}

// newPooledClient : client Halo servi par le pool, pour les commandes qui n'ont pas besoin
// d'un moteur de sync complet (archivage de films, passe kill-source en ligne, rejeu des
// événements de surbrillance). Rend aussi le fermeur du pool, à `defer`.
//
// `gamertag` n'est plus un « prêteur de token » : il ne sert qu'à épingler les endpoints
// privacy-gated (aucun n'est appelé par ces commandes) et à nommer le joueur dans les
// journaux. Les chunks de film, les stats et l'historique sont publics (PolicyAnyPublic),
// donc servis par n'importe quel token du parc — un gamertag sans token propre passe.
func newPooledClient(
	ctx context.Context,
	cfg *config.AppConfig,
	gamertag string,
	rps int,
) (*go_sync.PooledHaloClient, func(), error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil, nil, fmt.Errorf("chargement db_profiles.json: %w", err)
	}
	provider := auth_platform.NewSISUProvider()
	pool, err := buildCLITokenPool(ctx, cfg, provider, players, 0, rps)
	if err != nil {
		return nil, nil, err
	}
	xuid := ""
	for _, p := range players {
		if strings.EqualFold(p.Gamertag, gamertag) || strings.EqualFold(p.PlayerSlug, gamertag) {
			xuid = p.XUID
			break
		}
	}
	return go_sync.NewPooledHaloClient(pool, gamertag, xuid, rps), pool.Close, nil
}
