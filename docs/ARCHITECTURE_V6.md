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

## Player identity registries (ADR 0035)

A player is known to four registries with different lifecycles and sensitivity:

| Registry | File | Keyed by | Holds |
|---|---|---|---|
| Tracked profiles | `db_profiles.json` | slug/gamertag per title | `db_path`, `sync_enabled`, `initial_max_matches`, `auth_only` |
| Accounts | `data/auth/users.json` | username | role, password hash, linked xuid/gamertag |
| Credentials | `data/auth/watcher_tokens/{xuid}.json` | xuid | refresh/access/XSTS tokens (ADR 0023) |
| Live tracking | watcher daemon (in memory) | gamertag x title | pollers, rebuilt from profiles at boot |

The **xuid is the only join key** between them; gamertag and slug are display values and
the file-system path component. The `PlayerDirectory` port
(`internal/service/playerdirectory/`) is the single composite read model
(`GET /admin/identities`, admin "Identities" section, `levelup identity list`) and the single
write path for onboarding (`Onboard`, the only caller of `ProfileService.CreatePlayer`) and
purge (`Purge`, which never touches the shared warehouse). Three invariants are enforced by
ratchets in `internal/archlint/`: no sync and no live tracking without a tracked profile
(`domain.ProfileGate` on the sync coordinator, the watcher daemon and the Xbox SSO), the
instance lock is read only through `authz.InstanceLocked`, and `playerdirectory` never
imports DuckDB. When the auth is enforced and `app_settings.json` lacks the keys, the
defaults are `instance_locked=true` and `can_self_provision=false`; an admin is exempt from
both on `POST /setup/players`.

## Key tables (high-level)

### metadata.duckdb

- `asset_translations`: names and descriptions per language, one row per `(asset_id, asset_type, lang)` (`lang` = Halo BCP-47 tag: `en-US`, `fr-FR`, …) — **added v6.3**. Types `map`, `playlist`, `pair`, `game_variant` (14 languages, `levelup populate-assets`), plus the curated texts that left their `_fr` / `_en` column pairs (ADR 0040 D-13, D-15): `achievement`, `achievement_locked`, `milestone` (name = title, description = condition), `citation`, `challenge_template`, `preset_arc`, and for Halo 5 `commendation`, `team_color`. Readers load every language and choose with `Text.Pick` (request language, then `en`)
- `medal_translations`: medal name and description per `(medal_name_id, lang)`
- `challenge_definitions`: versioned Halo challenge definitions (`challenge_path` + `content_hash`) with category, difficulty, threshold and XP rewards
- `challenge_translations`: localized challenge titles and descriptions in all languages exposed by the CMS (BCP-47, `en-US` fallback)
- `weapon_name_labels` `(title_slug, weapon_key, lang, name)` and `weapon_label_names` `(weapon_id, lang, name)`: weapon display names, one row per language (`lang` = language code of `config/locales.toml`); `weapon_labels` (weapon_id, filmshell UBIGINT) keeps only its non-text columns
- `career_ranks`: rank tier definitions (`title_en` / `subtitle_en` are the single English source; the names per language are `career_rank_translations`)
- `citation_mappings`: medal → citation mappings (no text column: the citation texts are `asset_translations` rows of type `citation`)
- `mode_name_tr` / `mode_*`: game mode translations (legacy overrides, superseded by `asset_translations` for map/playlist/pair/variant names)

### shared_matches_v2.duckdb

Core tables:
- `match_registry`: one row per match
- `match_participants`: per-player stats for all matches

SQL views (`ensure_resolution_views()`):
- `v_match_full`: `SELECT mr.*` over `match_registry` (`internal/migration` `VMatchFullViewSQL`). The registry carries the names `map_name`, `pair_name`, `playlist_name`, `game_variant_name` and the `*_fr` columns `map_name_fr`, `pair_name_fr`, `playlist_name_fr`, kept in the shared database and read as registry values; the names in every other language are read from `meta.asset_translations` at query time (ADR 0040)
- `v_gamertag_lookup`: XUID → current gamertag (FULL OUTER JOIN `xuid_aliases` + `match_participants` + `match_kill_events_latest`)

> `v_killer_victim_full` was **dropped on 2026-08-02** and is no longer a guaranteed v6 view.
> Its two LEFT JOINs re-joined `v_gamertag_lookup` to produce columns that already carried those
> names, and it only "worked" through DuckDB's silent renaming of homonymous columns. Its single
> reader (match-view Q20) now reads the table directly and returns the same six columns. Killer →
> victim pairs live in `killer_victim_pairs` (historical, still read) and in `match_kill_events`
> — read the latter through its `match_kill_events_latest` view only (ADR 0026).

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

### HTTP API (always mounted since 2026-08-02)

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
dictionaries; the `features/*/fallback.i18n.ts` entries were dropped on
2026-08-02 along with the files themselves).

Pages migrated as of 2026-04-26: Career (encounters), Home (KPI bar, challenges
list, spartan identity), Match View (scoreboard), Synthesis (top weeks),
Compare (delta cards), Media (mode categories), Objectifs (challenge tiers,
cadences, prestige levels), Communauté (leaderboard tier), Session Detail
(outcomes).

As of 2026-08-02 (v7.3 lot 2) the TOML manifests are the SINGLE source of these
labels: the `features/{home,media,prestige}/fallback.i18n.ts` dictionaries and
the FR/EN tables of `lib/i18n/metricLabel.ts` were deleted, and the rollout flag
that made the endpoint optional was removed. Missing key = the humanized raw key
(`humanizeMetricKey`), never a second dictionary — enforced by
`apps/web/src/lib/i18n/no-field-label-dictionary.test.ts`. Remaining local
dictionaries (`kpi.i18n.ts`, `highlights.i18n.ts`, `compare/i18n.ts`) are
pre-existing debt scheduled for the same migration.

### Capability-aware degradation

Each `TitleDataAdapter` exposes a `Capabilities() games.CapabilityMap` reflecting
the per-title support of product capabilities. A `Load*` call on a capability
marked `not_exposed` returns `games.ErrCapabilityNotSupported`, which downstream
services translate into an explicit `not_supported_reason` field rather than a
silent empty payload.

See [`.ai/V7/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md`](../.ai/V7/PLAN_MULTI_TITLE_ADAPTERS_AND_MAPPINGS.md)
for the design rationale and [`tools/mappings/CHANGELOG.md`](../tools/mappings/CHANGELOG.md)
for the TOML schema versioning history.

## Bricks / libs (ADR 0041)

Code that is autonomous by nature (a file format, a stream decoder, a protocol) lives in **Go
modules of its own** under `libs/<brick>/`, in this repository, linked to the application by the
`go.work` at the root (no `replace` directive). The application stays the module `levelup/go-api`
(`apps/go-api`); a brick is `github.com/JGtm/LevelUp/libs/<brick>`, with its packages at the first
level of the module (an `internal/` package could not be imported by the application).

```text
go.work                  use ./apps/go-api ./libs/film ./libs/gamefiles ./libs/himap
apps/go-api/             the application (module levelup/go-api)
libs/film/               the "film decoder" brick (ADR 0034): the facade decfilm, the publication layer
                         replay, the closed layers under internal/ (source, profile, grammar, facts)
                         and its catalogues — imports libs/gamefiles and libs/himap only
libs/gamefiles/          the "game files" brick: himodule, ooz (GPLv3, never linked by cmd/server),
                         hinavmesh, mapvar — imports nothing from LevelUp
libs/himap/              the "map images" brick: package himap (map geometry and top-down background
                         images read from the game files) — imports libs/gamefiles only
```

Rules, held by ratchets: a brick imports nothing from `levelup/go-api` and only bricks of lower
rank (`archlint/briques_imports_test.go`, tests included); `go list -deps` in the module shows no
`levelup/go-api` package; no file extracted from the installed game enters a brick; the `go` line
is the same in `go.work`, `apps/go-api/go.mod`, every `libs/*/go.mod` and the `Dockerfile` image.
`./...` never crosses a module boundary: `make libs-test` and the CI steps "brique film", "brique gamefiles" and "brique himap"
(plus the `film*` shards of the coverage matrix) cover the bricks, `go mod tidy` does not run in `apps/go-api`. A brick is importable from outside by the
tag `libs/<brick>/vX.Y.Z`. Commands: [`COMMANDS.md`](COMMANDS.md) § "Go modules under `libs/`"; decision and the
checklist for adding a brick: [`adr/0041-go-modules-under-libs.md`](adr/0041-go-modules-under-libs.md).

### Halo API brick (ADR 0042)

The code that **talks to** the Halo services lives in two internal packages of `apps/go-api`, not in a
module: `internal/haloapi/xblauth` (the exchange chain Microsoft token -> Xbox Live -> XSTS -> Spartan) and
`internal/haloapi/infinite` (addresses, transport, one file per service, raw JSON DTOs, compressed film
bytes). They hold no state and no policy and import nothing of LevelUp outside `internal/haloapi`: the HTTP
`Doer` (guarded per `netguard` surface, consulted before the limiter), the retry policy (`Retry`, three
policies kept), the addresses, the Azure client, the title descriptor, the tokens and the language are all
passed in. What decides stays in LevelUp: `platform/auth` (store, pooling, refresh loops;
`platform/auth/tokencache` for the 50-minute per-player cache), `games/halo_infinite/haloadapter` (DTO ->
`domain`, caches, the `Provider` and `Client` facades), `sync/filmfetch` (inflation, finalisation, local film
cache). The table of internal bricks is `archlint/briques_internes_test.go`; `domain`, `analysis` and
`games/canonical` never reach the brick. Decision: [`adr/0042-halo-api-protocol-vs-exploitation.md`](adr/0042-halo-api-protocol-vs-exploitation.md).
