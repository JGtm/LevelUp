// Package main — sync_v2_wiring.go : construction du CycleOrchestrator V2
// (ADR 0020 D6.5). Isolé dans son propre fichier pour faciliter la
// suppression au D8 cleanup (rm sync_v2_wiring.go + 1 if-block dans main.go).
//
// Le câblage est défensif : retourne nil si une dépendance manque ; le
// scheduler.shouldUseV2() ne retournera pas true tant que l'orchestrator
// n'est pas câblé, donc V1 reste actif par défaut.
//
// T1 (audit defaultRunnerFactory) : engineFactory mirror EXACT des
// options câblées par scheduler.defaultRunnerFactory pour garantir la
// parité runtime V1↔V2 sur le post-sync (sessions, achievements,
// progression V2, media scan, etc.).
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/pool"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	settingsplatform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	syncpkg "levelup/go-api/internal/sync"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// SyncV2WiringDeps regroupe les dépendances brutes nécessaires à V2.
// Construit dans main.go ; passé à buildSyncV2Orchestrator.
//
// Cohérence forte avec defaultRunnerFactory (scheduler/auto_sync.go:200)
// — toute nouvelle option WithX sur SyncEngine doit être ajoutée ici
// ET dans engineFactory ci-dessous pour préserver la parité V1↔V2.
type SyncV2WiringDeps struct {
	Cfg            *config.AppConfig
	PathResolver   *titlePkg.PathResolver
	TitleSlug      string
	TokenPool      pool.Pool
	BatchQueue     *persist.BatchQueue
	MetaDB         *sql.DB
	SharedDB       *sql.DB
	TokenProvider  auth.TokenProvider
	Settings       *settingsplatform.Store // pour FriendsLoader + MediaScanHook
	PostSyncRunner port.PostSyncRunner     // pour WithPostSyncRunner (progression V2)
}

// buildSyncV2Orchestrator construit l'orchestrator V2 avec ses 6
// dépendances réelles (V1-bridge). Retourne nil si une dépendance
// critique manque (le scheduler tombera en V1).
//
// Pour D8 cleanup : supprimer ce fichier + le bloc if-call dans main.go.
func buildSyncV2Orchestrator(deps SyncV2WiringDeps) syncv2.CycleOrchestrator {
	if deps.TokenPool == nil || deps.BatchQueue == nil {
		slog.Warn("sync.v2 wiring: pool ou queue absente → orchestrator non câblé")
		return nil
	}

	// Adapter KnownLoader : ouvre la stats.duckdb du joueur en RO.
	playerDBOpener := func(_ context.Context, gamertag string) (*sql.DB, func(), error) {
		path := deps.PathResolver.PlayerDBPath(deps.TitleSlug, gamertag)
		db, err := duckdbpkg.OpenReadOnly(path)
		if err != nil {
			return nil, nil, err
		}
		// OpenReadOnly est cached process-wide ; Close() décrémente refCount.
		return db.SQLDb(), func() { _ = db.Close() }, nil
	}
	knownLoader := syncv2.NewKnownLoader(playerDBOpener, deps.SharedDB)

	// HaloClient factory : pinned client par joueur via pool.
	clientFactory := func(gamertag, xuid string) syncv2.HaloClient {
		c := syncpkg.NewPooledHaloClient(deps.TokenPool, gamertag, xuid, 0)
		return c
	}
	matchListProvider := syncv2.NewMatchListProvider(clientFactory, "matchmaking", 25, 20)
	sharedFetcher := syncv2.NewSharedMatchFetcher(clientFactory)
	playerEnrFetcher := syncv2.NewPlayerEnrichmentFetcher()

	// CycleBatchPersister : utilise BuildBatchFromRawForV2 + BatchQueue partagée.
	persister := syncv2.NewCycleBatchPersister(deps.TitleSlug, map[string]syncv2.PlayerProfile{}, deps.BatchQueue, 0)

	// PostSyncRunner : construit un SyncEngine V1 PARITY-COMPLETE via
	// engineFactory qui mirror exactement defaultRunnerFactory (T1 audit).
	engineFactory := buildSyncEngineFactoryParityComplete(deps)
	postSyncRunner := syncv2.NewPostSyncRunner(engineFactory, playerDBOpener, deps.SharedDB, clientFactory)

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

// buildSyncEngineFactoryParityComplete construit un SyncEngineFactory
// qui mirror EXACTEMENT scheduler.defaultRunnerFactory (auto_sync.go:204).
//
// Garantit la parité runtime V1↔V2 sur le post-sync :
//   - WithSharedProvider (mode B-swap si activé)
//   - WithFriendsLoader (recalcul sessions / is_with_friends)
//   - SetCustomClient (pool token pinned)
//   - WithPostSyncRunner (notifications delta + progression V2 Ascension)
//   - WithMediaScanHook (indexation captures)
//   - WithBatchPersistMode + WithBatchQueue (path INSERT-only Phase 2.3)
//   - WithCSRSeasonID (CSR snapshot)
//
// CRITIQUE : toute modification de defaultRunnerFactory doit être
// répliquée ICI sous peine de divergence runtime V1↔V2.
func buildSyncEngineFactoryParityComplete(deps SyncV2WiringDeps) syncv2.SyncEngineFactory {
	return func(_ context.Context, p syncv2.PlayerProfile) (*syncpkg.SyncEngine, error) {
		engine := syncpkg.NewSyncEngine(deps.Cfg.RepoRoot, p.Gamertag, p.XUID, &domain.HaloTokens{}, deps.TokenProvider)

		// 1. SharedProvider (B-swap si LEVELUP_USE_SHARED_PROVIDER=1)
		if deps.Cfg.SharedProvider != nil {
			engine.WithSharedProvider(deps.Cfg.SharedProvider)
		}

		// 2. FriendsLoader (sessions auto-recompute is_with_friends)
		if deps.Settings != nil {
			engine.WithFriendsLoader(func() ([]string, error) {
				cfg, lerr := deps.Settings.Load()
				if lerr != nil {
					return nil, lerr
				}
				return cfg.FriendGamertags, nil
			})
		}

		// 3. Custom client pinned via pool
		if deps.TokenPool != nil {
			pooledClient := syncpkg.NewPooledHaloClient(deps.TokenPool, p.Gamertag, p.XUID, 0)
			engine.SetCustomClient(pooledClient)
		}

		// 4. PostSyncRunner V1 (notifications delta + progression V2 Ascension)
		// PlayerSlug = gamertag (cf. config.go:284 — gamertag dans db_profiles.json)
		if deps.PostSyncRunner != nil {
			engine.WithPostSyncRunner(deps.PostSyncRunner, p.Gamertag)
		}

		// 5. MediaScanHook (indexation captures post-sync)
		if deps.Settings != nil {
			engine.WithMediaScanHook(service.BuildMediaScanHook(deps.Cfg.RepoRoot, p.Gamertag,
				func() string {
					cfg, _ := deps.Settings.Load()
					if cfg != nil {
						return cfg.MediaCapturesBaseDir
					}
					return ""
				},
				func() string {
					cfg, _ := deps.Settings.Load()
					if cfg != nil {
						return cfg.UserTimezone
					}
					return ""
				},
			))
		}

		// 6. BatchPersistMode + BatchQueue (Phase 2.3/4.7/4.9 — INSERT-only async)
		if os.Getenv("LEVELUP_PERSIST_BATCH") != "0" {
			engine.WithBatchPersistMode(true)
			if deps.BatchQueue != nil && os.Getenv("LEVELUP_PERSIST_BATCH_ASYNC") != "0" {
				engine.WithBatchQueue(deps.BatchQueue)
			}
		}

		// 7. CSRSeasonID (CSR snapshot post-sync)
		if deps.Cfg.CurrentCSRSeasonID != "" {
			engine.WithCSRSeasonID(deps.Cfg.CurrentCSRSeasonID)
		}

		return engine, nil
	}
}
