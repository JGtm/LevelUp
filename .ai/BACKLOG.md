# BACKLOG — Tâches et TODO centralisés

> Mis à jour le 2026-03-15.

---

## ✅ v6 — Migration UI → DuckDBRepository (complète)

> Toutes les pages UI n'émettent plus de requêtes DuckDB directement.
> Architecture v6 DB-abstraction en place côté UI.

| Phase | Périmètre | État |
|-------|-----------|------|
| Phase 1 | Accès critiques (`player_provisioning`, `cache_filters`, `multiplayer`) | ✅ |
| Phase 2 | Pages `career`, `career_lusr`, `explorer` + `CareerMixin` | ✅ |
| Phase 3 | Pages `career_encounters`, `career_top_matches`, `match_view_encounters`, `media_library`, `session_compare` | ✅ |

**Exception intentionnelle** : `teammates_synergy._db_has_xuid` — scanne des chemins DB arbitraires en boucle, impossible à proxifier via le repo.

> **Dette documentée (priorité basse)** : le fallback de `_resolve_player_db_path` ouvre N connexions DuckDB (O(n) sur `data/players/*/stats.duckdb`) pour identifier une DB par son xuid dans `sync_meta`. Il serait possible de le remplacer par un appel `repo.resolve_gamertag(xuid)` (1 requête `shared.xuid_aliases`) pour obtenir le nom de dossier canonique. Ratio gain/effort défavorable pour l'instant : cas rare en pratique (<10 joueurs), et passer `repo` jusqu'aux callsites nécessiterait de refactoriser `_build_squad_player_list` (déjà PLR0913) en TypedDict/dataclass. À traiter si le scan devient mesurable ou si le nombre de joueurs enregistrés dépasse ~20.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-15 | Phase 3 v6 : migration complète `duckdb_read_only` UI → repo — 7 fichiers migrés, 17 tests + 9 tests antagonistes, 4764 tests passent |
| 2026-03-15 | Phase 2 v6 : `career`, `career_lusr`, `explorer` migrés + `CareerMixin` créé |
| 2026-03-15 | Migration last_match : requêtes directes → DuckDBRepository (`load_player_match_enrichment`, `is_abandoned_match`) — 12 tests |
| 2026-03-15 | Fixes Phase 1 v6 : `player_provisioning.py` bare connect, `cache_filters.py` `_get_connection()` privé, `multiplayer.py` dead code — 6 tests |
| 2026-03-15 | Couche résolution gamertag→XUID : `lookup_xuid_for_gamertag()` dans `src/utils/xuid.py` + `GamertagResolverMixin` — 9 fichiers migrés, 11 tests |
| 2026-03-15 | v5.8 Wave 5 : nettoyage i18n playlists/modes obsolètes → `metadata.duckdb` |
| 2026-03-15 | v5.8 Wave 4 : suppression `highlight_events.gamertag` + helper `resolve_medal_name` |
| 2026-03-15 | v5.8 Wave 3 : nettoyage wrappers XUID + dead code outcomes → `Outcome` enum |
| 2026-03-15 | v5.8 Wave 2 : migration consommateurs directs (gamertags, KV pairs, assets) |
| 2026-03-15 | v5.8 Wave 1 : vues SQL `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full` + `GamertagResolverMixin` |
| 2026-03-15 | Fix weapon-parser : corrélation globale — taux `fire_event` 15% → 95% |
| 2026-03-15 | Navigation last_match : boutons ◀/▶ entre matchs filtrés |
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
| 2026-03-08 | i18n : clés `PAIR_FR` restaurées, 342 entrées redondantes supprimées |
| 2026-03-08 | Kwargs legacy SyncScope dépréciés + `scope=SyncScope(...)` opérationnel |
| 2026-03-08 | `career.py` migré vers `get_cached_repository_st()` |
| 2026-03-08 | Perf UI — vues matérialisées lazy, pagination SQL, projections fines, `@fragment_if_available` |
| 2026-03-08 | CI/CD — détection de régression + pre-commit hook |
| 2026-02-25 | v5.3 LUSR stabilisation + UI Carrière |
| 2026-02-20 | v5.2 : Filtres intent-based + Stats PvE Firefight |
| 2026-02-17 | Release v5.1 — architecture shared-only |
| 2026-02-15 | Remédiation P0/P1 sécurité SQL + conformité Streamlit |
