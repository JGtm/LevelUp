# ADR 0016 — SharedDBProvider with dynamic RO↔RW swap (B-swap)

**Date** : 2026-05-19
**Status** : Accepted (Provider available behind flag, default-on at commit 9)
**Branch** : `fix/auto-sync-different-configuration` (commits 1-9b)
**Related** : ADR 0013 (LeasedWriter), ADR 0009 (expvar monitoring)

## Context

The Go API runs a sync engine and HTTP handlers concurrently in the same process. They share `shared_matches_v2.duckdb`, which stores match registry / participants / medals / events / weapon kills — the canonical cross-player data.

Three independent file handles coexisted on this DB :
1. **Global RO** opened at boot (`main.go:209` legacy), shared by all HTTP handlers.
2. **N RO handles** opened by the player pool (`pool.go:152` — one ATTACH per player conn).
3. **Ad-hoc RW** opened by the sync engine (`sync/schema.go::OpenSharedDB`) on every `RunDelta`.

The DuckDB-Go v2 driver refuses #3 while #1 + #2 are alive, raising the error :

```
Can't open a connection to same database file with a different configuration than existing connections
```

This made `auto_sync RunDelta` flaky in production (observed on gamertag `Madina97294`, log 2026-05). A trade-off comment in `main.go:197-208` documented the situation : *the sync may fail rather than blocking HTTP handlers*.

## Decision

Eliminate the trade-off by introducing a **SharedDBProvider** which owns the global handle and dynamically swaps RO ↔ RW around writer leases.

### Interface

```go
type Provider interface {
    Get(ctx context.Context) (*sql.DB, func(), error)
    AcquireWriter(ctx context.Context) (*WriterHandle, error)
    State() State
    Path() string
    Subscribe(fn Subscriber) func()
    Close() error
}
```

### State machine

```
StateRO ──AcquireWriter ok──► StateDraining ──readers=0──► StateRW
   ▲                                                          │
   │                                                       Release()
   │                                                          ▼
StateError ◄─reopen KO─ StateReopening ◄─close RW ok──────────┘
                  │ reopen ok
                  └──► StateRO
```

- `atomic.Int64` reader counter + `ready chan struct{}` (re-closed on each return to `StateRO`).
- Hot path `Get()` reads state via `atomic.LoadPointer` (zero-cost). Mutex taken only on transitions.

### Coordination with pool's ATTACH

A subtle DuckDB-Go quirk : when a player conn does `ATTACH 'shared.duckdb' AS shared`, the file handle is locked at driver level. A direct RW open (Provider's swap) fails with "Unique file handle conflict" *unless* the ATTACH is explicitly DETACHed first.

**Mechanism** : `Provider.Subscribe(fn)` emits events on every transition. The pool subscribes and :
- On `PreSwapToRW` : DETACH `shared` from all player conns synchronously (`pool_swap_hook.go::PrepareForSharedSwap`).
- On `RWToRO` / `ErrorToRO` : invalidates pool's idle conns so subsequent queries re-ATTACH lazily.

A POC (`poc_swap_diagnostic_test.go::S5`) confirmed that `Reopen()` does NOT release the ATTACH at driver level — only an explicit DETACH does. This was a critical finding that unblocked the architecture.

### Test seeds & root-level naming

Queries routed through `SharedReader.Get()` access a direct conn to `shared_matches_v2.duckdb` whose schema is `main`, not `shared`. The auto-attach propagation from `pool.attachShared` historically masked this (queries using `shared.X` prefix worked transparently on any conn in the process).

**Migration rule** : queries via SharedReader must reference tables at root level (no `shared.` prefix). Test seeds (`seedSharedDBSchema`) expose **both** the `shared.X` schema (for legacy paths via pdb.Player+ATTACH) and root-level views (for SharedReader paths).

### Metrics (expvar)

| Key | Type | Semantics |
|---|---|---|
| `shared_provider_state{state}` | Map gauge | 1 on current state |
| `shared_provider_swap_total{direction}` | Map counter | `ro_to_rw`, `rw_to_ro` |
| `shared_provider_swap_duration_ms_total{direction}` | Map int64 | sum (avg derived) |
| `shared_provider_swap_failures_total{reason}` | Map counter | `reopen_ro`, `acquire_writer`, `panic` |
| `shared_provider_get_wait_ms_total` | Int64 | sum of Get waits |
| `shared_provider_get_timeout_total` | Int64 | `ErrSwapTimeout` counter |
| `shared_provider_readers_in_use` | Int64 | live gauge |

All keys initialized to zero at boot (pattern from `dblease/metrics.go`).

## Migration scope (commits 8c–8k.13)

14 repos migrated to consume `SharedReadDB() SharedReader` :
- **Pure shared-only** : filters_repo, explorer_repo, fanout_repo, engagement_score_repo, citations_repo, medals_by_xuid_repo, weapon_kills_repo, highlight_events_repo, match_exclusion_repo (GetMatchRegistryInfo only), squad_repo (3 methods : LoadImpactEvents, LoadMainTeamParticipants, LoadSynthesisHeatmap), career_repo (Q26/Q27), compare_repo (5 methods), prestige_baseline_provider, catalog_repo.
- **Cross-DB split+merge** (3 round-trips + Go merge) : filters_repo::LoadMatchesForFilters, match_history_repo::LoadAll, player_matches_repo::Load, campaign_repo::CampaignSampleProvider, catalog_repo::playlistsPlayedByXUID.

**Remaining cross-DB queries on pdb.ReadDB() with ATTACH** (deferred to post-9 follow-up) :
- `squad_repo::LoadTopTeammates` (Q29 : pme ⨝ match_participants x2)
- `squad_repo::LoadSquadMatches` (Q30 : shared + subquery medals + LEFT JOIN pme)
- `squad_repo::LoadTeammateMatches` (Q31 : match_participants x2 + v_match_full)
- `squad_repo::LoadSynthesisMatches` (Q33b : shared + LEFT JOIN pme)
- `squad_repo::LoadMapStatsForSquad` (Q42 : shared + LEFT JOIN pme)

For these 5 queries, `attachShared` remains in place as an explicit, documented fallback — see the comment in `pool.go::attachShared`.

## Consequences

### Positive

- **Bug fixed** : `auto_sync RunDelta` no longer races against HTTP RO opens.
- **Predictable HTTP latency** : during a swap, `Get()` waits up to `readyTimeout` (default 30s) then returns `ErrSwapTimeout` → handler maps to 503 Retry-After.
- **Coverage of all DB locking corner cases** : `T9` (`pool_attach_swap_integration_test.go`) validates the DETACH/REATTACH chain end-to-end. `T5` burst validates 5 syncs × 20 HTTP readers in 2s.
- **Future-proof** : the Provider is per-path (Manager indexed by absolute path), prêt for multi-titre.

### Negative

- **Single point of contention** : during a swap, all HTTP readers wait. Acceptable because syncs take seconds, not minutes (delta sync).
- **Test seed complexity** : test fixtures double-write into pdb.Player AND pdb.Shared for shared.* tables. Aliased via `for _, db := range []*DB{pdb.Player, pdb.Shared}` pattern.
- **Migration scope** : 14 repos touched, ~700 lines of split+merge code added (player_matches_repo alone is ~200 lines). 5 squad_repo cross-DB methods still pending — they're functional but on the legacy ATTACH path.

### Mitigations

- Production rollout : flag `LEVELUP_USE_SHARED_PROVIDER` defaults `0` (legacy) at commits 1-8l. Enabled by default at commit 9. Compteur `shared_provider_swap_failures_total` monitored ; rollback by setting flag to `0` if needed.
- Metrics dashboard : Grafana `/debug/vars` integration documented in `apps/go-api/internal/observability/README.md`.

## Cross-references

- `apps/go-api/internal/platform/duckdb/sharedprovider/` — Provider implementation
- `apps/go-api/internal/platform/duckdb/pool.go::PrepareForSharedSwap` — DETACH hook
- `apps/go-api/internal/platform/duckdb/poc_swap_diagnostic_test.go` — Reopen vs DETACH POC
- `apps/go-api/internal/platform/duckdb/pool_attach_swap_integration_test.go` (T9)
- `apps/go-api/internal/platform/duckdb/pool_sync_burst_integration_test.go` (T5)
- `apps/go-api/internal/sync/engine.go::acquireSharedWriter` — sync side
- `.ai/thought_log.md` — commits 8a–8k.13 detailed history
