# Project Map - LevelUp

> Ce fichier est la cartographie vivante du projet. L'agent IA doit le consulter et le mettre à jour.

## ⚠️ Limitations Connues

**IMPORTANT** : Consulter `.ai/API_LIMITATIONS.md` avant d'implémenter des fonctionnalités liées aux armes.

- **Weapon Stats par arme** : NON DISPONIBLE dans l'API (vérifié 2026-02-02)
- **Film Chunks** : NON EXPLOITABLES pour l'identification d'armes
- **SQLite** : PROSCRIT - Tout le code doit utiliser DuckDB uniquement. Aucun fallback SQLite (0 `import sqlite3` dans src/)
- **Pandas** : PROSCRIT - Utiliser **Polars** uniquement pour DataFrames/séries. Audit : `.ai/PANDAS_TO_POLARS_AUDIT.md`, `.ai/CONSOLIDATED_AUDITS_AND_ROADMAP.md`

## Architecture Multi-Joueurs (v5.1)

En v5.1, les stats coéquipiers sont chargées depuis `shared.match_participants` (plus besoin d'accéder aux DBs individuelles).

Le sync écrit dans les player DBs : `player_match_enrichment` + `personal_score_awards` uniquement.

## État Actuel (2026-02-25) — v5.3 Release

### Historique des versions

- **v5.1** : Architecture Shared DB, éradication SQLite/Pandas, cleanup tables legacy ✅
- **v5.2** : Filtres intent-based, Stats PvE Firefight (`shared_pve.duckdb`), Scoreboard, palette Okabe-Ito ✅
- **v5.3** : LUSR/CSR TrueSkill 2 per-groupe, Notifications Discord, 20 tests corrigés ✅

### Architecture v5.3

```
data/
├── players/                    # Enrichissements uniquement (~4 MB/joueur)
│   └── {gamertag}/
│       ├── stats.duckdb       # player_match_enrichment, awards, antagonists, citations,
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
- `src/data/repositories/duckdb_repo.py` : Repository principal DuckDB
- `src/data/repositories/factory.py` : Factory pattern
- `src/data/sync/engine.py` : Moteur de synchronisation
- `src/data/media_indexer.py` : Indexation médias (scan delta, associations, thumbnails), chargement pour UI

### Analyse
- `src/analysis/killer_victim.py` : Calcul antagonistes
- `src/analysis/antagonists.py` : Agrégation rivalités
- `src/analysis/sessions.py` : Détection sessions
- `src/analysis/performance_score.py` : Score de performance (percentile 0-100)
- `src/analysis/playlist_groups.py` : 6 groupes Halo Infinite (ranked/arena/btb/tactical/social/fun), détection par `pair_name`/`playlist_name` — v5.3
- `src/analysis/skill_rating_config.py` : Constantes TrueSkill 2 (K_ELO, tiers Bronze→Onyx, COMPOSITE_WEIGHTS, get_tier_for_rating) — v5.3
- `src/analysis/skill_rating.py` : Algorithme LUSR — `PlayerState` par groupe, `compute_composite_score()`, `trueskill_update()` Elo-style, `compute_skill_ratings_batch()` séquentiel — v5.3
- `src/analysis/skill_rating_calibration.py` : Calibration des poids COMPOSITE_WEIGHTS via grid search vs `team_mmr` API — v5.3

### UI
- `src/ui/pages/` : Pages du dashboard (career.py ajouté Sprint 3B)
- `src/ui/components/` : Composants réutilisables (career_progress_circle.py ajouté Sprint 3B)
- `src/ui/streamlit_modern.py` : Wrappers compatibilité Streamlit moderne (fragment_if_available, PLOTLY_CLEAN_CONFIG) — Sprint 8ter
- `src/ui/vectorize_helpers.py` : Helpers vectorisation Polars (build_mapping, replace map_elements) — Sprint 8ter
- `src/ui/filter_state.py` : Filtres intent-based v5.2 (`FilterPreferences`, `_detect_filter_mode()`, `reconcile_filter_prefs()`, persist JSON)
- `src/utils/discord_notifier.py` : Notifications Discord post-sync/backfill (failsafe, stdlib uniquement) — v5.3
- `src/visualization/` : Graphiques Plotly (palette Okabe-Ito v5.2, `plot_lusr_timeseries()` v5.3)

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
| `antagonists` | Top killers/victimes agrégés |
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

| Document | Contenu |
|----------|---------|
| `docs/INSTALL.md` | Installation |
| `docs/CONFIGURATION.md` | Configuration |
| `docs/ARCHITECTURE.md` | Architecture technique |
| `docs/DATA_ARCHITECTURE.md` | Architecture données |
| `docs/SYNC_GUIDE.md` | Guide synchronisation |
| `docs/BACKUP_RESTORE.md` | Backup/Restore |
| `docs/FAQ.md` | Questions fréquentes |

### Documentation IA (.ai/)

| Document | Contenu |
|----------|---------|
| `.ai/DATA_KILLER_VICTIM.md` | Guide killer/victim et antagonistes |
| `.ai/DATA_MATCH_RANK.md` | Rang d'un joueur lors d'un match (API vs recalcul, tie-breaker) |
| `.ai/sprints/SPRINT_GAMERTAG_ROSTER_FIX.md` | Sprint correction gamertags et roster |
| `.ai/API_LIMITATIONS.md` | Limitations connues de l'API |

## Problèmes Connus

Aucun problème bloquant connu.

## État technique final (v5.3)

- **3323 tests** passent, 0 échecs
- **Architecture DuckDB v5.3** : shared_matches + shared_pve + player enrichments
- **Polars** comme moteur DataFrame (0 Pandas dans code métier)
- **0 SQLite** dans le code runtime
- **Streamlit ≥1.37** avec @st.fragment, st.navigation, column_config
- **Taille player DB** : ~4 MB (vs ~30 MB en v5.0)

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

**2026-02-25** : **v5.3.0** — LUSR/CSR TrueSkill 2 per-groupe, Notifications Discord, 3323 tests
**2026-02-20** : **v5.2.0** — Filtres intent-based, Stats PvE shared_pve.duckdb, Scoreboard, Okabe-Ito
**2026-02-17** : **v5.1.0 Release** — Documentation finale, archivage, release tag
