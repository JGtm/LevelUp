— Tâches et TODO centralisés

> Mis à jour le 2026-03-19.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
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

### Bug — Frags vs. détail armes incohérent (sentinels + `reconciled_as`)

**Symptôme** : Pour un joueur (ex. Chocoboflor), le total de frags affiché dans le scoreboard est inférieur à la somme du détail armes (3 needler + 2 melee + 6 sidekick = 11 > frags). Certains kills semblent comptés deux fois.

---

#### Phase 0 — Investigation & validation du diagnostic (avant tout fix)

> Objectif : confirmer ou infirmer chaque hypothèse du diagnostic ci-dessous grâce à des requêtes SQL et une lecture de code rigoureuse, avant toute modification.

##### H1 — Vérifier que des sentinels `weapon_id ∈ {0,1,2}` portent un `reconciled_as` non-null en base

```sql
-- Sur shared_matches.duckdb
SELECT weapon_id, reconciled_as, attribution_path, confidence, COUNT(*) AS n
FROM weapon_kills
WHERE weapon_id IN (0, 1, 2)
  AND reconciled_as IS NOT NULL
GROUP BY ALL
ORDER BY weapon_id, n DESC
LIMIT 50;
```

**Attendu si vrai** : au moins une ligne retournée. Ces lignes sont les candidats au double-comptage.

**Si zéro ligne** : le diagnostic est infirmé — la vue `v_weapon_kills` ne peut pas produire de faux positifs par ce chemin. Investiguer alors la logique de `_enrich_with_grenade_melee` indépendamment (H3).

---

##### H2 — Vérifier que le filtre `EXCLUDED_WEAPON_IDS` opère sur l'`effective_weapon_id` post-COALESCE (pas sur `weapon_id` brut)

Lecture de code à confirmer dans `src/ui/pages/match_view_weapon_kills.py` :

```python
# _build_weapon_kills_df() — le DF reçu de load_weapon_kills_for_player()
# a déjà weapon_id = effective_weapon_id (alias SQL). Vérifier l'alias exact :
```

```sql
-- _weapon_kills_repo.py, load_weapon_kills_for_player :
SELECT match_id,
       effective_weapon_id AS weapon_id,   -- ← l'alias écrase weapon_id brut
       COUNT(*)::INTEGER AS kills
FROM shared.v_weapon_kills
WHERE xuid = ? AND match_id IN (...)
  AND effective_weapon_id IS NOT NULL
GROUP BY match_id, effective_weapon_id
```

**Conclusion** : le filtre Polars `~pl.col("weapon_id").is_in(EXCLUDED_WEAPON_IDS)` compare `effective_weapon_id` (post-COALESCE) à `{0, 1, 2}`. Un sentinel `weapon_id=1` avec `reconciled_as=sidekick_id` aura `effective_weapon_id=sidekick_id` → **il passe le filtre**. H2 **confirmée** par lecture de code seule.

---

##### H3 — Vérifier que `_enrich_with_grenade_melee` ajoute bien les valeurs API indépendamment du film

```python
# _weapon_kills_repo.py, load_grenade_melee_kills :
SELECT COALESCE(SUM(grenade_kills), 0),
       COALESCE(SUM(melee_kills), 0)
FROM shared.match_participants
WHERE xuid = ? AND match_id IN (...)
```

Source : colonnes API `match_participants.melee_kills` / `grenade_kills` — **indépendantes** du film.

**Requête de quantification** : pour un match donné et un joueur suspect, comparer `melee_kills` API vs. kills film avec `weapon_id=1` ayant `reconciled_as IS NOT NULL` :

```sql
-- Valeur API
SELECT melee_kills, grenade_kills
FROM shared.match_participants
WHERE match_id = '<MATCH_ID>' AND xuid = '<XUID>';

-- Kills film sentinels melee avec reconciled_as
SELECT weapon_id, reconciled_as, COUNT(*) AS n_kills_film
FROM weapon_kills
WHERE match_id = '<MATCH_ID>' AND xuid = '<XUID>'
  AND weapon_id IN (0, 1, 2)
  AND reconciled_as IS NOT NULL
GROUP BY weapon_id, reconciled_as;
```

**Attendu si double-comptage** : `n_kills_film` (via `reconciled_as`) apparu dans le détail armes **ET** `melee_kills`/`grenade_kills` API > 0 pour les mêmes kills.

---

##### H4 — Vérifier l'asymétrie du sentinel `3` (absent de `EXCLUDED_WEAPON_IDS`)

```sql
-- Lignes weapon_id=3 dans v_weapon_kills
SELECT COUNT(*) AS n, COUNT(DISTINCT match_id) AS n_matchs
FROM shared.v_weapon_kills
WHERE weapon_id = 3 OR effective_weapon_id = 3;
```

`EXCLUDED_WEAPON_IDS = frozenset({0, 1, 2})` — l'ID 3 (4e catégorie traitée par le script de fix CAT 2) **n'est pas exclu** dans l'UI. Si des lignes `weapon_id=3` existent, elles alimentent le détail armes sans correspondre à une arme réelle identifiable.

---

##### H5 — Vérifier que `_fix_weapon_kills_sentinel.py` cible bien la DB de production

```python
# Chercher DB_PATH dans le script
DB_PATH = "data/warehouse/shared_matches_v2.duckdb"  # ← ancienne convention ?
```

Si le chemin est `shared_matches_v2.duckdb` et non `shared_matches.duckdb`, **le script ne peut pas s'exécuter sur la DB active** sans modification manuelle de ce paramètre.

---

##### Tableau de synthèse attendu après investigation

| Hypothèse | Méthode | Statut attendu |
|-----------|---------|----------------|
| H1 — Sentinels `{0,1,2}` avec `reconciled_as` non-null présents | Requête SQL H1 | À vérifier |
| H2 — Filtre `EXCLUDED_WEAPON_IDS` opère sur post-COALESCE | Lecture code | **Confirmée** (code verbatim) |
| H3 — `_enrich_with_grenade_melee` source indépendante (API) | Lecture code + SQL H3 | **Confirmée** (code verbatim) |
| H4 — Sentinel `3` absent de `EXCLUDED_WEAPON_IDS` | Requête SQL H4 | À vérifier |
| H5 — Script fix cible mauvaise DB | Lecture code | **Confirmée** (chemin `_v2`) |

> **Décision go/no-go** : si H1 retourne au moins une ligne, le double-comptage est avéré et les fixs décrits ci-dessous sont valides. Si H1 est vide, rouvrir l'investigation sur la source du delta (ex. kills API vs. kills film sans sentinel, ou bug de déduplication dans le GROUP BY).

---

**Cause identifiée** : Double-comptage via sentinels corrompus.

1. `v_weapon_kills` expose `COALESCE(reconciled_as, weapon_id) AS effective_weapon_id`.
2. Des lignes avec `weapon_id IN (0, 1, 2)` (melee/grenade/vehicle) ont un `reconciled_as` non-null pointant vers une arme réelle (ex. sidekick). Ces lignes passent le filtre `EXCLUDED_WEAPON_IDS` dans `_build_weapon_kills_df()` et sont comptées comme kills d'arme.
3. Dans le même temps, `match_participants.melee_kills` / `grenade_kills` compte ces mêmes kills → `_enrich_with_grenade_melee()` les rajoute une seconde fois.

**Fichiers concernés** :
- `src/ui/pages/match_view_weapon_kills.py` → `_build_weapon_kills_df()` : filtrer en plus `weapon_id NOT IN (0,1,2)` (ou `attribution_path != 'none'`) avant export.
- `src/data/repositories/_weapon_kills_repo.py` → `load_weapon_kills_for_player()` : ajouter clause `AND attribution_path != 'none'` aux requêtes (ou filtrer sur `weapon_id > 3`).
- Script de nettoyage existant : `scripts/_fix_weapon_kills_sentinel.py` (CAT 2) peut normaliser les lignes corrompues déjà en base.

**Fix minimal recommandé** : Dans `load_weapon_kills_for_player()` (et `load_weapon_kills_for_match()`), exclure les lignes dont `weapon_id` est un sentinel même si `reconciled_as` est non-null — soit via `AND (weapon_id IS NULL OR weapon_id > 3)`, soit via `AND attribution_path != 'none'`.

---

### Perf — Film chunks : augmenter `_MAX_CONCURRENT_CHUNKS`

**Fichier** : `src/data/services/weapon_extraction_service.py`

Passer de 5 à 7 (puis 10 si stable) connexions concurrentes au CDN Azure. Objectif : ~14s → ~8s par match.

⚠️ Non confirmé sans mesure : vérifier d'abord que les 429 sont gérés avec retry exponentiel avant d'augmenter. Tester sur 5+ matchs à 7 concurrent, mesurer taux d'erreur, puis décider.

---

### UI — Notation de session escouade (en-tête Page Coéquipiers)

**Objectif** : Remplacer les métriques "Tendance K/D" (bloc `st.metric` par joueur, lignes ~134–173 de `teammates.py`) par un en-tête de session d'équipe plus riche et soigné. Pas d'emojis.

**Périmètre** : vues 1 coéquipier (`render_single_teammate_view`) et multi (`render_multi_teammate_view`). Affiché uniquement quand ≥ 4 matchs communs.

#### A — Scores individuels par joueur (côte à côte)

Réutiliser **`compute_session_performance_score_v2_ui`** + **`render_performance_score_card`** sur les matchs communs filtrés (`sub` pour le joueur principal, `_friend_df` pour chaque coéquipier). Une carte par joueur en `st.columns`. Badge ▲/▼ si un joueur se démarque de la moyenne d'équipe.

```python
perf_me = compute_session_performance_score_v2_ui(sub)
perf_f1 = compute_session_performance_score_v2_ui(friend_sub)
```

#### B — Score d'équipe agrégé

Sous les cartes individuelles, un score collectif unique :

```
Score équipe = moyenne(scores individuels)
             + bonus_winrate   (+5 si win_rate_équipe > 60 %)
             + bonus_cohesion  (+5 si min(K/D individuel) > 1.0)
             + bonus_équilibre (+3 si std(kills par joueur) < seuil)
```

Score plafonné à 100. Afficher via `render_performance_score_card(label="Équipe", ...)`. Créer `compute_squad_performance_score(scores: list[dict]) -> dict` dans `src/analysis/performance_score.py`.

#### D — Grade de carnage (ludique, sans emojis)

Au-dessus ou à côté du score d'équipe, afficher un grade textuel en majuscules :

| Score | Grade |
|------:|-------|
| ≥ 88 | LÉGENDAIRE |
| ≥ 75 | CARNAGE |
| ≥ 60 | SOLIDE |
| ≥ 45 | MOYEN |
| < 45 | DIFFICILE |

Ajouter `SQUAD_GRADE_THRESHOLDS` dans `src/analysis/performance_config.py`. Clés i18n `squad_grade_*` dans `src/ui/translations.py`. Style : `font-size: 1.6rem`, majuscules, couleur selon grade (même palette que `get_score_color`).

#### Fichiers impactés

| Fichier | Modification |
|---------|-------------|
| `src/ui/pages/teammates.py` | Supprimer bloc tendance K/D ; ajouter appel `render_squad_session_header()` |
| `src/ui/pages/teammates_views.py` | Idem si appelé depuis les vues single/multi |
| `src/analysis/performance_score.py` | Ajouter `compute_squad_performance_score()` |
| `src/analysis/performance_config.py` | Ajouter `SQUAD_GRADE_THRESHOLDS` |
| `src/ui/components/performance.py` | Ajouter `render_squad_session_header()` |
| `src/ui/translations.py` | Clés `squad_grade_*` |

---

### CI — Échecs permanents : scripts exclus par `.gitignore`

**Diagnostic (2026-03-20)** : Trois fichiers référencés dans `.github/workflows/ci.yml` et dans les tests ne sont jamais poussés sur GitHub car couverts par la règle `check_*.py` / `diagnose_*.py` du `.gitignore`.

| Fichier manquant | Référencé dans | Impact |
|------------------|----------------|--------|
| `scripts/check_code_size.py` | Job `quality` (ci.yml L118) + `test_code_quality.py::test_no_new_size_violations` | Jobs `quality` **et** `test` en rouge |
| `scripts/check_imports.py` | Job `quality` (ci.yml L121, `\|\| true`) | Erreur silencieuse uniquement |
| `tests/test_page_router_smoke.py` | Job `streamlit-smoke` (ci.yml L79) | Job `streamlit-smoke` en rouge |
| `tests/test_page_router_regressions.py` | Job `streamlit-smoke` (ci.yml L79) | Job `streamlit-smoke` en rouge |

> Note : `test_page_router_smoke.py` et `test_page_router_regressions.py` ne sont **pas** couverts par le `.gitignore` — ces fichiers n'ont simplement jamais été créés ou ont été supprimés.

#### Option A — Renommer les scripts (recommandée)

Renommer pour sortir du pattern `check_*.py` :

```
scripts/check_code_size.py  →  scripts/enforce_size_limits.py
scripts/check_imports.py    →  scripts/validate_imports.py
```

Puis mettre à jour les références dans :
- `.github/workflows/ci.yml` (lignes `quality` job)
- `tests/test_code_quality.py` (`subprocess.run([..., "scripts/check_code_size.py"])`)

#### Option B — Ajouter des exceptions dans `.gitignore`

Dans `.gitignore`, sous la règle `check_*.py` :

```gitignore
# Scripts de diagnostic temporaires
check_*.py
diagnose_*.py
# Exceptions CI permanentes
!scripts/check_code_size.py
!scripts/check_imports.py
```

#### Pour les fichiers de tests manquants

Créer `tests/test_page_router_smoke.py` et `tests/test_page_router_regressions.py` (stubs minimes suffisent), **ou** retirer ces deux fichiers de la commande pytest dans le job `streamlit-smoke` si la feature page-router n'est pas encore implémentée.
