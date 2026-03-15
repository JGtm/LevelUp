# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-15.

---

## � v6 — Instructions de merge (requêtes directes UI à migrer)

> En v6, les pages UI n'émettront plus de requêtes DuckDB directement.
> Les éléments ci-dessous devront être refactorisés au moment du merge vers la couche de données v6.

### Page Dernier match / match_view

| Fichier | Fonction | DB cible | Requête directe |
|---|---|---|---|
| `src/ui/pages/match_view_logic.py` | `load_enrichment()` | `stats.duckdb` | `SELECT ... FROM player_match_enrichment WHERE match_id = ?` |
| `src/ui/pages/match_view_logic.py` | `detect_abandoned_match()` | `shared_matches.duckdb` | `SELECT COUNT(*) FROM match_participants WHERE match_id = ?` |
| `src/ui/cache_loaders.py` | `cached_load_player_match_result()` | `shared_matches.duckdb` | `match_participants` |
| `src/ui/cache_loaders.py` | `cached_load_match_medals_for_player()` | `shared_matches.duckdb` | `medals_earned` |
| `src/ui/cache_loaders.py` | `cached_load_highlight_events_for_match()` | `shared_matches.duckdb` | `highlight_events` |
| `src/ui/cache_loaders.py` | `cached_load_match_player_gamertags()` | `shared_matches.duckdb` | `xuid_aliases` |
| `src/ui/cache_loaders.py` | `cached_load_match_rosters()` | `shared_matches.duckdb` | composition équipes |
| `src/ui/cache_loaders.py` | `cached_get_match_skill_rank()` | `stats.duckdb` | `match_skill_rank` |

**Action v6** : remplacer chaque appel par un endpoint de la couche de données v6 (ex: `MatchRepository.get_match_detail(match_id)`). Les `cached_load_*` de `cache_loaders.py` seront remplacés par des appels au repository avec cache Streamlit au même niveau.

---

## �🔴 v5.8 — Couche d'abstraction résolution IDs (branche `refactor/id-resolution-cleanup`)

> **Plan détaillé** : [PLAN_ABSTRACTION_RESOLUTION.md](PLAN_ABSTRACTION_RESOLUTION.md)
> **Branche** : `refactor/id-resolution-cleanup` créée depuis `analysis/weapon-parser-rewrite` (v5.7)
>
> **Objectifs** :
> 1. Centraliser toute résolution ID → nom affiché via 3 vues SQL + fonctions Python
> 2. Détecter les incohérences (même XUID = 2 gamertags différents selon la page, map_name stale)
> 3. Éliminer les redondances (~260 emplacements lisant des colonnes dupliquées dans 3-5 tables)
> 4. Garantir un point unique de modification (1 vue SQL, pas 35 fichiers)

| Wave | Commits | Volets | Status |
|:----:|:-------:|--------|:------:|
| 1 | 1–2 | Vues SQL (`v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`) + refactor resolver | ⬜ |
| 2 | 3–5 | Migration consommateurs directs (gamertags, KV pairs, assets) | ⬜ |
| 3 | 6–7 | Nettoyage wrappers XUID + dead code outcomes | ⬜ |
| 4 | 8–9 | Drop `highlight_events.gamertag` + helper médailles | ⬜ |

---

## 🔴 Dette Technique (code source)

### Centralisation des helpers de résolution d'IDs

> → **Traité dans v5.8** : voir [PLAN_ABSTRACTION_RESOLUTION.md](PLAN_ABSTRACTION_RESOLUTION.md) § Volets A (gamertags), B (outcomes), D (médailles).
>
> Résumé : 5 fonctions XUID fragmentées → 1 vue SQL + 1 mixin Python. 3 fonctions outcomes → 1 canonique. Helper médailles manquant → créé.

---

### Migration : noms d'assets résolus → IDs bruts en BDD

> → **Traité dans v5.8** : voir [PLAN_ABSTRACTION_RESOLUTION.md](PLAN_ABSTRACTION_RESOLUTION.md) § Volet C (assets).
>
> Résumé : vue `v_match_full` avec COALESCE JOIN metadata, mise à jour `mv_player_matches`, migration des requêtes directes. Colonnes `*_name` conservées comme cache, supprimables à terme via modification de la vue seule.

---

### Cohérence XUID↔Gamertag — source de vérité et stale data

> → **Traité dans v5.8** : voir [PLAN_ABSTRACTION_RESOLUTION.md](PLAN_ABSTRACTION_RESOLUTION.md) § Volets A (étapes 1-4) et E (killer/victim).
>
> Résumé : vue `v_gamertag_lookup` (xuid_aliases prioritaire, match_participants fallback), refactor cascade resolver, suppression `highlight_events.gamertag`, vue `v_killer_victim_full`.
> Décision : `match_participants.gamertag` conservé comme fallback. `highlight_events.gamertag` supprimé.

---

---

## 🟡 Table `weapon_names` dans `metadata.duckdb`

**Contexte** : `weapon_kills.weapon_id` stocke des entiers bruts (ex: `17584332298403800991`). La résolution vers un nom lisible (`Mk51 Sidekick`) est actuellement uniquement en Python via `WEAPON_INT_TO_NAME` dans `src/analysis/_weapon_data.py`. Les requêtes SQL directes (MCP, scripts ad-hoc) retournent des IDs illisibles.

**Objectif** : insérer une table `weapon_names(weapon_id UBIGINT PRIMARY KEY, name VARCHAR)` dans `metadata.duckdb`, populée depuis le dict Python via une migration.

**Plan** :
1. Créer `ensure_weapon_names(conn)` dans `src/data/sync/migrations.py` — crée la table si absente + upsert depuis `WEAPON_INT_TO_NAME`
2. Créer `src/data/migration/steps/add_weapon_names.py` avec `target_db="shared"` → ou `metadata`
3. Ajouter l'import dans `src/data/migration/steps/__init__.py`

**Note archi** : la table est un cache de lecture — `WEAPON_INT_TO_NAME` reste la source de vérité. La migration fait un `INSERT OR REPLACE` complet à chaque fois (idempotente). Si le dict évolue, une nouvelle migration (ou re-run de l'existante avec `force`) suffit.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-15 | Fix weapon-parser : corrélation globale — taux `fire_event` 15% → 95% (`_global_correlation.py`, marker 0x26 fixe) |
| 2026-03-15 | Qualité logs weapon_parser : `conf='sentinel'`/`'no_weapon'` + format COMPLETE compact (`match=XXXXXXXX… k=N \| H M L s ? \| fe fa s \| warn`) |
| 2026-03-13 | Couverture tests `migrations.py` (lacunes v5.5–v5.7) |
| 2026-03-13 | Conflit `shared_matches.duckdb` — sync depuis UI Streamlit |
| 2026-03-13 | Hover thumbnail sur les noms de cartes (tableaux HTML) |
| 2026-03-13 | Détection de langue système dans `LevelUp.sh` / `LevelUp.bat` |
| 2026-03-13 | [UI] Heatmap performance par joueur × carte — Page Teammates |
| 2026-03-13 | [UI] Performance par carte vs historique — vues escouade et joueur |
| 2026-03-13 | Audit Pandas → Polars — résidus nettoyés |
| 2026-03-13 | Kwargs legacy SyncScope — nettoyage différé |
| 2026-03-13 | Traductions FR manquantes dans migration metadata |
| 2026-03-13 | Images citations d'armes incorrectes |
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
| 2026-03-08 | Kwargs legacy SyncScope — dépréciés + `scope=SyncScope(...)` opérationnel |
| 2026-03-08 | `career.py` migré vers `get_cached_repository_st()` (plus de `duckdb.connect()` nu) |
| 2026-03-12 | `custom_rules.py:103` — TODO disparu, `compute_annexion_forcee` implémentée |
| 2026-03-12 | Pandas éliminé du code métier — résidus légitimes uniquement |
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
