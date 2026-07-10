# Sync Call Tree (Go)


> Code-level call graph of the **Go** sync pipeline. For the operational guide
> (settings, CLI, troubleshooting) read [SYNC_GUIDE.md](SYNC_GUIDE.md) first; this
> file is the map from entry point to the persist layer.
>
> The Python/Streamlit sync (`streamlit_app.py`, `src/ui/sync.py`, `src/data/sync/`)
> no longer exists. Sync runs entirely inside `apps/go-api`.

There are **two entry points**, both backed by the same `SyncEngine` and token
pool, plus an opt-in **V2 cycle orchestrator** that replaces the per-player loop.

Key files: `internal/scheduler/auto_sync.go`, `internal/sync/engine.go`,
`internal/sync/engine_batch_path.go`, `internal/persist/builder.go`,
`internal/sync/v2/cycle.go`.

## Entry point A — Auto-sync scheduler (periodic)

```text
AutoSyncScheduler.Run(ctx)                         (scheduler/auto_sync.go)
  └─ for each ticker.C  (interval from app_settings.json)
       ├─ settings.Load()
       │     ├─ skip tick if !cfg.SpnkrAutoSyncEnabled
       │     └─ resolveInterval(SpnkrAutoSyncIntervalMinutes|Hours)  → ticker.Reset
       │
       └─ RunOnceTrigger(ctx, "tick")
            │  (RunOnce(ctx) = same with trigger "manual", e.g. admin endpoint)
            │
            ├─ players := cfg.LoadPlayers()            (db_profiles.json)
            ├─ players = domain.SyncablePlayers(players)  ← drop sync_enabled=false
            ├─ warnStaleGateClaims(ctx)                 ← leaked-claim heartbeat
            │
            ├─ if shouldUseV2():  runOnceV2(ctx, players)   → see Entry point C
            │     └─ fallback to V1 on ErrNotImplemented / error
            │
            └─ errgroup (limit = pool size) — one goroutine per player:
                 └─ syncPlayer(ctx, p)  → syncOutcome {OK|Skipped|Failed}
                      │
                      ├─ checkSyncPreconditions(ctx, p)
                      │     ├─ pool != nil  &&  pool.HasPlayer(gamertag)
                      │     ├─ !ActivityChecker.IsPlayerActive(gamertag)  ← yield to watcher
                      │     └─ player DB exists (skipped for live-only titles)
                      │
                      ├─ SyncGate.TryClaimT(slug, gamertag)   ← cross-source dedup
                      │     └─ defer release()   (skip → retried next tick)
                      │
                      ├─ ctx = ctxkeys.WithTitleSlug(ctx, slug)   ← MT-11 / PMT-3
                      │
                      ├─ runner := RunnerFactory(ctx, gamertag, xuid)
                      │     │   default = defaultRunnerFactory → BuildEngine(...)
                      │     │   (live-only titles → liveRunner / livesync.AcquireRunner)
                      │     │
                      │     └─ BuildEngine(ctx, gamertag, xuid)   (the ONLY engine wiring funnel)
                      │           ├─ NewSyncEngineForTitle(RepoRoot, slug, gamertag, xuid, ...)
                      │           ├─ WithSharedProvider   (if cfg.SharedProvider, B-swap)
                      │           ├─ WithFriendsLoader    (settings.FriendGamertags)
                      │           ├─ SetCustomClient(NewPooledHaloClient(pool, gamertag, xuid))
                      │           ├─ WithPostSyncRunner(postSyncRunner, gamertag)
                      │           ├─ WithMediaScanHook(...)
                      │           ├─ WithBatchQueue(batchQueue)    if LEVELUP_PERSIST_BATCH_ASYNC != "0"
                      │           ├─ WithCSRSeasonID / WithAssetNameResolution(pool)
                      │           └─ returns *sync.SyncEngine
                      │
                      ├─ runner.RunDelta(ctx, domain.DefaultSyncOptions())   → see SyncEngine.run
                      │
                      ├─ record counters (MatchesInserted / Skipped / MedalsInserted / PostSync)
                      ├─ ConsecutiveZeroInserts++ (reset if inserted>0)  ← warn ≥ threshold(6)
                      └─ runEventsConvergencePass(ctx, gamertag, xuid)
                            └─ BuildEngine(...).RunEventsConvergence(ctx, nil, limit)   ← bounded heal
```

Diagnostics for every cycle are exposed at `GET /api/v1/_diag/auto-sync/snapshot`
(`AutoSyncScheduler.Snapshot()`).

The **presence watcher** (`internal/watcher`) is the event-driven sibling: on
match completion it enqueues a delta sync for one player through the same
`SyncEngine` (wired via `syncTrigger.WithEngineFactory` → the same `BuildEngine`,
guaranteeing watcher/scheduler parity). It then converges on `SyncEngine.run`
below.

## Engine core — `SyncEngine.run` (V1, per player)

```text
SyncEngine.RunDelta(ctx, opts)  → run(ctx, opts, isDelta=true)   (sync/engine.go)
SyncEngine.RunFull(ctx, opts)   → run(ctx, opts, isDelta=false)
  │
  ├─ opts.Validate()                                   ← fail-fast
  ├─ postSyncFinalizer = postSyncRunner.BeforeSync(...)  (snapshot before sync)
  │     └─ defer finalizer AFTER writer leases (LIFO → shared back to RO first)
  │
  ├─ ── write leases / DB handles ──
  │     ├─ dblease.AcquireWriterCtx(playerDBPath, KindPlayer)
  │     ├─ OpenPlayerDB(playerDBPath)
  │     ├─ acquireSharedWriter(ctx)   (Provider B-swap, else OpenSharedDB direct)
  │     └─ metadata DB via OpenReadForQuery   (best-effort name enrichment)
  │
  ├─ ── fetch + delta walk ──
  │     ├─ HEAD / watermark check  (delta: stop at first known match)
  │     ├─ load known match ids (shared × player_match_enrichment × awards)
  │     ├─ PooledHaloClient.GetMatchHistory / GetMatchStats (parallel fetch)
  │     └─ per match → persistFetchedMatch(...)
  │           └─ submitMatchAsBatch(...)   → see Persist layer (INSERT-only, seul chemin)
  │
  └─ ── post-sync (best effort) ──
        ├─ refresh aggregates / career rank / sync_meta watermark
        ├─ performance scores · sessions · citations · dominance · LUSR (v2 canonical)
        └─ finalizerArmed = true   → postSyncFinalizer runs on success only
              (delta notifications + Ascension progression V2)
```

## Persist layer — Collect → Persist (INSERT-only, ART-safe)

```text
submitMatchAsBatch(ctx, sharedDB, playerDB, result, fm)   (sync/engine_batch_path.go)
  │
  ├─ build the batch with persist.BatchBuilder   (persist/builder.go)
  │     SetMatch · AddParticipants · AddMedals · AddHighlightEvents · AddWeaponKills
  │     AddKillerVictim · AddKillPositions · AddXUIDAliases · AddMatchCSRs
  │     AddCommendations · SetEnrichment · SetSkillRank · AddLUSRComponents
  │     AddCitations · AddPersonalScoreAwards · SetCareerProgression · SetSession
  │     AddPVEStats · AddModeNameTranslations
  │       └─ Build() → *persist.MatchBatch
  │
  ├─ if batchQueue != nil:   batchQueue.Submit(batch)   ← async, durable WAL, boot recovery
  │                            (LEVELUP_PERSIST_BATCH_ASYNC)
  │
  └─ else (synchronous):
        ├─ SharedPersister.Persist(ctx, sharedDB, batch)   ← INSERT-only on shared tables
        └─ PlayerPersister.Persist(ctx, playerDB, batch)   ← INSERT-only on player tables
```

No concurrent `UPDATE` / `INSERT ... ON CONFLICT DO UPDATE` on the shared/state
tables — append-only by construction (ADR
[0019](adr/0019-collect-persist-architecture.md),
[0026](adr/0026-append-only-art-eradication.md)).

## Entry point C — V2 cycle orchestrator (sole cycle engine)

The sole engine driver of the auto-sync cycle since the V1 pipeline was removed
(2026-07). `RunOnceTrigger` → `shouldUseV2()` (= orchestrator wired) → `runOnceV2`
→ `CycleOrchestratorImpl.Run` for engine titles; live-only titles (Halo 5) go through
`syncPlayer`→`liveRunner`. No orchestrator wired → structural `syncPlayer` safety net.

```text
CycleOrchestratorImpl.Run(ctx, players)            (sync/v2/cycle.go)
  ├─ Phase 1  RunDiscovery   — parallel per player, read-only: known ids + API pages
  ├─ Phase 2  RunDedup       — single, pure: union of unknown match ids
  ├─ Phase 3  RunFetchShared — errgroup(FetchSharedParallelism, default 8): GetMatchStats/unique match
  ├─ Phase 4  RunFetchPlayer — parallel per player (FetchPlayerParallelism, default 4): awards/scores
  ├─ Phase 5  RunPersist     — single writer: one mega-batch (shared + player) → CycleBatchPersister
  └─ Phase 6  RunPostSync    — parallel per player: heals / films / citations / dominance
        → CycleResult { PerPlayer, UniqueMatches, PhaseDurations }
```

Each phase records `expvar` metrics (`sync_v2_phase_duration_ms_*`,
`sync_v2_cycle_*`) and structured logs (`event=sync.v2.cycle.phase`). Per-player
errors are collected in `CycleResult.PerPlayer` instead of aborting the cycle.
