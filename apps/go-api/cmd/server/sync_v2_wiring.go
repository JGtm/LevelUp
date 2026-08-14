// Package main — sync_v2_wiring.go : construction du CycleOrchestrator V2
// (ADR 0027 D6.5). Isolé dans son propre fichier pour faciliter la
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
	"strconv"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
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
	"levelup/go-api/internal/sync/replayartifacts"
	"levelup/go-api/internal/sync/snapshot"
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
	// PrestigeHook (optionnel) ré-évalue les défis Prestige actifs après le post-sync
	// de chaque joueur (Phase 6). = PrestigeBundle.RunPostSync. Nil → no-op.
	PrestigeHook func(ctx context.Context, playerSlug, titleSlug string)
	// ReplayEnqueue (optionnel) met la construction d'un rejeu dans la file durable
	// (= ServiceRegistry.EnqueueReplayBuild). Nil → le placement « worker » dégrade
	// en « aucune construction », journalisé.
	ReplayEnqueue replayartifacts.EnqueueFunc
}

// buildSyncV2Orchestrator construit l'orchestrator V2 avec ses 6
// dépendances réelles (V1-bridge). Retourne nil si une dépendance
// critique manque (le scheduler tombera en V1).
//
// Mode dry-run via LEVELUP_SYNC_V2_DRYRUN=1 : remplace persister +
// postSyncRunner par des stubs qui LOG les données qu'ils auraient
// écrites mais ne touchent PAS la DB. Permet de valider le pipeline
// (Discovery → Dedup → FetchShared → FetchPlayer) en conditions
// réelles sans aucun risque pour les données.
//
// Pour D8 cleanup : supprimer ce fichier + le bloc if-call dans main.go.
func buildSyncV2Orchestrator(deps SyncV2WiringDeps) syncv2.CycleOrchestrator {
	if deps.TokenPool == nil || deps.BatchQueue == nil {
		slog.Warn("sync.v2 wiring: pool ou queue absente → orchestrator non câblé")
		return nil
	}

	dryRun := os.Getenv("LEVELUP_SYNC_V2_DRYRUN") == "1"
	if dryRun {
		slog.Warn("sync.v2: MODE DRY-RUN ACTIVÉ — aucune écriture DB ne sera effectuée",
			"event", "sync.v2.dryrun.active")
	}

	// getSharedDB retourne la connexion shared courante via le cache process-wide.
	// Appelé à l'instant T (LoadKnown, RunPostSync) — après chaque swap provider
	// RO→RW→RO, LookupCachedDB retourne la connexion fraîche rouverte en RO.
	// Évite le pointeur fixe capturé au boot (deps.SharedDB) qui devient stale
	// après le premier cycle (le swap ferme l'ancienne *sql.DB).
	sharedPath := deps.PathResolver.SharedDBPath(deps.TitleSlug)
	getSharedDB := func() *sql.DB {
		if cached, ok := duckdbpkg.LookupCachedDB(sharedPath); ok {
			return cached.SQLDb()
		}
		return nil
	}

	// Adapter KnownLoader : lit les match_ids connus dans la stats.duckdb du
	// joueur (Phase 1 = lecture pure). OpenReadForQuery réutilise le handle déjà
	// en cache (RW ou RO) s'il existe — la lecture marche sur un handle RW.
	// Forcer OpenReadOnly échouait par intermittence quand un autre subsystem
	// (pool / career live / backup cron) tenait déjà la DB joueur en RW dans le
	// même process (DuckDB refuse RO+RW concurrents) → discovery "failed" sans
	// nouveau match inséré (incident 2026-06-01, même classe que RC-A / ADR-0016).
	playerDBOpenerRO := func(_ context.Context, gamertag string) (*sql.DB, func(), error) {
		path := deps.PathResolver.PlayerDBPath(deps.TitleSlug, gamertag)
		return duckdbpkg.OpenReadForQuery(path)
	}
	knownLoader := syncv2.NewKnownLoader(playerDBOpenerRO, getSharedDB)

	// CRITIQUE — Adapter PostSyncRunner : ouvre la stats.duckdb du joueur
	// en READ-WRITE car les heals post-sync UPDATE/INSERT sur 14+ tables
	// (sessions, performance_chain, engagement_*, citations, dominance_flag,
	// achievements, etc.). Bug observé 2026-05-25 19:41 : utiliser le
	// playerDBOpenerRO ici causait des ERROR "Cannot execute UPDATE on
	// database attached in read-only mode" qui silencieusement perdaient
	// toutes les écritures post-sync.
	playerDBOpenerRW := func(_ context.Context, gamertag string) (*sql.DB, func(), error) {
		path := deps.PathResolver.PlayerDBPath(deps.TitleSlug, gamertag)
		db, err := duckdbpkg.OpenReadWrite(path)
		if err != nil {
			return nil, nil, err
		}
		return db.SQLDb(), func() { _ = db.Close() }, nil
	}

	// HaloClient factory : pinned client par joueur via pool.
	clientFactory := func(gamertag, xuid string) syncv2.HaloClient {
		c := syncpkg.NewPooledHaloClient(deps.TokenPool, gamertag, xuid, 0)
		return c
	}
	matchListProvider := syncv2.NewMatchListProvider(clientFactory, "matchmaking", 25, 20)
	sharedFetcher := syncv2.NewSharedMatchFetcher(clientFactory)
	playerEnrFetcher := syncv2.NewPlayerEnrichmentFetcher()

	// CycleBatchPersister : réel ou dry-run logger.
	// playerBySlug est désormais passé via CycleBatch.PlayerBySlug
	// (renseigné par l'orchestrator à chaque cycle).
	var persister syncv2.CycleBatchPersister
	if dryRun {
		persister = newDryRunPersister()
	} else {
		// Résolution autonome des noms d'assets au sync V2 (primary write) : on passe
		// le POOL UNIFIÉ (deps.TokenPool, la même source que tous les syncs) — GameCMS
		// exige un token. Gate LEVELUP_SYNC_RESOLVE_ASSETS appliqué côté sync. Écriture
		// asset_translations via le handle metadata RW partagé (deps.MetaDB).
		persister = syncv2.NewCycleBatchPersister(deps.TitleSlug, deps.BatchQueue, 0,
			deps.MetaDB, deps.TokenPool, getSharedDB)
	}

	// RC-A fix (2026-06-01) : le PostSyncRunner doit acquérir le shared en
	// READ-WRITE (writer provider), pas le handle RO caché via getSharedDB. Les
	// heals events/skill/weapon/registry post-sync écrivent dans shared ; avec le
	// handle RO, toutes ces écritures échouaient "attached in read-only mode"
	// (silencieusement avant le fail-fast Phase 1a). Même classe de bug que le fix
	// player DB RO→RW du 2026-05-25, jamais corrigé côté shared.
	//
	// AcquireSharedWriterStandalone route via deps.Cfg.SharedProvider.AcquireWriter
	// (B-swap) ou OpenSharedDB legacy. Le release rend le writer (RW→RO).
	acquireSharedRW := func(ctx context.Context) (*sql.DB, func(), error) {
		// Étape 0 attribution : le post-sync V2 est le principal détenteur de la
		// fenêtre RW. Le label le rend mesurable (carte admin + watchdog).
		//
		// IfAbsent (et pas WithDBWriterLabel nu) : en mode bursts, l'appelant est
		// SharedAccess.Write, qui a DÉJÀ posé le label fin "sync_v2_postsync/<step>"
		// (weapons, events, psa_aliases). Écraser ici ramenait les 3 étapes sous le
		// même label grossier — les WARN watchdog prod n'attribuaient plus la
		// fenêtre à l'étape fautive. Seuls les call-sites nus (chemin pinned de
		// rollback LEVELUP_POSTSYNC_BURST=0) reçoivent le label de base.
		ctx = ctxkeys.WithDBWriterLabelIfAbsent(ctx, "sync_v2_postsync")
		return syncpkg.AcquireSharedWriterStandalone(ctx, deps.Cfg.SharedProvider, sharedPath)
	}

	// PostSyncRunner : réel ou dry-run no-op logger.
	// CRITIQUE : playerDBOpenerRW (pas RO) + acquireSharedRW (pas getSharedDB RO).
	var postSyncRunner syncv2.PostSyncRunner
	if dryRun {
		postSyncRunner = &dryRunPostSyncRunner{}
	} else {
		engineFactory := buildSyncEngineFactoryParityComplete(deps)
		runner := syncv2.NewPostSyncRunner(engineFactory, playerDBOpenerRW, acquireSharedRW, clientFactory)
		// Étape 1 contention : lecteur RO du shared (provider.Get, avec release)
		// pour les segments Read du pipeline en mode bursts. Sans provider
		// (legacy), les lectures retombent sur des bursts Write.
		if deps.Cfg.SharedProvider != nil {
			runner = runner.WithSharedReader(deps.Cfg.SharedProvider.Get)
		}
		postSyncRunner = runner
	}

	// Phase 6bis — producteur de snapshot immuable (durabilité / lecture découplée du
	// B-swap). Inconditionnel et best-effort : un échec de cut n'invalide pas le cycle.
	// Le cutter ouvre shared + player DB en RO via OpenReadForQuery (handle cached), donc
	// hors fenêtre RW (cut déclenché après libération du write-lease post-sync).
	keep := 0 // 0 → défaut appliqué par NewSnapshotCutter
	if v := os.Getenv("LEVELUP_SNAPSHOT_KEEP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			keep = n
		}
	}
	snapshotCutter := snapshot.NewSnapshotCutter(deps.PathResolver, keep)

	return syncv2.NewCycleOrchestrator(
		knownLoader,
		matchListProvider,
		sharedFetcher,
		playerEnrFetcher,
		persister,
		postSyncRunner,
		syncv2.CycleConfig{},
	).WithSnapshotProducer(snapshotCutter).
		WithPrestigeHook(deps.PrestigeHook)
}

// ─── Dry-run stubs (mode validation sans écriture DB) ─────────────────

// dryRunPersister implémente CycleBatchPersister en loggant ce qu'il
// aurait écrit, sans rien toucher en DB.
type dryRunPersister struct{}

func newDryRunPersister() syncv2.CycleBatchPersister { return &dryRunPersister{} }

func (p *dryRunPersister) PersistCycle(ctx context.Context, batch syncv2.CycleBatch) error {
	enrichmentsCount := 0
	for _, m := range batch.Enrichments {
		enrichmentsCount += len(m)
	}
	slog.InfoContext(ctx, "sync.v2 DRY-RUN: PersistCycle SKIP (no write)",
		"event", "sync.v2.dryrun.persist",
		"cycle_id", batch.CycleID,
		"matches_would_persist", len(batch.Matches),
		"enrichments_would_persist", enrichmentsCount,
	)
	for mID, sd := range batch.Matches {
		slog.InfoContext(ctx, "sync.v2 DRY-RUN: match récupéré (would persist)",
			"event", "sync.v2.dryrun.match",
			"match_id", mID,
			"fetcher", sd.Fetcher,
			"has_stats", sd.Stats != nil,
			"has_skill", len(sd.Skill) > 0,
			"has_highlights", sd.HasHighlights,
			"highlight_bytes", len(sd.HighlightChunk),
		)
	}
	return nil
}

// dryRunPostSyncRunner implémente PostSyncRunner en no-op.
type dryRunPostSyncRunner struct{}

func (r *dryRunPostSyncRunner) RunPostSync(ctx context.Context, p syncv2.PlayerProfile, insertedIDs []string) (syncv2.PlayerPostSyncResult, error) {
	slog.InfoContext(ctx, "sync.v2 DRY-RUN: PostSync SKIP (no heal, no write)",
		"event", "sync.v2.dryrun.postsync",
		"player", p.PlayerSlug,
		"inserted_ids_count", len(insertedIDs),
	)
	return syncv2.PlayerPostSyncResult{
		PlayerSlug: p.PlayerSlug,
	}, nil
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
		// MT-11 / PMT-3 : le profil porte le titre → écrit dans les DB du bon
		// titre (parité avec le path V1 BuildEngine). Slug vide → DefaultSlug.
		engine := syncpkg.NewSyncEngineForTitle(deps.Cfg.RepoRoot, p.TitleSlug, p.Gamertag, p.XUID, &domain.HaloTokens{}, deps.TokenProvider)

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

		// 2b. Fil de l'eau des artefacts de rejeu 2D (lot 6 v7.5). Le hook s'installe
		// toujours ; c'est LUI qui décide (construire ici / mettre en file / rien),
		// d'après replay_build_location relu à chaque cycle. Parité avec
		// scheduler.BuildEngine.
		if deps.Settings != nil {
			engine.WithReplayArtifacts(replayartifacts.NewHook(deps.Cfg, deps.Settings, deps.ReplayEnqueue))
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
				func() bool {
					var v *bool
					if cfg, _ := deps.Settings.Load(); cfg != nil {
						v = cfg.MediaDeleteSourceAfterTranscode
					}
					return config.ResolveMediaDeleteSource(
						os.Getenv(config.EnvMediaDeleteSource), v, deps.Cfg.IsProduction())
				},
			))
		}

		// 6. BatchQueue async (Phase 4.9 — INSERT-only via WAL + worker). Le batch
		// INSERT-only est le seul chemin d'écriture depuis D1b ; seul le layer async
		// reste optionnel via LEVELUP_PERSIST_BATCH_ASYNC (cf. main.go pour le
		// cycle de vie du kill-switch).
		if deps.BatchQueue != nil && deps.Cfg.PersistBatchAsync {
			engine.WithBatchQueue(deps.BatchQueue)
		}

		// 7. CSRSeasonID (CSR snapshot post-sync)
		if deps.Cfg.CurrentCSRSeasonID != "" {
			engine.WithCSRSeasonID(deps.Cfg.CurrentCSRSeasonID)
		}

		return engine, nil
	}
}
