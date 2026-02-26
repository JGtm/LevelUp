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

- `shared.match_registry`: one row per match
- `shared.match_participants`: per-player stats for all matches
- player `player_match_enrichment`: performance_score, session_id, etc.
