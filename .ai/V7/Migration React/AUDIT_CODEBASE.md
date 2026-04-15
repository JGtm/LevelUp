# AUDIT_CODEBASE.md — Ce qu'on garde, adapte ou supprime

> Audit exhaustif de la surface `src/` et des fichiers racine vis-à-vis de la migration.
> Source : PLAN_MIGRATION_FASTAPI_REACT.md § 1, 2, 3, 4
> Ce fichier est figé après l'audit initial. Les mises à jour reflètent les suppressions effectives lors de la décommission.

---

## Chiffres de surface (audit initial)

- ~185 fichiers dans `src/ui/`
- ~26 fichiers dans `src/app/`
- ~47 modules Plotly dans `src/visualization/`
- ~121 imports Streamlit dans `src/`

---

## 1. Ce qu'on garde tel quel

### Couche data et repositories

Conserver telle quelle, exposée ensuite via FastAPI :

- `src/data/repositories/duckdb_repo.py`
- `src/data/repositories/factory.py`
- La majorité des mixins et helpers dans `src/data/repositories/`

Pourquoi : accès DuckDB déjà structuré, logique de résolution XUID/gamertag déjà en place, connaissances schéma déjà encodées.

### Couche d'analyse pure

Conserver telle quelle :

- `src/analysis/` — zéro ou quasi-zéro dépendance UI, bonne séparation algorithmique, forte valeur métier

Modules représentatifs : `match_cadence.py`, `match_intensity.py`, `performance_score.py`, `sessions.py`, `skill_rating.py`, `friends_impact.py`, `objective_participation.py`, `weapon_parser.py`

### Couche sync et migrations

Conserver telle quelle :

- `src/data/sync/` — aucune raison de réécrire ce moteur pour changer d'UI
- `scripts/sync.py`
- `scripts/backfill_data.py`

Fichiers clés : `engine.py`, `migrations.py`, `scope.py`

### Services de page déjà bien extraits

Conserver et réutiliser dans l'API :

- `src/data/services/timeseries_service.py`
- Autres services dans `src/data/services/`

### Modèles et schémas

Conserver et réutiliser :

- Modèles Pydantic et dataclasses métier existants
- Schémas de domaine et résultats de service

---

## 2. Ce qu'on garde avec adaptation

### Visualisations Plotly

Conserver la logique de génération des figures, changer le mode de livraison.

- `src/visualization/` — conserver
- `src/visualization/theme.py` — conserver

Adaptation : retourner `fig.to_plotly_json()` au lieu de rendre via `st.plotly_chart`. Créer des endpoints de figures ou d'agrégats.

### Authentification

Conserver la logique, adapter l'exécution :

- `src/auth/provider.py`
- `src/auth/_msal.py`
- `src/auth/_halo_exchange.py`

Adaptation : sortir du cache process local pour les tokens, gérer la session côté API, prévoir cookies httpOnly ou session backend.

### i18n

Conserver le catalogue, adapter la source de vérité :

- `src/ui/i18n/`

Adaptation : ne plus dépendre de Streamlit pour la langue courante — recevoir explicitement la langue depuis le frontend quand nécessaire.

### Configuration et exploitation

Adapter :

- `launcher.py`
- `run.sh`
- `Dockerfile`
- `pyproject.toml`
- `README.md`

---

## 3. Ce qu'il faut réécrire ou retirer

### Shell applicatif Streamlit

Retirer ou remplacer :

- `streamlit_app.py`
- `streamlit_app_v7.py`
- `src/app/page_router.py`
- La logique de navigation basée sur `st.navigation`, `st.switch_page` et `st.query_params`

### État de session Streamlit

Retirer ou remplacer :

- `src/app/session_keys.py`
- `src/app/state.py`
- Les usages intensifs de `st.session_state` dans `src/app/` et `src/ui/`

Remplacement cible : query params URL, TanStack Query, Zustand, localStorage.

### Cache UI Streamlit

Retirer ou remplacer :

- `src/ui/_cache_core.py`
- `src/ui/_cache_loading.py`
- `src/ui/_cache_queries.py`
- `src/ui/_cache_sessions.py`
- `src/ui/cache.py`
- `src/app/cache_control.py`

Remplacement cible : cache HTTP + invalidation backend pour ressources coûteuses, TanStack Query frontend.

### Pages et composants fortement couplés

Ces modules ne se portent pas, ils se remplacent :

- `src/ui/pages/`
- `src/ui/layout/`
- `src/ui/components/` dont les composants Streamlit purs
- `src/app/filters_render.py`
- `src/app/filters.py`
- `src/app/sidebar.py`
- `src/ui/streamlit_modern.py`

### Médias et lightbox

Réécriture complète recommandée :

- `src/ui/components/media_lightbox.py`
- `src/ui/pages/media_v2_grid.py`
- `src/ui/pages/media_v2.py`
- `src/ui/pages/media_library_render.py`

### Browser storage serveur

Retirer ou refondre :

- `src/ui/components/browser_storage/__init__.py`

Remplacement cible : localStorage natif, IndexedDB si besoin.

---

## 4. Modules mixtes à découper avant ou pendant la migration

Ces fichiers mélangent encore logique métier, orchestration et rendu UI. **Ils doivent être découpés pour être migrables proprement** — tenter de migrer ces pages sans les avoir d'abord extraites produira soit une réécriture métier dans le front, soit un endpoint illisible.

| Fichier | Problème principal |
|---|---|
| `src/ui/pages/timeseries.py` | Calculs Polars, logique de downsampling et rendu Streamlit mélangés |
| `src/ui/pages/teammates.py` | Chargement multi-DB, enrichissements et composants Streamlit intriqués |
| `src/ui/pages/explorer.py` | Fuzzy search, cascade de filtres et gestion de l'état de navigation locale |
| `src/ui/pages/session_compare.py` | Sélection A/B, calculs de contexte historique et rendu |
| `src/ui/pages/match_view.py` | Callbacks injectés, tabs, loaders et rendu en une seule fonction |
| `src/app/filters_render.py` | Logique de résolution des sessions mélangée avec les widgets Streamlit |
| `src/ui/pages/media_v2_grid.py` | Index media, groupement, likes et lightbox couplés |
| `src/ui/pages/home_mission_control.py` | Agrégation multi-source, battle pass/challenges, cache process-level |

Modules déjà mieux préparés pour l'extraction (logique déjà séparée du rendu) :

- `src/ui/pages/match_view_logic.py`
- `src/ui/pages/session_compare_logic.py`
- `src/ui/pages/home_mission_control_logic.py`
- `src/data/services/timeseries_service.py`

**Règle** : avant de lancer un slice, vérifier que le module Python source de vérité correspondant est déjà découplé de Streamlit. Si ce n'est pas le cas, extraire d'abord la logique métier dans `src/data/services/` ou `src/analysis/` — puis exposer via FastAPI.

### Fichiers `src/app/` à lire impérativement avant Slice 0b

> **Ces fichiers contiennent la logique réelle de résolution des filtres, l'état applicatif et les sessions.** Toute implémentation de `filters/resolve` qui ne les a pas étudiés produira des régressions.

| Fichier | Contenu critique | Slice impacté |
|---------|------------------|---------------|
| `src/app/filters_render.py` | `GAP_MINUTES_FIXED = 120`, shadow keys, logique de résolution sessions, cascade de filtres, widgets couplés | **0b** |
| `src/app/filter_state.py` | État courant des filtres, transitions, scope | **0b** |
| `src/app/state.py` | `AppState` dataclass, contexte global | **0a, 0b** |
| `src/app/session_keys.py` | `SK` — toutes les clés `session_state` centralisées | **0a, 0b, 1** |
| `src/app/sidebar.py` | Mode d'interaction filtre (sidebar Streamlit → barre filtres React) | **0b** |
| `src/app/cache_control.py` | Invalidation cache — à remplacer par TanStack Query | **0a** |

---

## 5. Impact hors code applicatif

### Build et runtime

- Nouveau frontend avec `package.json`
- Nouveau build multi-étapes pour le frontend
- Nouveau point d'entrée backend FastAPI
- Nouveau mode dev local pour lancer API + web
- Refonte partielle du Dockerfile

### Documentation

- Documentation d'installation
- Documentation de lancement local
- Documentation de déploiement
- Documentation des endpoints si OpenAPI non suffisante

---

## 6. Surfaces legacy à absorber plutôt qu'à migrer

- `win_loss.py` : déjà redistribué entre Timeseries et Synthesis — pas de migration 1:1
- `media_tab.py` / `media_library.py` : remplacés par `media_v2.py`
- Anciens labels de page legacy servant seulement aux redirects
