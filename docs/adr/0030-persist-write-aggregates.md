# ADR 0030 — Persist write aggregates: compile-time enforcement of the anti-ART boundary

**Status**: Accepted (2026-07-12)

**Branch**: `docs/adr-aggregates-title-boundary`

**Completes / hardens**: [ADR 0019](0019-collect-persist-architecture.md) (Collect -> Persist), [ADR 0026](0026-append-only-art-eradication.md) (append-only + `_latest` views)

**Relates to**: [ADR 0013](0013-leased-writer-enforcement.md) (leased writer), [ADR 0025](0025-title-agnostic-minimal-viable-window.md) D-MV2 (canonical-typed repositories)

---

## Context

The DuckDB ART index bug (#23046, `Failed to delete all rows from index`) FATAL-invalidated
production databases (incident 2026-05-24 on `match_skill_rank`). Two ADRs eradicated it:

- **ADR 0019** moved per-match writes onto a Collect -> Persist path: `persist.BatchBuilder`
  assembles a `MatchBatch`, `BatchQueue.Submit` durably WAL-logs it, and INSERT-only
  `Persister`s commit it. No concurrent `UPDATE`/`UPSERT` on the critical tables.
- **ADR 0026** made the state tables append-only (technical PK + clock column, INSERT-only
  writes, reads through a `<table>_latest` view), so the ART delete-from-index path is
  unreachable by construction.

These invariants hold **at runtime**, but nothing enforces them **at compile time**. Three
concrete leak vectors remain, verified on the current tree:

1. **Batches are mutable after `Submit()`.** `MatchBatch`, `SharedBatch`, `PlayerBatch`
   (`internal/persist/batch.go`) are plain structs with **exported** fields. Once
   `builder.Build()` (`internal/persist/builder.go:185`) hands a `*MatchBatch` to
   `BatchQueue.Submit` (`internal/persist/queue.go:167`), the caller still holds the
   pointer and can mutate the aggregate that the worker is persisting. The builder itself
   is reusable (chained setters, no invalidation), so a batch can be built, submitted, then
   silently altered or resubmitted.

2. **Raw `OpenReadWrite` bypass.** `duckdb.OpenReadWrite` (`internal/platform/duckdb/db.go:286`)
   returns a read-write handle to any DB file, outside the lease/persist path. It is called
   from ~25 non-test sites today. Most are legitimate (the persister itself at
   `internal/persist/combined_persister.go:125`, the provider writer, the pool, sync schema
   bootstrap), but the surface is open: a new service can open a critical DB read-write and
   run an `UPDATE`/`ON CONFLICT` with no guard rail catching it.

3. **Direct post-sync writes.** Several sync-layer files write critical DBs directly,
   outside the batch path: `internal/sync/writes.go` (`UPDATE match_registry`,
   `INSERT player_match_enrichment`, ...), `internal/sync/career.go`
   (`INSERT player_csr_snapshots`, an append-only table), `internal/sync/engagement.go` and
   `internal/sync/performance.go` (`UPDATE match_registry` and multi-row enrichment updates).
   These are guarded today only by the file-level greps of
   `internal/sync/no_art_patterns_test.go` and the statement-level scan of
   `internal/sync/append_only_state_guard_test.go`.

Additionally, **there is no guard rail on the read side of append-only tables.** ADR 0026
mandates reading through `<table>_latest`, but nothing fails a build when a query does
`FROM match_skill_rank` (raw) instead of `FROM match_skill_rank_latest`. A raw read silently
serves stale rows (documented trap, ADR 0026 "Pieges"). This is the one invariant with no
net at all.

The ART bug is not theoretical and not volume-gated: a single concurrent `ON CONFLICT DO
UPDATE` suffices (ADR 0019). A regression here is a production outage, so the runtime
invariant needs a compile-time / ratchet backstop.

## Decision

Make the persist write-aggregate a **closed, single-package construction** and add the
missing read-side guard rail. Enforcement lives entirely **inside `internal/persist`** plus
two ratchet tests; no new package, no signature cascade through business repositories.

### D-1 — Opaque batch types constructed only via `BatchBuilder`

Privatize the fields of `MatchBatch`, `SharedBatch`, `PlayerBatch` inside
`internal/persist`. The aggregate becomes opaque to callers: it can only be built by
`BatchBuilder` (which lives in the same package and can populate unexported fields), and
callers outside `persist` cannot construct or mutate one. The persister reads the fields
directly (same package). No new package is introduced.

**Serialization constraint (must be handled in the implementing lot).** The batch is
JSON-serialized to the WAL for crash durability (ADR 0019). Go's `encoding/json` does not
marshal unexported fields, so privatizing them requires the package to own serialization
explicitly -- either a package-private serialization DTO (exported mirror used only by the
WAL codec) or custom `MarshalJSON`/`UnmarshalJSON` on the opaque types. This is an
implementation detail, not an open decision: the WAL format stays byte-compatible and is
covered by `internal/persist/batch_roundtrip_test.go`.

### D-2 — Pilot on the PlayerEnrichment family first

Generalize incrementally. Start with the **PlayerEnrichment** family
(`internal/persist/post_sync_enrichment_persister.go`), because it is the smallest surface,
it is exactly where the real bypasses live (`BatchUpdateColumn`/`BatchUpdateMulti` INSERT a
partial row tagged with the owning `stage`), and its merge-on-read `stage` invariant is the
subtlest to get right. `SharedMatch` (the full `SharedBatch`) is a later generalization; its
cross-DB lease sequencing is owned by ADR 0027, not this ADR.

### D-3 — Dated allowlist + ratchet test for `OpenReadWrite`

Guard `duckdb.OpenReadWrite` with a **dated allowlist plus a ratchet test**, reusing the
pattern already institutionalized by `internal/archlint/no_raw_outcome_literal_test.go`
(decreasing allowlist) and the `TODO(expiry:YYYY-MM-DD)` lint of
`internal/archlint/todo_expiry_test.go`. The initial allowlist is the **current set of call
sites**, each entry dated; **new entries are forbidden** without a dated justification, on the
same model as the (now empty) allowlists in `internal/sync/no_art_patterns_test.go`. The goal
is not to remove the legitimate call sites (persister, provider, pool, schema bootstrap) but
to freeze the surface so no new raw read-write handle to a critical DB appears unnoticed.

### D-4 — Read-side guard rail for `_latest` views

Add a ratchet test forbidding `FROM <append_only_table>` (raw) outside `internal/persist` and
the view definitions, using the **same harness** as `internal/sync/no_art_patterns_test.go`
and `internal/sync/append_only_state_guard_test.go` (source grep / AST-lite, comment-stripped,
migrations/cmd/scripts excluded). The set of protected tables reuses `tablesProtegees` from
`no_art_patterns_test.go` (single source of truth).

**Option rejected -- database-level revoke.** Enforcing "readers must use the view" via a
revoke on the base table is not available: embedded DuckDB has no multi-role ACL to revoke a
`SELECT` from one code path while granting it to the persister in the same process. A
source-level ratchet is the only enforceable mechanism here.

The typed read helper that this guard steers callers toward is the role of ADR 0025 D-MV2
(canonical-typed repositories reading `_latest` views); this ADR references it and does not
duplicate it.

### D-5 — Post-`Submit()` immutability via ownership transfer

Make construction one-shot: `Build()` (or `Submit()`) transfers ownership of the aggregate
and invalidates the builder, so a builder cannot be reused and a submitted batch cannot be
mutated. Combined with D-1 (opaque fields), the mutate-after-submit error becomes
**impossible to express** in caller code, at zero memory cost (no defensive copy needed --
the caller simply has no handle to mutate).

## Positioning versus ADR 0013 (no signature cascade)

ADR 0013 explicitly **rejected** a "strict compile-time guard via signature changes"
(alternative A): threading `port.DBExecutor` through every repository write method would
cascade through 5+ mocks and 146 Prestige tests and would couple the `prestige` metier
package to `internal/port/`. That rejection stands.

ADR 0030 does **not** reintroduce that cascade. It encapsulates batch construction **inside
a single package** (`internal/persist`): the compile-time guarantee comes from Go's package
visibility (unexported fields, same-package builder), not from changing the signatures of
business repositories. No business package gains a new dependency; the `port` interfaces are
untouched. The two ADRs are complementary: 0013 serializes writers via the lease at runtime;
0030 closes the construction of the persist aggregate at compile time. Neither imposes the
signature cost 0013 declined.

## Consequences

### Positive

- The mutate-after-submit and reuse-builder classes of bug become unrepresentable (D-1, D-5).
- The raw-read-write surface is frozen and dated; new leaks fail CI (D-3).
- The last unguarded invariant -- raw reads of append-only tables -- gets a net (D-4).
- Zero cost to the runtime path and to business-layer signatures (positioning vs 0013).

### Costs / risks

- D-1 forces the WAL serialization to be owned explicitly inside `persist` (DTO or custom
  marshaler). Contained, covered by the existing round-trip test.
- D-4 is a source-grep ratchet, not a semantic proof: it is file/statement-level like its
  siblings and shares their known limits (comment-stripping, no full AST). Acceptable -- it
  is a backstop for the code-review, not a replacement.
- Incremental rollout (D-2) means SharedMatch stays partially open until its dedicated lot.

## Execution sketch (future lots, out of scope of this ADR)

This ADR records the decisions; it changes no code. Recommended sequencing, to be planned
**after** the ADRs are accepted:

1. **PlayerEnrichment pilot (D-1/D-2/D-5)** as a dedicated lot **after the current audit lot
   closes**, to avoid churn on `builder.go` during the audit; land with a byte-identical
   golden on the WAL format.
2. **`OpenReadWrite` allowlist + ratchet (D-3)** seeded from the current call sites, each
   dated.
3. **`_latest` read guard (D-4)** reusing the `no_art_patterns_test.go` harness and
   `tablesProtegees`.
4. **SharedMatch generalization of D-1** last (its cross-DB lease sequencing is ADR 0027).

## References

- ADR 0019 -- Collect -> Persist (Persisters, BatchQueue, WAL).
- ADR 0026 -- append-only ART eradication + `_latest` views.
- ADR 0013 -- leased writer enforcement (alternative A, rejected signature cascade).
- ADR 0025 D-MV2 -- canonical-typed repositories (typed `_latest` read helper).
- `internal/persist/doc.go` -- "Hors scope MatchBatch" section (the aggregate boundary:
  which writes belong to a batch and which stay ad-hoc post-sync).
- `internal/persist/batch.go`, `internal/persist/builder.go` -- current batch types and
  builder (exported fields, reusable builder -- the state D-1/D-5 close).
- `internal/persist/post_sync_enrichment_persister.go` -- D-2 pilot surface (stage
  merge-on-read).
- `internal/sync/no_art_patterns_test.go`, `internal/sync/append_only_state_guard_test.go`
  -- existing anti-ART ratchet harness reused by D-3/D-4.
- `internal/archlint/no_raw_outcome_literal_test.go`, `internal/archlint/todo_expiry_test.go`
  -- dated-allowlist / dated-debt ratchet pattern reused by D-3.
- `internal/platform/duckdb/db.go` (`OpenReadWrite`) -- the guarded entry point (D-3).
