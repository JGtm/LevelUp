# Weapon Kill Attribution — Technical Reference

This document describes how LevelUp attributes each kill in a match to the
weapon that caused it. The attribution is derived **offline** from the match
film (theater) chunks fetched from the Halo film CDN — the Halo stats APIs do
not expose a per-kill weapon. The whole pipeline is implemented in Go under
`apps/go-api`.

> Scope note: per-kill weapon attribution is **best-effort, not authoritative**.
> It reconstructs the weapon from film replication data and reconciles it
> against the API kill totals. Read [Confidence Levels](#confidence-levels) and
> [Known Limitations](#known-limitations) before relying on it.

---

## Table of Contents

1. [Overview](#overview)
2. [Code Map](#code-map)
3. [Pipeline](#pipeline)
4. [Attribution Paths](#attribution-paths)
5. [Confidence Levels](#confidence-levels)
6. [Weapon ID (WID) Structure](#weapon-id-wid-structure)
7. [Storage — `weapon_kills` (append-only)](#storage--weapon_kills-append-only)
8. [Read Path — `v_weapon_kills` and labels](#read-path--v_weapon_kills-and-labels)
9. [Running the Backfill](#running-the-backfill)
10. [Adding / Resolving a Weapon ID](#adding--resolving-a-weapon-id)
11. [Known Limitations](#known-limitations)
12. [Deep-Dive: Film Reverse-Engineering](#deep-dive-film-reverse-engineering)

---

## Overview

For a given match, the engine downloads the film, scans the binary replication
chunks for weapon-fire and weapon-held events, correlates each kill (from
`highlight_events`) to a weapon, reconciles the result against the API kill
counts, and writes one row per kill into the shared `weapon_kills` table.

The aggregated result (kills per weapon, with EN/FR labels) is read back
through the `v_weapon_kills` view and the `metadata.weapon_labels` table, and
surfaced in the match view and the favorite-weapon home KPI.

---

## Code Map

| Concern | Location |
|---------|----------|
| Pipeline orchestration (per match / all participants / batch) | `internal/sync/backfill_weapons.go` |
| DB write (append-only INSERT) | `internal/sync/writes.go` — `InsertWeaponKills`, `MarkWeaponKillsDone` |
| Chunk scan (fire events, held-weapon timeline) | `internal/analysis/weapon_scanner.go`, `internal/analysis/weapon_parser.go` |
| Weapon ID map, timings, fusions, sentinels | `internal/analysis/weapon_data.go` |
| Kill -> weapon correlation | `internal/analysis/weapon_correlation.go` |
| API reconciliation | `internal/analysis/weapon_reconciliation.go` |
| Attribution result struct | `internal/analysis/kill_attribution.go` |
| Aggregated read repository | `internal/platform/duckdb/weapon_kills_repo.go` |
| Label / role resolution (metadata) | `internal/platform/duckdb/weapon_resolver.go` |
| Display-name source (per title, keyed by `weapon_key`) | `config/titles/{slug}/mappings/weapon_names.toml` + loader `internal/games/mappings/loader_weapon_names.go` + boot seed `internal/games/halo_infinite/migrations/weapon_name_labels.go` (`ReconcileWeaponNameLabels`) |
| Schema (table + view) | `internal/games/halo_infinite/migrations/steps_shared_core.go` (`add_weapon_kills`, `add_weapon_kills_reconciled_as`) |
| Append-only conversion | `internal/migration/steps_shared_append_only_weapon_kills.go` |
| CLI backfill | `apps/go-api/cmd/levelup` — `backfill --weapons` |
| Label seeding | `apps/go-api/cmd/seed-weapon-labels` |

---

## Pipeline

Entry point: `sync.BackfillWeaponKillsForMatchAll(ctx, client, sharedDB, matchID)`
in `internal/sync/backfill_weapons.go` (the `...ForMatch` variant runs a single
player; `...ForMatches` is the lease-acquiring batch method on `SyncEngine`).

Steps:

1. **Download film** — `client.GetMatchFilm(ctx, matchID)` returns the binary
   chunks. If the film is gone (404/410), the match is marked
   `weapon_kills_no_film` and skipped (films expire for older matches).
2. **Build held-weapon timelines** — `analysis.BuildWeaponTimelines(chunks)`
   produces, per chunk and per player index, the weapon bytes held at that
   chunk's snapshot moment (`Timeline`), plus a per-chunk swap-detection set
   (`SwapPIs`) and the chunk time intervals (`Timing`).
3. **Scan fire events** — `analysis.ScanFireEventsAll(...)` over every chunk
   collects all players' weapon-fire events with estimated timestamps.
4. **Load kills** — `getAllKillsForMatch` reads `highlight_events`
   (`event_type LIKE '%kill%'`, with melee/grenade flags) from the shared DB,
   plus a `xuid -> player_index` map (`getXuidToPI`, ordered by `team_id, rank`).
5. **Correlate** — `analysis.CorrelateKillsGlobal(...)` attributes each kill to
   a weapon (see [Attribution Paths](#attribution-paths)).
6. **Reconcile** — `analysis.ReconcileAPIAggregates` adjusts attributions and
   confidence against the API kill totals from `match_participants`.
7. **Write** — `InsertWeaponKills` writes one append-only generation of rows per
   `(match_id, xuid)`; `MarkWeaponKillsDone` sets the registry bit **only if at
   least one row was inserted** (a guard added after a 2026-05 incident where
   marking the bit on a 0-row extraction silently emptied ~1010 matches).

Writes are serialized by the shared write lease + `MaxOpenConns(1)`; correlation
and film download are parallelized network-only
(`weaponBackfillParallelism = 24`).

---

## Attribution Paths

Each kill gets an `attribution_path` (constants in
`internal/analysis/kill_attribution.go`):

| Path | Scope | Source | Notes |
|------|-------|--------|-------|
| `fire_event` | Per-player, timing-based | Weapon-fire events scanned from chunks, matched to the closest preceding fire event within the weapon's timing window | Highest precision when a fire event is found |
| `formula_a` | Per-player, snapshot-based | The weapon *held* by the player (`Timeline`) at the chunk covering the kill time, falling back to the previous chunk | Coarser (chunk granularity); demoted to `medium` if a swap was detected in that chunk |
| `none` | Fallback | No fire event and no usable held-weapon snapshot | Stored with `confidence = none` |

Melee and grenade kills are flagged from the `highlight_events` event type and
are **not** weapon-ID attributed via the film; their per-match counts come from
the `melee_kills` / `grenade_kills` columns on `match_participants`
(see [Read Path](#read-path--v_weapon_kills-and-labels)).

---

## Confidence Levels

Constants in `internal/analysis/weapon_correlation.go`
(`confidenceHigh/Medium/Low/None`). `ComputeConfidence(weaponID, deltaMS)` uses
the weapon's timing window (`GetTiming`, from `weapon_data.go`):

| Value | Meaning |
|-------|---------|
| `high` | Fire event within the weapon's `SwapMS`, or a confirmed snapshot reconciled against the API total |
| `medium` | Matched but with swap/timing ambiguity (delta within `TravelMax`) |
| `low` | Weak match (delta beyond `TravelMax`) |
| `none` | Kill could not be attributed |

`ReconcileAPIAggregates` (`weapon_reconciliation.go`) compares the attributed
totals against `match_participants.kills` and promotes/demotes confidence and
weapon IDs so the per-weapon counts stay consistent with the authoritative API
totals.

---

## Weapon ID (WID) Structure

A WID is the 8 filmshell weapon bytes read as a **big-endian `uint64`**
(`hexToUint64` in `internal/analysis/weapon_data.go`). DuckDB stores it as
`UBIGINT` — some real WIDs (e.g. `f408190f42c9679f`) have bit 63 set and exceed
`2^63`, which is why the write path casts a decimal string to `UBIGINT` rather
than binding a Go `uint64` (the duckdb-go driver rejects high-bit-set uint64s).

Structure of the 8 bytes:

- **Bytes 1-4 (high 32 bits): the weapon identity** — unique per weapon
  type/variant.
- **Bytes 5-8 (low 32 bits): a family/variant suffix.** The common suffix
  `42c9679f` (`CommonWeaponSuffix` in `weapon_data.go`) covers most standard
  weapons; special families share their high bytes but differ in the suffix:

| Family | High bytes (identity) | Behaviour |
|--------|------------------------|-----------|
| Energy Sword | `4ff3937e` | Same identity, suffix differs per skin (Duelist, Bloodblade, Infected) |
| Gravity Hammer | `841ac5e5` | Same identity, suffix differs per variant (Diminisher of Hope, Rushdown) |

Cosmetic variants are folded onto their canonical weapon via `WeaponFusionMap`
/ `WeaponFusionMapID`. Sentinel IDs `0` (grenade), `1` (melee), `2` (vehicle)
are reserved and excluded from weapon aggregation.

The authoritative WID list (confirmed hex -> name) lives in `weaponEntries`
inside `weapon_data.go`, and is mirrored as research notes in
`.ai/REFERENCE_WEAPON_IDS.md`.

---

## Storage — `weapon_kills` (append-only)

Table `weapon_kills` lives in the shared DB
(`data/warehouse/shared_matches_v2.duckdb`). Base columns
(`steps_shared_core.go`, migration `add_weapon_kills` +
`add_weapon_kills_reconciled_as`):

| Column | Type | Description |
|--------|------|-------------|
| `match_id` | VARCHAR | Halo match UUID |
| `xuid` | VARCHAR | Killer XUID |
| `time_ms` | INTEGER | Kill timestamp (ms from match start) |
| `weapon_id` | UBIGINT | Film-attributed WID |
| `reconciled_as` | UBIGINT | API-reconciled override (NULL if none) |
| `delta_ms` | INTEGER | Fire-event-to-kill delta (NULL if snapshot path) |
| `confidence` | VARCHAR | `high` / `medium` / `low` / `none` |
| `attribution_path` | VARCHAR | `fire_event` / `formula_a` / `none` |
| `swap_detected` | BOOLEAN | A weapon swap occurred near the kill |
| `delayed_damage` | BOOLEAN | Projectile travel may have inflated the delta |
| `player_index` | INTEGER | Resolved film player index |

**Append-only (#23046 hardening).** The table was converted to append-only
(`internal/migration/steps_shared_append_only_weapon_kills.go`): two columns
were added — `generation_id BIGINT` and `written_at TIMESTAMP`. Each
`InsertWeaponKills` call allocates one generation from
`weapon_kills_generation_seq` shared by all rows of that `(match_id, xuid)`
write, and INSERTs (no `DELETE`). This removed the previous
DELETE-then-INSERT, which triggered the DuckDB ART index bug on the
multi-writer shared DB. There is no technical PK; correctness comes from the
generation sequence + read-time superseding, not from row-level constraints.

> Some inline comments in `backfill_weapons.go` still describe the legacy
> DELETE-then-INSERT model — the append-only conversion supersedes them.

---

## Read Path — `v_weapon_kills` and labels

Readers never read `weapon_kills` directly. The canonical read surface is the
view **`v_weapon_kills`**, which:

- adds `effective_weapon_id = COALESCE(reconciled_as, weapon_id)`, and
- keeps, per `(match_id, xuid)`, **only the rows of the latest generation**
  (`DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC)`).

`internal/platform/duckdb/weapon_kills_repo.go`
(`LoadWeaponKillsAggregated`) aggregates `v_weapon_kills` by
`(xuid, effective_weapon_id)` with `COUNT(*)`, excluding sentinels
`effective_weapon_id NOT IN (0,1,2)`. When `IncludeGrenadeMelee = true`, it
UNION-ALLs the `grenade_kills` / `melee_kills` totals from `match_participants`
under the sentinel IDs `0` and `1`.

Labels (EN/FR) and roles are attached in Go via `weapon_resolver.go` — the
metadata DB is separate, so it cannot be SQL-joined to the shared DB.

**Display name — single source keyed by `weapon_key` (V72-06).** The display name
is resolved `weapon_id → weapon_ids → weapon_key → {en, fr}` from
`metadata.weapon_name_labels` (`title_slug`, `weapon_key`, `name_en`, `name_fr`),
seeded at boot from `config/titles/{slug}/mappings/weapon_names.toml`
(`ReconcileWeaponNameLabels`). All raw id variants of one weapon collapse to a
single translation (kills the `FRAG GRENADE` vs `Frag Grenade` mismatch). Label
priority: `weapon_name_labels.name_fr > .name_en > weapon_labels.name_fr >
weapon_labels.name_en` — the last two only cover ids **without** a `weapon_key`
(sentinels `0/1/2`, unknowns). The `weapons` registry no longer carries a display
name (`name_fr` removed); it provides only the dimensions (role/class/family/faction).
`weapon_labels.name_en` still drives the image URL (`AssetURLAdapter`).

If `weapon_kills` / `v_weapon_kills` is absent (e.g. a title that does not
support it), the repo returns `games.ErrCapabilityNotSupported`.

---

## Running the Backfill

From `apps/go-api` (CGO toolchain required for the DuckDB driver):

```bash
# Backfill weapon_kills for all eligible players' missing matches
go run ./cmd/levelup backfill --weapons

# Reprocess matches even if already marked done
go run ./cmd/levelup backfill --weapons --force
```

Match selection is bitmask-driven on `match_registry.backfill_completed`:
bit `1<<21` = weapon_kills done, bit `1<<22` = no-film. Without `--force`,
matches that already have either bit set are skipped
(`findMissingWeaponMatches`).

> Each run processes at most **30 missing matches per player** (`LIMIT 30` in
> `findMissingWeaponMatches`) and does not loop. Re-run the command until it
> reports `matches=0` to drain a large backlog.

If `metadata.weapon_labels` is empty (some prebuilt DBs), repair it (stop the
API server first — it holds metadata.duckdb RW):

```bash
go run -tags cgo ./cmd/seed-weapon-labels
```

---

## Adding / Resolving a Weapon ID

When a new weapon ships, or an unresolved WID is positively identified:

1. Add the entry to `weaponEntries` in
   `apps/go-api/internal/analysis/weapon_data.go` (hex -> name). Place it in the
   right group (standard / Energy Sword family / Gravity Hammer family /
   grenade). If the weapon class is new, add a `WeaponTimingByName` entry.
2. Add the weapon to the canonical registry (`weaponRegistryWeapons` +
   `weaponRegistry*IDs` in `internal/games/halo_infinite/migrations/weapon_registry.go`)
   so the raw id resolves to a `weapon_key`, **and** add the display name (en/fr)
   keyed by that `weapon_key` in `config/titles/{slug}/mappings/weapon_names.toml`.
   A registry `weapon_key` without a `weapon_names.toml` entry (or vice versa) fails
   the completeness guard (`weapon_names_completeness_test.go`). `weapon_labels` is
   still seeded (EN name for the image URL), but no longer carries the display name.
3. Re-run `go run ./cmd/levelup backfill --weapons --force` for affected
   matches if you want existing rows re-attributed; new matches pick up the
   mapping automatically.

> Caution: do not add WIDs speculatively from byte-family similarity. Only add
> WIDs positively identified from reliable asset sources or direct chunk
> inspection — a wrong entry misattributes across all matches.

---

## Known Limitations

- **Films expire.** Older matches have no film (404/410) and can never be
  attributed; they are marked `no_film`.
- **`fire_event` is per-fire, `formula_a` is per-chunk snapshot.** Snapshot
  attribution is coarse and is demoted on detected swaps within a chunk.
- **Reconciliation aligns totals, not individual kills.** API reconciliation
  guarantees the per-weapon counts match the API kill total, not that every
  single kill is correctly attributed.
- **Melee/grenade weapon type is not film-resolved** beyond the
  `MELEE`/`GRENADE` distinction; counts come from `match_participants`.
- Attribution is therefore a **statistical reconstruction**, suitable for
  weapon-usage breakdowns and favorite-weapon, not for forensic per-kill
  claims.

---

## Deep-Dive: Film Reverse-Engineering

The binary structure of the film chunks, the dead-state / kill-feed decoding,
and the catalogue of weapon IDs are documented separately and are **not
duplicated here**:

- `.ai/RESEARCH_THEATER_RE.md` — film/theater chunk structure, dead-state and
  kill-feed reverse-engineering notes.
- `.ai/REFERENCE_WEAPON_IDS.md` — weapon ID catalogue and resolution research.
