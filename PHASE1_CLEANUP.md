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
