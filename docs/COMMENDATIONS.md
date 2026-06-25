# Commendations — Architecture & Guide

French version: [FR/CITATIONS.md](FR/CITATIONS.md)

LevelUp implements a **DuckDB-first commendations system** (called "citations" in the French docs and in the code). The whole pipeline is Go: a seed table of rules, a stateless compute engine, and a sync-time backfill.

> Stack note: the legacy Python implementation (`src/analysis/citations/`, `scripts/populate_citation_mappings.py`) has been removed. Everything below runs in the Go backend (`apps/go-api`).

## Architecture

### Tables

| Table | Database | Description |
|-------|----------|-------------|
| `citation_mappings` | `metadata.duckdb` | Rule registry (1 row = 1 commendation). Seeded by Go. |
| `match_citations` | player `stats.duckdb` | Computed delta per match × commendation (append-only). |

The rule registry is the single source of truth for *what* a commendation is; `match_citations` stores the *computed* per-match progression.

### `citation_mappings` schema

Definition lives in `internal/ops/seed.go` (`SeedCitationMappings`):

```sql
CREATE TABLE IF NOT EXISTS citation_mappings (
    citation_name_norm    VARCHAR PRIMARY KEY,
    citation_name_display VARCHAR NOT NULL,
    mapping_type          VARCHAR NOT NULL DEFAULT 'medal',
    medal_id              UBIGINT,
    medal_ids             VARCHAR,        -- CSV of medal_name_id
    stat_name             VARCHAR,        -- match_participants column, or weapon_kills:<NameEN>
    award_name            VARCHAR,        -- personal_score_awards.award_name
    award_category        VARCHAR,
    custom_function       VARCHAR,        -- name dispatched in citations_custom.go
    composite_children    VARCHAR,        -- JSON array of child norm-keys
    enabled               BOOLEAN NOT NULL DEFAULT TRUE,
    image_path            VARCHAR,
    category              VARCHAR,
    description           VARCHAR,
    tier_targets          VARCHAR,        -- CSV of milestone targets, e.g. 10,20,30,50,100
    subcategory           VARCHAR
);
```

The `enabled` column gates a commendation: the engine reads `WHERE enabled IS NOT FALSE`. The seed is **ART-safe** (SELECT-then-INSERT-or-UPDATE, no `ON CONFLICT`); no index on `medal_id`/`mapping_type` because the seed mutates them.

### `match_citations` (append-only)

Results are written append-only (DuckDB ART eradication, ADR 0026): each rewrite of a match allocates one generation via `match_citations_generation_seq` and the current state is read through the view `match_citations_latest` (MAX generation per `match_id`). There is **no** `DELETE` or `ON CONFLICT`.

```sql
INSERT INTO match_citations (match_id, citation_name_norm, value, generation_id) VALUES (?, ?, ?, ?)
```

A match with no active citation gets a single sentinel row `('_processed', 0, gen)` so it leaves the candidate pool — **but** only once its `highlight_events` are loaded (`match_registry.events_loaded`), so film-delayed events do not freeze a match at zero.

## Mapping types

| `mapping_type` | Source (per match) | Compute |
|----------------|--------------------|---------|
| `medal` | `shared.medals_earned` (`medal_name_id`, `count`) | `medal_id` count, or **sum** of counts over `medal_ids` (CSV). |
| `stat` | `shared.match_participants` column = `stat_name` | `int(value)`. |
| `pve_stat` | `shared_pve.pve_match_stats` (merged into stats) | same as `stat`, graceful if PvE absent. |
| `weapon_stat` | `v_weapon_kills` joined to `weapon_labels` | `stat_name` is `weapon_kills:<NameEN>`; per-weapon kill count. |
| `award` | player `personal_score_awards_latest` (`award_name`) | `SUM(award_count)` for the exact `award_name`. |
| `custom` | `domain.CitationContext` | dispatched by name in `citations_custom.go` (see below). |
| `composite` | — | not computed per match; aggregated as +1 per child mastered. |

Engine: `internal/analysis/citations_engine.go` (`ComputeFullMatchCitations`), 0 DB access — inputs are `domain.CitationContext`, output is `[]domain.CitationMatchDelta`.

### Progression semantics (R1–R8)

- **Leaf**: `delta = min(raw, max(tier_targets) − cumulPre)`. Zero once mastered. No `tier_targets` ⇒ `delta = raw`, uncapped.
- **Composite**: +1 per child that crosses its final tier during this match. No `tier_targets` ⇒ `max = len(children)`.
- **Meta** (composite of composites): same rule, applied in iterative passes.

Invariants V1–V4 are enforced by `internal/sync/citations_checks.go` after a full recompute:

- V1 — no `value ≤ 0` in `match_citations`.
- V2 — leaf cumulative ≤ `max(tier_targets)`.
- V3 — composite cumulative ≤ `effectiveMax(tier_targets, len(children))`.
- V4 — per-match composite value ≤ `len(children)`.

### Custom functions (`citations_custom.go`)

Registered for the Halo Infinite title via `init()` (`RegisterCustomDispatcher`) to avoid an import cycle. 12 functions, each `func(domain.CitationContext) int`:

| Function | Logic |
|----------|-------|
| `compute_bulldozer` | KDA > 8 in Slayer/Assassin, excludes Firefight/BTB. Returns 0 or 1. |
| `compute_wins_ctf` | Win (`outcome == OutcomeWin`) in CTF. |
| `compute_wins_slayer` | Win in Slayer/Assassin. |
| `compute_wins_strongholds` | Win in Strongholds/Bases. |
| `compute_wins_firefight` | Win in Firefight (registered; not bound to any enabled commendation). |
| `compute_annexion_forcee` | Scans `highlight_events`: streaks of 3 `mode` events without a player `death`. Fallback `awards["zone_captured"] / 3`. |
| `compute_flag_em_down` | Sum exact awards `runner_stopped`, `Flag Carrier Kill`, … |
| `compute_hijack` | Awards prefixed `hijacked_` + containing `hijack`/`skyjack`. |
| `compute_vandalism` | Awards prefixed `destroyed_` + containing `destroyed`/`destruction`. |
| `compute_wraith_destroyer` | Sum exact `destroyed_wraith`, `Wraith Destroyed`. |
| `compute_mongoose_destroyer` | Sum exact `destroyed_mongoose`, `Mongoose Destroyed`. |
| `compute_warthog_destroyer` | Sum exact warthog + rocket warthog destroyed awards. |

## Data flow

```text
Sync match → BackfillMatchCitations (internal/sync/citations.go)
              ├─ loadFullCitationMappings (metadata.citation_mappings)
              ├─ buildCitationContext (medals + stats + weapon_kills + awards + events)
              ├─ ComputeFullMatchCitations (analysis) → deltas
              └─ writeCitations → match_citations (new generation)
                                       ↓
                            match_citations_latest (read) → HTTP API → UI
```

## Seed the rules

The rule registry is seeded from `internal/ops/seed_citation_data.go` (`defaultCitationMappings`, the authoritative data table — see `COMMENDATIONS_REFERENCE.md`).

```bash
# Build, then seed citation_mappings into metadata.duckdb
levelup seed citation-mappings
```

The seed is idempotent: existing rows are UPDATEd (refreshing `image_path`, `tier_targets`, `description`, …), new ones INSERTed.

## Backfill the computed values

```bash
# Incremental backfill for one player (only un-processed matches)
levelup backfill --gamertag YourGamertag --citations

# Force recompute of already-processed matches
levelup backfill --gamertag YourGamertag --citations --force

# All players
levelup backfill --all --citations

# Full recompute + V1–V4 invariant checks
levelup backfill --gamertag YourGamertag --citations-recompute-all

# Composite-only (additive, no recompute from shared_matches)
levelup backfill --gamertag YourGamertag --composite-only
```

## Add a new commendation

1. Add a `CitationMapping` literal in `defaultCitationMappings()` (`internal/ops/seed_citation_data.go`):
   - `medal` → set `MedalID` (or `MedalIDs` CSV).
   - `stat` / `pve_stat` → set `StatName`.
   - `weapon_stat` → set `StatName: "weapon_kills:<NameEN>"`.
   - `award` → set `AwardName` (+ `AwardCategory`).
   - `custom` → set `CustomFunction` and implement + register it in `citations_custom.go`.
   - `composite` → set `CompositeChildren` (JSON array of child norm-keys).
   - Always set `Display`, `Category`, `Description`, `TierTargets`, `ImagePath`, `Enabled`.
2. Re-seed: `levelup seed citation-mappings`.
3. Recompute: `levelup backfill --all --citations-recompute-all`.
4. Enable/disable later by flipping `Enabled` in the seed (or `citation_mappings.enabled`) and re-seeding.

## FAQ

**How do I change an existing rule?** Edit the literal in `seed_citation_data.go`, re-run `levelup seed citation-mappings`, then `levelup backfill --citations-recompute-all`.

**Disk impact?** Roughly one row per active commendation (value > 0) per match, plus one sentinel for matches with no active citation. Old generations are retained (append-only) until compaction.

**How do I diagnose issues?** Run `levelup backfill --gamertag X --citations-recompute-all`: it reports any V1–V4 invariant violation.
