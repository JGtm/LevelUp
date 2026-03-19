— Tâches et TODO centralisés

> Mis à jour le 2026-03-19.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-19 | **Migration `b5>>4`** — `scan_fire_events_b5` implémenté, `fire_seq%n_players` supprimé, `map_b2_to_player`/`group_events_by_pi`/`POV_PLAYER_INDEX` retirés, 25 nouveaux tests — 4968 tests passent. Relancer `--force-weapons --all` pour re-extraire. |
| 2026-03-19 | **Backfill enrichissement** JGtm + Madina97294 — 8 matchs du 18 mars rattrapés (performance_score, sessions, citations) |
| 2026-03-19 | **Fix 11 — Fan-out multi-joueurs** : `FanoutEnrichmentMixin` (`_engine_fanout.py`) + branchement dans `engine.py` après `_detach_shared_from_player_conn()`. Résout le manquement d'enrichissement local pour les joueurs qui ne sync pas eux-mêmes. |
| 2026-03-19 | **Fix 10 — Performance vs historique** : `performance_score` ajouté à `COLUMNS_COMMON` + JOIN `player_match_enrichment` dans `load_matches_as_polars` + `df_history` propagé dans `WinLossService` |
| 2026-03-19 | **Fix 9 — Radar escouade** : `radar_squad_ids` sauvegardé avant filtre UI ; DFs historiques séparés (`radar_me_df/f1/f2/f3`) passés à `render_trio_synergy_radar` |
| 2026-03-19 | **Fix 8 — Heatmap monochrome** : `compute_map_breakdown` lit `performance_score` depuis la colonne quand présente (fallback percentile supprimé pour les joueurs enrichis) |
| 2026-03-19 | **Fix 7 — Performance vue 1 coéquipier** : `enrich_with_performance_score` appelé pour `me_df` et `friend_df` dans `render_single_teammate_view` |
| 2026-03-19 | **Fix 6 — MediaFileStorageError icônes rang** : images rang converties en data URI base64 dans `career.py` (IDs Streamlit éphémères éliminés) |
| 2026-03-19 | **Fix 5 — Joueurs fantômes** : `_is_ghost_player` requiert la présence des clés stat + filtre appliqué uniquement dans `filter_encounter_xuids` (scoreboard non filtré — joueurs légitimes à 0 stats conservés) |
| 2026-03-19 | **Fix 4 — ratio=kda** : `ratio = pl.col("kda").alias("ratio")` dans `_finalize_polars_df` + `p.kda AS ratio` dans `_query_teammate_shared_stats` — source unique API, plus de recalcul |
| 2026-03-19 | **Fix 3 — Matrice d'impact** : `.unique(maintain_order=True)` dans `friends_impact_heatmap.py` |
| 2026-03-19 | **Fix 2 — Bots bid(33.0)** : `get_bot_name()` appelé dans `_build_encounter_rows` avant le fallback `xuid[:8]` |
| 2026-03-19 | **Fix 1 — ColumnNotFoundError map_name** : `mr.map_name` ajouté au SELECT de `load_friend_match_details` + `_FRIEND_DF_EMPTY_SCHEMA` mis à jour |
| 2026-03-19 | **Bonus — `resolve_weapon_display` fusion avant DB** : la fusion map est appliquée (étape 0) avant le lookup `weapon_labels`, évitant que M392 Bandit / Fuel Rod SPNKr contournent leur regroupement canonique |
| 2026-03-16 | Audit post-V6 : `weapon_kills` bit sync + logging, `v_gamertag_lookup` systématique, `shared_matches_v2.duckdb` production, LEGACY SyncScope supprimés, 17 nouveaux tests — 4799 tests passent |
| 2026-03-16 | Sprint refactor : splits fonctions/modules >80/500L, `_teammates_trio_helpers`, `_match_relations`, `_roster_loader` helpers, `render_trio_charts` DRY |
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
| 2026-03-13 | [UI] Heatmap performance par joueur × carte — Page Teammates |
| 2026-03-13 | [UI] Performance par carte vs historique — vues escouade et joueur |
| 2026-03-08 | Bug #0 : match invisible post-sync — suppression `_filters_loaded_*` dans `_clear_app_caches()` |
| 2026-03-08 | Perf UI — vues matérialisées lazy, pagination SQL, projections fines, `@fragment_if_available` |

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

### Migration `b5>>4` — Attribution armes (player_index définitif)

**Statut : NON MIGRÉ en production (2026-03-19)**

La formule `fire_seq % n_players` (actuellement en WD) est approximative. `b5 >> 4` est la solution définitive validée sur 3 matchs / 282 kills (84–95% de confiance). Voir `.ai/PLAYER_INDEX_FIRE-EVENTS_RESOLUTION.md` et `.ai/PLAN_B5_PRODUCTION_MIGRATION.md`.

**Fichiers à modifier** :

| Fichier | Changement |
|---------|-----------|
| `src/analysis/_weapon_scanners.py` | Extraire `b5` à chaque event, retourner `player_index = b5 >> 4`. Dédup par `byte_pos` proximity (≤2 bytes), pas par `(fire_counter, weapon)`. |
| `src/analysis/weapon_parser.py` | `scan_fire_events_all` : supprimer `n_players` et `ev["player_index"] = ev["fire_seq"] % n_players`. Le `player_index` vient du scan b5. |
| `src/data/services/weapon_extraction_service.py` | Supprimer `n_players = len(all_participants) or 8` passé à `scan_fire_events_all`. |

**Ce qui NE change PAS** : `correlate_kills_global` (filtre `ev["player_index"] == killer_pi` reste), `detect_pi_from_metadata`, `PLAYER_METADATA`.

**Après migration** : lancer `--weapons --force-weapons --all` pour re-extraire tous les matchs avec la formule définitive.

---

### Perf — Film chunks : augmenter `_MAX_CONCURRENT_CHUNKS`

**Fichier** : `src/data/services/weapon_extraction_service.py`

Passer de 5 à 7 (puis 10 si stable) connexions concurrentes au CDN Azure. Objectif : ~14s → ~8s par match.

⚠️ Non confirmé sans mesure : vérifier d'abord que les 429 sont gérés avec retry exponentiel avant d'augmenter. Tester sur 5+ matchs à 7 concurrent, mesurer taux d'erreur, puis décider.

---

### Noms et descriptions des médailles/citations en BDD

**Statut : À planifier**

Actuellement les noms et descriptions de médailles sont dans des fichiers JSON statiques (`static/medals/medals_{lang}.json`, `medals_descriptions_{lang}.json`). Les noms de citations sont dans `src/ui/translations.py`. Pas de table `medals` peuplée dans `metadata.duckdb`.

**Objectif** : centraliser noms et descriptions (médailles + citations) dans `metadata.duckdb` pour :
- Requêtes SQL directes avec JOIN (pas de résolution Python post-query)
- Source unique de vérité éditable sans redéployer le code
- Support futur de langues supplémentaires

**Approche envisagée** :
1. Créer/peupler la table `medals` dans `metadata.duckdb` (colonnes : `medal_name_id`, `name_fr`, `name_en`, `description_fr`, `description_en`, `is_custom`)
2. Créer une table `citation_descriptions` (ou enrichir `citation_mappings`)
3. Script de population depuis les JSON existants (`scripts/populate_medal_metadata.py`)
4. Migration step pour les nouvelles colonnes
5. Les JSON restent en fallback read-only (backward compat)
