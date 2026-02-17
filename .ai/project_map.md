# Project Map - LevelUp

> Ce fichier est la cartographie vivante du projet. L'agent IA doit le consulter et le mettre à jour.

## ⚠️ Limitations Connues

**IMPORTANT** : Consulter `.ai/API_LIMITATIONS.md` avant d'implémenter des fonctionnalités liées aux armes.

- **Weapon Stats par arme** : NON DISPONIBLE dans l'API (vérifié 2026-02-02)
- **Film Chunks** : NON EXPLOITABLES pour l'identification d'armes
- **SQLite** : PROSCRIT - Tout le code doit utiliser DuckDB v4 uniquement. Aucun fallback SQLite. Audit : `.ai/SQLITE_TO_DUCKDB_AUDIT.md`
- **Pandas** : PROSCRIT - Utiliser **Polars** uniquement pour DataFrames/séries. Audit : `.ai/PANDAS_TO_POLARS_AUDIT.md`, `.ai/CONSOLIDATED_AUDITS_AND_ROADMAP.md`

## ⚠️ RÈGLE CRITIQUE : Chargement Multi-Joueurs

**NE JAMAIS** passer le xuid d'un coéquipier à `load_df_optimized(db_path, xuid)` !
Le xuid est IGNORÉ pour DuckDB v4 et ça charge toujours depuis `db_path`.

**TOUJOURS** utiliser `_load_teammate_stats_from_own_db(gamertag, match_ids, db_path)`
pour charger les stats d'un coéquipier depuis **SA propre DB**.

```python
# ❌ FAUX - Charge depuis db_path (joueur principal), pas le coéquipier
teammate_df = load_df_optimized(db_path, teammate_xuid)

# ✅ CORRECT - Charge depuis data/players/{gamertag}/stats.duckdb
teammate_df = _load_teammate_stats_from_own_db(gamertag, match_ids, db_path)
```

Voir `src/ui/pages/teammates.py` pour l'implémentation de référence.

## État Actuel (2026-02-02)

### Phases Complétées

- **Phase 1** : Stabilisation architecture hybride ✅
- **Phase 2** : Migration vers DuckDB Unifiée ✅
- **Phase 3** : Enrichissement des Données (antagonistes) ✅
- **Phase 4** : Optimisations Avancées ✅
  - Vues matérialisées (`mv_map_stats`, `mv_mode_category_stats`, etc.)
  - Lazy loading et pagination
  - Backup/Restore Parquet avec compression Zstd
  - Partitionnement temporel
  - Refonte système de synchronisation (DuckDBSyncEngine)
- **Phase 5** : Enrichissement Visuel & API ✅
  - Career Rank & Stats Armes
  - Correctifs modes/playlists
  - Graphes Radar & Étiquettes
  - Nouvelles représentations statistiques
  - Watcher/Daemon Thumbnails
- **Phase 6** : Documentation & Branding "LevelUp" ✅
  - README.md complet
  - Guides d'installation et configuration
  - Documentation technique mise à jour
  - Branding LevelUp appliqué

### Architecture Cible v4

```
data/
├── players/                    # Données par joueur
│   └── {gamertag}/
│       ├── stats.duckdb       # DB DuckDB persistée
│       └── archive/           # Archives temporelles
│           ├── matches_2023.parquet
│           └── archive_index.json
├── warehouse/
│   └── metadata.duckdb        # Référentiels partagés
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
- `src/analysis/performance_score.py` : Score de performance

### UI
- `src/ui/pages/` : Pages du dashboard (career.py ajouté Sprint 3B)
- `src/ui/components/` : Composants réutilisables (career_progress_circle.py ajouté Sprint 3B)
- `src/ui/streamlit_modern.py` : Wrappers compatibilité Streamlit moderne (fragment_if_available, PLOTLY_CLEAN_CONFIG) — Sprint 8ter
- `src/ui/vectorize_helpers.py` : Helpers vectorisation Polars (build_mapping, replace map_elements) — Sprint 8ter
- `src/visualization/` : Graphiques Plotly

## Tables DuckDB

### Base Joueur (stats.duckdb)

| Table | Description |
|-------|-------------|
| `match_stats` | Faits des matchs |
| `medals_earned` | Médailles par match |
| `teammates_aggregate` | Stats coéquipiers |
| `antagonists` | Top killers/victimes |
| `player_match_stats` | Données MMR/skill |
| `highlight_events` | Événements film |
| `xuid_aliases` | Mapping XUID→Gamertag |
| `killer_victim_pairs` | Paires killer→victim avec timestamps |
| `match_participants` | Tous les joueurs par match : xuid, team_id, outcome, rank, score, kills, deaths, assists. Identifiant = xuid ; gamertag via xuid_aliases. Voir .ai/MATCH_PARTICIPANTS.md. |
| `career_progression` | Historique rangs |
| `sync_meta` | Métadonnées sync |
| `media_files` | Fichiers médias indexés (status, thumbnail_path, capture_end_utc) |
| `media_match_associations` | Média ↔ match ↔ xuid (map_name, match_id) |
| `mv_*` | Vues matérialisées |

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
| `duckdb` | >=0.10.0 | Moteur unique |
| `polars` | >=0.20.0 | DataFrames |
| `pydantic` | >=2.5.0 | Validation |
| `streamlit` | >=1.28.0 | Interface |

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

### 🔴 CRITIQUE - Données Manquantes en BDD (2026-02-05)

**Priorité** : HAUTE  
**Status** : 🔍 EN EXPLORATION

**Problèmes identifiés** :
1. Noms des cartes, modes et playlists non enregistrés (`playlist_name`, `map_name`, `pair_name`, `game_variant_name` sont NULL)
2. Noms des joueurs par match non récupérés correctement
3. Joueurs non affectés à l'équipe adverse
4. Nom de l'équipe adverse non récupéré
5. Valeurs "attendues" pour frags et morts non récupérées (`kills_expected`, `deaths_expected`, `assists_expected` sont NULL)

**Commit de référence** : `1a6115007272619985485be0f94cc69e6be5c2d2` (fonctionnait correctement)

**Documentation** :
- Diagnostic : `.ai/diagnostics/CRITICAL_DATA_MISSING_2026-02-05.md`
- Exploration : `.ai/explore/CRITICAL_DATA_MISSING_EXPLORATION.md`

**Fichiers concernés** :
- `src/data/sync/transformers.py` : Extraction des données depuis JSON
- `src/data/sync/engine.py` : Synchronisation et insertion en BDD
- `src/data/repositories/duckdb_repo.py` : Récupération depuis BDD

## Sprint en Cours

**Sprint Final** : Tous les sprints S0-S12 sont **livrés** ✅
📄 `.ai/PLAN_UNIFIE.md`

### Sprints livrés
- **S0** ✅ Bugs urgents (tri session, nettoyage filtres)
- **S1** ✅ Nettoyage scripts (113→16 actifs) + archivage .ai/
- **S2** ✅ Migration Pandas→Polars core (performance_score, backfill, sessions, killer_victim)
- **S3** ✅ Damage participants + Page Carrière
- **S4** ✅ Médianes, Frags, Modes, Médias, Coéquipiers refonte
- **S5** ✅ Score de Performance v4
- **S6** ✅ Nouvelles stats Phase 1 (Timeseries + Corrélations)
- **S7** ✅ Nouvelles stats Phase 2-3 (V/D + Dernier match)
- **S8** ✅ Nouvelles stats Phase 4 (Coéquipiers comparaisons)
- **S9** ✅ Suppression code legacy + Migration Pandas complète
- **S10** ✅ Nettoyage données + Refactoring backfill
- **S11** ✅ Finalisation, tests d'intégration, documentation
- **S12** ✅ Heatmap d'Impact & Cercle d'Amis

### État technique final
- **1065+ tests** passent (hors intégration)
- **Architecture DuckDB v4** unifiée
- **Polars** comme moteur DataFrame (migration Pandas complète)
- **Backfill modulaire** (scripts/backfill/)
- **15 tests d'intégration** nouvelles stats

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

**2026-02-12** : **v4.1 Release** — Sprints 11-12 livrés (tests intégration, documentation, heatmap impact)
**2026-02-11** : Sprints 3A+3B livrés (damage participants + page Carrière) + Sprint 4 partiel (4.0-4.2)
**2026-02-10** : Sprints 0-2 livrés (bugs, nettoyage, migration Polars core)
**2026-02-05** : Sprint Gamertag & Roster Fix + Documentation killer_victim
**2026-02-01** : Phase 6 terminée - Documentation & Branding "LevelUp"
