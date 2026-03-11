# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-08 — **Backlog 100% traité**.

---

## 🔴 Dette Technique (code source)

### Cleanup kwargs legacy SyncScope
> Supprimer les 30+ kwargs individuels marqués `LEGACY` une fois tous les appelants migrés vers `scope=SyncScope(...)`.

| Fichier | Ligne | Nature |
|---------|-------|--------|
| [scripts/backfill/detection.py](../scripts/backfill/detection.py#L46) | L46 | `# TODO(cleanup): supprimer ces kwargs quand tous les appelants…` |
| [scripts/backfill/orchestrator.py](../scripts/backfill/orchestrator.py#L104) | L104 | idem |
| [scripts/backfill/orchestrator.py](../scripts/backfill/orchestrator.py#L435) | L435 | idem |
| [scripts/backfill/orchestrator.py](../scripts/backfill/orchestrator.py#L774) | L774 | idem |

**Condition de suppression** : Tous les appelants externes (scripts, tests, UI) passent `scope=SyncScope(...)`.

---

### Migration `career.py` vers DuckDBRepository
> `src/ui/pages/career.py` L27 et L69 utilise `duckdb.connect()` directement (bypass `DuckDBRepository`).  
> SQL correctement paramétré → pas de risque injection, mais dette d'architecture traçable.

**Action** : Refactorer pour passer par `get_cached_repository_st()`.

---

### TODO `custom_rules.py:103`
> [`src/analysis/citations/custom_rules.py`](../src/analysis/citations/custom_rules.py#L103) — amélioration future dépendant de données API non disponibles actuellement.  
> Conservé volontairement en l'état jusqu'à disponibilité des données.

---

### Traductions FR manquantes dans migration metadata
> [`scripts/migration/migrate_metadata_to_duckdb.py`](../scripts/migration/migrate_metadata_to_duckdb.py#L72) L72 — `# TODO: ajouter traductions FR`

---

## 🟠 Performance UI (Roadmap optimisations profondes)

> Contexte : ROG Ally (Ryzen Z1), DuckDB CPU-bound, Streamlit re-renders.  
> Source : `thought_log.md` [2026-02-26].

### 1. Vues matérialisées DuckDB — reconstruction hors UI 📋
- **Problème** : `mv_map_stats`, `mv_mode_category_stats`, `mv_session_stats` reconstruites à chaque rafraîchissement (full-table scan `match_participants`).
- **Gain estimé** : −70% temps d'affichage pages stats.
- **Approche** : Déclencher la reconstruction uniquement dans `engine.py` post-sync, pas dans l'UI.

### 2. Lazy-loading `match_view` 📋
- **Problème** : Toutes les sections (scoreboard, nemesis, KD timeline, médailles, roster) chargées même si non consultées.
- **Gain estimé** : −40% premier rendu d'un match.
- **Approche** : `st.tabs` + `@fragment_if_available` + session state par onglet actif.

### 3. Pagination / virtualisation liste de matchs 📋
- **Problème** : 2000+ matchs → `mv_player_matches` chargé entièrement en Polars avant filtrage Python.
- **Gain estimé** : −50% RAM + temps chargement initial.
- **Approche** : Pousser filtres (map, mode, outcome, date range) en SQL DuckDB avec `LIMIT/OFFSET`.

### 4. Pré-calcul `performance_score` au sync 📋
- **Problème** : `compute_relative_performance_score` appelé à l'affichage pour certains contextes.
- **Approche** : Auditer les call sites, s'assurer que l'UI lit toujours depuis `player_match_enrichment.performance_score`.

### 5. Projections Polars fines par page 📋
- **Problème** : `load_df_optimized` charge `COLUMNS_COMMON` (30+ colonnes) pour pages n'en utilisant que 5-8.
- **Gain estimé** : −30% mémoire.
- **Approche** : Étendre les projections par page dans `cache_loaders.py` aux pages sans projection fine.

---

## 🟡 i18n — Câblage `t()` dans l'UI Streamlit

> Source : `thought_log.md` [2026-02-25] — traductions EN remplies (Phase 1b ✅), câblage UI non fait.

- [ ] Câbler la fonction `t()` dans les pages/widgets Streamlit
- [ ] Modifier `src/ui/translations.py` pour utiliser le registre i18n
- [ ] Supprimer (ou archiver) les commentaires `⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO"` dans :
  - `src/ui/i18n/common.py`
  - `src/ui/i18n/pages.py`
  - `src/ui/i18n/viz.py`
  - `src/ui/i18n/widgets.py`
  - `src/ui/i18n/cli.py`

---

## 🟡 Hover thumbnail sur les noms de cartes (tableaux HTML)

> Commencé le 2026-03-11 — bloqué sur le rendu.

**Objectif** : Au survol d'un nom de carte dans les tableaux HTML (Historique, Explorer, Win/Loss), afficher la miniature correspondante `static/maps/*.jpg|png`.

**Ce qui a été fait** :
- `enableStaticServing = true` activé dans `.streamlit/config.toml`
- `map_thumb_url()` + `_build_map_url_index()` (lru_cache) ajoutés dans `match_table_html.py`
- Cellule `map_name` injecte `<span class='map-cell' data-thumb-url='...'>`
- `win_loss_table_style.py` et `_render_map_table()` réécrits en HTML pur (sans pandas `.style`), avec coloration win/loss/ratio/performance conservée
- Tooltip JS `position:fixed` injecté via `_MAP_TOOLTIP_SCRIPT` dans `load_css()` pour contourner le clipping `overflow-x:auto` du `.os-table-wrap`

**Problème restant** : Le tooltip ne s'affiche pas en pratique — cause probable : le JS injecté via `st.markdown(unsafe_allow_html=True)` est sandboxé par Streamlit (les `<script>` inline sont retirés du DOM). Il faut soit un composant custom Streamlit (`st.components.v1.html`), soit utiliser les images en base64 encodées directement dans une fausse balise `<img>` qui contourne le sandbox.

**Pistes** :
- [ ] Encoder les miniatures en base64 et les injecter directement dans un `<img src='data:image/...'>`  via `st.components.v1.html()` dans un composant dédié
- [ ] Ou utiliser `st.components.v1.html()` pour rendre le tableau entier avec JS (hors contexte Streamlit markdown sandbox)

---

## 🟡 CI/CD & Outillage

> Source : `scripts/demo_regression_detection.py` L122-123.

- [ ] Ajouter la détection de régression au CI/CD (`.github/workflows/`)
- [ ] Créer un pre-commit hook pour la détection de régression

---

## � Connexion Xbox via OAuth (Streamlit)

> Source : `.ai/plan-xboxLogin.prompt.md` — plan rédigé le 2026-02-24, **non démarré**.

Ajouter un flux d'authentification Xbox (OAuth Microsoft) dans l'app Streamlit.  
Mécanisme : Microsoft redirige vers `http://localhost:8501/?code=XXXX` → `st.query_params["code"]` → échange contre tokens SPNKr → profil créé automatiquement.  
Tokens stockés par joueur dans `sync_meta` (`oauth_refresh_token`).

**Prérequis** : Ajouter `http://localhost:8501` dans Azure Portal → App Registration → Redirect URIs (action manuelle unique).

### Étapes

- [ ] **1.** Créer `src/ui/xbox_login.py` — `build_xbox_auth_url()`, `exchange_code_for_refresh_token()`, `resolve_player_identity()`, `create_player_profile()`, `store_refresh_token_in_db()`, `load_refresh_token_from_db()`
- [ ] **2.** Modifier `streamlit_app.py` (~L431-450) — détecter `code` + `state` dans `st.query_params`, vérifier CSRF, déclencher `handle_xbox_callback()` via `ThreadPoolExecutor`
- [ ] **3.** Créer `src/ui/pages/login.py` — bouton "Se connecter avec Xbox 🎮" (`st.link_button`), spinner, confirmation/erreur
- [ ] **4.** Modifier `src/ui/profile_api_tokens.py → ensure_spnkr_tokens()` — lire `refresh_token` depuis `sync_meta` si `db_path` de session disponible (tokens par session, pas globaux)
- [ ] **5.** Modifier sidebar `streamlit_app.py` (~L453) — indicateur "Connecté en tant que {gamertag}" + bouton "Changer de compte"
- [ ] **6.** Modifier `src/data/sync/engine.py` — préserver la clé `oauth_refresh_token` dans `sync_meta` lors des syncs
- [ ] **7.** Enregistrer la page `login` dans la navigation (`streamlit_app.py` ou `src/app/routing.py`)
- [ ] **8.** Écrire `tests/test_xbox_login.py` — mock HTTP pour `exchange_code_for_refresh_token`

**Décisions clés** : `st.query_params` comme callback receiver (natif, pas de serveur HTTP séparé) · tokens dans `sync_meta` (isolation par joueur, multi-user) · `SPNKR_AZURE_REDIRECT_URI=http://localhost:8501` (variable déjà supportée).

---

## �🟢 Améliorations futures / Backlog bas

- **Audit Pandas → Polars** : des usages Pandas résiduels subsistent à la frontière avec Streamlit/Plotly. Voir audit à jour dans les commentaires de code.
- **START_HERE.md** : Le fichier `.ai/START_HERE.md` référence des phases 5-10 d'une migration v5 qui semblent antérieures à v5.1. À vérifier si encore pertinent ou à archiver dans `.ai/archive/`.
- **Benchmark perf** : Réaliser un benchmark avant/après les optimisations UI profondes ci-dessus (outil : `scripts/benchmark_pages.py`).

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-08 | Bug #0 : match invisible post-sync — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | Bug #1 : `win_rate` unifié sur `NULLIF(WIN+LOSS, 0)` dans `analytics.py` et `trends.py` |
| 2026-03-08 | Bug #5 : NaN-check fragile dans `match_view.py` → `is not None` |
| 2026-03-08 | Dette #2 : guard obsolète `_PERF_SCORE_AVAILABLE` supprimé dans `_performance.py` |
| 2026-03-08 | Dette #3 : dead code `_ensure_performance_score_column()` supprimé |
| 2026-03-08 | Dette #4 : magic number `outcome == 4` → `Outcome.DID_NOT_FINISH` |
| 2026-03-08 | Dette #6 : magic SQL `2`/`3` → constantes `_WIN`/`_LOSS` dans `analytics.py` |
| 2026-03-08 | i18n-1 : clés tronquées `PAIR_FR` restaurées dans `translations.py` |
| 2026-03-08 | i18n-2 : 342 entrées redondantes supprimées de `PAIR_FR` (399 → 57) |
| 2026-03-08 | i18n-3 : doublon `tm_session_trend` supprimé dans `widgets.py` |
| 2026-03-08 | Kwargs legacy SyncScope — dépréciés + `scope=SyncScope(...)` opérationnel ; kwargs conservés pour rétro-compat (suppression conditionnelle : quand tous les appelants migrés) |
| 2026-03-08 | `career.py` migré vers `get_cached_repository_st()` (plus de `duckdb.connect()` nu) |
| 2026-03-08 | Perf UI — vues matérialisées reconstruites uniquement post-sync dans `engine.py` |
| 2026-03-08 | Perf UI — lazy-loading `match_view` via `st.tabs` + `@fragment_if_available` |
| 2026-03-08 | Perf UI — pagination SQL `LIMIT/OFFSET` sur `mv_player_matches` |
| 2026-03-08 | Perf UI — projections Polars fines par page dans `cache_loaders.py` |
| 2026-03-08 | i18n câblage `t()` dans les pages/widgets Streamlit |
| 2026-03-08 | CI/CD — détection de régression + pre-commit hook |
| 2026-02-26 | Quick wins perf UI (cache TTL, `@lru_cache`, `@st.cache_data`) |
| 2026-02-25 | v5.3 LUSR stabilisation + UI Carrière |
| 2026-02-25 | i18n Phase 1b — traductions EN registres |
| 2026-02-20 | v5.2 : Filtres intent-based + Stats PvE Firefight |
| 2026-02-17 | Release v5.1 — architecture shared-only |
| 2026-02-15 | Remédiation P0/P1 sécurité SQL + conformité Streamlit |
