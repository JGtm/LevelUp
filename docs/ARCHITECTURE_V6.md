# Architecture LevelUp v6 — DuckDB Shared Matches + i18n Assets

French version: [FR/ARCHITECTURE_V6.md](FR/ARCHITECTURE_V6.md)

> **Version** : 6.3.0 — **Mise à jour** : 2026-04-12

LevelUp uses a DuckDB v6 architecture based on **shared matches** and **centralized i18n via `asset_translations`**:

- `data/warehouse/shared_matches_v2.duckdb` — all match data shared across all players
- `data/warehouse/metadata.duckdb` — referentials: asset names (14 langs), weapons, career ranks, citations
- `data/players/{gamertag}/stats.duckdb` — per-player enrichments only

## Databases

```text
data/
  warehouse/
    metadata.duckdb
    shared_matches_v2.duckdb
    shared_pve.duckdb
  players/
    {gamertag}/
      stats.duckdb
```

## Key tables (high-level)

### metadata.duckdb

- `asset_translations`: localized names for maps, playlists, pairs and game variants — 14 BCP-47 languages (`en-US`, `fr-FR`, …) — **added v6.3** — populated by the Go metadata migrations (`internal/games/halo_infinite/migrations/`) during metadata seeding
- `challenge_definitions`: versioned Halo challenge definitions (`challenge_path` + `content_hash`) with category, difficulty, threshold and XP rewards
- `challenge_translations`: localized challenge titles and descriptions in all languages exposed by the CMS (BCP-47, `en-US` fallback)
- `weapon_labels`: weapon_id (filmshell UBIGINT) → `name_en`, `name_fr` — added v5.4
- `career_ranks`: rank tier definitions
- `citation_mappings`: medal → citation mappings
- `mode_name_tr` / `mode_*`: game mode translations (legacy overrides, superseded by `asset_translations` for map/playlist/pair/variant names)

### shared_matches_v2.duckdb

Core tables:
- `match_registry`: one row per match
- `match_participants`: per-player stats for all matches

SQL views (`ensure_resolution_views()`):
- `v_match_full`: `match_registry` enriched with i18n names from `meta.asset_translations` — 8 LEFT JOINs (en-US + fr-FR × map/playlist/pair/variant). Columns: `map_name`, `map_name_fr`, `game_variant_name`, `game_variant_name_fr`, etc.
- `v_gamertag_lookup`: XUID → current gamertag (FULL OUTER JOIN `xuid_aliases` + `match_participants`)
- `v_killer_victim_full`: killer/victim pairs with resolved gamertags

> **Important**: `v_match_full` requires `metadata.duckdb` to be ATTACHed as `meta` in the same connection for the i18n JOINs to work. `DuckDBRepository` handles this automatically.

### stats.duckdb (per player)

- `player_match_enrichment`: performance_score, session_id, etc.
- `challenge_snapshots`: append-only per-player challenge state history (active/completed/upcoming, progress, XP, expiry), deduplicated on state change

## OpenSpartan import

LevelUp accepts a one-time SQLite upload from [OpenSpartan Workshop](https://github.com/OpenSpartan) (community Halo tracker by Den Delimarsky, credited in `docs/ACKNOWLEDGMENTS.md`), so players who already tracked matches there before switching can backfill that history. The import parses the file and writes matches into `shared_matches_v2.duckdb` through the same `persist.SharedPersister` path as live sync — no ad hoc SQL. Code: `internal/openspartan/` (reader) + `internal/openspartan/mapper/` (row mapping) + `internal/service/openspartan_import_service.go` (orchestration) + `internal/api/handlers/openspartan_import.go` (auth-gated upload endpoint, off in demo mode) + `OpenSpartanImportCard.tsx` (onboarding UI). The name comes from the third-party project it reads from, not from LevelUp's own naming.

---

## Multi-Title Architecture (Sprint 44)

LevelUp supports multiple game titles via a **title-aware data layout**. Each title has its own isolated data tree:

```text
data/
  titles/
    halo_infinite/          # default title
      warehouse/
        metadata.duckdb
        shared_matches_v2.duckdb
      players/
        {gamertag}/
          stats.duckdb
    halo_mcc/               # second title (example)
      warehouse/
        metadata.duckdb
        shared_matches_v2.duckdb
      players/
        {gamertag}/
          stats.duckdb
  warehouse/                # legacy flat layout (backward compat)
  players/                  # legacy flat layout (backward compat)
```

### Key components

| Component | Role |
|-----------|------|
| `TitleRegistry` | In-memory registry of known titles (slug, name, status, capabilities) |
| `PathResolver` | Resolves all file paths relative to a title slug (`TitleDataDir`, `SharedDBPath`, `PlayerDBPath`, etc.) |
| `TitleExtractor` middleware | Reads `X-LevelUp-Title` header / session / fallback → injects `title_slug` into request context |
| `db_profiles.json` v3 | Title-scoped player profiles: `{ "version": "3.0", "profiles": { "<title_slug>": { "<gamertag>": {...} } } }` |

### Routing strategy

The API uses **header-based** title selection (`X-LevelUp-Title`). URLs remain unchanged (`/api/v1/players/{slug}/...`). The middleware injects the title into the request context, and all downstream services (PlayerResolver, ProfileService, etc.) use it to scope data access.

### Frontend

The `appShellStore` tracks `currentTitleSlug` and provides `switchTitle()` which:
1. POSTs `/session/context` with the new title
2. Re-bootstraps the app
3. Resets player-scoped caches

The API client sends `X-LevelUp-Title` header for non-default titles.

### Backward compatibility

- `PathResolver` provides `Legacy*` methods (`LegacySharedDBPath`, `LegacyPlayerDir`, etc.) for the flat `data/warehouse/` layout
- `db_profiles.json` v2.1 files are auto-detected and read as implicit `halo_infinite` profiles
- `LoadPlayers()` without a title filter returns players from all titles

---

## Canonical services schema + semantic adapters (Phase A–E plan multi-titres)

On top of the title-aware storage layout, LevelUp exposes a canonical services
schema and two title adapters per title. This decouples the product services
from the per-title DuckDB schema and from the per-title labels/units.

```text
HTTP handler → product service → games.Resolver
                                    ├─ Data(slug)     → games.TitleDataAdapter
                                    └─ Semantic(slug) → games.TitleSemanticAdapter
```

### Packages

| Package | Role |
|---------|------|
| `internal/games/canonical/` | `FieldKey` enum (59 keys), enums (`Outcome`, `MatchType`, `RatingType`, `Bucket`, `GroupBy`), scopes (`StatsScope`, `TimeseriesQuery`, `CareerOptions`), match/career/timeseries types — all stable, agnostic, used by services |
| `internal/games/mappings/` | Strict TOML loader (`go-toml/v2`), validation (locales, formats, `display_order` collisions, unit conversions), `FieldMappingSet`, registry |
| `internal/games/halo_infinite/` | HI implementation: `DataAdapter` (wraps existing repos), `SemanticAdapter` (wraps `FieldMappingSet`), `AssetURLAdapter` (composes `/static/...` URLs) |
| `internal/games/synthetic_title_b/` | Synthetic test corpus, isolated cross-title tests only — never referenced by production code |
| `internal/games/{adapter,resolver}.go` | `TitleDataAdapter` + `TitleSemanticAdapter` + `TitleAssetURLAdapter` interfaces, `StaticResolver` |
| `internal/assets/static/` | Pure URL/path composition for `/static/{folder}/{titleSlug}/{id}{ext}` — no title knowledge, no I/O, table-driven tests |

### TOML mappings (versioned in Git)

Three TOML files per title under `config/titles/{slug}/mappings/`:

```text
config/
  titles/
    halo_infinite/
      mappings/
        fields.toml           # 59 FieldKey × labels EN/FR + format + group + display_order
        assets.toml           # modes / challenge_tier / cadence / challenge_status / medal_tier / prestige_level
        outcomes.toml         # win / loss / tie / dnf — labels + color_token (design system)
    synthetic_title_b/
      mappings/
        fields.toml           # synthetic test corpus
        assets.toml           # divergent labels for cross-title isolation tests
        outcomes.toml         # divergent labels (Triomphe / Défaite / Match nul / Forfait)
```

`fields.toml` is mandatory; `assets.toml` and `outcomes.toml` are optional (their absence is silent). Each TOML carries `[meta].schema_version` (cf. `tools/mappings/CHANGELOG.md`).

The `Registry.LoadFromConfigDir()` boot loads all three files per title. A failure on any file emits `mappings_validation_failed` (Error) and aggregates into the returned errors slice — but a failed title does not block the others.

#### Decision: no hot-reload (dev or prod)

The plan §7.3 reserved a `GAMES_HOT_RELOAD=true` mode for live TOML reload in dev. We have **deliberately not implemented it** (verified 2026-04-26):

- Prod: hot-reload is forbidden by §7.3 — the semantic layer is a versioned contract that only changes via PR + golden parity. Cost = a redeploy per label change, acceptable for a few dozen FieldKey.
- Dev: a TOML edit with the current setup means `Ctrl+C` + `air` again (~3-5s rebuild + reboot). At our edit frequency (~1 TOML edit per sprint outside of new-title onboarding), the gain (~5s/edit, no production impact) does not justify the cost (fsnotify watcher Windows-friendly + `Registry.Reload()` method + ETag invalidation in the `/field-mappings` handler + tests for race conditions).

Consequence: the log event `mappings_hot_reloaded` from plan §8.1 is intentionally absent (8/9 events emitted, this one is the 9th). To revisit if/when:
- A second real title onboarding requires intensive TOML iteration, or
- Catalog volume grows (e.g., medals, weapons families) and live label tuning becomes valuable.

### HTTP API (behind `MULTI_TITLE_API_ENABLED=true`)

- `GET /api/v1/titles/{slug}/field-mappings?locale=fr` — exposes the
  `FieldMappingSet` of a title with ETag + `Cache-Control: max-age=300`.

> Note: the former proof-of-concept route `GET /api/v1/titles/{slug}/preview/career`
> was removed (orphaned on the frontend, cf. `server.go` + `multi_title_smoke_test.go`).

### Frontend hooks (Phase D + Phase finition)

```ts
import {
  useFieldLabel,    // FieldKey  → 'Éliminations' (kills FR) / 'Kills' (EN) / 'kills' (fallback)
  useAssetLabel,    // (kind,id) → 'Classé' (mode.Ranked FR) / 'Ranked' (EN) / 'Ranked' (fallback id)
  useOutcomeLabel,  // outcome key → 'Victoire' (win FR) / 'Win' (EN) / 'win' (fallback key)
  useAssetMapping,  // DTO complet (label + color_token + icon + display_order)
  useOutcomeMapping,
} from '@/lib/i18n/fieldMappings'
```

All hooks read `currentTitleSlug` and `locale` from `appShellStore`, share a
single TanStack Query cache (`staleTime: Infinity` — versioned in Git, no
hot-reload), and fall back gracefully on the raw key/id if the endpoint is
absent (flag off, 404, network error).

Components consume these hooks instead of hardcoding labels. The frontier
TOML vs i18n React (cf. plan §6.9) is enforced by `tools/lint-no-hardcoded-fields.mjs`,
which scans 277 files and rejects any literal matching a label declared in
`fields.toml`, `assets.toml`, or `outcomes.toml` (whitelist for fallback
dictionaries under `features/*/fallback.i18n.ts`).

Pages migrated as of 2026-04-26: Career (encounters), Home (KPI bar, challenges
list, spartan identity), Match View (scoreboard), Synthesis (top weeks),
Compare (delta cards), Media (mode categories), Objectifs (challenge tiers,
cadences, prestige levels), Communauté (leaderboard tier), Session Detail
(outcomes). Dictionaries `kpi.i18n.ts`, `highlights.i18n.ts`, `compare/i18n.ts`
keep their FR/EN labels as fallback for `MULTI_TITLE_API_ENABLED=false`.

### Capability-aware degradation

Each `TitleDataAdapter` exposes a `Capabilities() games.CapabilityMap` reflecting
the per-title support of product capabilities. A `Load*` call on a capability
marked `not_exposed` returns `games.ErrCapabilityNotSupported`, which downstream
services translate into an explicit `not_supported_reason` field rather than a
silent empty payload.

See [`.ai/V7/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`](../.ai/V7/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md)
for the design rationale and [`tools/mappings/CHANGELOG.md`](../tools/mappings/CHANGELOG.md)
for the TOML schema versioning history.
