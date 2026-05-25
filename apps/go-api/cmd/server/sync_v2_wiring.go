// Package main — sync_v2_wiring.go : construction du CycleOrchestrator V2
// (ADR 0020 D6.5). Isolé dans son propre fichier pour faciliter la
// suppression au D8 cleanup (rm sync_v2_wiring.go + 1 if-block dans main.go).
//
// Le câblage est défensif : retourne nil si une dépendance manque ; le
// scheduler.shouldUseV2() ne retournera pas true tant que l'orchestrator
// n'est pas câblé, donc V1 reste actif par défaut.
package main

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// buildSyncV2Orchestrator construit l'orchestrator V2 avec ses 6
// dépendances réelles (V1-bridge). Retourne nil si une dépendance
// critique manque (le scheduler tombera en V1).
//
// Pour D8 cleanup : supprimer ce fichier + le bloc if-call dans main.go.
func buildSyncV2Orchestrator(
	cfg *config.AppConfig,
	pr *titlePkg.PathResolver,
	titleSlug string,
	tokenPool pool.Pool,
	batchQueue *persist.BatchQueue,
	metaDB *sql.DB,
	sharedDB *sql.DB,
	tokenProvider auth.TokenProvider,
) syncv2.CycleOrchestrator {
	if tokenPool == nil || batchQueue == nil {
		slog.Warn("sync.v2 wiring: pool ou queue absente → orchestrator non câblé")
		return nil
	}

	// Adapter 1 — KnownLoader : ouvre la stats.duckdb du joueur en RO.
	playerDBOpener := func(ctx context.Context, gamertag string) (*sql.DB, func(), error) {
		path := pr.PlayerDBPath(titleSlug, gamertag)
		db, err := duckdbpkg.OpenReadOnly(path)
		if err != nil {
			return nil, nil, err
		}
		// OpenReadOnly est cached process-wide ; Close() décrémente refCount.
		return db.SQLDb(), func() { _ = db.Close() }, nil
	}
	knownLoader := syncv2.NewKnownLoader(playerDBOpener, sharedDB)

	// Adapter 2/3/4 — HaloClient factory : pinned client par joueur via pool.
	clientFactory := func(gamertag, xuid string) syncv2.HaloClient {
		c := syncpkg.NewPooledHaloClient(tokenPool, gamertag, xuid, 0)
		return c // *PooledHaloClient satisfait syncv2.HaloClient (narrow subset of sync.HaloClient)
	}
	matchListProvider := syncv2.NewMatchListProvider(clientFactory, "matchmaking", 25, 20)
	sharedFetcher := syncv2.NewSharedMatchFetcher(clientFactory)
	playerEnrFetcher := syncv2.NewPlayerEnrichmentFetcher() // no-op D6.3 (cf. doc)

	// Adapter 5 — CycleBatchPersister : utilise BuildBatchFromRawForV2 +
	// BatchQueue partagée. Le slug→profile map est construit à la volée
	// à chaque cycle par l'orchestrator (cf. cycle.go playerBySlug), donc
	// ici on initialise vide et le persister gère le lookup au PersistCycle.
	persister := syncv2.NewCycleBatchPersister(titleSlug, map[string]syncv2.PlayerProfile{}, batchQueue, 0)

	// Adapter 6 — PostSyncRunner : construit un SyncEngine V1 par joueur
	// via SyncEngineFactory + délègue à RunPostSyncForV2.
	engineFactory := func(ctx context.Context, p syncv2.PlayerProfile) (*syncpkg.SyncEngine, error) {
		tokens := &domain.HaloTokens{}
		engine := syncpkg.NewSyncEngine(cfg.RepoRoot, p.Gamertag, p.XUID, tokens, tokenProvider)
		if cfg.CurrentCSRSeasonID != "" {
			engine.WithCSRSeasonID(cfg.CurrentCSRSeasonID)
		}
		return engine, nil
	}
	postSyncRunner := syncv2.NewPostSyncRunner(engineFactory, playerDBOpener, sharedDB, clientFactory)

	// Construire l'orchestrator avec config defaults (8/4/0).
	return syncv2.NewCycleOrchestrator(
		knownLoader,
		matchListProvider,
		sharedFetcher,
		playerEnrFetcher,
		persister,
		postSyncRunner,
		syncv2.CycleConfig{},
	)
}
