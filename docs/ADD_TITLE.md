# Adding a New Game Title to LevelUp

French version: [FR/ADD_TITLE.md](FR/ADD_TITLE.md)

> **Architecture**: title-aware v7 — `data/titles/<slug>/` layout, config-driven title
> registry (`config/titles/<slug>/`). See [ADR 0008](adr/0008-db-schema-multi-title-and-xuid-global.md)
> (filesystem-path isolation, no `title_id` column) and
> [ADR 0025](adr/0025-title-agnostic-minimal-viable-window.md) (title-agnostic refactor).

LevelUp supports multiple game titles. This guide is the complete, end-to-end procedure
to onboard a new title — from CLI scaffolding to the adapter wiring that actually serves
its data. For the cross-cutting foundations (canonical types, adapters, i18n manifests,
ECharts wrappers) referenced below, see [FOUNDATIONS_GUIDE.md](FOUNDATIONS_GUIDE.md).

---

## How a title is registered (read this first)

There are **two registration mechanisms**, and the canonical one is config-driven:

1. **Built-in default** — `halo_infinite` is hard-wired in `NewRegistry()`
   (`apps/go-api/internal/domain/title/registry.go`). It is byte-identical and robust
   even without any config. **You do not touch this for a new title.**

2. **Config-driven (the path for every additional title)** — at boot,
   `cmd/server/main.go` calls:

   ```go
   title.SetDefaultRegistry(title.NewRegistryFromConfig(cfg.RepoRoot, slog.Default()))
   ```

   `NewRegistryFromConfig` first builds the built-in registry, then
   `LoadTitlesIntoRegistry` scans `config/titles/*/` and registers every directory that
   carries a `title.toml`. **Dropping `config/titles/<slug>/title.toml` is what registers
   an additional title — zero recompilation of the registry.**

> The `levelup add-title` command prints a Go snippet for `registry.go`, but for an
> **additional** title the authoritative source of truth is `title.toml`. Editing
> `registry.go` is only how the *default* title is declared. Prefer the `title.toml`
> route; an invalid manifest is logged and skipped without blocking server boot.

---

## Quick start — scaffold the on-disk layout

`levelup` is the ops CLI (`apps/go-api/cmd/levelup`). Run it as a built binary or via
`go run ./cmd/levelup` from `apps/go-api`.

```bash
# Minimum: game name only (slug is derived from the name)
levelup add-title --name "Halo MCC"

# With all options
levelup add-title \
  --name "Halo MCC" \
  --slug halo_mcc \
  --capabilities matchmaking,media,ranked \
  --xbox-id 976923 \
  --steam-id 976730
```

**Flags** (from `cmd_title.go`):

| Flag             | Required | Default              | Notes                                              |
|------------------|:--------:|----------------------|----------------------------------------------------|
| `--name`         | **Yes**  | —                    | Full game name, e.g. `"Halo MCC"`                   |
| `--slug`         | No       | derived from `--name`| Lowercase `[a-z][a-z0-9_]*[a-z0-9]`; cannot be `halo_infinite` |
| `--capabilities` | No       | `matchmaking,media`  | Comma-separated coarse capabilities (see below)    |
| `--xbox-id`      | No       | empty                | Xbox Title ID (for watcher presence + achievements)|
| `--steam-id`     | No       | empty                | Steam App ID                                       |

The command:
1. Derives the slug from the name (`"Halo MCC"` → `halo_mcc`).
2. Creates `data/titles/<slug>/warehouse/` and `data/titles/<slug>/players/`.
3. Creates `apps/web/public/titles/<slug>/` (frontend header-image folder).
4. Creates and initializes `shared_pve.duckdb` **only if** `firefight` is in `--capabilities`.
5. Adds the empty `"<slug>"` section to `db_profiles.json` (atomic, via the dbprofiles store).
6. Prints the `registry.go` snippet and the frontend image reminder.

It does **not** write `config/titles/<slug>/` — you create the manifest and mappings
yourself (Steps 1–3 below). DuckDB file creation and schema migrations happen at the
next server startup.

---

## Steps — overview

| Step | Action | Who |
|-----:|--------|-----|
| 0 | Scaffold dirs + `db_profiles.json` section | Automated by `levelup add-title` |
| 1 | Write `config/titles/<slug>/title.toml` (descriptor) | **Manual** |
| 2 | Declare capabilities (coarse in `title.toml`, fine in `mappings/capabilities.toml`) | **Manual** |
| 3 | Write the mapping TOMLs (`fields`, `assets`, `outcomes`) | **Manual** |
| 4 | (If serving data) Write a `TitleDataAdapter` + register it | **Manual, Go** |
| 5 | Add players to `db_profiles.json` | **Manual** |
| 6 | Start the server (DB creation + migrations + boot discovery) | Automatic |
| 7 | Add header banner images for the home page | **Manual** |
| 8 | (Optional) Pre-populate `metadata.duckdb` referentials | Manual, title-specific |

---

## Step 1 — Write `config/titles/<slug>/title.toml`

This manifest is the descriptor parsed by `LoadTitleManifest`
(`internal/domain/title/config_loader.go`). It is the equivalent of a `registry.go`
`Register(...)` call, externalized to config.

```toml
# config/titles/halo_mcc/title.toml
[meta]
title_slug     = "halo_mcc"   # must match the directory name
schema_version = 1            # must be > 0

[title]
name          = "Halo: The Master Chief Collection"
provider      = "halo_mcc"
status        = "coming_soon" # active | coming_soon | archived
icon_url      = ""
xbox_title_id = "976923"      # watcher presence + achievements (empty if N/A)
steam_app_id  = "976730"      # empty string if N/A
placement_matches = 0         # ranked placement count; 0 = consumer applies its default
csr_season_id     = ""        # fixed CSR season overlay; empty = global fallback

# Coarse capabilities — see Step 2.
capabilities = [
  "matchmaking",
  "media",
  "ranked",
]
```

Validation rules enforced by `LoadTitleManifestFromBytes`:

- `[meta].title_slug`, if present, **must equal the directory name**.
- `[meta].schema_version` must be `> 0`.
- `[title].name` is required; missing `provider` defaults to the slug.
- `[title].status` must be `active | coming_soon | archived`.
- `[title].is_default` is **forbidden** for additional titles (reserved for `halo_infinite`).
- Every entry in `capabilities` must be a known coarse capability (Step 2) — an unknown
  one fails validation and the title is skipped (logged `title_manifest_invalid`).

### Effect of `status`

| Status         | Behaviour                                                               |
|----------------|-------------------------------------------------------------------------|
| `active`       | Title fully enabled, served, provisioned at boot, adapters wired        |
| `coming_soon`  | Discovered + listed in the title switcher ("coming soon"), but **not served** — `RequireActiveTitle` returns `503 title_unavailable` |
| `archived`     | Excluded from the switcher (`NonArchived()` filters it out)             |

`ValidateTitle(slug)` passes for any registered title regardless of status; runtime
gating of title-scoped routes is done by the `RequireActiveTitle` middleware
(`IsActive()` ⇒ `Status == active`), not by the title resolver.

---

## Step 2 — Declare capabilities

There are **two distinct capability vocabularies** — do not confuse them:

### A. Coarse capabilities (`title.Capability`) — in `title.toml`

Declared in `[title].capabilities`, validated against `knownCapabilities`
(`config_loader.go`). These gate **product surfaces / middleware** and feed the title
switcher. Known values (mirror of the `Cap*` constants in `registry.go`):

| Value                  | Meaning                                                       |
|------------------------|---------------------------------------------------------------|
| `matchmaking`          | Ranked/social matchmaking stats                               |
| `firefight`            | Co-op PvE / Firefight (drives `shared_pve.duckdb` creation)   |
| `forge`                | Custom maps & modes                                           |
| `media`                | Screenshots and video clips                                   |
| `ranked`               | CSR / skill rating                                            |
| `career`               | Career rank progression                                       |
| `season_pass`          | Season pass / Battlepass progression                          |
| `asset.images`         | Asset Drawer thumbnails (maps & weapons)                      |
| `achievements`         | Xbox achievements page                                        |
| `engagement`           | In-match engagement score + coefficients                      |
| `lusr`                 | LevelUp internal rating (LUSR v2)                             |
| `world.leaderboard`    | World leaderboards                                            |
| `native_kill_mechanics`| Native per-kill mechanics (assassinations, spartan abilities) |

The `levelup add-title --capabilities` flag only understands the first six
(`matchmaking,firefight,forge,media,ranked,career`) when emitting the `registry.go`
snippet; declare the richer set directly in `title.toml`.

### B. Fine capabilities (`games.CapabilityKey`) — in `mappings/capabilities.toml`

These gate the **`Load*` methods of the title's `TitleDataAdapter`** (i.e. exactly
which data surfaces the adapter is wired to serve). The 16 known keys live in
`internal/games/adapter.go` (constantes) ; `AllCapabilityKeys()` est dans `internal/games/capabilities.go` :

`match.history`, `match.detail.core`, `match.skill.snapshot`, `career.progression`,
`career.rank_catalog`, `pve.firefight_stats`, `analytics.timeseries`,
`match.scoreboard.extra`, `citations.engine`, `engagement.score`,
`battlepass.progression`, `challenges.surface`, `match.events.timeline`,
`match.killfeed.per_kill`, `match.events.spatial`, `commendations.native`.

Each value is one of `supported | degraded | not_exposed`:

```toml
# config/titles/halo_mcc/mappings/capabilities.toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[capabilities]
"match.history"      = "supported"   # only if LoadMatchSummaries is actually wired
"match.detail.core"  = "supported"
"career.progression" = "supported"
"analytics.timeseries" = "not_exposed"
# ... remaining keys default to not_exposed
```

> **Golden rule** (from `halo_5/mappings/capabilities.toml`): a key may be `supported`
> or `degraded` **only if its `Load*` method is genuinely wired**. `CapabilityMap.Has()`
> returns true for `supported`/`degraded`; if the method is a stub, every call would
> fail. Unwired surfaces must stay `not_exposed`.

`CapabilityMapFromMappings` (`internal/games/capabilities.go`) validates each key and
status at boot and aggregates errors — an unknown key or invalid status is rejected.

---

## Step 3 — Write the mapping TOMLs (semantic layer)

Under `config/titles/<slug>/mappings/`, alongside `capabilities.toml`:

### `fields.toml` — canonical metric definitions, units, labels, ordering

```toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[fields.kills]
labels        = { en = "Kills", fr = "Éliminations" }
storage_unit  = "count"
display_unit  = "count"
format        = "integer"
display_order = 10
group         = "combat"

[fields.accuracy]
labels        = { en = "Accuracy", fr = "Précision" }
storage_unit  = "ratio"
display_unit  = "percent"
format        = "percent_2"
display_order = 40
group         = "combat"
```

### `assets.toml` — modes, tiers, etc. (bilingual labels + display order)

```toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[assets.mode.slayer]
labels = { en = "Slayer", fr = "Massacre" }
display_order = 10

[assets.mode.ctf]
labels = { en = "Capture the Flag", fr = "Capture du drapeau" }
display_order = 20
```

Keys (`kind/id`) are free-form; validation requires non-empty values and no
`display_order` collisions.

### `outcomes.toml` — win/loss/tie/dnf labels + semantic color tokens

```toml
[meta]
title_slug     = "halo_mcc"
schema_version = 1

[outcomes.win]
labels = { en = "Victory", fr = "Victoire" }
color_token = "outcome.positive"

[outcomes.loss]
labels = { en = "Defeat", fr = "Défaite" }
color_token = "outcome.negative"

[outcomes.tie]
labels = { en = "Tie", fr = "Égalité" }
color_token = "outcome.neutral"

[outcomes.dnf]
labels = { en = "DNF", fr = "Abandon" }
color_token = "outcome.neutral"
```

Use semantic color **tokens** (never raw hex) — see
[ADR 0011](adr/0011-canonical-vs-semantic-adapter-separation.md) for the
canonical (raw data) vs semantic (i18n/labels) vs asset-URL adapter boundary.

---

## Step 4 — Adapters (only if the title serves data)

The adapter layer (`internal/games/`) projects a title's native source onto the
cross-title canonical schema (`internal/games/canonical/`). Three adapter interfaces
exist (`adapter.go`):

| Interface              | What you write for a new title                                  |
|------------------------|------------------------------------------------------------------|
| `TitleSemanticAdapter` | **Nothing.** The shared `GenericSemanticAdapter` (`semantic_adapter.go`) wraps your TOMLs — there is no per-title semantic code. |
| `TitleDataAdapter`     | **A Go package** (e.g. `internal/games/halo_mcc/`) projecting your source (live API or DuckDB) onto `canonical.*`. This is the real work. |
| `TitleAssetURLAdapter` | Optional — only if asset URL naming diverges from the default.   |

### Wiring the data adapter

Additional active titles are wired by `registerAdditionalTitles`
(`internal/api/server_titles_additional.go`), which iterates `titleRegistry.Active()`
and dispatches by slug through the `additionalTitleRegistrars` map. Add your title:

```go
var additionalTitleRegistrars = map[string]additionalTitleRegistrar{
    halo5.TitleSlug:    registerHalo5Adapters,
    halo_mcc.TitleSlug: registerHaloMCCAdapters, // your new registrar
}
```

Your `registerHaloMCCAdapters` builds the `GenericSemanticAdapter` from the mapping
sets, converts `capabilities.toml` via `CapabilityMapFromMappings`, constructs your
`TitleDataAdapter`, and registers both on the `StaticResolver`
(`RegisterSemantic` / `RegisterData`) — see `registerHalo5Adapters` as the reference.
This Go change requires a rebuild (`make go-api-build`).

> If a title is `active` but has **no registrar**, the server logs
> `additional_title_no_adapter_registrar` and the title is not served. A
> `coming_soon` title needs no adapter (it is never served).

### Data writes: the Collect → Persist architecture (ADR 0019)

A new title that syncs matches **must not write per-match rows with raw
`ExecContext`**. All per-match writes on a shared or player DB go through the
`internal/persist` layer, which is **INSERT-only** — never a concurrent
`UPSERT` / `ON CONFLICT DO UPDATE` on the critical tables (that is what corrupted
production DuckDB ART indexes, ADR 0019 / 0026):

- **Collect** the match into a `persist.MatchBatch` via `persist.NewBatchBuilder(...)`
  (`SetMatch` / `AddParticipants` / `AddMedals` / `AddMatchCSRs`).
- **Persist** it with `persist.NewSharedPersister(db).Persist(ctx, batch)` (shared,
  atomic, idempotent — re-persisting an existing `match_id` is a no-op), or a
  `PlayerPersister` for player-scoped enrichments / per-match ratings.
- **Append-only tables** (`match_skill_rank`, `match_csrs`, `player_csr_snapshots`,
  `pve_match_stats`, …) are written INSERT-only with a `written_at` stamp and **read
  exclusively through their `<table>_latest` view** (a raw read serves stale rows,
  ADR 0026).

Reference implementation for a new title's live sync:
`internal/games/halo_5/livesync/csr_match.go` (per-match CSR/skill written via
`PlayerPersister.PersistPerMatchRating`). Hierarchy: `games/<slug>/client.go` (fetch)
→ `games/<slug>/livesync/*` (map to persist inputs) → `internal/persist/*` (write).
Never bypass persist from the client or livesync layer.

---

## Step 5 — Add players to `db_profiles.json`

The empty title section is created by `add-title`; players are added manually inside it.

```json
{
  "version": "3.0",
  "profiles": {
    "halo_infinite": { },
    "halo_mcc": {
      "MyGamertag": {
        "db_path":         "data/titles/halo_mcc/players/MyGamertag/stats.duckdb",
        "xuid":            "2533274800000000",
        "waypoint_player": "MyGamertag"
      }
    }
  }
}
```

`cfg.LoadPlayers(titleSlug)` navigates directly to `profiles[titleSlug]`, so the key
**must exactly match** the slug. The `players/<gamertag>/` subdirectory and
`stats.duckdb` are created automatically on the player's first sync.

---

## Step 6 — Start the server

On startup:

1. `NewRegistryFromConfig` discovers `config/titles/<slug>/title.toml` and registers it
   (logged `title_registered_from_config`).
2. Schema migrations run per database. `OpenReadWrite` creates a `.duckdb` file if the
   **parent directory exists** (hence `data/titles/<slug>/warehouse/` must already be
   present — `add-title` creates it).

| Database                   | Auto-created? | Notes                                              |
|----------------------------|:-------------:|----------------------------------------------------|
| `metadata.duckdb`          | Yes           | `OpenReadWrite` creates the file if absent         |
| `shared_matches_v2.duckdb` | Yes           | Same                                               |
| `shared_social.duckdb`     | Yes           | Same                                               |
| `shared_pve.duckdb`        | **No**        | Migrations run only if the file already exists     |

Migrations are tracked in a `schema_migrations` table and are idempotent (never run
twice). `shared_pve.duckdb` is created by `add-title` when `firefight` is in
`--capabilities`; to bootstrap it manually, open and close it once so the file exists.

### Routing requests to the new title

The `TitleExtractor` middleware resolves the active title per request:

1. **`X-LevelUp-Title` header** — if the slug is registered, it is used
2. **Current session** (`CurrentTitleSlug`) — persisted server-side
3. **Fallback** — `halo_infinite`

```bash
curl -H "X-LevelUp-Title: halo_mcc" \
     http://localhost:8000/api/v1/players/MyGamertag/pages/home
```

No router changes are needed — all `/api/v1/players/{player_slug}/...` routes are
title-aware. Note: an `active` title without a wired adapter (Step 4) resolves but
returns `503 title_unavailable` on data routes via `RequireActiveTitle`.

---

## Step 7 — Add header banner images

`add-title` creates `apps/web/public/titles/<slug>/`. Drop the header visuals there
(`.webp` or `.png`), then register them in
`apps/web/src/features/home/HomeHeroBanner.tsx` under `HEADER_IMAGES_BY_TITLE`:

```ts
const HEADER_IMAGES_BY_TITLE: Record<string, string[]> = {
  halo_infinite: [ /* … */ ],
  halo_mcc: [
    '/titles/halo_mcc/header-1.webp',
    '/titles/halo_mcc/header-2.png',
  ],
}
```

If no images are provided the banner silently shows nothing (empty-array fallback).

---

## Step 8 — (Optional) Pre-populate referential data

`metadata.duckdb` referential tables (weapon labels, career ranks, map names, etc.)
start empty for a new title. Population is title-specific — check the available ops
CLIs (`apps/go-api/cmd/*`) and the title's adapter for what it expects.

---

## What degrades gracefully via capabilities

Capabilities make missing data a first-class, non-fatal state rather than an error:

- A `TitleDataAdapter` `Load*` method returns `ErrCapabilityNotSupported` when its fine
  capability (Step 2.B) is not `supported`/`degraded`. Callers treat this as a
  degradation signal, not a failure.
- Coarse capabilities (Step 2.A) gate whole product surfaces. Examples observed in code:
  no `season_pass` ⇒ the career/season-pass tab is hidden and degrades to
  `FeatureUnavailable`; no `world.leaderboard` ⇒ the leaderboard page returns empty
  `200`; no `native_kill_mechanics` ⇒ the front hides assassination / spartan-ability
  sections.
- Semantic gaps degrade too: an empty `RankCatalog` ⇒ consumers show the raw `rank_id`;
  `nil` `Assets()`/`Outcomes()` ⇒ callers fall back gracefully.

Declaring only what the title truly serves is therefore the correct way to ship an
incremental integration: start `coming_soon` with everything `not_exposed`, then flip
keys to `supported` as each `Load*` method is wired.

---

## Validation checklist

```bash
# Global DB integrity check
levelup healthcheck

# Gate check (tables, views, migrations)
levelup gate-check --gamertag MyGamertag
```

Manual verification:

- [ ] `config/titles/<slug>/title.toml` exists and is valid (boot logs `title_registered_from_config`, not `title_manifest_invalid`)
- [ ] `registry.Get("<slug>")` returns a non-nil descriptor
- [ ] `config/titles/<slug>/mappings/{fields,assets,outcomes,capabilities}.toml` present
- [ ] `data/titles/<slug>/warehouse/` exists and contains the `.duckdb` files
- [ ] `db_profiles.json` has the entry under `profiles["<slug>"]`
- [ ] `schema_migrations` tables are populated in each database
- [ ] `GET /api/v1/bootstrap` lists the title in `titles`
- [ ] For an `active` title: a player request with `X-LevelUp-Title: <slug>` returns `200` (or `503` if the adapter is not yet wired)
- [ ] `apps/web/public/titles/<slug>/` contains at least one header image referenced in `HomeHeroBanner.tsx`

---

## Reference: path resolution

All paths for a title are resolved by `title.NewPathResolver(repoRoot)`
(`internal/domain/title/registry.go`):

| Method                         | Resolved path                                |
|--------------------------------|----------------------------------------------|
| `TitleDataDir(slug)`           | `data/titles/<slug>/`                        |
| `WarehouseDir(slug)`           | `data/titles/<slug>/warehouse/`              |
| `SharedDBPath(slug)`           | `…/warehouse/shared_matches_v2.duckdb`       |
| `MetadataDBPath(slug)`         | `…/warehouse/metadata.duckdb`                |
| `SharedPVEDBPath(slug)`        | `…/warehouse/shared_pve.duckdb`              |
| `SharedSocialDBPath(slug)`     | `…/warehouse/shared_social.duckdb`           |
| `PlayerDir(slug, gamertag)`    | `…/players/<gamertag>/`                      |
| `PlayerDBPath(slug, gamertag)` | `…/players/<gamertag>/stats.duckdb`          |
| `GlobalXuidAliasesDBPath()`    | `data/global/xbox_aliases.duckdb` (global)   |

The `xuid → gamertag` alias DB is **global**, not per-title (it is a Microsoft/Xbox
identity, title-independent — [ADR 0008](adr/0008-db-schema-multi-title-and-xuid-global.md)).
No `filepath.Join(repoRoot, "data", ...)` is allowed outside `PathResolver` — always
use these methods.
