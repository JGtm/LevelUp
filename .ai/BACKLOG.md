# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-09.

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

## 🟠 Conflit `shared_matches.duckdb` — sync depuis UI Streamlit

> Source : audit logs 2026-03-09 — 339 warnings/sync, app stable, pas de panne fonctionnelle.  
> **Ne pas traiter tant que le sync n'est pas stable depuis ≥ 1 semaine.**  
> Signal de déclenchement : sync retourne `None` pour shared sur plusieurs runs consécutifs.

### Contexte

Quand le sync est lancé depuis l'UI Streamlit, `_engine_connections.py::_get_shared_connection()` tente d'ouvrir `shared_matches.duckdb` en **R/W direct**. Simultanément, Streamlit maintient une connexion **R/O + ATTACH** sur le même fichier via `@st.cache_resource` (ttl=3600, `get_cached_repository_st`). DuckDB refuse qu'un même fichier soit ouvert sous deux modes dans le même processus.

Le retry appelle `release_all_db_connections()` (WeakSet), mais le repo du cache Streamlit rétablit sa connexion R/O dès le rerun suivant → **cycle conflit → release → reconnexion R/O → conflit**.

---

### Option A — Fix minimal dans `DuckDBRepository._get_connection()` ⭐ Recommandée

**Mécanique** : Si `_sync_mode.is_set()` est actif ET que `shared_matches` est attaché → le DETACHER immédiatement avant de retourner la connexion. Le repo continue à fonctionner pour les requêtes ne touchant pas shared ; shared sera réattaché automatiquement à la fin du sync via `end_sync_mode()`.

**Fichiers** : `src/data/repositories/duckdb_repo.py` uniquement (~10-15 lignes dans `_get_connection()`).

**Effort** : S  
**Risque** : Faible — ne touche pas au moteur de sync, pas de nouveau fichier.  
**Effet de bord** : Pendant le sync, les requêtes UI nécessitant shared retournent données partielles (déjà le cas aujourd'hui via `SharedDBUnavailableError`).

```python
# Dans _get_connection(), après avoir obtenu self._connection :
if _sync_mode.is_set() and "shared" in self._attached_dbs:
    try:
        self._connection.execute("DETACH shared")
        self._attached_dbs.discard("shared")
    except Exception:
        pass
```

---

### Option B — Hook pré-sync enregistrable

**Mécanique** : Avant d'ouvrir la connexion R/W, l'engine appelle tous les hooks enregistrés. L'UI Streamlit enregistre un hook qui appelle `st.cache_resource.clear()` → vide le cache de tous les repos, plus aucun conflit possible.

**Fichiers** :
- `src/data/sync/_engine_connections.py` → `register_pre_sync_hook()` + appel avant `_open_shared()`
- `src/ui/_cache_core.py` → `clear_cached_repositories()` exposant `st.cache_resource.clear()`
- `streamlit_app.py` → `register_pre_sync_hook(clear_cached_repositories)` au démarrage

**Effort** : M  
**Risque** : Moyen — 3 fichiers modifiés dont `_engine_connections.py` (zone sensible).  
**Effet de bord** : `st.cache_resource.clear()` vide **tous** les caches resource, pas seulement les repos → cold start de ~100ms après chaque sync.

---

### Option C — Ouvrir shared en R/O pour le sync (refactoring profond)

**Mécanique** : Supprimer `duckdb.connect(shared, read_only=False)`. À la place, écrire dans shared via `ATTACH shared AS s (READ_WRITE)` depuis la connexion player, ou passer par un contexte de connexion partagé unique géré par un singleton.

**Effort** : XL — refactoring complet du sync engine  
**Risque** : Élevé  
**Verdict** : À réserver à une refonte complète du moteur de sync.

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

- [x] Câbler la fonction `t()` dans les pages/widgets Streamlit — 53 fichiers utilisent `from src.ui.i18n`
- [x] Modifier `src/ui/translations.py` pour utiliser le registre i18n — utilise `from src.ui.i18n.data_labels import label, load_domain`
- [x] Supprimer (ou archiver) les commentaires `⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO"` dans :
  - `src/ui/i18n/common.py`
  - `src/ui/i18n/pages.py`
  - `src/ui/i18n/viz.py`
  - `src/ui/i18n/widgets.py`
  - `src/ui/i18n/cli.py`

---

## 🟡 CI/CD & Outillage

> Source : `scripts/demo_regression_detection.py` L122-123.

- [x] Ajouter la détection de régression au CI/CD (`.github/workflows/`) — intégré dans `ci.yml` (step "Run filters/visualization non-regression")
- [ ] Créer un pre-commit hook pour la détection de régression

---

## � Connexion Xbox via OAuth (Streamlit)

> Source : `.ai/plan-xboxLogin.prompt.md` — plan rédigé le 2026-02-24, **non démarré**.

Ajouter un flux d'authentification Xbox (OAuth Microsoft) dans l'app Streamlit.  
Mécanisme : Microsoft redirige vers `http://localhost:8501/?code=XXXX` → `st.query_params["code"]` → échange contre tokens SPNKr → profil créé automatiquement.  
Tokens stockés par joueur dans `sync_meta` (`oauth_refresh_token`).

**Prérequis** : Ajouter `http://localhost:8501` dans Azure Portal → App Registration → Redirect URIs (action manuelle unique).

### Étapes

- [x] **1.** Créer `src/ui/xbox_oauth.py` — toutes les fonctions présentes (`build_xbox_auth_url`, `exchange_code_for_refresh_token`, `resolve_player_identity`, `store_refresh_token`, `load_refresh_token`) + UI dans `src/ui/xbox_oauth_ui.py`
- [x] **2.** Modifier `streamlit_app.py` — `_handle_xbox_oauth_callback()` (L332) détecte `code`+`state` dans `st.query_params`, vérifie CSRF, appelle `run_xbox_oauth_callback()`
- [x] **3.** Intégré dans `src/ui/pages/settings.py` (`render_xbox_login_section()`) et `src/ui/pages/setup_wizard.py` (`_render_xbox_flow()`) — pas de page standalone mais approche plus cohérente
- [x] **4.** `src/ui/profile_api_tokens.py` — `_load_refresh_token_from_db()` comme fallback dans `get_api_tokens()`
- ~~**5.** Sidebar `streamlit_app.py` — indicateur "Connecté en tant que {gamertag}" + bouton "Changer de compte"~~ — abandonné (pas de valeur suffisante)
- [x] **6.** `engine.py` préserve `oauth_refresh_token` — `_update_sync_meta()` écrit uniquement des clés spécifiques (`last_sync_at`, etc.), la clé oauth est préservée par défaut
- [x] **7.** Navigation assurée via les pages settings et setup_wizard (toutes deux enregistrées dans le routing)
- [x] **8.** `tests/test_xbox_oauth.py` + `tests/test_xbox_oauth_callback_e2e.py` (9 tests) existent

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
| 2026-03-09 | Fix logging : `basicConfig` déplacé dans `main()` de `scripts/sync.py` — élimine 142 artefacts de test dans `sync.log` |
| 2026-03-09 | Fix migrations : `_create_index_safe()` — branche DEBUG pour tables absentes — élimine ~2134 warnings/sync dans `sync.log` |
| 2026-03-09 | Fix oauth : cache process-level `_rt_cache` dans `load_refresh_token()` — 185 ouvertures DuckDB/session → 1 par joueur |
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
