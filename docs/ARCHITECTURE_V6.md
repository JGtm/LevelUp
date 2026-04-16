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

- `asset_translations`: localized names for maps, playlists, pairs and game variants — 14 BCP-47 languages (`en-US`, `fr-FR`, …) — **added v6.3** — populated by `scripts/populate_asset_translations.py`
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
