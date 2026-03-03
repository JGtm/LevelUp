# Phase 1 — Nettoyage du code mort

> **Branche** : `refactor/phase1-dead-code-cleanup`
> **Date de début** : 2 mars 2026
> **Objectif** : Supprimer le code mort, les duplications triviales et les vestiges legacy (~1 500 lignes)

---

## Checklist des tâches

### 1. Supprimer `src/app/routing.py` (module mort)
- [x] Retirer les imports/réexports de `src/app/__init__.py`
- [x] Supprimer le fichier `src/app/routing.py`
- [x] Adapter les tests `test_app_module.py`, `test_pending_page_navigation_regressions.py`, `test_query_params_routing_regressions.py`
- **Impact** : ~252 lignes supprimées + 3 fichiers tests adaptés

### 2. Nettoyer `src/ui/cache_social.py` (fonctions mortes)
- [x] Supprimer `cached_load_friends` (retourne toujours `[]`)
- [x] Supprimer `cached_get_match_session_info` (retourne toujours `None`)
- [x] Retirer les réexports de `src/ui/cache_loaders.py` et `src/ui/cache.py`
- [x] Garder `cached_load_top_teammates_optimized` (seule fonction utile)
- **Impact** : ~40 lignes mortes supprimées

### 3. Nettoyer `src/ui/multiplayer.py` (fonctions mortes)
- [x] Supprimer `is_multi_player_db` (retourne toujours `False`)
- [x] Supprimer `list_players_in_db` (retourne toujours `[]`)
- [x] Supprimer `get_unique_xuids_from_matchstats` (retourne toujours `[]`)
- [x] Supprimer `get_player_display_name` (retourne toujours `None`)
- [x] Supprimer `render_player_selector` (jamais exécuté)
- [x] Vérifier les imports avant suppression
- **Impact** : ~100 lignes mortes supprimées

### 4. Supprimer les branches SQLite legacy dans `src/ui/cache_loaders.py`
- [x] Retirer les branches `# Legacy SQLite non supporté` dans chaque fonction cachée
- **Impact** : ~100 lignes mortes supprimées

### 5. Supprimer les 13 aliases inutiles dans `streamlit_app.py`
- [x] Supprimer les `_xxx = xxx` (L163-175)
- [x] Remplacer les usages des aliases par les fonctions originales dans le même fichier
- **Impact** : ~13 lignes + clarté

### 6. Supprimer les guards `POLARS_AVAILABLE` (7 fichiers)
- [x] `src/analysis/cumulative.py` (8 checks)
- [x] `src/analysis/win_streaks.py` (4 checks)
- [x] `src/analysis/objective_participation.py` (1 check)
- [x] `src/visualization/performance.py` (5 checks)
- [x] `src/visualization/antagonist_charts.py` (4 checks)
- [x] `src/visualization/objective_charts.py` (3 checks)
- **Impact** : ~60 lignes de boilerplate supprimées

### 7. Nettoyer `src/data/repositories/factory.py`
- [x] Supprimer `is_migration_complete()` (retourne toujours `True`)
- [x] Supprimer `get_default_mode()` (retourne toujours `DUCKDB`)
- [x] Garder `RepositoryMode` (encore importé ailleurs — à retirer en Phase 2)
- **Impact** : ~25 lignes mortes supprimées

### 8. Extraire `_to_polars()` dans `src/utils/polars_compat.py`
- [x] Créer `src/utils/polars_compat.py` avec la version robuste
- [x] Remplacer les 9 copies par un import unique
- **Impact** : 9 copies → 1, ~50 lignes de duplication éliminées

---

## Validation

- [x] `python -m pytest --ignore=tests/integration -q` passe (**3500 passed, 66 skipped**)
- [ ] L'app démarre (`streamlit run streamlit_app.py`)

---

## Fichiers modifiés

| Fichier | Action |
|---------|--------|
| `src/app/routing.py` | **Supprimé** |
| `src/app/__init__.py` | Imports routing retirés |
| `src/ui/cache_social.py` | Fonctions mortes supprimées |
| `src/ui/cache_loaders.py` | Branches SQLite + réexports nettoyés |
| `src/ui/cache.py` | Réexports morts retirés |
| `src/ui/multiplayer.py` | Fonctions mortes supprimées |
| `streamlit_app.py` | Aliases supprimés |
| `src/analysis/cumulative.py` | Guards POLARS_AVAILABLE supprimés |
| `src/analysis/win_streaks.py` | Guards POLARS_AVAILABLE supprimés |
| `src/analysis/objective_participation.py` | Guards POLARS_AVAILABLE supprimés |
| `src/visualization/performance.py` | Guards POLARS_AVAILABLE supprimés |
| `src/visualization/antagonist_charts.py` | Guards POLARS_AVAILABLE supprimés |
| `src/visualization/objective_charts.py` | Guards POLARS_AVAILABLE supprimés |
| `src/data/repositories/factory.py` | Vestiges legacy supprimés |
| `src/utils/polars_compat.py` | **Créé** — _to_polars centralisé |
| `src/app/filters.py` | Import _to_polars centralisé |
| `src/app/filters_render.py` | Import _to_polars centralisé |
| `src/app/helpers.py` | Import _to_polars centralisé |
| `src/app/kpis.py` | Import _to_polars centralisé |
| `src/app/kpis_render.py` | Import _to_polars centralisé |
| `src/app/page_router.py` | Import _to_polars centralisé |
| `src/analysis/stats.py` | Import _to_polars centralisé |
| `src/analysis/maps.py` | Import _to_polars centralisé |
| `tests/test_app_module.py` | Adapté (routing retiré) |
| `tests/test_pending_page_navigation_regressions.py` | **Supprimé** (testait routing mort) |
| `tests/test_query_params_routing_regressions.py` | **Supprimé** (testait routing mort) |
| `tests/test_multiplayer.py` | Adapté (fonctions mortes retirées) |
| `tests/test_navigation_state_regressions.py` | Adapté (tests routing retirés) |
| `src/ui/pages/objective_analysis.py` | Guard POLARS_AVAILABLE supprimé |

---

## Pour reprendre le travail

1. `git checkout refactor/phase1-dead-code-cleanup`
2. Vérifier la checklist ci-dessus — les tâches cochées sont terminées
3. Lancer `python -m pytest --ignore=tests/integration -q` pour valider
4. Quand tout passe : `git add -A && git commit -m "refactor: phase 1 — dead code cleanup"`

---
---

# Phase 2 — Élimination des violations DRY

> **Branche** : `refactor/phase2-dry-violations`
> **Prérequis** : Phase 1 mergée
> **Objectif** : Centraliser les patterns dupliqués (identité, Plotly, connexions DuckDB, error handling, session state, dates)

---

## Checklist des tâches

### 1. Consolider la résolution d'identité (6 copies → 1) ✅
- [x] `source.py::_default_identity_from_secrets()` → délègue à `data_loader.default_identity_from_secrets()`
- [x] `main_helpers.py::resolve_xuid_from_input()` → délègue à `data_loader.resolve_xuid_input()`
- [x] `main_helpers.py::propagate_identity_to_env()` → délègue à `data_loader.propagate_identity_env()`
- [x] Imports inutilisés nettoyés (`parse_xuid_input`, `resolve_xuid_from_db`)
- [x] Fix résiduel Phase 1 : `_to_polars` retiré de `cache.py` re-exports
- [x] Test `test_facade_reexports` adapté
- **Impact** : -43 lignes, 3 clones éliminés, source.py -19 lignes
- **Note** : Unification `PlayerIdentity` (profile.py NamedTuple vs state.py dataclass) reportée en Phase 4

### 2. Centraliser les configs Plotly (~57 inline → constantes) ✅
- [x] Remplacer `config={"displayModeBar": False}` → `PLOTLY_CLEAN_CONFIG` (30 occurrences, 12 fichiers)
- [x] Remplacer `config={"staticPlot": True}` → `PLOTLY_STATIC_CONFIG` (26 occurrences, 12 fichiers)
- [x] Remplacer conditionnel inline dans `teammates_synergy.py`
- [x] Imports `PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG` ajoutés dans 12 fichiers
- [x] Test `test_static_plot_count_in_pages` adapté (compte les constantes, pas les inline)
- **Impact** : 57 magic dicts éliminés, cohérence avec CLAUDE.md §9

### 3. Adopter le context manager `duckdb_read_only()` ✅
- [x] `duckdb_read_write()` créé dans `src/utils/db.py`
- [x] **Bug fix** : `multiplayer.py::_get_duckdb_connection()` — connexion leakée via `__enter__()` sans `with` → remanié avec `with duckdb_read_only()`
- [x] `discord_notifier.py` — 3 bare `duckdb.connect()` convertis en `with duckdb_read_only()`
- [x] `commendations.py` — `try/finally` converti en `with duckdb_read_only()`
- [x] Audit des fichiers restants : déjà convertis (career_ranks, sync, match_view, xuid, participation_radar, teammates_service)
- **Impact** : 1 bug leak fix, 4 connexions sécurisées, `duckdb_read_write()` disponible

### 4. Créer un context manager `safe_chart_render()` ✅
- [x] `src/ui/chart_utils.py` créé avec `safe_chart_render(error_key="error_chart")`
- [x] Pattern adopté dans `win_loss.py` (5 blocs try/except → `with safe_chart_render()`)
- [x] Supporte les clés d'erreur personnalisées (`"career_gauge_error"`, etc.)
- **Impact** : Module prêt à l'emploi, ~30 blocs restants convertibles progressivement
- **Note** : Conversion des 30+ blocs restants = travail mécanique pour PRs futures

### 5. Centraliser l'accès session state (`get_page_context()`) ✅
- [x] `get_page_context()` créé dans `src/app/state.py` → retourne `(db_path, xuid, waypoint_player)`
- [x] Résolution XUID intelligente : `player_xuid > xuid > xuid_input`
- [x] Exporté dans `src/app/__init__.py`
- [x] Pattern adopté dans `timeseries.py` (remplacement de 2 accès session_state)
- **Impact** : Helper prêt, ~17 accès restants convertibles progressivement

### 6. Centraliser les formats de date (39 strftime → constantes) ✅
- [x] `src/ui/date_formats.py` créé avec 12 constantes (FR, ISO, ticks)
- [x] 25 remplacements dans 8 fichiers :
  - [x] `timeseries_combat.py` : 10 formats remplacés (8× tick, 2× datetime FR)
  - [x] `timeseries.py` : 3× tick
  - [x] `match_bars.py` : 2× (tick + short datetime)
  - [x] `distributions_outcomes.py` : 4× ISO date
  - [x] `teammates_helpers.py` : 2× datetime FR
  - [x] `match_history.py` : 1× datetime FR
  - [x] `session_compare_charts.py` : 1× datetime FR short year
  - [x] `sessions.py` : 2× date FR
- **Impact** : 25 magic strings → constantes, format modifiable en un seul endroit

---

## Validation Phase 2

- [x] `python -m pytest --ignore=tests/integration -q` → **3499 passed, 66 skipped**
- [x] L'app démarre (`streamlit run streamlit_app.py`)

---

## Fichiers créés

| Fichier | Contenu |
|---------|---------|
| `src/ui/chart_utils.py` | Décorateur `@safe_chart` |

---
---

# Phase 3 — Découpage des God Classes / God Functions

> **Branche** : `refactor/phase3-god-class-split`
> **Prérequis** : Phase 2 mergée
> **Objectif** : Découper les fichiers/fonctions monolithiques (engine.py 2551L, main() 582L, etc.)

---

## Checklist des tâches

### 1. Découper `src/data/sync/engine.py` (2 551 lignes → 6 modules)
- [ ] Créer `src/data/sync/match_processing.py` :
  - [ ] Extraire `_process_matches()`, `_process_single_match()`, `_process_known_match()` (238L), `_process_new_match()` (217L)
  - ~700 lignes extraites
- [ ] Créer `src/data/sync/shared_writes.py` :
  - [ ] Extraire `_insert_shared_registry()`, `_insert_shared_participants()`, `_update_match_participant_bits()`, `_insert_shared_events()`, `_insert_shared_medals()`, `_insert_shared_aliases()`
  - ~230 lignes extraites
- [ ] Créer `src/data/sync/performance.py` :
  - [ ] Extraire `_compute_and_update_performance_score()` (126L), `batch_compute_performance_scores()` (131L), `_ensure_performance_score_column()`
  - ~270 lignes extraites
- [ ] Créer `src/data/sync/skill_rating.py` :
  - [ ] Extraire `_upsert_csr_rating()`, `batch_compute_lusr()` (186L)
  - ~270 lignes extraites
- [ ] Créer `src/data/sync/career.py` :
  - [ ] Extraire `sync_career_rank()`, `_save_career_rank()`, `get_career_rank_history()`, `get_latest_career_rank()`
  - ~120 lignes extraites
- [ ] Créer `src/data/sync/aggregates.py` :
  - [ ] Extraire `_refresh_aggregates_async()`, `refresh_aggregates()`, `get_sync_status()`
  - ~100 lignes extraites
- [ ] Garder `engine.py` comme orchestrateur (~600L) :
  - `DuckDBSyncEngine.__init__`, `sync()`, `close()`, `__enter__`/`__exit__`
  - Imports des sous-modules
- [ ] Vérifier que `src/data/sync/__init__.py` expose correctement les fonctions publiques
- [ ] Adapter les tests `tests/test_sync*.py`
- **Impact** : 2 551L → ~600L dans engine + 6 modules cohérents, testabilité améliorée

### 2. Découper `main()` dans `streamlit_app.py` (582 lignes → fonctions nommées)
- [ ] Extraire la phase d'initialisation (secrets, identity, db_path) → `_initialize_app()`
- [ ] Extraire le bloc de sync + background worker → `_start_background_services()`
- [ ] Extraire le chargement du DataFrame + filtres → `_load_and_filter_data()`
- [ ] Extraire le dispatch de pages → `dispatch_current_page()` (déléguer à `page_router.dispatch_page`)
- [ ] Dé-nester `_tailscale_worker()`, `worker()`, `_index_media_for_player()` → fonctions top-level
- [ ] Dé-nester `_background_media_indexing()` → fonction top-level dans `src/app/media_worker.py`
- [ ] `main()` réduit à ~80L : init → load → filter → dispatch
- **Impact** : 582L → ~80L + fonctions nommées lisibles, débogage facilité

### 3. Découper `render_media_library_page()` dans `media_library.py` (248 lignes)
- [ ] Extraire la section filtres → `_render_media_filters()`
- [ ] Extraire la section grille → `_render_media_grid()` (existe partiellement)
- [ ] Extraire le player média → `_render_media_player()`
- [ ] Fonction principale réduite à ~50L : orchestration uniquement
- **Impact** : 248L → ~50L + 3 fonctions de ~60L

### 4. Découper `render_session_comparison_page()` (127 lignes)
- [ ] Extraire la sélection de sessions → `_render_session_selector()`
- [ ] Extraire la comparaison KPIs → `_render_comparison_kpis()`
- [ ] Extraire la section charts → `_render_comparison_charts()`
- **Impact** : 127L → ~40L + 3 sous-fonctions

### 5. Refactoriser `duckdb_repo.py` — grouper les méthodes (1 536 lignes)
- [ ] Extraire les méthodes d'archivage → `src/data/repositories/_archive.py`
  - `load_matches_from_archives()` (108L), `get_archive_info()` (76L), `archive_old_matches()`
- [ ] Extraire les méthodes de métadonnées → `src/data/repositories/_metadata.py`
  - `_build_metadata_resolution()` (75L), `_resolve_metadata()`, `_get_metadata_connection()`
- [ ] Garder `duckdb_repo.py` comme façade (~800L) avec les méthodes courantes
- **Impact** : 1 536L → ~800L + 2 modules spécialisés, facilite la navigation

---

## Validation Phase 3

- [ ] `python -m pytest --ignore=tests/integration -q` passe
- [ ] Pas de circular imports (`python scripts/check_imports.py`)
- [ ] L'app démarre et toutes les pages fonctionnent
- [ ] Aucune fonction > 150 lignes après découpage

---

## Fichiers créés

| Fichier | Contenu |
|---------|---------|
| `src/data/sync/match_processing.py` | Traitement des matchs (connu/nouveau) |
| `src/data/sync/shared_writes.py` | Écritures dans shared_matches.duckdb |
| `src/data/sync/performance.py` | Calcul des performance scores |
| `src/data/sync/skill_rating.py` | CSR/LUSR rating |
| `src/data/sync/career.py` | Progression de carrière |
| `src/data/sync/aggregates.py` | Vues matérialisées + sync status |
| `src/app/media_worker.py` | Background media indexing |
| `src/data/repositories/_archive.py` | Archivage matchs |
| `src/data/repositories/_metadata.py` | Résolution métadonnées |

---
---

# Phase 4 — Qualité, Typage et Patterns

> **Branche** : `refactor/phase4-quality-patterns`
> **Commit** : `9c0a21a`
> **Prérequis** : Phase 3 mergée
> **Objectif** : Corriger les incohérences de typage, migrer AppSettings vers Pydantic, remplacer les magic numbers

---

## Checklist des tâches

### 1. Migrer `AppSettings` vers Pydantic v2 (`src/ui/settings.py`) ✅
- [x] Remplacer `@dataclass` par `pydantic.BaseModel` pour `AppSettings`
- [x] Ajouter les `Field(default=..., ge=...)` avec validations
- [x] Supprimer les ~160 lignes de coercition manuelle dans `load_settings()` (`_coerce_bool`, `_coerce_int`)
- [x] Utiliser `model_validate(data)` pour charger depuis JSON
- [x] Ajouter `model_config = ConfigDict(extra="ignore", str_strip_whitespace=True)`
- [x] Typage `Literal` pour `spnkr_refresh_match_type`, `repository_mode`, `lang`, `discord_lang`, `cli_lang`
- [x] Validators : `_strip_none_values`, `_normalize_match_type`, `_normalize_lang`, `_clamp_non_negative`, `_clamp_positive`
- [x] `model_validator(mode="after")` pour migration legacy media dirs
- [x] `save_settings()` : `dataclasses.asdict()` → `model_dump()`
- [x] Tester la rétrocompatibilité avec `app_settings.json` existant (14/14 tests passent)
- **Impact** : ~160 lignes de boilerplate supprimées, validation automatique

### 2. Remplacer les magic numbers `outcome` par l'enum `Outcome` ✅
- [x] Remplacer `outcome == 2` par `outcome == Outcome.WIN` dans 7 fichiers :
  - [x] `src/data/domain/models/stats.py` (propriétés `is_win`, `is_loss`)
  - [x] `src/visualization/match_impact_timeline.py`
  - [x] `src/visualization/friends_impact_heatmap.py`
  - [x] `src/ui/pages/match_view.py`
  - [x] `src/ui/pages/session_compare_logic.py`
  - [x] `src/data/services/win_loss_service.py`
  - [x] `src/analysis/performance_score.py`
- [x] Comparaisons Polars `pl.col("outcome") == Outcome.WIN` — fonctionne nativement (IntEnum)
- [x] Test de régression existant dans `test_refdata.py` : `assert Outcome.WIN == 2`
- **Impact** : 0 magic number `outcome` restant dans `src/`
- **Note** : `src/analysis/stats.py`, `cumulative.py`, `win_streaks.py` ne contenaient pas de magic numbers outcome

### 3. Vérifier les annotations `-> None` (11 fonctions) ✅
- [x] Audit complet des 11 fonctions : **toutes les annotations `-> None` sont correctes**
- Les `return <value>` détectés dans le grep initial appartenaient à des **fonctions internes** (closures), pas aux fonctions elles-mêmes :
  - `_render_period_section()` : `return` nu (early exit) ✅
  - `_render_map_table()` : pas de return ✅
  - `render_comparison_radar_chart()` : pas de return ✅
  - `render_participation_trend_section()` : `return` nus (early exits) ✅
  - `render_nemesis_section()` : `return` nu (early exit) ✅
  - `render_roster_section()` : `return` nu (early exit) ✅
  - `render_friends_history_table()` : pas de return ✅
  - `render_trio_synergy_radar()` : `return` nus (early exits) ✅
  - `save_filter_preferences()` : pas de return ✅
  - `render_metric_comparison_row()` : pas de return ✅
  - `_render_history_table()` : pas de return ✅
- **Impact** : 0 annotation à corriger — typage déjà honnête

### 4. Nettoyage des `hasattr` code-smells pour vérification DF ✅
- [x] Remplacer `hasattr(df, "is_empty") and df.is_empty()` par `df.is_empty()` dans :
  - [x] `src/ui/pages/match_view_players.py` (2 occurrences)
- [x] Seul `hasattr` restant : `hasattr(st, "query_params")` (compatibilité Streamlit — légitime)
- **Impact** : 2 code-smells résolus

### 5. Créer une constante `CORE_STAT_COLUMNS` (4 copies → 1) ✅
- [x] Défini dans `src/config.py` :
  ```python
  CORE_STAT_COLUMNS: list[str] = ["start_time", "kills", "deaths"]
  ```
- [x] Remplacé dans 4 fichiers :
  - [x] `src/ui/pages/teammates.py`
  - [x] `src/ui/pages/session_compare.py`
  - [x] `src/data/services/timeseries_service.py`
  - [x] `src/analysis/cumulative.py`
- **Note** : `src/data/query/trends.py` était un faux positif (set de métriques SQL, pas les mêmes colonnes).
  La liste réelle est `["start_time", "kills", "deaths"]` (sans `"assists"`).
- **Impact** : 4 copies → 1 constante

### 6. `duckdb_read_only()` — déjà adopté (Phase 2.3) ✅
- [x] Phase 2.3 a adopté `duckdb_read_only()` : utilisé activement dans 5+ fichiers
- **Impact** : Rien à faire — déjà fait

---

## Validation Phase 4

- [x] `python -m pytest --ignore=tests/integration -q` → **3499 passed, 66 skipped**
- [x] `app_settings.json` charge correctement avec le nouveau `AppSettings` Pydantic
- [x] L'app démarre et se comporte identiquement

---
---

# Récapitulatif global

| Phase | Objectif | Lignes impactées | Commit | Statut |
|-------|----------|:----------------:|--------|--------|
| **Phase 1** | Code mort & duplications triviales | **-839** | `df4da3c` | ✅ Terminé |
| **Phase 2** | Violations DRY (identité, Plotly, DuckDB, charts, sessions, dates) | **~-400** | `145c0a0` | ✅ Terminé |
| **Phase 3** | God classes/functions (duckdb_repo → 8 modules) | **+148 / -198** | `1dbde50` | ⚠️ Partiel (3.5 fait, 3.1-3.4 restent) |
| **Phase 4** | Qualité & patterns (Pydantic, enum, typage, constantes) | **+148 / -198** | `9c0a21a` | ✅ Terminé |

## Travail restant

| Tâche | Description | Priorité |
|-------|-------------|----------|
| **3.1** | Split `engine.py` (2 551L → ~600L + 6 modules) | Haute |
| **3.2** | Split `main()` dans `streamlit_app.py` (582L → ~80L) | Haute |
| **3.3** | Split `media_library.py` (filtres, player) | Basse |
| **3.4** | Split `session_compare` (noms internes) | Basse |
