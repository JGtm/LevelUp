# ADR 0013 — Leased writer enforcement for DuckDB writes

**Date** : 2026-05-05
**Status** : Accepted
**Branch** : `refactor/leased-writer-enforcement` (commits 0-8)

## Context

The Go API runs a sync engine (Halo data ingestion) and HTTP handlers concurrently
in the same process. Both write to per-player DuckDB files (`stats.duckdb`) and
to shared DBs (`shared_social.duckdb`, `shared_matches_v2.duckdb`,
`metadata.duckdb`).

Until this refactor, only the sync engine acquired write leases via
`sync.AcquireLeaseCtx(ctx, path)` (a `sync.Mutex` keyed by file path). HTTP
handlers (Prestige, Notifications, Social, Media) wrote directly through their
repositories, bypassing the lease. This was a tribal-knowledge convention, not
materialized in the type system — and Prestige HTTP handlers added later did not
follow it.

**Concrete risk (P1)** : during a long sync (~30s for delta, several minutes for
full), a `POST /challenges` could:
- Hit a DuckDB-level file lock (process-external lock from another process or
  an open conn pool).
- Race against the sync engine's `match_participants` writes, producing
  inconsistent reads.
- Fail with HTTP 500 (driver-level "database is locked"), confusing users.

**P2** (notifications) and **P3** (media likes / match favorites): no atomic
guarantees between paired writes (`media_files.liked` vs `media_likes`).

## Decision

**Materialize the lease as a typed `*LeasedWriter` and route all HTTP writes
through it via Option pattern or wrappers — without modifying the existing
repository interfaces.**

### Architecture

```
HTTP write call
    │
    ▼
Service (Prestige / Notifications / Social / Media)
    │
    ├─ acquire *LeasedWriter via configured WriterAcquirer
    │   └─ Returns ErrDBLocked if mutex unavailable within timeout
    │       → Handler maps to HTTP 503 + Retry-After: 5
    │
    ├─ defer w.Release()
    │
    ▼
Repository (CRUD pure, signature unchanged)
    │
    └─ db.Exec(...) on the same *sql.DB whose path is mutex-protected
```

Two patterns coexist:

1. **Wrapper** (commit 2-3, Prestige): the API layer wraps `prestige.Service`
   with `LazyPrestigeService` which acquires the lease before delegating to the
   inner service. Justified for services with large API surfaces (16 methods).

2. **Option** (commits 4-6, Notifications/Social/Media): the service exposes
   `WithWriterAcquirer(...)` Option at construction. The acquirer is invoked
   internally before each write. Justified for services with small API
   surfaces — avoids passe-plat duplication.

Both patterns share the same underlying `dblease.leaseMutex(path)` map, so the
sync engine (still using legacy `sync.AcquireLeaseCtx`) coordinates
automatically with new HTTP handlers.

### Type system

- `port.DBExecutor` (Exec / Query / QueryRow) — satisfied by `*sql.DB` AND
  `*sql.Tx`. What repositories take in atomic write methods (commit 6).
- `port.DBWriter` (`DBExecutor + BeginTx`) — satisfied by `*sql.DB` and
  `*dblease.LeasedWriter`. What services manipulate when they need to open a
  transaction.
- `*sql.Tx` deliberately does **not** satisfy `DBWriter` (no nested
  transactions in `database/sql`). Verified by compile-time checks in
  `writer_test.go`.

### Atomicity (commit 6)

Media likes (P3) need `media_files.liked` and `media_likes` to stay consistent.
The service opens `tx := writer.BeginTx(ctx, nil)` then calls
`repo.SetMediaLikeAtomic(ctx, tx, ...)`. Repository executes both SQL
statements via the `port.DBExecutor` (which is the `*sql.Tx`) — rollback if
either fails. The repository's existing methods stay unchanged for
backwards compatibility.

### Observability

Metrics exposed via `expvar` on `/debug/vars`, **bounded cardinality** by
`Kind` (player / shared_matches / shared_social / metadata) — not by individual
file path (would explode in multi-user, cf. ADR-0009).

- `dblease_acquire_total{kind}` — successful acquisitions.
- `dblease_acquire_timeout_total{kind}` — `ErrDBLocked` returns.
- `dblease_wait_duration_ms_total{kind}` — cumulative wait time. Average via
  division by `acquire_total` in observability dashboard.
- `dblease_writers_in_use{kind}` — currently held writers (gauge).

## Alternatives considered

### A. Strict compile-time guard via signature changes

**Rejected** — modifying every repo write method to take `port.DBExecutor`
(e.g. `ChallengeRepo.Create(ctx, exec, c)`) would cascade through 5+ mocks and
146 Prestige tests, and would couple the metier package `prestige` to
`internal/port/`. The functional guarantee (mutex serialization) is achieved
without this cost via the wrapper / option patterns. The compile-time guard is
delivered later via the lint check in commit 8.

### B. Reentrant lease via context token

**Rejected** — would have allowed the sync hook (Prestige) to "see" the lease
already held by the sync engine and skip re-acquisition. But propagating tokens
through services is harder to debug than explicit signature propagation, and
`sync.Mutex` non-reentrancy stays a stronger invariant. We instead rely on the
explicit invariant: services don't acquire on `EvaluateForUser` (called by sync
hook) — documented in `internal/sync/lease.go` and `prestige_setup.go`.

### C. Single-writer goroutine (channel queue)

**Rejected** — would have eliminated the lease entirely by routing all writes
through one goroutine per DB. But loses synchronous error propagation for HTTP
handlers (a `POST /challenges` error would arrive asynchronously). Kept the
lease pattern instead.

## Consequences

### Positive

- P1 (Prestige), P2 (Notifications), P3 (Media atomicity) all resolved.
- HTTP handlers return `503 + Retry-After: 5` cleanly during sync, instead of
  500 or corrupted state.
- Existing tests preserved bit-for-bit (no signature changes on shared
  interfaces).
- Sync engine integration is automatic via shared mutex map (no changes to
  the sync engine's 11 `AcquireLeaseCtx` call sites).
- Observability: cardinality-bounded expvar metrics, leak detection helper
  (`dblease.AssertNoLeasedWriters(t)`).

### Negative / debt

- **Compile-time guard not strict** : a future developer could add a new
  service that writes via `db.Exec` without acquiring a lease. Mitigated by:
  (a) ADR documentation, (b) lint script that greps for `OpenReadWrite` in
  service/handlers (commit 8).
- **Sync engine still uses legacy `sync.AcquireLeaseCtx`** : the new
  `dblease.AcquireWriterCtx` API would feed `dblease_acquire_total{kind}`
  metrics, but migrating 11 call sites in `sync/engine.go`,
  `sync/backfill_weapons.go`, `sync/friends_recompute.go` risks breaking ~63
  sync tests. Deferred to a future commit. Functionally equivalent — the
  shared mutex map ensures coordination.
- **Atomic test cgo-only** : `Atomic_Success` and `Atomic_Rollback` tests for
  media likes need real DuckDB `:memory:` tx (cgo). Documented as integration
  tests pending.
- **Fairness not guaranteed** : `dblease.leaseMutex` uses `TryLock + sleep`
  polling, not FIFO. Under very high contention, starvation possible. Tracked
  in package docstring.

### Operational

- Restart sequence unchanged.
- New dependency on stdlib `expvar` (already in use, ADR-0009).
- Frontend: no changes needed — the new 503 with `Retry-After` is transparent
  to existing clients (browsers retry naturally).

## Implementation

8 sequential commits on `refactor/leased-writer-enforcement`:

| Commit | SHA | Scope |
|---|---|---|
| 0 | baseline | tests baseline (1662 tests) + plan v3 + check script |
| 1 | LeasedWriter | type + `port.DBExecutor`/`DBWriter` + metrics + leak helper |
| 2 | Prestige Player HTTP | P1 — Create/Update/Abandon/Arc/Pilot |
| 3 | Squad/PP shared_social | P1 sync hook differred + Squad |
| 4 | Notifications via Option | P2 |
| 5 | Match favorites via Option | shared_social write |
| 6 | Media likes atomique | P3 — transaction with `*sql.Tx` |
| 7 | Sync invariant + tests | coordination tests, no behavior change |
| 8 | ADR + lint check | this ADR + grep CI script |

41 new tests + 3 benchmarks added across the branch. 0 existing tests removed.

## References

- Plan complet : `.ai/V7/PLAN_DB_WRITE_CONCURRENCY.md` v3
- ADR-0009 (expvar monitoring multi-user) — cardinality bounded
- ADR-0005 (Prestige phased activation) — context for P1 urgency
- `internal/sync/lease.go` — invariant documentation
- [apps/go-api/scripts/check_lease_enforcement.sh](../../apps/go-api/scripts/check_lease_enforcement.sh) — CI lint script
