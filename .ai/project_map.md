# Project Map - LevelUp

> Ce fichier est la cartographie vivante du projet. L'agent IA doit le consulter et le mettre à jour.

> 📋 **Tâches et TODO centralisés** : voir `.ai/BACKLOG.md`

## ⚠️ Limitations Connues

**IMPORTANT** : Consulter `.ai/API_LIMITATIONS.md` avant d'implémenter des fonctionnalités liées aux armes.

- **Weapon Stats par arme** : NON DISPONIBLE dans l'API (vérifié 2026-02-02)
- **Film Chunks** : NON EXPLOITABLES pour l'identification d'armes
- **SQLite** : PROSCRIT - Tout le code doit utiliser DuckDB uniquement. Aucun fallback SQLite (0 `import sqlite3` dans src/)
- **Pandas** : PROSCRIT - Utiliser **Polars** uniquement pour DataFrames/séries. Audit : `.ai/PANDAS_TO_POLARS_AUDIT.md`, `.ai/CONSOLIDATED_AUDITS_AND_ROADMAP.md`

## Architecture Multi-Joueurs (v5.1)

En v5.1, les stats coéquipiers sont chargées depuis `shared.match_participants` (plus besoin d'accéder aux DBs individuelles).

Le sync écrit dans les player DBs : `player_match_enrichment` + `personal_score_awards` uniquement.

## État Actuel (2026-03-05) — v5.4 Refactoring

### Historique des versions

- **v5.1** : Architecture Shared DB, éradication SQLite/Pandas, cleanup tables legacy ✅
- **v5.2** : Filtres intent-based, Stats PvE Firefight (`shared_pve.duckdb`), Scoreboard, palette Okabe-Ito ✅
- **v5.3** : LUSR/CSR TrueSkill 2 per-groupe, Notifications Discord, 20 tests corrigés ✅
- **v5.4** : i18n split, logging centralisé, SyncScope cleanup, refactoring modules >500L (Phases 0-6, 72 sous-modules) ✅

### Architecture v5.3

```
data/
├── players/                    # Enrichissements uniquement (~4 MB/joueur)
│   └── {gamertag}/
│       ├── stats.duckdb       # player_match_enrichment, awards, citations,
│       │                      #   match_skill_rank (LUSR/CSR par match)
│       └── archive/           # Archives temporelles
├── warehouse/
│   ├── metadata.duckdb        # Référentiels (playlists, maps, medals, ranks)
│   ├── shared_matches.duckdb  # Matchs centralisés (registry, participants, events, medals)
│   └── shared_pve.duckdb      # Stats PvE Firefight (pve_match_stats) — v5.2
└── backups/                   # Backups Parquet
```

## Modules Clés

### Accès aux Données
- `src/data/repositories/duckdb_repo.py` : Repository principal DuckDB (splitté: `_awards_repo`, `_diagnostic_repo`, `_legacy_compat`, `_match_queries_helpers`, `_match_queries_polars`, `_metadata_resolution`, `_schema_introspection`, `_archives_repo`, `_events_repo`, `_medals_repo`, `_gamertag_resolver`)
- `src/data/repositories/factory.py` : Factory pattern
- `src/data/sync/engine.py` : Moteur de synchronisation (8 mixins MRO : `_shared_writes`, `_performance`, `_skill_rating`, `_career`, `_aggregates`, `_match_processing`, `_engine_connections`, `_engine_schema` + `_protocol.py`)
- `src/data/media_indexer.py` : Indexation médias (splitté: `media_helpers`, `media_loaders`, `media_thumbnails`)

### Analyse
- `src/analysis/killer_victim.py` : Calcul antagonistes (splitté: `_killer_victim_polars`, `_kv_types`)
- `src/analysis/antagonists.py` : Agrégation rivalités
- `src/analysis/sessions.py` : Détection sessions
- `src/analysis/performance_score.py` : Score de performance (splitté: `_performance_relative`, `_performance_session`)
- `src/analysis/objective_participation.py` : Participation objectifs (splitté: `_objective_helpers`, `_objective_profile`, `_objective_summary`)
- `src/data/sync/transformers/` : Package (7 sous-modules: `_helpers`, `_match`, `_skill`, `_events`, `_medals`, `_personal_scores`, `_pve`)

### Visualisation & UI (splits phases 4-6)
- `src/visualization/antagonist_charts.py` : Charts antagonistes (splitté: `_antagonist_kv`, `_antagonist_duels`)
- `src/ai/rag.py` : RAG IA (splitté: `_rag_models`, `_rag_github`, `_rag_chunker`)
- `src/data/repositories/refdata.py` : Référentiels (splitté: `_refdata_personal_scores`)
- `src/app/cache_filters.py` : Cache & filtres (splitté: `_cache_loading`, `_cache_sessions`)
- `src/app/filters_render.py` : Rendu filtres (splitté: `_filters_apply`, `_filters_period`, `_filters_session`, `_filters_cascade`)
- `src/visualization/session_compare_charts.py` : Comparaison sessions (splitté: `_session_compare_history`)

### Infrastructure transversale (v5.4)
- `src/data/sync/_protocol.py` : `_SyncProtocol` — contrat Protocol pour les 8 mixins engine
- `src/app/_page_context.py` : `PageContext` + `MatchViewParams` — types réels pour pages
- `src/app/session_keys.py` : `SessionKeys` / `SK` — clés session_state centralisées
- `src/data/query/_sql_fragments.py` : `WIN_RATE_EXPR`, `IS_WIN`, `IS_LOSS` centralisés
- `src/analysis/playlist_groups.py` : 6 groupes Halo Infinite — v5.3
- `src/analysis/skill_rating.py` / `skill_rating_config.py` / `skill_rating_calibration.py` : LUSR/CSR TrueSkill 2 — v5.3

### UI
- `src/ui/pages/` : Pages du dashboard
- `src/ui/pages/teammates_views.py` : Vues coéquipiers (splitté: `_teammates_trio.py`)
- `src/ui/components/radar_chart.py` : Radar charts (splitté: `_radar_participation`, `_radar_teammates`)
- `src/ui/cache_loaders.py` : Cache Streamlit (splitté: `_cache_core`, `_cache_queries`)
- `src/ui/sync.py` : UI sync (splitté: `_sync_utils`, `_sync_indicator`, `_sync_duckdb_ops`)
- `src/ui/streamlit_modern.py` : Wrappers Streamlit moderne
- `src/ui/filter_state.py` : Filtres intent-based v5.2
- `src/utils/discord_notifier.py` : Notifications Discord (splitté: `_discord_embed`, `_discord_queries`) — v5.3
- `src/utils/safe_types.py` / `async_compat.py` / `env.py` : Utilitaires partagés — v5.4
- `src/visualization/` : Graphiques Plotly
- `src/visualization/timeseries_combat.py` : Séries temporelles (splitté: `_timeseries_helpers`, `_timeseries_progression`)

## Tables DuckDB

### shared_matches.duckdb (centralisée)

| Table | Description |
|-------|-------------|
| `match_registry` | Registre central (1 ligne par match unique) |
| `match_participants` | Stats de tous les joueurs (31 colonnes, incl. MMR) |
| `highlight_events` | Événements filmés de tous les matchs |
| `medals_earned` | Médailles de tous les joueurs |
| `killer_victim_pairs` | Paires killer→victim |
| `xuid_aliases` | Mapping global XUID→Gamertag |

### Base Joueur stats.duckdb (v5.3 — enrichissements uniquement)

> 8 tables supprimées (v5.1) : match_stats, match_participants, highlight_events,
> medals_earned, killer_victim_pairs, player_match_stats, xuid_aliases, teammates_aggregate

| Table | Description |
|-------|-------------|
| `player_match_enrichment` | performance_score, session_id, is_with_friends (**SEULE table match**) |
| `personal_score_awards` | Awards objectifs (PersonalScores API) |
| `match_citations` | Citations calculées par match |
| `career_progression` | Historique rangs |
| `media_files` | Fichiers médias indexés (status, thumbnail_path, capture_end_utc) |
| `media_match_associations` | Média ↔ match ↔ xuid (map_name, match_id) |
| `sessions` | Sessions groupées |
| `sync_meta` | Métadonnées sync |
| `match_skill_rank` | Rating LUSR/CSR par match (PK=match_id — exclusif LUSR ou CSR) — **v5.3** |
| `mv_*` | Vues matérialisées (mv_player_matches, mv_map_stats, etc.) |

### Base Métadonnées (metadata.duckdb)

| Table | Description |
|-------|-------------|
| `playlists` | Définitions playlists |
| `game_modes` | Modes de jeu (FR/EN) |
| `medal_definitions` | Référentiel médailles |
| `career_ranks` | Rangs de carrière |

## Scripts Utilitaires

| Script | Description |
|--------|-------------|
| `scripts/sync.py` | Synchronisation SPNKr |
| `scripts/backup_player.py` | Export Parquet Zstd |
| `scripts/restore_player.py` | Import depuis backup |
| `scripts/archive_season.py` | Archivage temporel |
| `scripts/migrate_*.py` | Scripts de migration |

## Dépendances Critiques

| Package | Version | Usage |
|---------|---------|-------|
| `duckdb` | >=1.4.0 | Moteur unique |
| `polars` | >=1.38.0 | DataFrames |
| `pydantic` | >=2.5.0 | Validation |
| `streamlit` | >=1.37.0 | Interface (@st.fragment, st.navigation) |

## Points d'Entrée

- `streamlit_app.py` : Application principale
- `launcher.py` : Lanceur CLI

## Documentation

> Convention :
> - `docs/` = documentation EN (publique)
> - `docs/FR/` = sources FR
> - `docs/archive/` = docs conservées mais non traduites

| Document | Contenu |
|----------|---------|
| `docs/INSTALL.md` | Installation |
| `docs/CONFIGURATION.md` | Configuration |
| `docs/COMMANDS.md` | Commandes usuelles |
| `docs/ARCHITECTURE_V5.md` | Architecture DuckDB v5 |
| `docs/SYNC_GUIDE.md` | Guide synchronisation |
| `docs/BACKUP_RESTORE.md` | Backup/Restore |
| `docs/TESTING_V5.md` | Tests (v5) |
| `docs/FAQ.md` | Questions fréquentes |
| `docs/COMMENDATIONS.md` | Commendations (ex "citations") |
| `docs/COMMENDATIONS_REFERENCE.md` | Référentiel complet des commendations |

### Documentation IA (.ai/)

| Document | Contenu |
|----------|---------|
| `.ai/DATA_KILLER_VICTIM.md` | Guide killer/victim et antagonistes |
| `.ai/DATA_MATCH_RANK.md` | Rang d'un joueur lors d'un match (API vs recalcul, tie-breaker) |
| `.ai/sprints/SPRINT_GAMERTAG_ROSTER_FIX.md` | Sprint correction gamertags et roster |
| `.ai/API_LIMITATIONS.md` | Limitations connues de l'API |

## Problèmes Connus

Aucun problème bloquant connu.

## État technique (v5.4)

- **3693 tests** passent, 0 échecs
- **Architecture DuckDB v5.3** : shared_matches + shared_pve + player enrichments
- **Polars** comme moteur DataFrame (0 Pandas dans code métier)
- **0 SQLite** dans le code runtime
- **Streamlit ≥1.37** avec @st.fragment, st.navigation, column_config
- **Taille player DB** : ~4 MB (vs ~30 MB en v5.0)
- **Refactoring v5.4** : 72 nouveaux sous-modules (phases 0-6), 191 violations restantes (25 modules >500L, 166 fonctions >80L)

## Exploration Complète du Projet

Une exploration détaillée de tout le projet (modules, scripts, tests, docs) a été refaite le **2026-02-05** :

📄 **`.ai/explore/PROJECT_EXPLORE_2026-02-05.md`**

Contenu :
- Vue d’ensemble (stack, points d’entrée, règles critiques)
- Arborescence `src/` complète (rôle de chaque module : app, data, ui, analysis, visualization, db, ai, utils)
- Scripts catégorisés (~100) : sync, backup, migration, backfill, diagnostic, analyse/recherche, API, tests
- Tests listés par thème
- Documentation `docs/` et `.ai/`
- Structure données et config
- Flux d’entrée et dépendances
- Référence aux audits (SQLite, Pandas→Polars, problèmes connus)

Consulter ce fichier pour une cartographie exhaustive ; le présent `project_map.md` reste la cartographie vivante (état, problèmes, sprints).

## Dernière Mise à Jour

**2026-03-05** : **v5.4 Refactoring** — Phases 0-6 split modules >500L, 72 sous-modules, 3693 tests, baseline 191 violations
**2026-02-25** : **v5.3.0** — LUSR/CSR TrueSkill 2 per-groupe, Notifications Discord, 3323 tests
**2026-02-20** : **v5.2.0** — Filtres intent-based, Stats PvE shared_pve.duckdb, Scoreboard, Okabe-Ito
**2026-02-17** : **v5.1.0 Release** — Documentation finale, archivage, release tag
