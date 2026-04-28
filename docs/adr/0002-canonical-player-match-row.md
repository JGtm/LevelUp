# ADR 0002 — Canonical types for cross-title data flow

**Status** — Accepted (2026-04-28). Implemented in Phase A→F of `PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`.

**Deciders** — Guillaume (GS), based on the audit `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` and the multi-title strategy locked at sprint 44.

## Context

The Go API was originally Halo Infinite-only. Services queried DuckDB with title-specific column names (`bs_score`, `kdr_halo_3`, `csr_value`, etc.) and returned title-specific structs. As we plan to support multiple titles (Halo 5, MCC, ODST, etc.), every page needed:

- Conditional code branching on `slug == "halo_infinite"` — anti-pattern.
- Title-specific DTO fields the front had to interpret per-title.
- Risk of leaking title-specific column names through 6+ layers (handler → service → repo → frontend).

Without a canonical layer:

- Adding a second title would mean rewriting every page service.
- Frontend would have to handle `n × m` combinations (n titles × m pages).
- Tests would have to be duplicated per title.

## Decision

**Introduce a canonical type system in `internal/games/canonical/`** that all services consume, regardless of the current title.

Core types (non-exhaustive):

| Canonical type | Replaces |
|---|---|
| `MatchSummary` | per-title list rows with title-specific scoring |
| `MatchDetail` | per-title detail page payloads |
| `MatchParticipant` | per-title scoreboard rows |
| `PlayerStats` | per-title aggregated stats |
| `PlayerIdentity` | per-title identifier shapes |
| `CareerSnapshot` | per-title rank progression |
| `EncounterRow` | per-title teammate/enemy summaries |
| `AssetReference` | per-title asset (mode/map/playlist) with localized labels |
| `Outcome` enum | per-title result codes (win/loss/tie/dnf/cancelled) |
| `MatchType` enum | Ranked / Social / Custom / Firefight |

`FieldKey` constants in `canonical/fields.go` are the canonical names referenced by the i18n manifests (`config/titles/{slug}/mappings/fields.toml` — cf. ADR 0003).

## Architecture

```
                 ┌──────────────────────┐
                 │   service/* (Go)     │  ← uses canonical.* exclusively
                 └──────────┬───────────┘
                            │ depends on
                 ┌──────────▼───────────┐
                 │  TitleDataAdapter     │  ← interface in internal/games/
                 │  Load*() → canonical  │
                 └──────────┬───────────┘
                            │ implemented by
        ┌───────────────────┴───────────────────┐
        │                                       │
┌───────▼────────────┐               ┌──────────▼─────────┐
│ halo_infinite/     │               │ synthetic_title_b/  │
│ adapter_data.go    │               │ adapter_data.go     │
│ (DuckDB Halo cols) │               │ (test fixture)      │
└────────────────────┘               └────────────────────┘
```

## Consequences

### Positive

- **Title-agnostic services** — `match_view_service.go`, `career_service.go`, `synthesis_service.go`, etc. consume only `canonical.*`. Adding a second title = implementing the adapter, no service change.
- **Capability-based degradation** — `data.LoadMatchSummaries(ctx, ids)` returns `games.ErrCapabilityNotSupported` when the title doesn't have the data. Service degrades gracefully (cf. `arch-rules` skill).
- **Frontend simplification** — TS types in `lib/api/types.ts` mirror canonical structs. Single contract per page.
- **Tests by layer** — `analysis/` tested with canonical inputs (no DB), `service/` tested with mock `port.Repository`, `platform/duckdb/` tested with `:memory:` per title.

### Negative

- **Initial migration cost** — every page service had to be refactored to consume the adapter. Spread across Phase 1 + 2 of meta plan (~24 j-h).
- **Discovery friction** — new contributors must know to look in `canonical/` first. Mitigated by skill `canonical-types` (`.claude/skills/canonical-types/SKILL.md`).
- **Pointer-heavy types** — many fields are `*int` / `*float64` / `*AssetReference` (nullable when title doesn't supply them). Accessor patterns repetitive.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **Keep title-specific types per page** | Doesn't scale to a 2nd title. Combinatorial explosion. |
| **Single mega-DTO with all possible fields** | Bloated, weak typing, no degradation signal. |
| **Generics per-title** | Go generics insufficient at the time of decision; would require runtime type assertions. |
| **GraphQL schema federation** | Massive infra change for a desktop-first app with no GraphQL stack. |

## References

- Implementation: `apps/go-api/internal/games/canonical/` (~750 lines).
- Adapter interfaces: `internal/games/adapter.go` (`TitleDataAdapter` + `TitleSemanticAdapter`).
- Skill: `.claude/skills/canonical-types/SKILL.md` (developer reference).
- Plan source: `.ai/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md` § 4 (canonical schema) + § 5 (adapter pattern).
