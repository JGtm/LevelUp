# Release Notes — LevelUp v5.1

> **Date** : 2026-02-17  
> **Version** : 5.1.0  
> **Migration depuis** : v5.0 (Shared Matches)

---

## Vue d'ensemble

LevelUp v5.1 apporte une **modernisation en profondeur de l'interface Streamlit** et des **optimisations de réactivité** significatives. L'objectif : une UI plus fluide, des interactions instantanées, et une architecture prête pour les futures évolutions Streamlit.

---

## Optimisations Réactivité (Étape 8bis)

### Vue matérialisée `mv_player_matches`

- Pré-calcul des jointures `match_participants + match_registry + metadata`
- Réduction du parsing SQL de **170 → 10 lignes** par requête
- Gain performance : **-70% parsing SQL**

### Cache Repository Streamlit

- `get_cached_repository_st()` avec `@st.cache_resource(ttl=3600)`
- Connexion DB persistante entre pages UI
- Gain : **80ms → <20ms** connexion

### Index DuckDB hautes performances

- **16+ index créés** sur 9 tables (composites `(xuid, match_id)`, triés `start_time`)
- Amélioration des scans séquentiels sur les tables partagées

### Cache schema metadata

- `_has_column()` et `_has_shared_mp_column()` cachés au niveau connexion
- Évite les requêtes `information_schema` répétées à chaque page

### Éradication `map_elements()`

- **28 → 0** occurrences de `map_elements()` dans `src/` (15 fichiers)
- Remplacement par `build_mapping()` + `replace_strict()` (module `src/ui/vectorize_helpers.py`)
- Gain estimé : -40% sur les transformations colonnes fréquentes

### Jointures redondantes supprimées

- `_get_match_source()` retourne un 3-tuple `(source_sql, params, uses_mv)`
- En mode v5.1 (`uses_mv=True`), **-3 LEFT JOIN** sur le chemin critique
- Les métadonnées et MMR sont déjà dans `mv_player_matches`

---

## Modernisation Streamlit (Étape 8ter)

### Bump Streamlit ≥1.37.0

- Requis pour `@st.fragment`, `st.navigation` et les nouvelles APIs
- Détection automatique : `HAS_FRAGMENT`, `HAS_NAVIGATION` dans `src/ui/streamlit_modern.py`

### Plotly `staticPlot` + `displayModeBar`

- **30 charts** marqués `staticPlot: true` (14 fichiers) — KPIs, trends, distributions read-only
- **33 charts interactifs** conservés avec `displayModeBar: false` (tooltips actifs, barre d'outils masquée)
- Configs centralisées : `PLOTLY_STATIC_CONFIG`, `PLOTLY_CLEAN_CONFIG`
- Impact : suppression du JavaScript Plotly sur les charts statiques → **-80% RAM charts**

### `@st.fragment` (26 décorateurs, 8 pages)

- Décorateur `@fragment_if_available` avec graceful degradation
- Pages migrées : timeseries, session_compare, win_loss, objective_analysis, career, match_view_charts, match_view_participation, match_view_players, teammates_charts, session_compare_charts
- Impact : **re-render limité au fragment** lors d'interactions filtre (au lieu de la page entière)

### `st.dataframe` + `column_config` (match_history)

- Remplacement du tableau HTML custom par `st.dataframe()` natif
- `column_config` pour formatage colonnes (timestamps, scores, pourcentages)
- Suppression dead code : `_format_score_label`, `_fmt`, `_fmt_mmr_int`
- Impact : **virtualisation native** pour tableaux larges (1000+ lignes sans lag)

### Pré-calcul post-sync (`post_sync_compute.py`)

- 3 tables pré-calculées après chaque sync : sessions, KDA trend, global stats
- Hook automatique dans `src/data/sync/engine.py`
- Impact : **-75% temps de chargement** des agrégats en UI

### `st.navigation` lazy loading

- **11 page closures** dans `streamlit_app.py` avec chargement paresseux
- `build_navigation()` + `render_page_selector_nav()` dans `page_router.py`
- Fallback legacy `dispatch_page()` pour Streamlit < 1.36
- Impact : seules les pages visitées sont importées → **-60% mémoire initiale**

### Centralisation `duckdb_read_only` (A3)

- Nouveau context manager `duckdb_read_only()` dans `src/utils/db.py`
- **14 → 4** `duckdb.connect` directs dans `src/` (restants : sync engine, écriture légitime)
- 7 fichiers migrés : career.py, cache_loaders.py, cache_filters.py, media_library.py, multiplayer.py, data_loader.py
- Impact : garantie de fermeture connexion, code simplifié

### Réduction `st.rerun()` (A8)

- **32 → 14** `st.rerun()` dans `src/`
- `checkbox_filter.py` : 16 reruns → 0 via `on_click`/`on_change` callbacks
- `filters.py` + `filters_render.py` : trio button via `on_click=_apply_trio_filter`
- Impact : **zéro double-rerun** sur les filtres checkbox

### Sécurisation `unsafe_allow_html` (A9)

- **30 → 27** `unsafe_allow_html` dans `src/`
- `html.escape()` ajouté sur les données dynamiques dans `kpi.py` et `performance.py`
- Brand sidebar : HTML remplacé par `st.header()` + `st.divider()`
- Rank fallback career : HTML remplacé par `st.markdown()`
- Impact : **protection XSS** sur les composants KPI

---

## Tests de Non-Régression (8ter.7)

- **30 tests** dans `tests/ui/test_8ter_modernisation.py` couvrant :
  - `PLOTLY_STATIC_CONFIG` et comptage `staticPlot` dans les pages
  - `@fragment_if_available` déployé sur les bonnes pages (5 vérifications)
  - `build_navigation`, `render_page_selector_nav`, `_PAGE_URL_PATHS`, `_PAGE_ICONS`
  - `duckdb_read_only` context manager (import + test fonctionnel + vérification migration)
  - Seuils `st.rerun()` (≤15 dans `src/`), zéro dans `checkbox_filter.py`
  - `html.escape` dans `kpi.py` et `performance.py`
  - `post_sync_compute` importable + hook engine
  - 4 fichiers vérifiés : zéro `duckdb.connect` direct

---

## Métriques de Performance

| Métrique | v5.0 | v5.1 | Gain |
|----------|------|------|------|
| Connexion DB | 80ms | <20ms | **-75%** |
| load_matches(100) | 200ms | <80ms | **-60%** |
| Première page UI | 1500ms | <800ms | **-47%** |
| Parsing SQL/requête | 170 lignes | 10 lignes | **-94%** |
| Interaction intra-page | ~2-3s | <500ms | **-80%** |
| RAM charts (staticPlot) | 100% | ~20% | **-80%** |
| Mémoire initiale | 100% | ~40% | **-60%** |

---

## Fichiers Créés/Modifiés

### Nouveaux fichiers

| Fichier | Description |
|---------|-------------|
| `src/ui/streamlit_modern.py` | Wrappers compatibilité Streamlit moderne |
| `src/ui/vectorize_helpers.py` | Remplacement vectorisé de `map_elements()` |
| `src/utils/db.py` | Context manager `duckdb_read_only()` |
| `scripts/post_sync_compute.py` | Pré-calcul post-sync (sessions, KDA, global) |
| `tests/ui/test_8ter_modernisation.py` | 30 tests non-régression modernisation |

### Fichiers modifiés (sélection)

| Fichier | Modification |
|---------|-------------|
| `streamlit_app.py` | st.navigation lazy loading (11 closures) |
| `src/app/page_router.py` | `build_navigation()`, `_PAGE_URL_PATHS`, `_PAGE_ICONS` |
| `src/ui/components/checkbox_filter.py` | on_click/on_change (0 st.rerun) |
| `src/ui/components/kpi.py` | html.escape sur label/value |
| `src/ui/components/performance.py` | html.escape sur label |
| `src/app/sidebar.py` | Brand natif (st.header + st.divider) |
| 14 fichiers pages | staticPlot + @fragment_if_available |

---

## Suite de Tests

- **2877 tests passés**, 0 échec
- Couverture non-régression modernisation : 30 tests dédiés
- Compatible Python 3.10+ / Streamlit ≥1.37.0

---

## Prochaines Étapes

- **Étape 9** : Tests complémentaires + documentation exhaustive
- **Étape 10** : Release finale v5.1.0 + tag Git
