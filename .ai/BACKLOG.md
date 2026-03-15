# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-15.

---

## 🔵 v6 — Instructions de merge (requêtes directes UI à migrer)

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

## 🟡 Table `weapon_names` dans `metadata.duckdb`

**Contexte** : `weapon_kills.weapon_id` stocke des entiers bruts (ex: `17584332298403800991`). La résolution vers un nom lisible (`Mk51 Sidekick`) est actuellement uniquement en Python via `WEAPON_INT_TO_NAME` dans `src/analysis/_weapon_data.py`. Les requêtes SQL directes (MCP, scripts ad-hoc) retournent des IDs illisibles.

**Objectif** : insérer une table `weapon_names(weapon_id UBIGINT PRIMARY KEY, name VARCHAR)` dans `metadata.duckdb`, populée depuis le dict Python via une migration.

**Plan** :
1. Créer `ensure_weapon_names(conn)` dans `src/data/sync/migrations.py` — crée la table si absente + upsert depuis `WEAPON_INT_TO_NAME`
2. Créer `src/data/migration/steps/add_weapon_names.py` avec `target_db="metadata"`
3. Ajouter l'import dans `src/data/migration/steps/__init__.py`

**Note archi** : la table est un cache de lecture — `WEAPON_INT_TO_NAME` reste la source de vérité. La migration fait un `INSERT OR REPLACE` complet à chaque fois (idempotente).

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-15 | Résolution gamertag→xuid centralisée : `lookup_xuid_for_gamertag()` dans `src/utils/xuid.py` + `GamertagResolverMixin.resolve_xuid_from_gamertag()` — 5 fichiers migrés |
| 2026-03-15 | v5.8 Wave 5 : nettoyage i18n playlists/modes obsolètes → `metadata.duckdb` (`57a755c`, `b4ff066`) |
| 2026-03-15 | v5.8 Wave 4 : suppression `highlight_events.gamertag` (migration `drop_highlight_events_gamertag`) + helper `resolve_medal_name` (`src/analysis/_medal_data.py`) (`0a5c69c`, `ffdd959`) |
| 2026-03-15 | v5.8 Wave 3 : nettoyage wrappers XUID + dead code outcomes → `Outcome` enum |
| 2026-03-15 | v5.8 Wave 2 : migration consommateurs directs (gamertags, KV pairs, assets) |
| 2026-03-15 | v5.8 Wave 1 : vues SQL `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full` + `GamertagResolverMixin` |
| 2026-03-15 | Fix weapon-parser : corrélation globale — taux `fire_event` 15% → 95% (`_global_correlation.py`, marker 0x26 fixe) |
| 2026-03-15 | Qualité logs weapon_parser : `conf='sentinel'`/`'no_weapon'` + format COMPLETE compact (`match=XXXXXXXX… k=N \| H M L s ? \| fe fa s \| warn`) |
| 2026-03-15 | Navigation last_match : boutons ◀/▶ entre matchs filtrés (`_resolve_nav_index()`) |
| 2026-03-13 | Couverture tests `migrations.py` (lacunes v5.5–v5.7) |
| 2026-03-13 | Conflit `shared_matches.duckdb` — sync depuis UI Streamlit |
| 2026-03-13 | Hover thumbnail sur les noms de cartes (tableaux HTML) |
| 2026-03-13 | Détection de langue système dans `LevelUp.sh` / `LevelUp.bat` |
| 2026-03-13 | [UI] Heatmap performance par joueur × carte — Page Teammates |
| 2026-03-13 | [UI] Performance par carte vs historique — vues escouade et joueur |
| 2026-03-13 | Audit Pandas → Polars — résidus nettoyés |
| 2026-03-13 | Traductions FR manquantes dans migration metadata |
| 2026-03-13 | Images citations d'armes incorrectes |
| 2026-03-12 | `custom_rules.py:103` — `compute_annexion_forcee` implémentée |
| 2026-03-08 | Bug #0 : match invisible post-sync — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | Bug #1 : `win_rate` unifié sur `NULLIF(WIN+LOSS, 0)` dans `analytics.py` et `trends.py` |
| 2026-03-08 | Bug #5 : NaN-check fragile dans `match_view.py` → `is not None` |
| 2026-03-08 | Dette : guards obsolètes + dead code `_ensure_performance_score_column()` supprimés |
| 2026-03-08 | Magic numbers outcomes → `Outcome` enum + constantes `_WIN`/`_LOSS` SQL |
| 2026-03-08 | i18n : clés `PAIR_FR` restaurées, 342 entrées redondantes supprimées, doublon `tm_session_trend` retiré |
| 2026-03-08 | Kwargs legacy SyncScope dépréciés + `scope=SyncScope(...)` opérationnel |
| 2026-03-08 | `career.py` migré vers `get_cached_repository_st()` |
| 2026-03-08 | Perf UI — vues matérialisées lazy, pagination SQL, projections fines, `@fragment_if_available` |
| 2026-03-08 | CI/CD — détection de régression + pre-commit hook |
| 2026-02-25 | v5.3 LUSR stabilisation + UI Carrière |
| 2026-02-20 | v5.2 : Filtres intent-based + Stats PvE Firefight |
| 2026-02-17 | Release v5.1 — architecture shared-only |
| 2026-02-15 | Remédiation P0/P1 sécurité SQL + conformité Streamlit |
