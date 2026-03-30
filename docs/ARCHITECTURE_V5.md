# Architecture (DuckDB v5) — LevelUp

French version: [FR/ARCHITECTURE_V5.md](FR/ARCHITECTURE_V5.md)

LevelUp uses a DuckDB v5 “shared matches” architecture:

- `data/warehouse/shared_matches.duckdb` stores match-wide data for all players.
- `data/players/{gamertag}/stats.duckdb` stores only per-player enrichments.

## Databases

```text
data/
  warehouse/
    metadata.duckdb
    shared_matches.duckdb
    shared_pve.duckdb
  players/
    {gamertag}/
      stats.duckdb
```

## Key tables (high-level)

### metadata.duckdb

- `asset_translations`: localized names for maps, playlists, pairs and game variants — 14 BCP-47 languages (`en-US`, `fr-FR`, …) — **added v6.3** — populated by `scripts/populate_asset_translations.py`
- `weapon_labels`: weapon_id (filmshell UBIGINT) → `name_en`, `name_fr` — added v5.4
- `career_ranks`: rank tier definitions
- `citation_mappings`: medal → citation mappings
- `mode_name_tr` / `mode_*`: game mode translations (legacy overrides, superseded by `asset_translations` for map/playlist/pair/variant names)

### shared_matches.duckdb (v6: `shared_matches_v2.duckdb`)

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
