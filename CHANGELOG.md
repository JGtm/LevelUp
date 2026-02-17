# Changelog

Toutes les modifications notables de ce projet sont documentées ici.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/).

## [5.1.0] - 2026-02-17

### Added

- **Module `src/ui/streamlit_modern.py`** — Wrappers compatibilité Streamlit moderne
  - `fragment_if_available` : décorateur graceful-degradation pour `@st.fragment`
  - `PLOTLY_CLEAN_CONFIG` : config Plotly sans barre d'outils
  - `plotly_chart` : wrapper avec config propre par défaut
  - `HAS_FRAGMENT`, `HAS_NAVIGATION` : détection de version
- **Module `src/ui/vectorize_helpers.py`** — Remplacement vectorisé de `map_elements()`
  - `build_mapping()` : pré-calcul dict mapping sur valeurs distinctes
  - `vectorized_apply()` : apply vectorisé via `replace_strict()`
  - `safe_int_format()`, `format_score_pair()` : expressions Polars réutilisables
- **Helpers `get_shared_matches_path()`** — Fonctions centralisées dans `src/utils/paths.py`
  - `get_shared_matches_path()` : chemin absolu vers `shared_matches.duckdb`
  - `get_shared_matches_path_from_player()` : déduction depuis path joueur
- **Script `cleanup_legacy_tables.py`** — Suppression tables obsolètes
  - 9 tables supprimées : `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`, + 4 vues `mv_*`
  - Options : `--dry-run`, `--backup`, `--all`
  - Backups automatiques dans `backups/pre_cleanup/`
- **Vue matérialisée `mv_player_matches`** — Optimisation performance v5.1
  - Pré-calcul jointures match_participants + match_registry + metadata
  - Réduction parsing SQL de 170→10 lignes par requête
  - Gain performance : -70% parsing SQL
- **Cache Repository Streamlit** — `get_cached_repository_st()` avec `@st.cache_resource(ttl=3600)`
  - Connexion DB persistante entre pages UI
  - Gain : 80ms→<20ms connexion
- **Index DuckDB Performance** — 16+ index créés sur 9 tables
  - Index composites `(xuid, match_id)`, `(match_id, xuid)`
  - Index triés sur `start_time`
- **Cache schema metadata** — `_has_column()` et `_has_shared_mp_column()` cachés
  - Évite requêtes `information_schema` répétées
- **Scripts migration bannières LEGACY** — 5 scripts marqués + README.md
  - Bannière claire "HORS SERVICE POST-V5.1"
  - Documentation dans `scripts/migration/README.md`

### Changed

- **Bump Streamlit ≥1.37.0** — Requis pour `@st.fragment` et futures migrations `st.navigation`
- **Plotly `config={"displayModeBar": False}`** — Appliqué sur 69 `st.plotly_chart` (15 fichiers)
  - Suppression barre d'outils Plotly pour une UI plus propre
- **`@fragment_if_available`** — Décorateur appliqué sur 5 pages multi-charts
  - timeseries, session_compare, win_loss, objective_analysis, career
  - Réduit le re-render au fragment seul lors d'interactions filtre
- **`match_history.py` modernisé** — Remplacement HTML custom par `st.dataframe` + `column_config`
  - Suppression dead code : `_format_score_label`, `_fmt`, `_fmt_mmr_int`
  - Virtualisation native Streamlit pour tableaux larges
- **`st.navigation` lazy loading** — 11 page closures dans `streamlit_app.py`
  - `build_navigation()` + `render_page_selector_nav()` dans `page_router.py`
  - Fallback legacy `dispatch_page()` pour Streamlit < 1.36
  - Seules les pages visitées sont importées → -60% mémoire initiale
- **Centralisation `duckdb_read_only()`** — Context manager dans `src/utils/db.py`
  - 7 fichiers migrés (career, cache_loaders, cache_filters, media_library, multiplayer, data_loader)
  - `duckdb.connect` directs : 14 → 4 (restants : sync engine, écriture légitime)
- **Réduction `st.rerun()`** — 32 → 14 dans `src/`
  - `checkbox_filter.py` : 16 reruns → 0 via callbacks `on_click`/`on_change`
  - Trio button filters : `on_click=_apply_trio_filter`
- **Sécurisation `unsafe_allow_html`** — html.escape() sur données dynamiques
  - `kpi.py` et `performance.py` : XSS protection
  - `sidebar.py` brand : HTML → `st.header()` + `st.divider()`
- **Tests non-régression modernisation** — 30 tests dans `test_8ter_modernisation.py`
  - Couverture : staticPlot, fragments, st.navigation, duckdb_read_only, st.rerun, html.escape
- **Éradication complète `map_elements()`** — 28 occurrences remplacées dans 15 fichiers
  - Remplacement par `build_mapping()` + `replace_strict()` ou expressions Polars natives
  - Fichiers : filters.py, filters_render.py, win_loss.py, last_match.py, stats.py,
    match_view_charts.py, media_library.py, teammates_helpers.py, session_compare.py,
    session_compare_charts.py, duckdb_analytics.py, match_view.py, citations.py,
    teammates_service.py, media_indexer.py
- **Migration `xuid_aliases` → `shared_matches.duckdb`** — Source unique centralisée
  - 9 fichiers migrés pour lire depuis `shared.xuid_aliases` (13 955 rows)
  - Suppression fallbacks locaux `stats.duckdb`
  - Fichiers : `aliases.py`, `xuid.py`, `multiplayer.py`, `cache_loaders.py`, `engine.py`, `_roster_loader.py`, `sessions_backfill.py`, `sync.py`, `resolve_missing_gamertags.py`
- **`_get_match_source()`** retourne maintenant un 3-tuple `(source_sql, params, uses_mv)`
  - Permet skip jointures redondantes en mode v5.1
- **8+ fonctions cache_loaders** migrées vers `get_cached_repository_st()`
  - Suppression connexions neuves redondantes
- **Jointures metadata/MMR** skippées en mode v5.1 quand `uses_mv=True`
  - RC3/RC4 : -3 LEFT JOIN sur chemin critique

### Removed

- **Tables legacy player DBs** — 9 tables par joueur, données centralisées
  - `match_stats`, `medals_earned`, `highlight_events`, `player_stats`, `xuid_aliases`
  - Vues obsolètes : `mv_match_stats_with_context`, `mv_recent_matches`, `mv_team_stats`, `mv_opponent_stats`
  - 38 528 rows libérées sur 4 joueurs
- **Références SQLite runtime** — 0 `import sqlite3` dans `src/`
- **Références `metadata.db`** — Tout migré vers `metadata.duckdb`
- **Méthode dépréciée `attach_sqlite`** — Supprimée de duckdb_engine.py

### Performance

| Métrique | v5.0 | v5.1 | Gain |
|----------|------|------|------|
| Connexion DB | 80ms | <20ms | **-75%** |
| load_matches(100) | 200ms | <80ms | **-60%** |
| Première page UI | 1500ms | <800ms | **-47%** |
| Parsing SQL/requête | 170 lignes | 10 lignes | **-94%** |

---

## [5.0.0] - 2026-02-15

### Added

- **Architecture shared_matches.duckdb** — Base de données partagée centralisant les matchs de tous les joueurs
  - 6 tables : `match_registry`, `match_participants`, `highlight_events`, `medals_earned`, `xuid_aliases`, séquence `highlight_events_id_seq`
  - 14 index optimisés (match_id, xuid, start_time, composites)
  - Schéma DDL complet : `scripts/migration/schema_v5.sql`
  - Documentation : `docs/SHARED_MATCHES_SCHEMA.md`
- **Migration v4 → v5** — Scripts de migration incrémentale par joueur
  - `scripts/migration/create_shared_matches_db.py` : création de la DB partagée
  - `scripts/migration/migrate_player_to_shared.py` : migration par joueur
  - Résultat : 1289 matchs migrés, 285 partagés (22.1%), 0 orphelins
- **Détection matchs partagés dans Sync Engine** — Sync allégée pour matchs déjà connus
  - `_process_known_match()` : enrichissement personnel uniquement (économie 1-2 appels API/match)
  - `_process_new_match()` : sync complète vers shared (registry + participants + events + medals)
  - `extract_all_medals()` : extraction des médailles de TOUS les joueurs du match
  - `extract_match_registry_data()` : extraction données communes du match
- **ATTACH multi-DB dans DuckDBRepository** — Lecture transparente depuis `shared_matches.duckdb`
  - `shared_db_path` auto-détecté ou configurable
  - Queries natives `shared.match_participants`, `shared.match_registry`, `shared.medals_earned`
  - Propagation dans la factory repository
- **Sous-requête `_get_match_source()`** — Abstraction permettant à toutes les pages UI de lire depuis shared sans modification
- **Optimisations API Sync v5**
  - Parallélisation appels API skill + events (`asyncio.gather`)
  - Batching des insertions DB (commit tous les 10 matchs)
  - Performance scores calculés en batch post-sync
  - Rate limit optimisé (10 req/s, parallel_matches=5)
- **Citations DuckDB-first** — Nouveau système de citations stockées par match
  - `CitationEngine` : moteur de calcul et agrégation SQL
  - Table `citation_mappings` dans `metadata.duckdb` : 14 règles (8 existantes + 6 réintégrées)
  - Table `match_citations` dans chaque `stats.duckdb` joueur
  - Backfill CLI : `--citations` / `--force-citations` dans `scripts/backfill_data.py`
  - 6 citations objectives réintégrées : Défenseur du drapeau, Je te tiens !, Sus au porteur du drapeau, Partie prenante, À la charge, Annexion forcée
  - Colonne `enabled` dans `citation_mappings` pour désactivation sans suppression
  - Support V5 (shared_matches) dans `CitationEngine` avec fallback V4
  - Documentation : `docs/CITATIONS.md`
- **Framework de test MockStreamlit** — Fixture `MockStreamlit` dans `conftest.py` pour tester les pages UI en mode headless
- **+946 tests** ajoutés (S1→S7ter) — total 2768 passed, 0 failed, 38 skipped
- **Script de nettoyage post-migration** — `scripts/cleanup_player_dbs_v5.py`
  - Supprime les tables redondantes des player DBs après migration v5 (match_stats, match_participants, highlight_events, medals_earned)
  - Mode --dry-run pour simulation sans modification
  - Backup optionnel avant nettoyage
  - Validation automatique de l'existence de shared_matches.duckdb
  - VACUUM automatique pour récupération d'espace disque (-85% de taille en moyenne)
  - Documentation : `docs/CLEANUP_V5.md`
- **Documentation** : `docs/SHARED_MATCHES_SCHEMA.md`, `docs/SYNC_OPTIMIZATIONS_V5.md`, `docs/TESTING_V5.md`, `docs/ARCHITECTURE_V5.md`, `docs/MIGRATION_V4_TO_V5.md`, `docs/CLEANUP_V5.md`

### Changed

- **`DuckDBSyncEngine`** refactoré pour écrire dans `shared_matches.duckdb` (matchs, participants, events, médailles)
- **`DuckDBRepository`** refactoré avec ATTACH `shared_matches.duckdb` en read-only
  - `load_match_participants()` → lecture depuis `shared.match_participants`
  - `load_highlight_events()` → lecture depuis `shared.highlight_events`
  - `load_medals_for_match()` → lecture depuis `shared.medals_earned`
  - `load_matches()` → JOIN `shared.match_participants` + `shared.match_registry` + `player_match_enrichment`
- **Toutes les pages UI** utilisent `_get_match_source()` au lieu de `match_stats` directement
- **`render_h5g_commendations_section()`** utilise `CitationEngine` (agrégation SQL, ~90% plus rapide)
- **`render_citations_page()`** simplifié — ne pré-agrège plus les médailles/stats pour les citations
- **Filtrage des citations** piloté par `citation_mappings.enabled` (plus besoin du JSON d'exclusion)
- **Version `pyproject.toml`** bumpée de 3.0.0 à 5.0.0
- **Statut projet** : Development Status 4-Beta → 5-Production/Stable

### Removed

- **VIEWs de compatibilité v4** supprimées (`scripts/migration/remove_compat_views.py`)
- **Données dupliquées** dans les player DBs : `match_participants`, `highlight_events`, `medals_earned` centralisés dans shared
- **Shim `src/db/migrations.py`** — déprécié, supprimé en faveur de `src.data.sync.migrations`
- `CUSTOM_CITATION_RULES` dict (ancien `commendations.py`)
- `_compute_custom_citation_value()` (itérations lentes, remplacé par SQL)
- `load_h5g_commendations_tracking_rules()` (remplacé par `citation_mappings` DuckDB)
- Constantes `DEFAULT_H5G_TRACKING_ASSUMED_PATH` / `DEFAULT_H5G_TRACKING_UNMATCHED_PATH`
- Dépendance aux fichiers JSON de tracking commendations
- Logique d'exclusion JSON dans `render_h5g_commendations_section()`

### Fixed

- **Tests flaky Windows** : `tmp_dir` → `tmp_path` pour éviter DuckDB `WinError 32` (file locking)
- **Tests lazy_loading** : mode v4 forcé pour compatibilité

### Performance

| Métrique | v4 | v5 | Gain |
|----------|----|----|------|
| Stockage (4 joueurs) | 800 MB | 250 MB | **-69%** |
| DB size par joueur | 200 MB | 30 MB | **-85%** |
| Appels API (sync 4 joueurs) | 12 000 | 3 300 | **-72%** |
| Temps sync (100 matchs) | 45 min | 12 min | **-73%** |
| Temps/match (partagé) | 16s | 0.5s | **-97%** |
| Temps/match (nouveau) | 16s | 2-3s | **-81%** |
