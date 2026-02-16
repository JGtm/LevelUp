# Scripts de Migration — Archive Historique

**ATTENTION** : Ces scripts sont destinés uniquement a la migration depuis des architectures obsoletes (SQLite, DuckDB v4). Ils sont **HORS SERVICE** post-v5.1.

## Statut

| Script | Migration | Post-v5.1 |
|--------|-----------|-----------|
| `recover_from_sqlite.py` | SQLite -> DuckDB v4 | OBSOLETE |
| `migrate_player_to_duckdb.py` | SQLite -> DuckDB v4 | OBSOLETE |
| `migrate_all_to_duckdb.py` | SQLite -> DuckDB v4 (batch) | OBSOLETE |
| `migrate_metadata_to_duckdb.py` | SQLite -> DuckDB v4 | OBSOLETE |
| `migrate_player_to_shared.py` | DuckDB v4 -> v5 (shared matches) | OBSOLETE |

### Autres scripts de migration

| Script | Description | Post-v5.1 |
|--------|-------------|-----------|
| `create_shared_matches_db.py` | Creation initiale shared_matches.duckdb | OBSOLETE |
| `migrate_add_columns.py` | Ajout colonnes schema | OBSOLETE |
| `migrate_game_variant_category.py` | Migration game_variant_category | OBSOLETE |
| `migrate_highlight_events.py` | Migration highlight_events | OBSOLETE |
| `migrate_highlight_events_schema.py` | Migration schema highlight_events | OBSOLETE |
| `migrate_player_match_stats.py` | Migration player_match_stats | OBSOLETE |
| `migrate_aliases_to_db.py` | Migration aliases | OBSOLETE |
| `create_compat_views.py` | Creation vues compatibilite v4/v5 | OBSOLETE |
| `remove_compat_views.py` | Suppression vues compatibilite | OBSOLETE |
| `migrate_to_v5_final.py` | Migration finale vers v5 | OBSOLETE |

## Usage

Si vous devez absolument migrer depuis SQLite ou v4, consultez `docs/ARCHITECTURE_V5.md`.

**Recommandation** : Pour tout nouveau deploiement, demarrer directement en v5 (shared matches DuckDB).

## Architecture cible (v5.1+)

- **100% DuckDB** — zero SQLite en runtime
- **100% Polars** — zero Pandas dans le code metier
- Bases : `shared_matches.duckdb` + `metadata.duckdb` + `stats.duckdb` par joueur
