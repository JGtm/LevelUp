# ADR 0031 — Title data-source boundary and sync mutualization

**Status**: Accepted (2026-07-12)

**Branch**: `docs/adr-aggregates-title-boundary`

**Amends**: [ADR 0027](0027-sync-pipeline-v2-cycle-orchestrator.md) (sync pipeline V2 cycle orchestrator)

**Relates to**: [ADR 0012](0012-halo-only-adapters-extraction.md) (Halo-only adapters extraction, `internal/games/<slug>/`), [ADR 0025](0025-title-agnostic-minimal-viable-window.md) (Phases 1.5 / 1.6 prerequisites), [ADR 0008](0008-db-schema-multi-title-and-xuid-global.md) (FS path isolation)

---

## Context

Two title-specific data-source paths coexist today with no common seam, and the cost of a
third title is paid twice.

### Two orchestrations

- **Infinite** syncs through the V2 cycle orchestrator (ADR 0027): `AutoSyncScheduler`
  routes engine players (Infinite) to `v2.CycleOrchestrator` (`internal/sync/v2/`), which
  runs the 6-phase Discovery -> Dedup -> Fetch -> Persist -> Post-sync pipeline. ADR 0027 is
  still marked **Proposed**, but its outcome is in production: the `LEVELUP_SYNC_PIPELINE`
  flag and the V2->V1 fallback were removed (D1c, 2026-07-03), and V2 is the sole engine for
  Infinite players.
- **Halo 5** contours V2 entirely. It is a live-only title routed by `AutoSyncScheduler`
  through `syncPlayer` -> `internal/games/halo_5/livesync.Runner` (`RunDelta(ctx, opts)`).
  V2 is explicitly mono-title (Infinite) and does not route live-only titles (ADR 0027,
  D1c note).

There is no shared "title data source" interface between the two. Adding a third title means
choosing one of the two paths ad hoc, or writing a third.

### HTTP client duplication (re-measured on the current tree)

The Infinite HTTP client moved into a subpackage since ADR 0027 was written: it now lives at
`internal/sync/haloclient/` (2386 non-test lines total). Its retry/backoff/rate-limit core is
`halo_client_http.go` (219 lines): `doGet` (bounded retry loop), `backoff` (exponential +
`Retry-After` floor, ceiling), `rateWait` (`rate.Limiter`), `parseRetryAfter`. The constants
live in `halo_client.go`: `maxRetries = 4`, `retryBaseDelay = 800ms`, `backoffCeiling = 10s`.
`HTTPError` is declared at `halo_client.go:62`.

The Halo 5 client (`internal/games/halo_5/client.go`, 450 lines) re-implements the same
plumbing: `doGet` (lines 301-354), the binary variant `doGetBinary` (361-415, follows 302
redirects for PNG), `waitRetry` (419-431), `parseRetryAfterSeconds` (436-450), with constants
`h5MaxRetries = 4`, `h5RetryBaseDelay = 800ms`, `h5MaxBackoff = 10s` -- **identical values**
-- and a **second** `HTTPError` declaration (client.go:58), whose own comment states it is
"symetrique de sync.HTTPError, redeclare local pour eviter un couplage halo_5 -> sync". The
duplicated retry/backoff/rate-limit/error surface measures ~160 lines on the H5 side and
~140 lines of matching plumbing on the Infinite side.

This is **observed duplication, not speculative abstraction**: the two copies already exist
with identical constants, so the "rule of three occurrences" does not oppose extracting a
shared core (there are two live copies plus the pressure of a third title).

### A property to preserve

`internal/games/halo_5/client.go` imports **only** stdlib and `golang.org/x/time/rate` --
zero LevelUp import. That leaf cleanliness is deliberate (a title client should not depend on
the sync engine) and must survive any refactor. The Infinite client, by contrast, lives
inside the sync layer and imports `internal/domain`.

## Decision

Clarify the boundary **internally** (net packages, not a repository extraction) and mutualize
the shared plumbing without collapsing the two orchestrations into a speculative third one.

### D-1 — Per-title client packages + shared HTTP infra

Move `internal/sync/haloclient/` to `internal/games/halo_infinite/client/`, mirroring
`internal/games/halo_5/` and honoring the spirit of ADR 0012 (Halo-only code lives under
`internal/games/<slug>/`). The shared HTTP core goes into `internal/platform/httpx`
(infrastructure, not a title domain; the package does not exist yet). Written rule enforced by
an archlint import guard: **title clients import only stdlib, `golang.org/x/time/rate`, and
`internal/platform/httpx`**, and `httpx` is itself a leaf (stdlib + `x/time/rate` only). This
preserves the H5 zero-domain-import property and extends it to Infinite.

### D-2 — Minimal HTTP core (~150 lines)

`internal/platform/httpx` provides exactly: the bounded retry loop, exponential backoff with
`Retry-After` floor and ceiling, a `rate.Limiter`, and a unified `HTTPError` (replacing both
current declarations). Per-title auth is injected via a `RequestDecorator func(*http.Request)`
that each client supplies (Infinite: Spartan + clearance headers; H5: Spartan header +
`?auth=st` + `cpprestsdk` user-agent, no clearance). The H5 binary variant (`doGetBinary`,
redirect-following for PNG) stays a thin per-client wrapper over the same core. **No generic
options-bag client**: the core is the de-duplicated plumbing, nothing more.

### D-3 — `TitleSyncRunner` interface modeled on what exists

Introduce an interface shaped after the two current runners, not an aspirational one:

```
type TitleSyncRunner interface {
    TitleSlug() string
    RunCycle(ctx context.Context, players []Profile) (CycleReport, error)
    HandlesTitle(slug string) bool   // capability-based registration
}
```

`livesync.Runner` (H5) and the V2 orchestrator (Infinite) both become implementations behind
this seam; the scheduler dispatches by `HandlesTitle`.

**Rejected -- a fine source-level interface** (e.g. one that yields `canonical.MatchSummary`
per match). It would force the Infinite path to materialize a `canonical.MatchSummary` it
currently skips, which is a chantier of ADR 0027 (canonical mid-pipeline), not of this ADR.
`TitleSyncRunner` stays at the cycle granularity the two runners already expose.

### D-4 — Shared delta/watermark (`KnownSet`)

Extract a shared `KnownSet` / `KnownLoader` from the existing V2 implementation
(`internal/sync/v2/known_loader.go` -- union of `player_match_enrichment_latest` and
`shared.match_participants` for the player's xuid). H5 replaces its local `isKnown` predicate
(`internal/games/halo_5/livesync/{backfill,runner}.go`) with it. One concept -- "matches known
for `(title, player)`" -- instead of two.

### D-5 — Orchestration articulation (amends ADR 0027)

The target is **a single sync architecture parameterized by `titleSlug`**: V2 taking the
title as a parameter (that generalization is owned by ADR 0027, which this ADR amends).
`livesync.Runner` is a **transitional adapter** behind `TitleSyncRunner` (D-3), not a second
permanent architecture. **A third architecture is forbidden**: no new orchestrator, no
parallel pipeline. This ADR makes **no promise of an H5->V2 migration** -- whether and when
H5 folds into V2 is ADR 0027's decision, gated by its own phases. D-5 only fixes the seam
(`TitleSyncRunner`) so both paths present one face to the scheduler.

## Amends ADR 0027

ADR 0027 is marked *Proposed* (2026-05-25) but its core is in production (flag and fallback
removed, V2 sole Infinite engine). This ADR adds the multi-title dimension ADR 0027 deferred
(its "Multi-titre" section states the orchestrator takes `titleSlug` as a parameter but does
not define the cross-runner seam). We **propose transitioning ADR 0027 to "Accepted
(amended)"**, the amendment being: the V2 orchestrator and `livesync.Runner` unify behind the
`TitleSyncRunner` seam (D-3); the `titleSlug`-parameterized V2 is the target architecture;
`livesync.Runner` is transitional. This ADR does not itself edit ADR 0027's status -- that
acceptance is a human decision (the amendment is a long-term architecture commitment).

## Sequencing note

The pure move (D-1) and `httpx` extraction (D-2) should land **before Phase 1.6** (per-title
auth pool, ADR 0025 prerequisite): the `RequestDecorator` (D-2) is the natural attach point
for a per-`(titleSlug, gamertag)` token pool, so extracting it first gives Phase 1.6 a clean
hook. Multi-title V2 (the `titleSlug`-parameterized orchestrator of D-5) stays gated by
Phases 1.5 (per-title DDL) and 1.6 (auth pool key) per ADR 0025 -- this ADR does not unblock
them, it prepares the client/HTTP seam they will build on.

Recommended lot order (out of scope here, after acceptance): (1) D-1/D-2 pure move
(`git mv`, zero logic diff) + `httpx`, committed separately from any wiring; (2) D-3/D-4
interface + `KnownSet`; (3) `titleSlug`-parameterized V2 generalization, gated by ADR 0025
Phases 1.5/1.6.

## Multi-title register

MT-03 (world leaderboard) is already taken and done. This ADR creates **MT-27** -- "per-title
data-source interface (`TitleSyncRunner` + shared `httpx` + `KnownSet`)" -- in
`.ai/V7/PLAN_MULTITITRE_INDEX.md`, pointing to this ADR.

## Consequences

### Positive

- One shared HTTP core, one `HTTPError`, one delta concept -- ~150 duplicated lines removed
  from each client (D-2/D-4).
- Title clients become uniform leaves under `internal/games/<slug>/client/`, both with the
  zero-domain-import property (D-1).
- The scheduler sees one seam (`TitleSyncRunner`) instead of two ad-hoc paths (D-3); a third
  title implements the interface instead of forking an orchestration (D-5).

### Costs / risks

- D-1 is a package move touching import paths across the sync layer; must be a pure
  `git mv` + import rewrite with zero behavior change (golden fetch byte-identical), separate
  from any wiring commit.
- `httpx` becomes a new shared infra dependency of both title clients; kept a strict leaf by
  the archlint import guard (D-1).
- The Infinite `doGet` carries film/blob helpers (`downloadBlob`) that are Infinite-specific;
  only the retry/backoff/rate/error core moves to `httpx`, the rest stays in the Infinite
  client.

## References

- ADR 0027 -- sync pipeline V2 cycle orchestrator (amended by this ADR).
- ADR 0012 -- Halo-only adapters extraction (`internal/games/<slug>/` boundary).
- ADR 0025 -- title-agnostic MVW; Phases 1.5 (per-title DDL) and 1.6 (auth pool key) gates.
- ADR 0008 -- multi-title FS path isolation.
- `internal/sync/haloclient/halo_client_http.go`, `halo_client.go` -- Infinite client retry
  core + constants + `HTTPError` (the source of D-1/D-2).
- `internal/games/halo_5/client.go` -- H5 client, zero-domain-import leaf, duplicated retry
  plumbing + second `HTTPError` (lines 58, 301-450).
- `internal/sync/v2/known_loader.go` -- V2 `KnownLoader` implementation (source of D-4).
- `internal/games/halo_5/livesync/runner.go`, `backfill.go` -- H5 `isKnown` local predicate
  (replaced by D-4) and `livesync.Runner` (transitional adapter of D-5).
- `.ai/V7/PLAN_MULTITITRE_INDEX.md` -- MT register (MT-27 added).
