# Sync Guide — LevelUp

French version: [FR/SYNC_GUIDE.md](FR/SYNC_GUIDE.md)

> How LevelUp keeps your Halo matches in sync. The backend is Go (`apps/go-api`); the front is React/Vite (`apps/web`). Sync is now **automatic** — there is no `python scripts/sync.py` anymore.

## Overview

Sync runs **inside the Go server**. Two independent loops keep the data fresh, both backed by the same `SyncEngine` and the same token pool:

- **Presence watcher** (`internal/watcher`) — event-driven. A daemon tracks each configured player's Xbox/Steam presence (RTA WebSocket + REST pollers). When a player finishes a match, a delta sync is queued for that player only. Low latency, near-real-time.
- **Auto-sync scheduler** (`internal/scheduler/auto_sync.go`) — periodic. On a fixed interval it runs a delta sync for every player in `db_profiles.json`, catching anything the watcher missed.

Manual CLI commands (`levelup sync-delta` / `sync-full` / `backfill`) exist for bootstrap, gap-filling and local recomputes, but day-to-day operation needs no manual action.

## Data Architecture (V6)

Match data is centralized in per-title **shared** databases; per-player **enrichments** stay in the player's own DB. Layout is title-agnostic under `data/titles/{slug}/` (default slug `halo_infinite`).

```
Halo API (SPNKr-compatible client, Go)
        |
        v
SyncEngine (internal/sync) + token Pool (internal/platform/auth/pool)
        |
        +-- new match -> data/titles/{slug}/warehouse/shared_matches_v2.duckdb
        |     match_registry         (one row per unique match)
        |     match_participants     (all players, incl. MMR)
        |     highlight_events       (film events)
        |     medals_earned          (medals)
        |     killer_victim_pairs    (kill pairs)
        |     xuid_aliases           (xuid -> gamertag)
        |
        +-- PvE / Firefight -> data/titles/{slug}/warehouse/shared_pve.duckdb
        |
        +-- enrichment -> data/titles/{slug}/players/{gamertag}/stats.duckdb
              player_match_enrichment (performance_score, session_id, is_with_friends)
              personal_score_awards   (objective awards)
              match_skill_rank        (LUSR / CSR per match)
              sync_meta               (sync state)
```

Full schema and rationale: [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md).

## Automatic Sync

### Presence watcher

Started by the server at boot (`internal/watcher` daemon). For each player it runs a presence FSM (RTA WebSocket, Steam/REST fallback) and, on match completion, enqueues a coordinated delta sync. Auth is delegated to the shared token pool. No configuration beyond having the player declared in `db_profiles.json` with a valid token (see Auth below).

### Auto-sync scheduler

`AutoSyncScheduler` reads `app_settings.json` at boot and on each tick. Relevant keys:

| Key (`app_settings.json`) | Meaning |
|---|---|
| `spnkr_auto_sync_enabled` | Master on/off switch. Must be `true` for the scheduler to act. |
| `spnkr_auto_sync_interval_hours` | Interval in hours (default 6 if unset). |
| `spnkr_auto_sync_interval_minutes` | Interval in minutes (takes precedence when set). |

Each cycle, for every player in `db_profiles.json`:
1. Skip if the player has no entry in the token pool, or if the watcher already has an active session for them.
2. Build a `PooledHaloClient` pinned to that player.
3. Run `SyncEngine.RunDelta` (parallel internal fetches). Repeated zero-insert cycles raise a warning (incident guard: 14 days of silent zero inserts in mai 2026).

Diagnostics are exposed via the admin endpoint `/api/v1/_diag/auto-sync/snapshot`.

## Sync Pipeline V2

The per-player engine (`RunDelta`/`RunFull`) is the default (V1). An opt-in **V2 cycle orchestrator** (`internal/sync/v2`) processes *all* players per cycle in 6 phases, removing the shared-writer serialization and guaranteeing correct cross-player dedup:

1. **Discovery** — parallel per player, read-only: load known IDs + paginate the API.
2. **Dedup** — single: union of unknown match IDs across players.
3. **FetchShared** — bounded errgroup: `GetMatchStats` per unique match.
4. **FetchPlayer** — parallel per player: awards/scores needing the player's own token.
5. **Persist** — single writer: one mega-batch (shared + player) in one transaction.
6. **PostSync** — parallel per player: heals, films, citations, etc.

V2 is the sole engine driver of the auto-sync cycle since the V1 pipeline was removed (2026-07). Engine titles (Infinite) go through the orchestrator; live-only titles (Halo 5) go through `syncPlayer`→`liveRunner`. If the orchestrator is not wired at boot (missing prereqs), the cycle falls back to a structural `syncPlayer` safety net. V2 shares the Persisters, schema and WAL.

## Delta vs Full

- **Delta** — fetches only matches newer than the last sync watermark. Fast, the default for both the watcher and the scheduler.
- **Full** — walks the last N API matches and inserts any that are missing (gap-filling). Use after a long outage, an import, or a watermark issue.

For each synced match the engine always pulls the full payload: stats, medals, personal scores, performance score, highlight events, per-match skill/MMR, and xuid -> gamertag aliases.

## Manual CLI

Build/run the `levelup` CLI from `apps/go-api/cmd/levelup` (requires the CGO toolchain for the DuckDB driver — see [testing.md](testing.md)). `LEVELUP_REPO_ROOT` is auto-detected if unset.

### Delta / Full sync

```bash
# Delta for one player
levelup sync-delta --gamertag YourGamertag [--max-matches 25] [--match-type matchmaking] [--rps 1]

# Delta for all configured players (uses the token pool)
levelup sync-delta --all [--max-matches 25] [--token-pool-size 0]

# Full (gap-fill) for one player or all
levelup sync-full --gamertag YourGamertag [--max-matches 150] [--match-type matchmaking] [--rps 1]
levelup sync-full --all [--token-pool-size 0]
```

| Flag | Applies to | Default | Notes |
|---|---|---|---|
| `--gamertag` | sync-delta, sync-full | — | Mutually exclusive with `--all`. |
| `--all` | sync-delta, sync-full | — | All players in `db_profiles.json` via the pool. |
| `--max-matches` | sync-delta / sync-full | 25 / 150 | Delta: max new matches inserted. Full: API matches walked. |
| `--match-type` | both | `matchmaking` | `all` \| `matchmaking` \| `custom` \| `local`. |
| `--rps` | both | 1 | Max API requests per second. |
| `--token-pool-size` | `--all` only | 0 | 0 = auto (all discovered sources), `MaxSize` of the pool. |

### Backfill (local recomputes & API backfills)

```bash
levelup backfill (--gamertag X | --all) <selector...> [--force] [--dry-run]
```

Selectors (one or more required):

| Selector | Needs Halo API | Description |
|---|---|---|
| `--engagement-scores` | No | Backfill engagement score. |
| `--citations` | No | Recompute `match_citations` from mappings + medals + stats + awards. |
| `--citations-recompute-all` | No | Full recompute (force) + invariant checks V1-V4. |
| `--composite-only` | No | Composite citations only (additive). |
| `--lusr` | No | Recompute LUSR (TrueSkill 2 + medal weights). `--dry-run` previews per playlist_group. |
| `--perf` | No | Recompute relative performance score (v5). |
| `--assists-model` | No | Per-mode OLS expected_assists model. |
| `--csr` | Yes | Per-match CSR via `GetMatchSkill` (RankRecap), idempotent. |
| `--shared-csr` | Yes (no API with `--dry-run`) | CSR of all participants of ranked matches into `shared.match_csrs`. |
| `--weapons` | Yes | `weapon_kills` from the film CDN. |
| `--compare-formulas` | No | Simulate 5 LUSR formula variants on `--last-n` matches (default 20). |

`--force` reprocesses already-persisted data. `--dry-run` is only valid with `--shared-csr` or `--lusr`. API-backed selectors refresh the player's Halo tokens via the OAuth refresh token (see Auth). LUSR recompute uses the v2 canonical path; v1 is dead.

Backfill is also exposed over HTTP (`POST /backfill/start`); the CLI is the local, server-free path.

## Auth

Tokens are the single source described in [adr/0023-auth-tokens-single-source.md](adr/0023-auth-tokens-single-source.md): `data/auth/watcher_tokens/{xuid}.json` via `MultiUserTokenStore`. The player must be declared in `db_profiles.json` (with `xuid`) first.

- Normal onboarding: Xbox SSO web flow -> `/auth/xbox/callback` persists the refresh token.
- Advanced onboarding: `go run ./apps/go-api/cmd/token-capture/ <Gamertag>` (device-code) or `go run ./apps/go-api/cmd/token-import/ <Gamertag>` (RT on stdin) writes directly to the store — no `.env.local` editing.

The `--all` sync paths and API-backed backfills resolve tokens through the pool (Discovery -> Resolver -> Pool), whose Discovery scans the token store (`data/auth/watcher_tokens/{xuid}.json`); the pool handles the OAuth refresh, persists the rotated refresh token back to that store, and caches Spartan tokens (~3h30). Single-player `levelup sync-delta/sync-full --gamertag` and `--csr/--weapons` backfills read the same store directly. Since ADR 0023 Phase 5 (2026-08-25) that store is the ONLY source: no env var, no `sync_meta`. Never re-capture tokens to fix a 401: a green sync means the tokens are good.

## Append-only / ART-safe writes

All per-match writes go through the Collect -> Persist architecture (one INSERT-only batch per cycle), and the critical state tables are append-only. This eradicates the DuckDB ART index corruption bug by construction. Do not reintroduce concurrent `UPDATE` / `INSERT ... ON CONFLICT DO UPDATE` on the shared/state tables. References:

- [adr/0019-collect-persist-architecture.md](adr/0019-collect-persist-architecture.md)
- [adr/0026-append-only-art-eradication.md](adr/0026-append-only-art-eradication.md)

## Ops Runbook (DuckDB cross-process lock)

DuckDB does not share an OS file-lock across processes. Running a CLI tool that opens a **shared** DB (metadata, shared_matches_v2, shared_pve, shared_social) in RW while the server holds its handle will fail with an `IO Error: Cannot open file ... used by another process`.

Rule: do not run `levelup sync-* / backfill` (or other shared-DB CLI tools) against shared databases while the server (`apps/go-api/server.exe` or `air`) is running. Stop the server first for cross-process shared writes. Full procedure and tool inventory: [RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md](RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md).

## Troubleshooting

| Symptom | Action |
|---|---|
| Auto-sync not running | Check `spnkr_auto_sync_enabled: true` in `app_settings.json`; inspect `/api/v1/_diag/auto-sync/snapshot`. |
| Player skipped (`not_in_pool`) | No token discovered for that player — onboard via SSO or `token-capture`/`token-import`. |
| Repeated zero inserts | Watch the scheduler warning; verify the `/matches` call uses `xuid(NNN)` not the raw gamertag, and the watermark is sane. |
| 401 on API backfill | Tokens stale in cache; do **not** re-capture. Let the pool refresh. See [adr/0023](adr/0023-auth-tokens-single-source.md). |
| `Cannot open file ... used by another process` | Cross-process DuckDB lock — stop the server before running the CLI (see Ops Runbook). |
