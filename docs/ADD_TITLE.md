# Adding a New Game Title to LevelUp

French version: [FR/ADD_TITLE.md](FR/ADD_TITLE.md)

> **Architecture**: title-aware v7 — `data/titles/<slug>/` layout

LevelUp is designed to support multiple game titles. This guide covers the complete
procedure to onboard a new title from code registration to first server startup.

---

## Quick start — automated command

```bash
# Minimum: game name only (slug is derived)
levelup add-title --name "Halo MCC"

# With all options
levelup add-title \
  --name "Halo MCC" \
  --slug halo_mcc \
  --capabilities matchmaking,media,ranked \
  --xbox-id 976923 \
  --steam-id 976730
```

The command:
1. Derives the slug from the name (`"Halo MCC"` → `halo_mcc`)
2. Creates `data/titles/<slug>/warehouse/` and `data/titles/<slug>/players/`
3. Creates and initializes `shared_pve.duckdb` if `firefight` is in `--capabilities`
4. Adds the empty `"<slug>"` section to `db_profiles.json`
5. Prints the Go snippet to paste into `registry.go` (requires a `make build` after)

DuckDB file creation and schema migrations happen automatically at next server startup.

---

## Steps — overview

| Step | Action | Who |
|-----:|--------|-----|
| 1 | Register the descriptor in `registry.go` + `make build` | **Manual** |
| 2 | Create on-disk directories + frontend image folder | Automated by `add-title` |
| 3 | Initialize `shared_pve.duckdb` (if Firefight) | Automated by `add-title --capabilities firefight` |
| 4 | Add the title section to `db_profiles.json` | Automated by `add-title` |
| 5 | Start the server (DuckDB creation + migrations) | Automatic at startup |
| 6 | Add header banner images for the home page | **Manual** |
| 7 | (Optional) Pre-populate `metadata.duckdb` referentials | Manual, title-specific |

---

## Step 1 — Register the title in `registry.go`

**File**: `apps/go-api/internal/domain/title/registry.go`

Add a `Register(...)` call inside `NewRegistry()`:

```go
r.Register(&TitleDescriptor{
    Slug:     "halo_mcc",                  // unique slug, lowercase, underscores
    Name:     "Halo: The Master Chief Collection",
    Provider: "halo_mcc",
    Status:   StatusComingSoon,            // active | coming_soon | archived
    Capabilities: []Capability{
        CapMatchmaking, CapMedia,
    },
    IsDefault:   false,
    XboxTitleID: "976923",                 // from Xbox title catalogue
    SteamAppID:  "976730",                 // from Steam (empty string if N/A)
})
```

### Available capabilities

| Constant        | Meaning                                    |
|-----------------|--------------------------------------------|
| `CapMatchmaking`| Ranked/social matchmaking stats            |
| `CapFirefight`  | Co-op PvE / Firefight mode                 |
| `CapForge`      | Custom map & mode support                  |
| `CapMedia`      | Screenshots and video clips                |
| `CapRanked`     | CSR / skill rating                         |
| `CapCareer`     | Career rank progression                    |

Only declare capabilities the title actually has — services gate their behaviour on
`HasCapability(...)`.

### Effect of `Status`

| Status         | Behaviour                                                              |
|----------------|------------------------------------------------------------------------|
| `active`       | Title fully enabled, routing middleware resolves it                    |
| `coming_soon`  | Registered but not yet routed (use for in-progress integrations)       |
| `archived`     | Read-only legacy access, no new sync                                   |

A title with any status is still **registered** in the `Registry` and its paths are
resolved by `PathResolver`. `ValidateTitle(slug)` passes for all statuses.

---

## Step 2 — Create the directory structure

**Automated by `levelup add-title`.**

If you are bootstrapping without the command:

```bash
mkdir -p data/titles/<slug>/warehouse
mkdir -p data/titles/<slug>/players
```

### Why these directories are required

`duckdb.OpenReadWrite(path)` creates a `.duckdb` file if it does not exist, but only
if the **parent directory already exists**. If `data/titles/<slug>/warehouse/` is
absent, the server will fail to open or create any database for that title.

### Resulting layout

```
data/
└── titles/
    └── <slug>/
        ├── warehouse/          ← DuckDB files are created here at startup
        │   ├── metadata.duckdb
        │   ├── shared_matches_v2.duckdb
        │   └── shared_social.duckdb
        └── players/            ← one sub-directory per player after first sync
```

---

## Step 3 — Add players to `db_profiles.json`

**The title section is added automatically by `levelup add-title`.**
Players are then added manually inside that section.

**File**: `db_profiles.json` (repo root)

Add a player entry under the title section:

```json
{
  "version": "3.0",
  "profiles": {
    "halo_infinite": { ... },
    "halo_mcc": {
      "MyGamertag": {
        "db_path":        "data/titles/halo_mcc/players/MyGamertag/stats.duckdb",
        "xuid":           "2533274800000000",
        "waypoint_player": "MyGamertag"
      }
    }
  }
}
```

`cfg.LoadPlayers(titleSlug)` navigates directly to `profiles[titleSlug]`, so the key
**must exactly match** the slug registered in Step 1.

The `players/<gamertag>/` subdirectory and `stats.duckdb` are created automatically
when the player's data is first synced via `RunPlayerMigrations`.

---

## Step 4 — Start the server

On startup, `runMigrations` in `cmd/server/main.go` runs schema migrations in order:

| Database                | Auto-created? | Notes                                                   |
|-------------------------|---------------|---------------------------------------------------------|
| `metadata.duckdb`       | Yes           | `OpenReadWrite` creates file if absent                  |
| `shared_matches_v2.duckdb` | Yes        | Same                                                    |
| `shared_social.duckdb`  | Yes           | Same                                                    |
| `shared_pve.duckdb`     | **No**        | Migrations run only if file already exists (`os.Stat`)  |

Each migration is tracked in a `schema_migrations` table inside the target database.
A migration that was already applied is **never run twice** (idempotent).

### PvE / Firefight support

`shared_pve.duckdb` is created automatically by `add-title` when `firefight` is in
`--capabilities`. If you are bootstrapping manually:

```bash
duckdb data/titles/<slug>/warehouse/shared_pve.duckdb ".quit"
```

The server will apply PvE migrations automatically at next startup.

---

## Step 5 — Route requests to the new title

The `TitleExtractor` middleware resolves the active title for every API request using
the following priority:

1. **`X-LevelUp-Title` request header** — if the slug is registered, it is used
2. **Current session** (`CurrentTitleSlug`) — persisted server-side
3. **Fallback** — `halo_infinite` (the default slug)

To direct requests to the new title from a client or `curl`:

```bash
curl -H "X-LevelUp-Title: halo_mcc" \
     http://localhost:8000/api/v1/players/MyGamertag/pages/home
```

No router changes are needed — all `/api/v1/players/{player_slug}/...` routes are
title-aware through the middleware and `ResolvePlayer`.

---

## Step 6 — Add header banner images

`add-title` creates the folder `apps/web/public/titles/<slug>/` automatically.
Drop the title's header visuals there (`.webp` or `.png`), then register them in
`apps/web/src/features/home/HomeHeroBanner.tsx` inside `HEADER_IMAGES_BY_TITLE`:

```ts
const HEADER_IMAGES_BY_TITLE: Record<string, string[]> = {
  halo_infinite: [ /* … */ ],
  halo_mcc: [
    '/titles/halo_mcc/header-1.webp',
    '/titles/halo_mcc/header-2.png',
  ],
}
```

If no images are provided the banner silently shows nothing (empty array fallback).

---

## Step 7 — (Optional) Pre-populate referential data

`metadata.duckdb` contains referential tables (weapon labels, career ranks, map names,
etc.). For a new title these tables start empty.

The population scripts vary by title — check `scripts/` for available importers and
follow the documentation specific to that game.

---

## Validation checklist

After completing the steps above, verify the configuration:

```bash
# Global DB integrity check
levelup healthcheck

# Gate check (tables, views, migrations)
levelup gate-check --gamertag MyGamertag
```

Or inspect each point manually:

- [ ] `registry.Get("<slug>")` returns a non-nil descriptor (rebuild done)
- [ ] `data/titles/<slug>/warehouse/` exists and contains the `.duckdb` files
- [ ] `db_profiles.json` has the correct entry under `profiles["<slug>"]`
- [ ] `schema_migrations` tables are populated in each database
- [ ] `GET /api/v1/bootstrap` returns the title in the `titles` list
- [ ] A player request with `X-LevelUp-Title: <slug>` returns HTTP 200
- [ ] `apps/web/public/titles/<slug>/` exists and contains at least one header image
- [ ] `HEADER_IMAGES_BY_TITLE` in `HomeHeroBanner.tsx` has an entry for `<slug>`

---

## Reference: path resolution

All paths for a title are resolved by `title.NewPathResolver(repoRoot)`:

| Method                         | Resolved path                                           |
|--------------------------------|---------------------------------------------------------|
| `TitleDataDir(slug)`           | `data/titles/<slug>/`                                   |
| `WarehouseDir(slug)`           | `data/titles/<slug>/warehouse/`                         |
| `SharedDBPath(slug)`           | `…/warehouse/shared_matches_v2.duckdb`                  |
| `MetadataDBPath(slug)`         | `…/warehouse/metadata.duckdb`                           |
| `SharedPVEDBPath(slug)`        | `…/warehouse/shared_pve.duckdb`                         |
| `SharedSocialDBPath(slug)`     | `…/warehouse/shared_social.duckdb`                      |
| `PlayerDir(slug, gamertag)`    | `…/players/<gamertag>/`                                 |
| `PlayerDBPath(slug, gamertag)` | `…/players/<gamertag>/stats.duckdb`                     |

No `filepath.Join(repoRoot, "data", ...)` is allowed outside `PathResolver` — always
use these methods.
