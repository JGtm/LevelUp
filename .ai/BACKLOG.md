— Tâches et TODO centralisés

> Mis à jour le 2026-03-30.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
| 2026-03-30 | **i18n — Table `asset_translations` peuplée dans `metadata.duckdb`** : 9 674 traductions (698 assets × 14 langues BCP-47). Script `populate_asset_translations.py` réécrit avec `_build_version_id_cache()` (version_id SPNKr requis, `""` → 404), parallélisme `asyncio.gather` sur les 14 langues, reprise possible. |
| 2026-03-30 | **Fix critique — `v_match_full` sans traductions en prod** : `_try_attach_meta_for_views()` cherchait `meta.maps` (table absente en v6) → toujours `None` → vue créée sans JOINs i18n. Fix : vérifier `meta.asset_translations`. `_create_v_match_full()` : suppression des 4 JOINs legacy (`meta.maps/playlists/playlist_map_mode_pairs/game_variants`), 8 JOINs `asset_translations` (en-US + fr-FR × 4 types). Vue recréée en prod : "Starboard"→"Tribord", "The Pit"→"La fosse", etc. |
| 2026-03-30 | **Docs — Renommage ARCHITECTURE_V5 → V6** : `git mv` + mise à jour contenu (titre, version 6.3.0, `shared_matches_v2.duckdb`). §6 asset_translations ajouté dans la version FR. Toutes les références mises à jour : `CLAUDE.md`, `README.md`, `README_FR.md`, `FR/README.md`, `FR/COMMANDS.md`, `.ai/project_map.md`, `.ai/START_HERE.md`. |
| 2026-03-30 | **Docs — CHANGELOG 6.3.0** : entrées EN + FR documentant `asset_translations`, refonte `v_match_full` v6, fix `_try_attach_meta_for_views`. |
| 2026-03-30 | **Normalisation des labels de modes de jeu (v6.2.1)** : `resolve_display_mode()` dans `src/analysis/mode_display.py`, colonne `canonical_category` dans `mode_prefix_names`, 29 overrides dans `mode_pair_overrides`, `translate_pair_name` délégue au resolver, fichier plat de contrôle généré et validé. |
| 2026-03-30 | **Audit KDA locaux → `efficiency` (v6.2.1)** : sémantiques séparées — `p.kda` API conservé per-match, agrégats session/carte/cumul renommés `efficiency`/`session_efficiency` ; clés i18n `efficiency`/`efficacité` ajoutées ; 6 modules `src/analysis/` mis à jour (`cumulative.py`, `stats.py`, `_performance_relative.py`, `_performance_relative_helpers.py`, `_performance_session.py`, `stats.py` domain model). |
| 2026-03-27 | **Bug — `index_media.py --force` levait `ConstraintError: Duplicate key`** : quand `force_rescan=True`, `existing` était laissé vide `{}` → toutes les entrées considérées "nouvelles" → INSERT sur des clés déjà présentes. Fix : `existing` est toujours chargé depuis la DB ; `force_rescan` contourne uniquement le filtre delta `mtime`. Ré-indexation JGtm (73 médias) exécutée avec succès après fix. |
| 2026-03-26 | **Bug critique — `mv_player_matches` recalcule le KDA au lieu de lire la valeur API** : vue recréait `(kills + assists/3)/deaths` au lieu de `COALESCE(p.kda, fallback)`. Fix : détection dynamique `has_kda_col` (même pattern `has_enemy_mmr`) + génération SQL conditionnelle. |
| 2026-03-26 | **UX — Score d'équipe supérieur aux scores individuels (En-tête Page Coéquipiers)** : carte équipe n'affichait pas les bonus collectifs. Fix : `_render_compact_team_card` calcule `bonus = score - base_avg` et affiche `"moy. X (+Y collectif)"` quand > 0. |
| 2026-03-26 | **Bug — Colonne "Dernière rencontre" incohérente (Page Match · Encounters)** : SQL `MAX(start_time)` incluait le match courant et les matchs futurs. Fix : `filter_past` CTE + `_fetch_match_start_time` helper + guard `days = max(0, delta.days)` + colonne renommée "Précédente rencontre" + "1ère rencontre" pour les nouvelles têtes. |
| 2026-03-26 | **Bug annexe — `datetime.utcnow()` déprécié dans `career_lusr.py`** : remplacé par `datetime.now(timezone.utc).replace(tzinfo=None)`. |
| 2026-03-26 | **Bug — Médias mal rattachés aux matchs (décalage fuseau horaire)** : `epoch(capture_end_utc)` → `epoch(timezone('UTC', capture_end_utc))` dans `associate_with_matches()` + EXIF naïf ignoré (heure locale caméra, pas UTC). Ré-indexation requise (faite pour JGtm le 2026-03-27). |
| 2026-03-26 | **Bug RÉCURRENT CRITIQUE — Session escouade absente du graphe "Évolution de la performance"** : root cause A (fanout ouvrait shared en R/W → conflit handle Streamlit) fixée via Phase J (`shared_read_only=True` dans `_engine_fanout.py`). Fix défensif LEFT JOIN dans `_performance_squad._join_perf_frames()`. Les deux chemins de fix documentés dans l'audit sont implémentés. |
| 2026-03-26 | **Bug — Stats coéquipiers absentes (Page Teammates)** : résolu par le fix fanout R/O (Phase J). La root cause était identique au bug session escouade — fanout silencieux → PME coéquipier non créées. À revalider sur la prochaine session de jeu. |
| 2026-03-26 | **Bug annexe — `get_sync_metadata` lit mauvaise DB** : `SELECT last_sync_at FROM meta.sync_meta WHERE xuid=?` → `SELECT value FROM sync_meta WHERE key='last_sync_at'` dans la player DB. Fix commité dans `_diagnostic_repo.py` (Phase F). |
| 2026-03-26 | **Piste — Crashes silencieux (Page Coéquipiers · Top medals)** : source principale (connexions zombies fanout R/W) supprimée par Phase J. Si non récurrent → archivé. |
| 2026-03-21 | **Bug — Frags vs. détail armes (double-comptage melee)** : melee kills filmés attribués à l'arme tenue + `melee_kills` API → double-comptage. Fix : remainder `api_total - film_kills` dans 3 fichiers + `load_total_kills_for_player()` + 2 nouveaux tests. |
| 2026-03-21 | **UI — Graphe stats/min escouade : morts sous l'axe** — `plot_per_minute_timeseries` : deaths tracées en négatif (`dpm_neg`), `customdata[5]` = valeur absolue, `hover_dpm_neg` i18n, ticks Y absolus via `build_symmetric_abs_ticks` (extrait dans `src/visualization/_permin_helpers.py`). `timeseries.py` à exactement 500L. |
| 2026-03-21 | **Maintenance — Nettoyage dossier `scripts/`** — 10 scripts investigation → `scripts/investigation/` + README ; `cleanup_legacy_tables.py` + `cleanup_player_dbs_v5.py` → `scripts/_archive/` ; `.tmp.*` supprimés. |
| 2026-03-21 | **CI — Scripts exclus par `.gitignore`** — `check_code_size.py` → `enforce_size_limits.py` ; `check_imports.py` → `validate_imports.py` ; stubs `test_page_router_smoke.py` + `test_page_router_regressions.py` créés. Références mises à jour dans `ci.yml`, `.pre-commit-config.yaml`, `test_code_quality.py`. |
| 2026-03-21 | **UI — Notation de session escouade (Page Coéquipiers)** — `compute_squad_performance_score()` dans `src/analysis/_performance_squad.py` ; `SQUAD_GRADE_THRESHOLDS` + `resolve_squad_grade()` dans `performance_config.py` ; `render_squad_session_header()` + `_render_squad_score_block()` dans `src/ui/components/performance.py` ; 7 clés i18n `squad_grade_*` dans `src/ui/i18n/pages/teammates.py` ; bloc tendance K/D remplacé dans `teammates.py` ; 18 tests unitaires. |
| 2026-03-21 | **Perf — `_MAX_CONCURRENT_CHUNKS`** : déjà à 50 en production (`weapon_extraction_service.py`). Tâche obsolète — objectif déjà atteint. |
| 2026-03-19 | **Medal definitions en BDD** — table `medal_definitions` dans `metadata.duckdb` (167 médailles, DB-first + JSON-fallback). Migration, script population, CLI `--medal-metadata`, `MedalsMixin.load_medal_definitions()` / `get_medal_label()`, UI DB-first dans `medals.py`, 16 tests unitaires + 4 intégration. Orphan `citations_{fr,en}.json` supprimés. |
| 2026-03-19 | **Phase 8 — Couche centralisée médailles** (`medal_definitions.py`) — `src/data/medal_definitions.py` source canonique unique ; `_medal_data.py` thin re-export ; `medals.py` wrapper `@st.cache_data` délégant ; `_medals_repo.py` délègue. 3 chemins DB indépendants → 1. Fallbacks JSON applicatifs supprimés de `medals.py`. JSON `static/medals/*.json` conservés (source pour `populate_medal_metadata.py`). 51 tests passent. Commit `88d5cf0`. |
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
| 2026-03-28 | [v6.2] Badges Remontada / Débandade / Contre-Remontada — `DominanceFlag` 3-5, `comeback_analysis.py`, `comeback_backfill.py`, `--comeback-badges` CLI |
| 2026-03-28 | [v6.2] Unification vue coéquipier unique → vue escouade — `f2_xuid` optionnel, suppression `render_single_teammate_view` |
| 2026-03-28 | [v6.2] Graphe combiné Frags↑/Morts↓ — `plot_trio_kills_deaths()`, axe Y symétrique, `safe_chart_render()` |

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

---

### [refacto] Supprimer title= de apply_halo_plot_style (finalisation Axe G)

**Noté le** : 2026-04-05
**Priorité** : Basse (warning en place, aucun impact fonctionnel)

**Contexte** : L'Axe G du plan V2 a externalisé ~74 titres Plotly vers `st.subheader()` et ajouté un `DeprecationWarning` dans `apply_halo_plot_style` quand `title=` est non vide. La prochaine étape est de supprimer complètement le param `title` de la signature et de la logique interne de `theme.py`, une fois que tous les call-sites auront été vérifiés. Vérifier avec `grep -rn "title=" src/visualization/` qu'il ne reste aucun appel actif avant de supprimer.

---

### [refacto] Migrer les pages timeseries vers SingleSeriesChartData

**Noté le** : 2026-04-05
**Priorité** : Basse

**Contexte** : `SingleSeriesChartData` (créé en Axe F) centralise la construction des séries temporelles single-joueur. La moitié des fonctions `plot_*` dans `src/visualization/timeseries*.py` construisent encore leurs traces manuellement. Les migrer réduirait la duplication et rendrait le downsampling automatique.

---

### [perf] Indexer les requêtes SQL non indexées identifiées dans les audits

**Noté le** : 2026-04-05
**Priorité** : Basse

**Contexte** : Plusieurs requêtes DuckDB sur `match_participants`, `medals_earned` et `weapon_kills` effectuent des scans complets alors qu'un index sur `(xuid, match_id)` accélérerait les lookups par joueur. À investiguer avec `EXPLAIN ANALYZE` sur les requêtes lentes identifiées lors des audits de performance.

---

### [backfill] Recalculer `match_citations` pour `spartan_carnage` (tous les joueurs)

**Noté le** : 2026-04-04
**Priorité** : Haute (données incorrectes en production — valeur actuelle = `max_killing_spree` brut au lieu de la somme des médailles de spree)

**Contexte** : Le mapping `spartan_carnage` était de type `stat` (source : `max_killing_spree`). Corrigé le 2026-04-04 : type → `medal`, `medal_ids` = `2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471` (Folie meurtrière, Massacre, Émeute, Carnage, Cauchemar, Croque-mitaine, Croque-mort, Démon). La DB `metadata.duckdb` doit être mise à jour quand Streamlit est arrêté (fichier verrouillé).

**Prérequis** : Arrêter Streamlit, puis exécuter :
```python
con.execute("""
    UPDATE citation_mappings
    SET mapping_type = 'medal',
        medal_ids = '2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471',
        stat_name = NULL,
        notes = 'Multijoueur — médailles de killing spree (5→40 kills sans mourir)',
        updated_at = CURRENT_TIMESTAMP
    WHERE citation_name_norm = 'spartan_carnage'
""")
```

**Backfill** :
```bash
python scripts/backfill_data.py --all --citations --force-citations
```

**Impact** : Toutes les lignes `match_citations` avec `citation_name_norm = 'spartan_carnage'` sont à recalculer pour tous les joueurs.

---

### [refacto] Traductions d'assets : supprimer les fallbacks Python, DB = source de vérité

**Noté le** : 2026-04-02
**Priorité** : Haute (cohérence + suppression de code mort — la DB est fiable, les dicts Python dupliquent sans valeur ajoutée)

**Contexte** : Audit des traductions d'assets (maps, playlists, modes, armes). Deux pipelines coexistent :
- **Pipeline SQL (bonne voie)** : `v_match_full` expose déjà `map_name_fr`, `playlist_name_fr`, `pair_name_fr`, `game_variant_name_fr` via JOIN sur `asset_translations` (metadata.duckdb). `weapon_labels` est la source de vérité des noms d'armes (EN + FR).
- **Pipeline Python (legacy)** : dicts statiques `WEAPON_INT_TO_NAME` / `WEAPON_NAME_FR` dans `_weapon_data.py`, fonctions `normalize_map_label` / `translate_pair_name` / `translate_playlist_name` qui repassent par SQL ou des heuristiques string au lieu de lire les colonnes déjà resolues dans le DataFrame.

**Décision** : La BDD est la source de vérité. Les fallbacks Python ne servent que si asset_translations est vide (cas onboarding premier lancement). En dehors de ce cas, tout duplicated est du code mort.

**Tâches concrètes** :

1. **Unifier `is_uuid_like`** — deux implémentations (`src/ui/translations.py::_is_uuid_like` et `src/app/helpers.py::is_uuid_like`). Déplacer dans `src/utils/strings.py`, supprimer les copies. *(15 min)*

2. **Supprimer `_normalize_mode_label` dans `teammates_helpers.py`** — version locale qui réimplémente `normalize_mode_label` de `src/app/helpers.py`. Remplacer par import direct. *(10 min)*

3. **Exploiter `map_name_fr` / `pair_name_fr` / `game_variant_name_fr` du DataFrame** — dans `_filters_apply.py`, lire ces colonnes en priorité (déjà dans le DF via `v_match_full`), fallback `map_name` si null. Supprimer le chemin `normalize_map_label` comme chemin principal (garder uniquement comme guard UUID). *(2h)*

4. **Extraire `st.session_state` hors de `normalize_mode_label`** — la fonction appelle `st.session_state` pour lire `app_settings` → couplage UI/métier. Passer `lang` et `normalize` en paramètres explicites. *(1h)*

5. **Armes : DB first, dicts Python en dernier recours** — inverser la priorité dans `_weapon_data.py::resolve_weapon_label()` : lire `weapon_labels` (metadata.duckdb) en premier, consulter `WEAPON_INT_TO_NAME` / `WEAPON_NAME_FR` uniquement si `weapon_id` absent de la table. À terme, supprimer les dicts statiques quand `weapon_labels` couvre 100% des armes connues. *(1h)*

6. **Cache `@st.cache_data` sur `translate_playlist_name`** — appelée sans cache via `build_mapping` à chaque rendu, alors que `_load_mode_tables` est déjà cachée. Wrapper TTL court (300s). *(30 min)*

**Ordre recommandé** : 1 → 2 → 4 → 3 → 5 → 6 (du moins risqué au plus impactant)

**Impact** : Suppression ~100L de code mort, cohérence garantie avec la BDD, testabilité améliorée (plus de `st.session_state` dans la logique).

---

### [refacto] `safe_chart_render` manquant sur les pages Career et LUSR

**Noté le** : 2026-04-02
**Priorité** : Haute (stabilité — une exception dans une figure crashe la page entière)

**Contexte** : Audit issu de la revue globale des visuels (`CHARTS_AND_TABLES.md`). Sur 81 appels `st.plotly_chart`, seulement ~4 sont enveloppés dans `with safe_chart_render():`. Les pages non protégées identifiées : `career.py` (L188, L224, L316), `career_lusr.py` (L176), `citations.py`, `match_view_charts.py`, `match_view_participation.py`.

**Solution** : Enrouler chaque `st.plotly_chart` exposé dans `safe_chart_render`. Ne pas toucher les appels déjà à l'intérieur du wrapper.

**Effort estimé** : 30 min.

---

### [refacto] Centraliser `_downsample_for_plot` dans `src/visualization/`

**Noté le** : 2026-04-02
**Priorité** : Haute (performance — pages `win_loss`, `teammates`, `career` envoient 500+ points bruts à Plotly)

**Contexte** : La fonction `_downsample_for_plot` (MAX_PLOT_POINTS=200) est définie et utilisée uniquement dans `src/ui/pages/timeseries.py`. Les autres pages timeline (`career_lusr`, `teammates_charts`, `win_loss`) n'appliquent aucun downsampling.

**Solution** : Déplacer `_downsample_for_plot` dans `src/visualization/theme.py` ou `src/visualization/_compat.py` (déjà importé partout). Appeler depuis toutes les fonctions `plot_timeseries*` et `plot_average_life` avant de construire les traces.

**Effort estimé** : 1h.

---

### [refacto] Découper `maps_outcome.py` (590 L → limite 500 L)

**Noté le** : 2026-04-02
**Priorité** : Moyenne (seul fichier `visualization/` en dépassement)

**Contexte** : `src/visualization/maps_outcome.py` fait 590 lignes — dépasse la limite projet de 500 L. Contient 4 fonctions indépendantes : `plot_map_lollipop`, `plot_map_outcome_timeline` (DÉSACTIVÉ), `plot_map_winrate_bullet`, `plot_map_perf_vs_history`.

**Solution** : Extraire vers `_maps_outcome_bullet.py` (bullet + perf_vs_history) et `_maps_outcome_lollipop.py` (lollipop + timeline), réexporter depuis `maps_outcome.py` pour ne pas casser les imports.

**Effort estimé** : 2h.

---

### [refacto] Magic numbers de hauteur → `PLOT_CONFIG.default_height`

**Noté le** : 2026-04-02
**Priorité** : Basse (cosmétique — 8 valeurs hardcodées : `320`, `420`, `400`)

**Contexte** : `match_bars.py` (×2), `timeseries_combat.py` (×4), `_timeseries_progression.py` (×2) utilisent des hauteurs numériques au lieu de `PLOT_CONFIG.default_height` ou d'une constante nommée (`HEIGHT_COMPACT = 320`, `HEIGHT_NORMAL = 420`).

**Solution** : Ajouter `HEIGHT_COMPACT` et `HEIGHT_NORMAL` dans `src/config.py` (ou `PLOT_CONFIG`) et remplacer les magic numbers.

**Effort estimé** : 15 min.

---

### [refacto] PLR0913 : réduire les fonctions à trop d'arguments dans `_perf_progression.py`

**Noté le** : 2026-04-02
**Priorité** : Basse (3 violations `# noqa: PLR0913` dans ce seul fichier, 13 au total dans `src/visualization/`)

**Contexte** : Les fonctions `plot_cumulative_kd_with_ci`, `plot_net_score_per_hour`, `plot_ewma_kd` et `plot_match_kill_death_timeline` ont 6–8 paramètres. Les suppressions `# noqa: PLR0913` masquent la dette.

**Solution** : Regrouper les paramètres optionnels de style/config dans un `@dataclass PlotOptions` ou `TypedDict` (couleurs, largeur de ligne, alpha CI, show_kde, etc.). Les paramètres de données restent positionnels.

**Effort estimé** : 3h (à faire par fichier, pas tout d'un coup).

---

###  Amélioration v7++ — Backfill multi-flags : vectoriser le calcul per-match des performance scores (v7+)

**Noté le** : 2026-03-26
**Priorité** : Basse (non bloquant — le chemin normal sync app est déjà vectorisé)

**Contexte** : Quand `--force-performance-scores` est combiné avec d'autres flags backfill (ex. `--medals --performance-scores`), la boucle séquentielle de l'orchestrateur appelle `compute_performance_score_for_match()` une fois par match. Cette fonction fait une requête SQL individuelle à chaque itération pour charger l'historique des 50 derniers matchs → ~1 req/match → lent sur un grand historique.

Le shortcut `_perf_force_only` (v6) bypasse cette boucle quand `--force-performance-scores` est le *seul* flag, mais pas quand combiné à d'autres.

**Solution envisagée** : Pré-charger l'historique complet en une seule requête avant la boucle (comme `batch_compute_performance_scores`), le passer en contexte à `compute_performance_score_for_match()`, et supprimer la requête SQL interne per-match.

**Impact** : Uniquement les backfills multi-flags. Le sync normal (`engine._run_post_sync_compute`) est déjà sur le chemin batch vectorisé.

---

### Script d'analyse des kills par arme pour un match donné (v7+)

**Noté le** : 2026-03-27
**Priorité** : Basse

**Contexte** : Outil de diagnostic/exploration permettant d'analyser en détail tous les kills d'un match donné, pour un joueur donné.

**Entrée** : `match_id` + `gamertag`

**Sortie** : Tableau avec, pour chaque kill :
- `match_id`
- Paire `killer` / `victim` (gamertag ou xuid si inconnu)
- `timestamp` en format `mm:ss`
- `weapon_id` (même si inconnu / non résolu)

**Ce que ça impliquerait** :
1. Requête sur `weapon_kills` (shared_matches_v2) jointure `killer_victim_pairs` + `xuid_aliases`
2. Résolution des gamertags via `v_gamertag_lookup`
3. Conversion `timestamp_ms` → `mm:ss`
4. Affichage : script CLI + éventuellement widget UI dans la page d'un match

**Complexité estimée** : Faible (données déjà disponibles dans `weapon_kills` + vues v6)

**Priorité** : Basse — outil de debug / exploration, non bloquant pour les features v7

---

### [investigation] Déterminer le vrai début de match (countdown non fiable)

**Noté le** : 2026-04-02
**Priorité** : Moyenne (impacte la précision du graphe premier frag/première mort dans la page Teammates et les stats temporelles en général)

**Contexte** : Le champ `MatchInfo.PlayableDuration` de l'API SPNKr est censé représenter la durée réelle du gameplay (hors countdown). On en déduit le countdown : `countdown = duration_seconds - playable_duration_seconds`, et par extension `real_start_time`.

Mais ce champ s'avère non fiable dans la pratique : pour de nombreux matchs (notamment Ranked Arena), `PlayableDuration = Duration` → countdown calculé = 0, même lorsqu'on sait qu'un temps de préparation existe. On ne peut donc pas distinguer "pas de countdown" de "API a renvoyé une valeur incorrecte".

**Impact actuel** : `time_ms` dans `highlight_events` est relatif au début du fichier film (qui inclut le countdown). Le graphe "Premier frag / Première mort" soustrait le countdown estimé, mais si ce dernier vaut 0 à tort, les valeurs restent décalées vers les premières secondes.

**Exploration effectuée le 2026-04-02** — dépôts [`dend/grunt`](https://github.com/dend/grunt) et [`dend/filmshell`](https://github.com/dend/filmshell) analysés. Résultats :

**Grunt API** : Le modèle `MatchInfo` (TypeScript, `match-info.ts`) confirme que les seuls champs temporels disponibles sont `StartTime`, `EndTime`, `Duration` et `PlayableDuration`. **Pas de `GameplayStartTime`** ni de champ similaire. L'API officielle n'expose rien de plus que ce qu'on utilise déjà.

**FilmShell — format binaire** : Pas d'event "spawn" explicite avec timestamp dans le format film. Le contenu est uniquement des frames de **position à ~60Hz** (index `frame` = position dans le flux binaire, pas un timestamp UTC). Le marqueur de frame est `A0 7B 42`. Deux pistes intéressantes :
- `FilmCustomData.FilmLength` (millisecondes) : longueur totale du film exposée par l'API film. Si cette valeur diffère de `Duration * 1000`, le film ne couvre pas le match complet. Si `FilmLength ≈ PlayableDuration * 1000`, cela confirmerait que le film démarre après le countdown — et donnerait une mesure de countdown indépendante.
- **Discontinuités de position** : le code filmshell filtre les grands sauts (`DISCONTINUITY_THRESHOLD = 4000`) en les sautant — ces sauts correspondent aux morts/respawn et au spawn initial. La **première discontinuité majeure** dans le film est le spawn initial du joueur (transition de la zone de lobby vers la carte). Son index de frame, divisé par le framerate (~60Hz calculé depuis `FilmLength`), donnerait `gameplay_start_ms`. **C'est la piste la plus prometteuse sans appel API supplémentaire.**

**Pistes restantes (sans exploration approfondie)** :

1. **`FilmLength` vs `Duration`** : comparer sur un batch de matchs connus. Si `FilmLength < Duration * 1000` systématiquement pour les matchs Ranked, cela prouverait que le film ne couvre pas le countdown → countdown = `Duration - FilmLength/1000`.

2. **Premier spawn via discontinuité filmshell** : réimplémenter en Python la détection du premier grand saut de coordonnées dans les chunks décompressés (marqueur `A0 7B 42`, offset +10/+11 pour coord1). L'index de frame du premier saut × (1000/Hz) = `countdown_ms`. Nécessite un POC Python sur les chunks déjà téléchargés.

3. **`highlight_events` — première occurrence globale** : `MIN(time_ms)` sur *tous* les événements du match (tous types, tous joueurs) est une borne supérieure du début réel. Utile en sanity check : si `countdown_calculé > MIN(time_ms)`, le countdown est surestimé.

4. **Table de valeurs typiques par mode** : Ranked Arena → countdown ~0-5s, BTB/Custom → 10-20s. Peut servir de fallback quand `PlayableDuration` semble incohérent.

**Ce qu'il faudrait faire** :
1. **(facile, ~1h)** Requête SQL sur les matchs connus : comparer `FilmLength` (si stocké dans `media_files`) vs `duration_seconds`. Si absents, télécharger quelques films de référence via filmshell sur des matchs avec countdown connu.
2. **(moyen, ~4h)** POC Python : lire les chunks `filmChunkN_dec` de filmshell, détecter le marqueur `A0 7B 42`, extraire les deltas coord1/coord2, identifier le premier grand saut → `countdown_ms`. Valider sur 3-4 matchs de référence.
3. Si validé : ajouter colonne `gameplay_start_ms` dans `match_registry`, backfillable pour les films déjà téléchargés.

**Référence** : Correction partielle appliquée en session 2026-04-02 (`_teammates_first_events_queries.py` + `_events_repo.py`), mais la source `PlayableDuration` reste non fiable pour les matchs Ranked Arena.

---

### Kills environnementaux — catégorie dédiée (v7++)

**Contexte** : La médaille **Kong** (kill via baril projeté) est actuellement comptée dans `GRENADE_MEDALS` faute d'une meilleure catégorie. Ce classement est approximatif — il est impossible de savoir avec certitude si l'API inclut ces kills dans `GrenadeKills` ou non.

**Idée** : Créer une catégorie `environmental_kills` (ou `environmental`) pour regrouper les kills causés par l'environnement sans arme tenue :
- Baril projeté (médaille **Kong**)
- Potentiellement : chutes provoquées, explosions de véhicules, etc.

**Ce que ça impliquerait** :
1. Nouvelle colonne `environmental_kills` dans `match_participants` (migration DuckDB)
2. Nouveau bit `ParticipantBits.ENVIRONMENTAL_KILLS` dans `constants.py`
3. Retirer `Kong` de `GRENADE_MEDALS` → nouvel ensemble `ENVIRONMENTAL_MEDALS`
4. Logique de réconciliation filmshell dédiée dans `_weapon_kills_repo.py`
5. Backfill pour l'historique existant
6. Affichage UI éventuel

**Complexité estimée** : Moyenne (surtout le backfill + validation que l'API expose bien des compteurs séparés)

**Priorité** : Basse — les barrel kills sont extrêmement rares, l'impact sur les stats est négligeable. À faire uniquement si on veut une exhaustivité totale des catégories de kills.

---

## 🔮 Roadmap v6.4+

---

### [v6.4] Score de forme — indice de progression court terme

**Noté le** : 2026-03-28 | **Priorité** : Moyenne

```
form_score = moy_perf_score(14 derniers matchs) - moy_perf_score(90 derniers matchs)
```

- Positif → en forme / Négatif → creux de forme
- Calculable par `mode_category` (Arena, BTB, Ranked)
- Données : `player_match_enrichment.performance_score` déjà disponible

**Implémentation** :
1. `compute_form_score(gamertag, anchor_date)` dans `src/analysis/performance_score.py`
2. Colonne `form_score FLOAT` dans `sessions` (migration)
3. Calculé au post-sync, affiché sur la page d'accueil / profil (sparkline 30j + indicateur ↑↓)

---

### [v6.4] Détection de changement de niveau (breakpoints)

**Noté le** : 2026-03-28 | **Priorité** : Basse

Moyenne mobile double (14j vs 90j) — croisements ascendant/descendant = pallier détecté.

**Implémentation** :
1. `detect_level_breakpoints(df: pl.DataFrame) -> list[Breakpoint]` dans `src/analysis/progression.py`
2. `Breakpoint(date, direction: "up"|"down", delta_perf, n_matches_confirmed)` — seuil ≥10 matchs consécutifs
3. Table `progression_breakpoints` dans `stats.duckdb`
4. Overlay "cap franchi" sur les courbes de tendance

---

### [v6.4] Page Adversaires — Head-to-head, Nemesis, Proie

**Noté le** : 2026-03-28 | **Priorité** : Moyenne

Nouvelle page dédiée aux adversaires récurrents.

**Données** : tout dans `shared_matches_v2.duckdb` — `match_participants`, `killer_victim_pairs`, `match_registry`, `v_gamertag_lookup`.

| Métrique | Source |
|----------|--------|
| `matches_vs` | `match_participants` |
| `win_rate_vs` | `match_registry.outcome` |
| `kills_on` / `deaths_from` | `killer_victim_pairs` |
| `nemesis_score` = `deaths_from / max(1, kills_on)` pondéré | dérivé |
| `prey_score` = `kills_on / max(1, deaths_from)` pondéré | dérivé |

**Implémentation** :
1. `src/data/services/rivals_service.py` — `load_rivals_stats(gamertag, min_matches=3, limit=50)`
2. Nouvelle page `src/ui/pages/rivals.py`
3. Filtres : mode_category, fenêtre temporelle (30j/90j/all)
4. Exclure bots (`xuid LIKE 'bid(%'`), min_matches configurable

---

### [v6.4] Discord — Résumé de session post-sync

**Noté le** : 2026-03-28 | **Priorité** : Basse

Bouton `📤` dans la sidebar, actif ≥5 min après `last_match_end_time` (configurable).

**Contenu embed** : W-L/win rate, meilleur match, top médaille, badge comeback, composition escouade, rôles de soirée (Champion 🏆 / Maillon Faible 🍌 via `compute_impact_scores()`).

**Données** :
- Colonne `discord_notified_at TIMESTAMP DEFAULT NULL` dans `sessions` (migration)
- `discord_session_notify_delay_minutes` dans `app_settings.json` (défaut : 5)
- `src/utils/discord_notifier.py` à étendre

**Opt-in** : visible uniquement si `discord_session_notify = true` ET webhook configuré.

---

### [v6.4] Clutch moments — kills décisifs

**Noté le** : 2026-03-28 | **Priorité** : Basse

Trois types de kills clutch, par ordre de fiabilité :

| Type | Définition | Données |
|------|-----------|---------|
| **Spree-stopper** | Kill sur joueur avec médaille de série dans ce match | `medals_earned` × `killer_victim_pairs` |
| **Comeback clutch** | Kill en match `DominanceFlag.COMEBACK` / `COUNTER_COMEBACK`, joueur top-2 killers | `match_registry.comeback_flag` × `match_participants` |
| **Last-minute** | Kill dans les 60 dernières secondes d'un Slayer à ≤2 pts d'écart | `killer_victim_pairs.timestamp_ms` × `match_registry` |

**Stockage** : colonnes `clutch_kills INTEGER` + `clutch_type TEXT` dans `player_match_enrichment`.
**Backfill** : `--clutch-kills` dans `scripts/backfill_data.py`, logique dans `src/analysis/clutch_analysis.py`.
**Limites** : spree-stopper approximatif (pas de timestamp par médaille) ; last-minute dépend de la couverture filmshell.

---

### [v6.4] [feat/teammates] Précision du timer premier frag/mort

**Noté le** : 2026-04-02 | **Priorité** : Basse

**Problème** : `time_ms` dans `highlight_events` est relatif au début du **lobby** (chargement inclus), pas au coup d'envoi du combat. Ce décalage est variable selon maps/modes (~30–45s), ce qui rend les tranches de 0–15s quasi vides et décale tout vers la droite.

**Solutions envisagées** :

1. **Normalisation par premier event du match** — soustraire `MIN(time_ms) OVER (PARTITION BY match_id)` (tous joueurs confondus) pour aligner sur le "premier sang". Simple, mais biaise : le joueur mort en premier aura toujours `adjusted = 0s`.

2. **Capturer le timing du premier mouvement** — utiliser un event de type "premier déplacement" ou "spawn" comme temps 0. Non disponible aujourd'hui dans l'API SPNKr/filmshell ; nécessiterait une investigation sur les event_types non exploités (ex: `mode` events déjà présents dans `highlight_events`).

**Aucune action immédiate** — documenter pour investigation future.

---

## 🆕 Ajouté le 2026-04-03 — v6.3.1

---

### [v6.3.1][feat] Option pour désactiver l'affichage des records (page Teammates)

**Noté le** : 2026-04-03 | **Priorité** : Basse

**Périmètre** : La feature records est active **uniquement sur la page Teammates**, sur 4 graphes + 1 bloc textuel.

**État actuel** : Les records s'affichent toujours si les données sont disponibles. Pas de toggle UI. Calculés sur l'historique complet du joueur (pas filtré par session/date).

**Rendus visuels concernés** :

| Graphe | Visuel record | Fonctions |
|--------|--------------|-----------|
| Trio Metric (assists, ratio, accuracy…) | Barres fantômes hachurées `/` au-dessus des barres réelles | `plot_trio_metric()` → `add_record_shapes()` |
| Trio Kills/Deaths | Barres fantômes hachurées (kills +, deaths −) | `plot_trio_kills_deaths()` → `add_record_shapes()` |
| Killing Spree (bar chart) | Barres fantômes hachurées | `plot_multi_metric_bars_by_match()` → `add_record_shapes()` |
| HS+PK Stacked | Lignes horizontales colorées | `plot_hs_pk_stacked()` → `add_overlay_record_shapes()` |
| Stats par minute | Texte "Record : X/min" | `_render_per_minute_stats()` |

**Calcul** : `compute_squad_records()` / `compute_squad_records_per_map()` dans `src/analysis/squad_records.py`, appelés dans `src/ui/pages/_teammates_trio.py` (L203–231) et `teammates_views.py` (L183–199).

**Solution** :
1. Ajouter `show_records: bool = True` dans `app_settings.json`
2. Lire ce flag depuis `AppSettings` (Pydantic v2)
3. Dans `_teammates_trio.py` : conditionner le calcul `compute_squad_records()` et `compute_squad_records_per_map()` — passer `records=None` aux graphes si désactivé
4. Dans `teammates_views.py` : même chose pour les 2 appels `compute_squad_records()`
5. Dans `_render_per_minute_stats()` : masquer le bloc textuel record si le flag est `False`
6. Exposer le toggle dans la sidebar ou page Paramètres

**Impact si désactivé** : zéro calcul de records (gain perf mineur), graphes affichent uniquement les barres réelles sans barres fantômes ni lignes horizontales.

**Effort estimé** : 1h.

---

### [v6.3.1][fix] Mode "Quick Play" dans la notification Discord

**Noté le** : 2026-04-03 | **Priorité** : Basse

**Contexte** : La notification Discord affiche le mode brut `"Quick Play"` au lieu du nom traduit/normalisé. La couche `resolve_display_mode()` / `translate_pair_name()` n'est apparemment pas appelée au moment de la construction de l'embed Discord.

**Solution** :
1. Dans `src/utils/discord_notifier.py`, appliquer `resolve_display_mode(mode_raw, lang="fr")` (ou `"en"` selon la config webhook) avant d'injecter le nom de mode dans l'embed
2. Vérifier que `canonical_category` (table `mode_prefix_names`) est bien attachée lors de la construction du message
3. Tester sur un embed local avec `mode_raw = "Quick Play"` → attendu : `"Jeu rapide"` (FR) ou `"Quick Play"` normalisé (EN)

**Effort estimé** : 30 min.

---

## ❓ À détailler par l'utilisateur

> Ces items sont trop vagues pour être implémentés sans plus de contexte. Décrire le comportement attendu avant de les planifier.

---

### [?] Revue de code par ChatGPT

**Noté le** : 2026-04-03

Processus/outillage à clarifier : quels fichiers ? quel périmètre ? via API ou copier-coller manuel ? objectif de la revue (sécurité, qualité, style) ?

---

### [?] Tester/corriger sync sur le site

**Noté le** : 2026-04-03

Préciser : quel environnement ("le site" = prod déployée ? staging ?) — quels symptômes observés — quels gamertags concernés.

---
