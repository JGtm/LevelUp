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

### 1. Consolider la résolution d'identité (6 copies → 1)
- [ ] Choisir `src/app/profile.py` comme source unique (`PlayerIdentity` + `get_identity_from_secrets()` + `resolve_xuid()`)
- [ ] Supprimer `default_identity_from_secrets()` dans `src/app/data_loader.py` (L44-83)
- [ ] Supprimer `_default_identity_from_secrets()` dans `src/ui/sections/source.py` (L20-43)
- [ ] Supprimer `resolve_xuid_input()` dans `src/app/data_loader.py` (L248-286) — clone exact de `resolve_xuid_from_input`
- [ ] Adapter `src/app/main_helpers.py` pour déléguer à `profile.resolve_xuid()`
- [ ] Rediriger tous les appelants vers `src/app/profile`
- [ ] Mettre à jour les tests
- **Impact** : ~200 lignes de duplication, 3 fichiers nettoyés

### 2. Centraliser les configs Plotly (~50 occurrences inline → constantes)
- [ ] Remplacer `config={"displayModeBar": False}` par `config=PLOTLY_CLEAN_CONFIG` dans :
  - [ ] `src/ui/pages/teammates_charts.py` (18 occurrences)
  - [ ] `src/ui/pages/timeseries.py` (~10 occurrences)
  - [ ] `src/ui/pages/win_loss.py` (2 occurrences)
  - [ ] `src/ui/pages/objective_analysis.py` (2 occurrences)
  - [ ] `src/ui/pages/session_compare.py` (1 occurrence)
- [ ] Remplacer `config={"staticPlot": True}` par `config=PLOTLY_STATIC_CONFIG` dans :
  - [ ] `src/ui/pages/timeseries.py` (~5 occurrences)
  - [ ] `src/ui/pages/win_loss.py` (3 occurrences)
  - [ ] `src/ui/pages/objective_analysis.py` (3 occurrences)
  - [ ] `src/ui/pages/teammates_views.py` (2 occurrences)
  - [ ] `src/ui/pages/teammates_impact.py` (1 occurrence)
  - [ ] `src/ui/pages/citations.py` (1 occurrence)
- [ ] Ajouter l'import `from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG` dans chaque fichier
- **Impact** : ~50 remplacements, cohérence avec les règles du projet (CLAUDE.md §9)

### 3. Adopter le context manager `duckdb_read_only()` (48 appels → pattern unifié)
- [ ] Auditer `src/utils/db.py` — vérifier que `duckdb_read_only()` est correct et prêt à l'emploi
- [ ] Créer `duckdb_read_write()` si nécessaire (context manager RW)
- [ ] Remplacer les appels `duckdb.connect(..., read_only=True)` par `with duckdb_read_only(path) as conn:` dans :
  - [ ] `src/data/media_indexer.py` (10 appels)
  - [ ] `src/data/sync/engine.py` (6 appels)
  - [ ] `src/data/sessions_backfill.py` (3 appels)
  - [ ] `src/utils/discord_notifier.py` (3 appels)
  - [ ] `src/analysis/citations/engine.py` (4 appels)
  - [ ] `src/analysis/skill_rating_calibration.py` (2 appels)
  - [ ] `src/ui/sync.py` (2 appels)
  - [ ] `src/ui/career_ranks.py` (2 appels)
  - [ ] `src/ui/commendations.py` (1 appel)
  - [ ] + 7 autres fichiers (~10 appels)
- **Impact** : ~48 blocs try/finally → `with`, connexions garanties fermées, ~100 lignes de boilerplate supprimées

### 4. Créer un décorateur `@safe_chart` (37 copies try/except → 1 décorateur)
- [ ] Créer `src/ui/chart_utils.py` avec :
  ```python
  def safe_chart(fn):
      """Décorateur attrapant les erreurs de rendering chart."""
      @wraps(fn)
      def wrapper(*args, **kwargs):
          try:
              return fn(*args, **kwargs)
          except Exception as e:
              st.warning(t("error_chart", error=e))
      return wrapper
  ```
- [ ] Appliquer `@safe_chart` aux fonctions de rendu chart dans :
  - [ ] `src/ui/pages/timeseries.py` (14 blocs)
  - [ ] `src/ui/pages/win_loss.py` (6 blocs)
  - [ ] `src/ui/pages/objective_analysis.py` (4 blocs)
  - [ ] `src/ui/pages/session_compare_charts.py` (3 blocs)
  - [ ] `src/ui/pages/teammates_views.py` (2 blocs)
  - [ ] `src/ui/pages/match_view_charts.py` (2 blocs)
  - [ ] `src/ui/pages/match_view_participation.py` (2 blocs)
  - [ ] + 5 autres fichiers (4 blocs)
- **Impact** : ~37 blocs try/except → `@safe_chart`, ~110 lignes

### 5. Centraliser l'accès session state (`db_path` + `xuid`) (16+ copies → 1 helper)
- [ ] Créer `get_page_context()` dans `src/app/helpers.py` ou `src/ui/session_helpers.py` :
  ```python
  def get_page_context() -> tuple[str, str]:
      """Retourne (db_path, xuid) depuis session_state."""
      db_path = st.session_state.get("db_path", "")
      xuid = st.session_state.get("player_xuid") or st.session_state.get("xuid", "")
      return db_path, xuid
  ```
- [ ] Unifier les clés session state : choisir `"player_xuid"` comme clé canonique
- [ ] Remplacer les 16+ occurrences du pattern `db_path = ...; xuid = ...` dans `src/ui/pages/`
- [ ] Vérifier les pages utilisant `"xuid_input"` — est-ce un concept différent ?
- **Impact** : ~50 lignes de boilerplate, cohérence des clés

### 6. Centraliser les formats de date (41 `strftime` → constantes)
- [ ] Créer les constantes dans `src/ui/formatting.py` (existe déjà) :
  ```python
  DATE_FMT_SHORT = "%m-%d %H:%M"      # Axes Plotly
  DATE_FMT_FR = "%d/%m/%Y %H:%M"      # Affichage FR complet
  DATE_FMT_FR_DATE = "%d/%m/%Y"        # Date seule FR
  DATE_FMT_TIME = "%H:%M"             # Heure seule
  ```
- [ ] Remplacer les 12 occurrences de `"%m-%d %H:%M"` (surtout `src/visualization/timeseries_combat.py`)
- [ ] Remplacer les 6 occurrences de `"%d/%m/%Y %H:%M"`
- [ ] Remplacer les occurrences restantes
- **Impact** : ~24 magic strings → constantes, changement de format en un seul endroit

---

## Validation Phase 2

- [ ] `python -m pytest --ignore=tests/integration -q` passe
- [ ] Aucune régression d'import (`python -c "from src.app import *; from src.ui import *"`)
- [ ] L'app démarre (`streamlit run streamlit_app.py`)

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
> **Prérequis** : Phase 3 mergée
> **Objectif** : Corriger les incohérences de typage, migrer AppSettings vers Pydantic, remplacer les magic numbers

---

## Checklist des tâches

### 1. Migrer `AppSettings` vers Pydantic v2 (`src/ui/settings.py`)
- [ ] Remplacer `@dataclass` par `pydantic.BaseModel` pour `AppSettings`
- [ ] Ajouter les `Field(default=...)` avec validations
- [ ] Supprimer les ~160 lignes de coercition manuelle dans `load_settings()` (`_coerce_bool`, `_coerce_int`)
- [ ] Utiliser `model_validate(data)` pour charger depuis JSON
- [ ] Ajouter `model_config = ConfigDict(extra="ignore")` pour ignorer les clés inconnues
- [ ] Tester la rétrocompatibilité avec `app_settings.json` existant
- **Impact** : ~160 lignes de boilerplate supprimées, validation automatique

### 2. Remplacer les magic numbers `outcome` par l'enum `Outcome`
- [ ] Remplacer `outcome == 2` par `outcome == Outcome.WIN` dans :
  - [ ] `src/analysis/stats.py`
  - [ ] `src/analysis/cumulative.py`
  - [ ] `src/analysis/win_streaks.py`
  - [ ] `src/data/domain/models/stats.py` (propriétés `is_win`, `is_loss`)
  - [ ] Tout autre fichier utilisant `outcome == 2`, `outcome == 3`
- [ ] Vérifier les comparaisons Polars (`pl.col("outcome") == 2`) — utiliser `.cast()` si besoin
- [ ] Ajouter un test de régression garantissant `Outcome.WIN == 2`
- **Impact** : ~15 magic numbers → enum, prévention de bugs futurs

### 3. Corriger les annotations `-> None` mensongères (11 fonctions)
- [ ] Corriger dans `src/ui/pages/win_loss.py` :
  - [ ] `_render_period_section()` L293 — vérifier le `return`, corriger en `-> None` avec early return ou `-> Something`
  - [ ] `_render_map_table()` L420
- [ ] Corriger dans `src/ui/pages/session_compare_charts.py` :
  - [ ] `render_comparison_radar_chart()` L266
  - [ ] `render_participation_trend_section()` L586
- [ ] Corriger dans `src/ui/pages/match_view_players.py` :
  - [ ] `render_nemesis_section()` L191
  - [ ] `render_roster_section()` L922
- [ ] Corriger dans `src/ui/pages/teammates_helpers.py` :
  - [ ] `render_friends_history_table()` L162
- [ ] Corriger dans `src/ui/pages/teammates_synergy.py` :
  - [ ] `render_trio_synergy_radar()` L189
- [ ] Corriger dans `src/ui/filter_state.py` :
  - [ ] `save_filter_preferences()` L259
- [ ] Corriger dans `src/ui/components/performance.py` :
  - [ ] `render_metric_comparison_row()` L147
- [ ] Corriger dans `src/ui/pages/match_history.py` :
  - [ ] `_render_history_table()` L180
- **Impact** : 11 annotations corrigées, typage honnête

### 4. Nettoyage des `hasattr` code-smells pour vérification DF
- [ ] Remplacer `hasattr(df, "is_empty") and df.is_empty()` par `df is None or df.is_empty()` dans :
  - [ ] `src/ui/pages/match_view_players.py` (L467, L490)
- [ ] Auditer les 5 occurrences de `df is None or df.is_empty()` — s'assurer qu'elles sont toutes cohérentes
- **Impact** : 2 code-smells résolus

### 5. Créer une constante `CORE_STAT_COLUMNS` (5 copies → 1)
- [ ] Définir dans `src/config.py` ou `src/data/domain/constants.py` :
  ```python
  CORE_STAT_COLUMNS = ["start_time", "kills", "deaths", "assists"]
  ```
- [ ] Remplacer les 5 listes hardcodées dans :
  - [ ] `src/ui/pages/teammates.py` (L167)
  - [ ] `src/ui/pages/session_compare.py` (L623)
  - [ ] `src/data/services/timeseries_service.py` (L136)
  - [ ] `src/analysis/cumulative.py` (L109)
  - [ ] `src/data/query/trends.py` (L334 — vérifier compatibilité)
- **Impact** : 5 copies → 1 constante

### 6. Supprimer `src/utils/db.py::duckdb_read_only()` si inutilisé, ou l'adopter (Phase 2)
- [ ] Si Phase 2.3 a adopté `duckdb_read_only()` : marquer comme fait
- [ ] Sinon : supprimer cette fonction dead code de `src/utils/db.py`
- **Impact** : Cohérence — pas de dead code utilitaire

---

## Validation Phase 4

- [ ] `python -m pytest --ignore=tests/integration -q` passe
- [ ] Pas d'erreurs mypy/pyright sur les fichiers modifiés
- [ ] `app_settings.json` charge correctement avec le nouveau `AppSettings` Pydantic
- [ ] L'app démarre et se comporte identiquement

---
---

# Récapitulatif global

| Phase | Objectif | Lignes impactées (est.) | Branches |
|-------|----------|:-----------------------:|----------|
| **Phase 1** ✅ | Code mort & duplications triviales | **-839** | `refactor/phase1-dead-code-cleanup` |
| **Phase 2** | Violations DRY (identité, Plotly, DuckDB, charts, sessions, dates) | **~-400** | `refactor/phase2-dry-violations` |
| **Phase 3** | God classes/functions (engine.py, main(), media_library) | **~+500 / -0** (redistribution) | `refactor/phase3-god-class-split` |
| **Phase 4** | Qualité & patterns (Pydantic, enum, typage, constantes) | **~-200** | `refactor/phase4-quality-patterns` |

**Estimation effort total** : ~3-4 jours de travail

**Ordre** : Phase 2 → Phase 3 → Phase 4 (séquentiel obligatoire — chaque phase dépend de la précédente)
