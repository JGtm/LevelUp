# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-11.

---

## ✅ Traité

### Citations d'armes — Refactoring catégories et images
> Traité le 2026-03-11 — commit `56c68d7` + `7158626`

- Images incorrectes retirées sur 6 citations (VK78 Commando, Fusil traqueur, Déchiqueteur, Empaleur, Calcineur, Crémateur)
- Covenant + Banished fusionnés en sous-catégorie **Paria** (6 armes)
- Nouvelle sous-catégorie **Forerunner** : Calcineur, Crémateur, Rayon de Sentinelle (nouvelle citation avec image H5G)
- Composites `covenant_weapons_mastery` + `banished_weapons_mastery` remplacés par `paria_weapons_mastery` + `forerunner_weapons_mastery`
- Nouveau composite général `all_weapons_mastery` — Maîtrise en armement (avec image)
- `_SUBCAT_ORDER` Arme mis à jour : Général > UNSC > Paria > Forerunner > Grenade
- i18n FR/EN mis à jour

---

## 🔴 Bugs actifs

### ~~Images citations d'armes incorrectes~~
> ✅ Traité le 2026-03-11 — voir section **Traité** ci-dessus.

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

### Migration : noms d'assets résolus → IDs bruts en BDD
> Dans `match_registry`, les noms d'assets sont stockés en parallèle des IDs bruts (redondance + risque de stale data). À terme, l'UI doit résoudre les noms à la lecture depuis `metadata.duckdb`, pas les lire depuis les colonnes `*_name`.

**Contexte** : Au moment de l'insertion (sync initial), les noms publics (ex. `"Aquarius"`, `"Ranked Arena"`) sont récupérés depuis l'API SPNKr et stockés directement en BDD — en plus de l'ID brut. La `weapon_kills` (v5.7) et `medals_earned` montrent le bon modèle : ID brut uniquement, résolution à la lecture.

**Colonnes concernées dans `shared_matches.duckdb`** :

| Table | Colonnes ID (OK) | Colonnes nom résolu (à migrer) |
|-------|-----------------|-------------------------------|
| `match_registry` | `map_id`, `playlist_id`, `pair_id`, `game_variant_id` | `map_name`, `playlist_name`, `pair_name`, `game_variant_name` |
| `match_participants` | `xuid` | `gamertag` (redondant avec `xuid_aliases`) |
| `highlight_events` | `xuid` | `gamertag` (peut devenir stale si alias change) |

**Modèles de référence (déjà corrects)** :
- `medals_earned.medal_name_id` → UBIGINT, résolution via `metadata.duckdb`
- `weapon_kills.weapon_id` → UBIGINT post v5.7 (migré depuis `weapon_name`)

**Actions** :
- [ ] Auditer les usages UI/query des colonnes `*_name` dans `match_registry` pour identifier ce qui lit directement le nom stocké vs ce qui joint `metadata.duckdb`
- [ ] Créer une vue `v_match_registry` qui résout les noms à la lecture via JOIN sur les tables de référence `metadata.duckdb` (maps, playlists, game_variants)
- [ ] Migrer les requêtes consommatrices (pages Streamlit, repositories) vers la vue — supprimer les colonnes `*_name` de `match_registry` une fois toutes les requêtes migrées
- [ ] `match_participants.gamertag` et `highlight_events.gamertag` : évaluer si ces colonnes sont utilisées en lecture directe ou si le JOIN sur `xuid_aliases` est systématique — supprimer si redondant
- [ ] Ajouter un test de non-régression : aucune colonne `*_name` dans les nouvelles tables shared (hors `xuid_aliases`)

**Complexité** : L (impact UI + repositories + migrations)  
**Fichiers clés** : [`src/data/sync/migrations.py`](../src/data/sync/migrations.py), [`src/data/sync/_shared_writes.py`](../src/data/sync/_shared_writes.py), [`src/data/sync/transformers/_match.py`](../src/data/sync/transformers/_match.py), `data/warehouse/shared_matches.duckdb`

---

### Couverture tests `migrations.py` (lacunes v5.5–v5.7)
> [`src/data/sync/migrations.py`](../src/data/sync/migrations.py) — ~1290 lignes, couverture actuelle ~60%. Trois blocs sans aucun test depuis les versions 5.5–5.7.

| Fonction | Version ajoutée | Couverture actuelle |
|----------|----------------|---------------------|
| `ensure_weapon_kills_table()` | v5.7 | ❌ Aucun test |
| `ensure_bot_teammate_column()` | v5.5 | ❌ Aucun test |
| `add_spartan_id_to_career_progression()` | v5.x | ❌ Aucun test |
| `_recreate_highlight_events_with_sequence()` | v5.x | ⚠️ Chemin idempotent non testé |

**Actions** :
- [ ] `ensure_weapon_kills_table()` : tester création de table, conversion `weapon_name→weapon_id`, type BIGINT→UBIGINT, idempotence
- [ ] `ensure_bot_teammate_column()` : tester ajout de colonne, valeur par défaut, idempotence (double appel = même schéma)
- [ ] `add_spartan_id_to_career_progression()` : tester ajout colonne, contrainte, idempotence
- [ ] `_recreate_highlight_events_with_sequence()` : tester le chemin déjà-migré (si `nextval` existe, pas de double création)
- [ ] Viser couverture ≥ 85% sur `migrations.py` (mesurer via `python -m pytest --cov=src/data/sync/migrations`)

**Complexité** : M  
**Fichiers** : [`tests/test_migrations.py`](../tests/test_migrations.py), [`src/data/sync/migrations.py`](../src/data/sync/migrations.py)

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

### 6. Scan bitstring POV FRAME-only 📋
- **Contexte** : `_scan_fire_events_bitstring()` scanne le chunk entier (~700 KB) alors que les fire events n'existent que dans les payloads FRAME (32% du chunk, ~230 KB). Les 68% restants (INIT_STATE ~155 KB, METADATA ~25 KB, headers…) ne peuvent pas contenir de fire events.
- **Gain mesuré** : −46% temps de scan bitstring (458 ms → 247 ms sur match 000d5950, 24 chunks).
- **Note** : Cette optimisation ne concerne que le POV — les fire events Section 2 sont exclusifs au joueur filmé. Les coéquipiers T1 restent sur Formula A (snapshots par chunk), le film ne contient tout simplement pas leurs fire events.
- **Approche** : Ajouter `extract_frame_data(chunk_data, packets)` dans `packet_index.py` → concatène les payloads FRAME + adapte l'estimateur de position. Modifier `_scan_player_chunks` pour extraire les FRAMEs avant de passer les données à `_scan_fire_events_bitstring`.
- **Coût secondaire** : concat FRAME payloads ~3 ms/match (négligeable vs 211 ms économisés).

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

**Actions restantes** :
- [ ] Remplacer le rendu `st.markdown` par `st.components.v1.html()` pour le tableau entier (contourne le sandbox JS Streamlit qui retire les `<script>` inline)
- [ ] Encoder les miniatures en base64 et les injecter directement dans les cellules `<img src="data:image/jpeg;base64,...">` (pas de dépendance au serveur de fichiers statiques)
- [ ] Améliorer `_build_map_url_index()` dans `match_table_html.py` : passer `lru_cache(maxsize=None)` (actuellement `maxsize=1`, très fragile) et normaliser les noms via `unicodedata.normalize('NFC', name)` pour gérer les accents
- [ ] Créer une table de correspondance explicite `nom API Halo → fichier PNG` pour les maps avec caractères spéciaux ou noms divergents

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
