— Tâches et TODO centralisés

> Mis à jour le 2026-03-25.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
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

---

## 🔄 Aucune tâche en cours

---

## 📋 Backlog

### 🔴 BUG RÉCURRENT CRITIQUE — Dernière session absente du graphe "Évolution de la performance d'escouade" (Page Coéquipiers)

**Signalé le** : 2026-03-25 (récurrent — déjà observé plusieurs fois après chaque sync)
**Gravité** : Critique — le graphe est la principale feature de tracking escouade. La session la plus récente (la seule qui intéresse l'utilisateur après un sync) n'apparaît jamais.
**Audité le** : 2026-03-25 (code vérifié en profondeur)

---

**Symptôme exact** : Après un sync, le graphe "Évolution de la performance d'escouade" (rendu par `render_squad_timeline` → `compute_squad_timeseries` → `plot_squad_performance_timeline`) n'affiche pas la dernière session. Les sessions précédentes sont visibles. Le problème se résout automatiquement lors du **sync suivant** du coéquipier.

---

**ROOT CAUSE CONFIRMÉE — Chemin A : conflit R/W sur `shared_matches.duckdb` dans le fanout**

`_get_shared_connection()` ([src/data/sync/_engine_connections.py](src/data/sync/_engine_connections.py#L164)) ouvre **toujours** en `read_only=False` :

```python
def _open_shared() -> duckdb.DuckDBPyConnection:
    return duckdb.connect(str(self._shared_db_path), read_only=False)  # ← TOUJOURS R/W
```

Or `batch_compute_performance_scores()` ([src/data/sync/_performance.py](src/data/sync/_performance.py#L158)) ne fait que des **SELECT** sur shared — il écrit exclusivement dans la player DB locale. Il n'a **aucun besoin du R/W**.

**Scénario en conditions réelles (joueurs avec des centaines de matchs)** :
1. On sync un joueur → insère dans shared (R/W) → fanout se déclenche
2. Le fanout crée un nouveau `DuckDBSyncEngine` pour le coéquipier → tente d'ouvrir shared en R/W
3. **Streamlit est actif** (dashboard ouvert pendant le sync — cas standard) → connexion R/O ouverte sur shared
4. DuckDB interdit R/W + R/O simultané → `"unique file handle conflict"`
5. `_get_shared_connection()` retente 1 fois après `gc.collect()` + `time.sleep(0.5)` → échoue aussi → retourne `None`
6. `batch_compute_performance_scores()` : `if shared_conn is None: return 0` → **0 scores calculés, silencieusement** (log WARNING uniquement : `"shared_connection ou xuid manquant"`)
7. Pas de lignes PME créées pour les matchs du coéquipier → `backfill_sessions` ne voit pas ces matchs → dernière session invisible

**Pourquoi ça se résout au sync suivant** : quand le coéquipier sync lui-même (invocation séparée de `sync.py`), c'est un processus distinct qui ouvre shared en R/W sans conflit (Streamlit n'est pas forcément actif à ce moment ou le timing est différent).

**Cascade amplificatrice** : le `_join_perf_frames()` ([src/analysis/_performance_squad.py](src/analysis/_performance_squad.py#L85)) utilise un INNER JOIN — si le coéquipier n'a pas de score, le match entier est supprimé du graphe, même si le joueur principal a un score valide.

---

**Chemins secondaires (aggravants, pas root cause)** :

**Chemin B — `backfill_sessions_for_player` early-exit** : conséquence du chemin A. Si aucune ligne PME n'est créée pour les nouveaux matchs (parce que `batch_compute_performance_scores()` a échoué silencieusement), `_load_matches_from_shared` ne les voit pas → sessions non recalculées.

**Chemin C — Seuil `MIN_MATCHES_FOR_RELATIVE = 10`** : non pertinent ici car le bug est observé sur des joueurs avec des centaines de matchs. Néanmoins, pour un coéquipier avec < 10 matchs dans shared, `batch_compute_performance_scores()` ferait `continue` sans créer de ligne PME → même effet que chemin A.

---

**Fix root cause — Ouvrir shared en R/O dans le fanout**

Le `DuckDBSyncEngine` n'a actuellement pas de paramètre pour contrôler le mode de connexion shared. Le fix :

1. **Ajouter `shared_read_only: bool = False`** au constructeur de `DuckDBSyncEngine` et le propager à `_get_shared_connection()`
2. **Dans `_run_other_player_enrichment()`** ([src/data/sync/_engine_fanout.py](src/data/sync/_engine_fanout.py#L163)), passer `shared_read_only=True` :
   ```python
   engine = DuckDBSyncEngine(
       player_db_path=player_db_path,
       xuid=xuid, gamertag=gamertag,
       shared_db_path=shared_path,
       shared_read_only=True,  # ← le fanout ne fait que LIRE shared
   )
   ```
3. **Dans `_get_shared_connection()`** ([src/data/sync/_engine_connections.py](src/data/sync/_engine_connections.py#L164)), utiliser `self._shared_read_only` :
   ```python
   def _open_shared() -> duckdb.DuckDBPyConnection:
       return duckdb.connect(str(self._shared_db_path), read_only=self._shared_read_only)
   ```

**Fix défensif complémentaire — LEFT JOIN dans `_join_perf_frames()`**

Indépendamment du fix R/O, le graphe devrait tolérer un score manquant chez un coéquipier au lieu de supprimer le match entier :
- Remplacer `how="inner"` par `how="left"` dans le join
- Supprimer le `.filter(pl.col(f"perf_{i}").is_not_null())` avant le join
- `squad_perf = list.mean()` ignore les nulls automatiquement → le score escouade sera la moyenne des joueurs disponibles
- Une session avec score partiel (2 joueurs sur 3) est **mieux** qu'une session invisible

**Fichiers impliqués :**
- [src/data/sync/_engine_connections.py](src/data/sync/_engine_connections.py) — `_get_shared_connection` : paramétrer `read_only`
- [src/data/sync/engine.py](src/data/sync/engine.py) — `__init__` : ajouter `shared_read_only`
- [src/data/sync/_engine_fanout.py](src/data/sync/_engine_fanout.py) — `_run_other_player_enrichment` : passer `shared_read_only=True`
- [src/analysis/_performance_squad.py](src/analysis/_performance_squad.py) — `_join_perf_frames` : INNER → LEFT join

---

### � Bug critique — `mv_player_matches` recalcule le KDA au lieu de lire la valeur API

**Signalé le** : 2026-03-25
**Gravité** : Critique — **impact fort**. L'API Halo peut retourner un KDA négatif (cas betrayals/suicides). La formule locale `(kills + assists/3) / deaths` est structurellement incapable de produire une valeur négative (kills, assists, deaths >= 0). Divergence non cosmétique : **changement de signe** propagé dans tous les graphes et calculs dérivés.
**Audité le** : 2026-03-25 (code vérifié en profondeur)

---

**Symptôme** : Le KDA/ratio affiché pour le joueur principal diverge de la valeur officielle API — peut être positif quand l'API renvoie un négatif, et inversement affecte tous les calculs dérivés (performance_score, timeseries, comparaisons coéquipiers…).

**Cause confirmée — recalcul local dans la vue `mv_player_matches`** :

La vue `mv_player_matches` dans `shared_matches.duckdb` (créée par `ensure_mv_player_matches_view()`, [src/data/sync/migrations.py](src/data/sync/migrations.py#L678)) **ne lit pas** la colonne `match_participants.kda` ; elle recalcule :

```sql
-- migrations.py L678 — recalcul dans mv_player_matches
CASE WHEN p.deaths > 0
THEN (CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0)
     / CAST(p.deaths AS FLOAT)
ELSE CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0
END AS kda
```

Or `match_participants.kda` (FLOAT, schéma v5+) est peuplée via `_extract_kda()` ([src/data/sync/transformers/_helpers.py](src/data/sync/transformers/_helpers.py#L101)) qui lit `stats_dict.get("KDA")` — la valeur officielle Halo.

**Chemin de la régression :**
1. API → `_extract_kda()` → stocké dans `match_participants.kda` ✅
2. `_get_match_source()` ([src/data/repositories/_match_queries.py](src/data/repositories/_match_queries.py#L59)) → lit `mv_player_matches` qui **recalcule** au lieu de lire `p.kda` ❌
3. `_finalize_polars_df()` ([src/data/repositories/_match_queries_polars.py](src/data/repositories/_match_queries_polars.py#L186)) crée `ratio` comme alias de ce `kda` recalculé → propagé partout

**Note contexte :** Le "Fix 4" du 2026-03-19 avait corrigé `ratio` pour les **coéquipiers** (`p.kda AS ratio` dans `_query_teammate_shared_stats`) mais pas pour le joueur principal (chemin `mv_player_matches`).

---

**⚠️ ACTION REQUISE : Audit exhaustif des recalculs KDA dans le codebase**

La vue `mv_player_matches` n'est potentiellement pas le seul endroit où le KDA est recalculé au lieu de lire la colonne API. Il faut auditer **tout le code** pour trouver d'autres occurrences du pattern `(kills + assists/3) / deaths` ou équivalent, y compris :
- Autres vues SQL dans `migrations.py`
- Requêtes dans les repositories (`_match_queries.py`, `_encounter_loader.py`, etc.)
- Calculs Python dans `src/analysis/` ou `src/data/services/`
- Performance score (`_performance.py`) qui utilise `kda` de l'historique shared

**Chaque recalcul doit être remplacé** par la lecture de `p.kda` (avec fallback COALESCE pour les anciens matchs sans valeur).

---

**Fix principal — vue `mv_player_matches`** :

Remplacer la clause recalculée dans `ensure_mv_player_matches_view()` ([src/data/sync/migrations.py](src/data/sync/migrations.py#L678)) :
```sql
COALESCE(p.kda,
    CASE WHEN p.deaths > 0
    THEN (CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0) / CAST(p.deaths AS FLOAT)
    ELSE CAST(p.kills AS FLOAT) + CAST(p.assists AS FLOAT) / 3.0
    END
) AS kda
```
La vue est recréée avec `CREATE OR REPLACE VIEW` (idempotent — aucune migration de données nécessaire).

**Fichiers impliqués (minimum — compléter après audit)** :
- [src/data/sync/migrations.py](src/data/sync/migrations.py) — `ensure_mv_player_matches_view` (~L678) : clause `kda` à corriger
- [src/data/repositories/_match_queries.py](src/data/repositories/_match_queries.py) — `_get_match_source` (~L59) : consomme la vue
- [src/data/repositories/_match_queries_polars.py](src/data/repositories/_match_queries_polars.py) — `_finalize_polars_df` (~L186) : propage `ratio`
- **+ tous les fichiers identifiés par l'audit**

---
### � UX — Score d'équipe supérieur à tous les scores individuels (En-tête Page Coéquipiers)

**Signalé le** : 2026-03-25
**Nature** : Pas un bug de calcul — **transparence manquante dans l'UI**.
**Audité le** : 2026-03-25 — comportement voulu (bonus collectifs), UX à clarifier.

---

**Symptôme** : Sur la session escouade du 12/02, la moyenne de performance d'équipe (carte "Score d'équipe", section tout en haut de la page Coéquipiers) est **supérieure au score des 3 joueurs individuels**. Visuellement trompeur.

**Diagnostic confirmé** : le score d'équipe inclut des **bonus collectifs** cumulables jusqu'à +13 points (`_compute_squad_bonuses()`, [src/analysis/_performance_squad.py](src/analysis/_performance_squad.py#L51)) :
- **+5** si win rate moyen d'équipe > 60 %
- **+5** si min(K/D) > 1.0
- **+3** si écart-type des kills < 3.0

C'est le **design voulu** — le score collectif récompense la cohésion au-delà des performances individuelles. Le problème est que la carte équipe n'affiche que le chiffre final sans mention des bonus.

**Décision : Option A — Afficher le détail du bonus sur la carte équipe** *(fix minimal, pas de logique changée)*
- Sur la carte "Score d'équipe", sous le score, ajouter une ligne : `moy. XX (+YY collectif)`
- Données déjà disponibles dans `squad_result["components"]["base_avg"]`
- `render_squad_session_header()` passe `{**squad_result, "score": avg_score}` à `_render_compact_team_card()` : `components` y est accessible

**Fichiers impliqués :**
- [src/ui/components/performance.py](src/ui/components/performance.py) — `_render_compact_team_card()`, `render_squad_session_header()`
- [src/ui/i18n/pages/teammates.py](src/ui/i18n/pages/teammates.py) — ajouter clés i18n pour le détail bonus

---

### 🐛 Bug — Colonne "Dernière rencontre" incohérente (Page Match · Encounters)

**Signalé le** : 2026-03-25
**Audité le** : 2026-03-25 — root cause identifiée + 2 bugs annexes confirmés.

---

**Symptôme** : Dans le tableau des encounters (page vue d'un match), la colonne "Dernière rencontre" affiche parfois "à venir" et ne montre pas la bonne information.

---

**ROOT CAUSE — La requête SQL ne filtre pas par rapport au match courant**

La colonne "Dernière rencontre" est calculée dans `_build_encounter_sql()` ([src/data/repositories/_encounter_loader.py](src/data/repositories/_encounter_loader.py#L51)) comme :

```sql
MAX(r.start_time) AS last_seen
```

…sur **tous les matchs** entre le joueur et l'adversaire, **y compris le match courant et les matchs postérieurs**. Conséquences :
1. Si le match qu'on regarde **est** le plus récent : `last_seen` = date du match courant → peu utile
2. Si on regarde un ancien match et qu'une rencontre plus récente existe : `last_seen` pointe vers le futur par rapport au match affiché → incohérent
3. Si le timestamp Halo est marginalement dans le futur (horloge serveur) : `timedelta.days == -1` → "à venir"

**Fix root cause — Filtrer les rencontres antérieures au match courant** :

1. **SQL** : dans `_build_encounter_sql()`, ajouter `AND r.start_time < ?` (timestamp du match courant) et exclure le `match_id` courant. Passer ces paramètres depuis le caller.

2. **`_relative_date()`** : ajouter un paramètre optionnel `reference_dt` (date du match affiché). Calculer le delta par rapport au match, pas par rapport à "maintenant". Si `reference_dt` est fourni, les messages deviennent : "même jour", "1 jour avant", "3 jours avant", etc.

3. **Guard défensif** : `if days < 0: days = 0` pour les cas marginaux d'horloge serveur.

4. **Renommer la colonne** : "Dernière rencontre" → **"Précédente rencontre"** pour clarifier qu'il s'agit de la rencontre la plus récente *avant* ce match.

5. **Cas "première rencontre"** : quand aucune rencontre antérieure n'existe (SQL retourne NULL pour `last_seen`), afficher **"1ère rencontre"** au lieu de "—".

**Clés i18n à ajouter/modifier** :
- `encounters_col_last_seen` → renommer en `encounters_col_prev_encounter` : "Précédente rencontre"
- `encounters_first_encounter` : "1ère rencontre"
- `rel_date_same_day` : "Même jour"
- `rel_date_days_before` : "{{days}} j. avant"
- Supprimer `rel_date_upcoming` (plus jamais atteint)

---

**Bug annexe #1 — `get_sync_metadata` lit une colonne inexistante (sync indicator toujours NULL)**

- **Localisation** : [src/data/repositories/_diagnostic_repo.py](src/data/repositories/_diagnostic_repo.py#L18) — `get_sync_metadata()`
- La requête actuelle : `SELECT last_sync_at FROM meta.sync_meta WHERE xuid = ?` — **incorrecte** : `sync_meta` est une table key-value `(key, value, updated_at)` dans la DB joueur (pas dans `meta`). L'Exception est silencieusement swallowée → `last_sync` vaut toujours `None`.
- **Fix** : Lire via `SELECT value FROM sync_meta WHERE key = 'last_sync_at'` dans la connexion courante (DB joueur), puis parser la chaîne ISO.
- **Fichier** : [src/data/repositories/_diagnostic_repo.py](src/data/repositories/_diagnostic_repo.py)

---

**Bug annexe #2 — `datetime.utcnow()` déprécié dans `career_lusr.py`**

- **Localisation** : [src/ui/pages/career_lusr.py](src/ui/pages/career_lusr.py#L136)
- `datetime.utcnow()` est déprécié depuis Python 3.12 (émet un `DeprecationWarning`, pas encore supprimé mais à corriger par hygiène).
- **Fix** : Remplacer par `datetime.now(timezone.utc).replace(tzinfo=None)` pour conserver un datetime naïf UTC.
- **Fichier** : [src/ui/pages/career_lusr.py](src/ui/pages/career_lusr.py)

---

**Fichiers impliqués :**
- [src/data/repositories/_encounter_loader.py](src/data/repositories/_encounter_loader.py) — `_build_encounter_sql` : filtrer `start_time < match_start_time`
- [src/ui/pages/match_view_encounters_logic.py](src/ui/pages/match_view_encounters_logic.py) — `_relative_date` : paramètre `reference_dt`, guard `days < 0`
- [src/ui/pages/match_view_encounters.py](src/ui/pages/match_view_encounters.py) — passer `match_start_time` au SQL + renommer colonne
- [src/data/repositories/_diagnostic_repo.py](src/data/repositories/_diagnostic_repo.py) — `get_sync_metadata` (bug annexe #1)
- [src/ui/pages/career_lusr.py](src/ui/pages/career_lusr.py) — `datetime.utcnow()` (bug annexe #2)
- i18n : clés `encounters_col_prev_encounter`, `encounters_first_encounter`, `rel_date_same_day`, `rel_date_days_before`

---
### 🐛 Bug — Médias mal rattachés aux matchs (décalage fuseau horaire)

**Signalé le** : 2026-03-25
**Symptôme** : Sur la page Médias, les captures/vidéos sont associées au mauvais match. Le décalage observé est de **−1h** (heure "API" = heure locale − 1h à Paris, UTC+1). Les fichiers sont rattachés au match qui précède le bon.
**Audité le** : 2026-03-25 — root cause identifiée (incohérence `epoch()` SQL vs `match_start_to_epoch()` Python).

---

**ROOT CAUSE — Incohérence entre le calcul epoch côté média (SQL) et côté match (Python)**

Le matching média↔match compare deux epochs calculés différemment :

**Côté médias** (SQL, [src/data/media_indexer.py](src/data/media_indexer.py#L371)) :
```sql
COALESCE(epoch(mf.capture_end_utc), mf.mtime_paris_epoch, mf.mtime)
```
`capture_end_utc` est un `TIMESTAMP` **naïf** dans DuckDB. Or DuckDB `epoch()` sur un TIMESTAMP naïf **l'interprète dans la timezone locale du système**. À Paris (UTC+1), un `14:00:00` UTC naïf → `epoch()` croit que c'est 14h CET → retourne l'epoch de `13:00:00 UTC` → **−1h**.

**Côté matchs** (Python, [src/data/media_helpers.py](src/data/media_helpers.py#L59) — `match_start_to_epoch()`) :
```python
if dt.tzinfo is None:
    dt = dt.replace(tzinfo=timezone.utc)  # ← assume explicitement UTC
return dt.timestamp()
```
Correct — les timestamps naïfs sont **explicitement traités comme UTC**.

**Résultat** : pour un même instant UTC, l'epoch du média est en avance de −1h (hiver) ou −2h (été) sur l'epoch du match → le matching temporel associe le média au **mauvais match** (celui qui précède).

**Note** : `mtime_paris_epoch` (le fallback) est stocké comme `meta["mtime"]` = `stat.st_mtime` = epoch UTC standard. Le nom `mtime_paris_epoch` est un **misnomer** — cette colonne contient un epoch UTC correct. Le bug ne touche que le chemin `epoch(capture_end_utc)`.

---

**Bug secondaire — EXIF datetime naïf stocké sans conversion UTC**

Pour les images avec EXIF ([src/data/media_helpers.py](src/data/media_helpers.py#L243)), si `exif_dt.tzinfo is None` (cas standard — EXIF ne transporte jamais de timezone), l'heure locale de l'appareil est stockée telle quelle dans `capture_end_utc` sans conversion UTC. Combiné au bug principal, les JPEG peuvent être décalés de 2h.

---

**Fix principal** : Remplacer dans `associate_with_matches()` ([src/data/media_indexer.py](src/data/media_indexer.py#L371)) :
```sql
-- Avant : interprète capture_end_utc en TZ locale
COALESCE(epoch(mf.capture_end_utc), mf.mtime_paris_epoch, mf.mtime)
-- Après : force l'interprétation UTC
COALESCE(epoch(timezone('UTC', mf.capture_end_utc)), mf.mtime_paris_epoch, mf.mtime)
```

**Fix secondaire EXIF** : Dans `get_file_metadata()` ([src/data/media_helpers.py](src/data/media_helpers.py#L243)), ignorer l'EXIF naïf et utiliser le fallback `mtime` (epoch UTC fiable) :
```python
# Si EXIF n'a pas de timezone → ne pas utiliser (c'est l'heure locale caméra, pas UTC)
if exif_dt.tzinfo is not None:
    exif_dt = exif_dt.astimezone(timezone.utc).replace(tzinfo=None)
    capture_end_utc = exif_dt
    capture_start_utc = exif_dt
# sinon : fallback sur mtime (déjà UTC)
```

**Après fix** : ré-indexer avec `python scripts/index_media.py --gamertag GAMERTAG --force` pour recalculer les associations.

**Fichiers impliqués :**
- [src/data/media_indexer.py](src/data/media_indexer.py) — `associate_with_matches` : requête SQL `epoch(...)`
- [src/data/media_helpers.py](src/data/media_helpers.py) — `get_file_metadata` : branche EXIF naïf
- [src/data/media_indexer_matchers.py](src/data/media_indexer_matchers.py) — `_associate_single_media` : logique de fenêtre temporelle (pas à modifier)

---

### 🐛 Bug — Stats coéquipiers absentes sur la session la plus récente (Page Teammates)

**Signalé le** : 2026-03-25
**Audité le** : 2026-03-25 — **probablement lié au bug "Session escouade absente" (conflit R/W fanout)**. À revalider après le fix fanout R/O.

---

**Symptôme** : Après une session de jeu, les stats des amis/coéquipiers n'apparaissent pas dans la vue Teammates (graphe heatmap, comparaison, armes…). Les sessions précédentes fonctionnent normalement.

**Lien avec le bug fanout** : si le fanout échoue silencieusement à cause du conflit R/W sur shared (voir bug "Session escouade absente" ci-dessus), les lignes PME du coéquipier ne sont pas créées → sessions non backfillées → session la plus récente invisible partout dans la page Teammates.

**À revalider après le fix fanout R/O.** Si le problème persiste, les pistes ci-dessous restent pertinentes :

**Piste #1 — `is_with_friends = NULL` dans `player_match_enrichment`**
- Lors du sync, `_insert_enrichment_row` (`src/data/sync/_engine_writes.py`) tente de résoudre les amis via `get_friends_xuids_for_backfill()`. Si la résolution échoue, `is_with_friends` est écrit `NULL` → session classifiée "solo" dans `_classify_sessions_solo_squad`.

**Piste #2 — Race condition cache / WAL DuckDB**
- `cached_query_matches_with_friend` n'a pas de TTL. Si le WAL n'est pas checkpointé au moment de la visite → `shared_ids` incomplet → matchs récents invisibles.

**Fichiers impliqués :**
- `src/data/sync/_engine_writes.py` — `_insert_enrichment_row` / `_load_friends_lazy`
- `src/data/sessions_backfill.py` — `get_friends_xuids_for_backfill`
- `src/app/_filters_session.py` — `_classify_sessions_solo_squad`
- `src/ui/_cache_queries.py` — `cached_query_matches_with_friend`