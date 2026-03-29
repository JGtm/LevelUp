— Tâches et TODO centralisés

> Mis à jour le 2026-03-28.

---

## ✅ Récemment complété (référence)

| Date | Item |
|------|------|
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


### 🟠 Normalisation des labels de modes de jeu — suppression des redondances (v6.2.1)

**Noté le** : 2026-03-28
**Priorité** : Moyenne

#### Problème

Le champ `game_variant_name` (format API Halo : `{prefix}:{mode}`) contient des redondances visibles dans toute l'UI :
- La **playlist** implique déjà le contexte → `BTB:Slayer` dans "Big Team Battle" = "Slayer" suffit
- La **`mode_category`** implique déjà le préfixe → `Arena:Slayer` dans Assassin = "Slayer" suffit
- Le format est parfois **inversé** (`CTF:Arena` vs `Arena:CTF`) selon les variants
- Quelques variants n'ont **pas de séparateur** (`CASTLE WARS`, `TFF | Survive The Undead`)

#### Mapping des redondances connues

| Préfixe (`game_variant_name`) | `mode_category` impliquée | Redondant si… |
|-------------------------------|--------------------------|---------------|
| `Arena` | Assassin | toujours |
| `BTB` | BTB | toujours |
| `Ranked` | Ranked | toujours |
| `Fiesta` | Fiesta | quand le mode est Slayer basique |
| `Firefight` | Firefight | toujours |
| `BTB Heavies` | BTB | **non** — "Heavies" est un qualificatif significatif à conserver |
| `Tactical` | Assassin | oui (sous-variant Arena) |
| `Community` / `Event` | Other | oui |

#### Architecture cible

**Principe** : normaliser à l'affichage, jamais au stockage. Le `game_variant_name` brut reste intact en DB.

**Couche de résolution** : fonction Python pure dans `src/analysis/` (0 accès DB, 0 Streamlit).

```
resolve_display_mode(
    game_variant_name: str,
    mode_category: str,
    lang: str,
    overrides: dict[str, str],          # depuis mode_pair_overrides (metadata)
    prefix_categories: dict[str, str],  # depuis mode_prefix_names étendu
    mode_translations: dict[str, str],  # depuis mode_name_tr (metadata)
) -> str
```

**Algorithme de résolution (priorité décroissante)** :
1. Lookup exact dans `mode_pair_overrides` → retourner le label override si trouvé
2. Si pas de `:` → retourner `game_variant_name` tel quel (variants sans séparateur)
3. Split sur `:` → `(left, right)`
4. Détecter le format inversé : si `right` est un préfixe connu (dans `mode_prefix_names`) et `left` ne l'est pas → `prefix=right`, `mode_name=left`
5. Si `canonical_category(prefix)` == `mode_category` du match → afficher seulement `mode_name` traduit
6. Sinon → afficher `label(prefix) + sep + label(mode_name)` traduit

#### Extension `mode_prefix_names` requise

Ajouter une colonne `canonical_category` (ou `implied_category`) mappant chaque préfixe vers sa `mode_category` :

| prefix | canonical_category |
|--------|--------------------|
| Arena | Assassin |
| BTB | BTB |
| BTB Heavies | BTB |
| Ranked | Ranked |
| Fiesta | Fiesta |
| Firefight | Firefight |
| Gruntpocalypse | Firefight |
| Tactical | Assassin |
| Community | Other |
| Event | Other |
| Husky Raid | Fiesta |
| Super Husky Raid | Fiesta |
| Super Fiesta | Fiesta |
| Assault | Assassin |

#### Validation humaine obligatoire

Avant de brancher la fonction dans l'UI, générer un **fichier plat de contrôle** (CSV ou tableau console) listant :

```
game_variant_name | mode_category | playlist_name | nb_matchs | → label_résolu
```

Le fichier doit être **relu et validé par l'utilisateur** avant toute intégration UI. Des corrections peuvent être apportées via des entrées supplémentaires dans `mode_pair_overrides`.

#### Implémentation

1. Migration `metadata.duckdb` : ajouter colonne `canonical_category` à `mode_prefix_names`
2. Écrire `resolve_display_mode()` dans `src/analysis/mode_display.py`
3. Script de génération du fichier plat de contrôle (CLI, sans toucher l'UI)
4. **Validation utilisateur** du fichier plat
5. Intégrer `resolve_display_mode()` dans les points d'affichage UI (filtres, top matches, profil, page match…)
6. Tests unitaires couvrant : format standard, format inversé, override, sans séparateur, qualificatif Heavies

---

### 🔴 Audit — Calculs KDA locaux dans `src/analysis/` à valider vs valeurs API  (v6.2.1)

**Noté le** : 2026-03-27
**Priorité** : Moyenne

**Contexte** : Suite au fix KDA (2026-03-27), les affichages per-match utilisent désormais exclusivement `p.kda` de l'API. Cependant, plusieurs modules dans `src/analysis/` calculent encore un KDA local à partir des totaux K/D/A pour des métriques agrégées (session, cumul, performance relative) :

- `src/analysis/cumulative.py:72` — `(kills + assists) / max(1, deaths)`
- `src/analysis/stats.py:102,180` — formules session
- `src/analysis/_performance_relative.py:75,77` — KDA relatif
- `src/analysis/_performance_relative_helpers.py:271` — KDA dérivé
- `src/analysis/_performance_session.py:263,362` — KDA session
- `src/data/domain/models/stats.py:54,103` — propriété calculée sur `MatchRow`

**Décision actée (2026-03-28)** : Séparer explicitement les deux sémantiques.

1. **Match / distribution / comparaison match-level** : utiliser exclusivement `p.kda` de l'API, tel quel, même si la valeur est négative.
2. **Session / période / carte / cumul agrégé** : utiliser un indicateur distinct nommé **`efficiency`** (code) / **`efficacité`** (UI FR) / **`efficiency`** (UI EN), dérivé des totaux, avec la formule `sum(K + A/3) / sum(D)`.

**Justification** : le champ API `kda` ne doit plus être traité implicitement comme un simple ratio mathématique agrégable. S'il peut être négatif, alors la moyenne des `kda` match par match décrit la moyenne d'une métrique API signée, pas un rendement global de session. Pour les agrégats lisibles par l'utilisateur, il faut donc conserver un indicateur séparé et explicitement nommé.

**⛔ Nommage obligatoire** : le terme `efficiency` / `efficacité` est **le seul terme autorisé** pour désigner cet agrégat. Les termes `ratio`, `FDA`, `KDA` ou `performance` sont **interdits** pour cette métrique afin d'éviter toute confusion avec la métrique API (`kda`) et le score de performance existant. Toute variable ou clé i18n doit utiliser `efficiency` (ex. `session_efficiency`, `combat_efficiency`).

**Consigne d'implémentation** :
- Conserver `kda` comme métrique API brute dans tous les flux per-match et percentiles relatifs.
- Renommer tous les agrégats dérivés des totaux en `efficiency` / `session_efficiency` (code) et `efficacité` / `efficacité de session` (UI FR).
- Ajouter les clés i18n `efficiency` EN et `efficacité` FR dans `src/ui/i18n/`.
- Audit UI/i18n à prévoir pour éviter qu'une moyenne de `kda` API soit affichée comme une efficacité de session.

---

### 🟡 Amélioration v7++ — Backfill multi-flags : vectoriser le calcul per-match des performance scores (v7+)

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


---

## 🔮 Roadmap v6.3

---

### [v6.3] Score de forme — indice de progression court terme

**Noté le** : 2026-03-28
**Priorité** : Moyenne

#### Problème

Le `performance_score` existant est un score relatif à l'historique global du joueur. Il dit "tu as bien joué ce match", mais pas "tu joues mieux qu'il y a 2 semaines". Aucun indicateur de progression court terme n'existe.

#### Concept

Un `form_score` calculé après chaque sync, représentant la **forme récente** comparée à la baseline long terme :

```
form_score = moy_perf_score(14 derniers matchs) - moy_perf_score(90 derniers matchs)
```

- > 0 : en progression ("en forme")
- < 0 : en régression ("creux de forme")
- Normalisé en percentile pour l'affichage (ex. "Top 20% de tes sessions")

Variante : calculer séparément par `mode_category` (Arena, BTB, Ranked) car la forme peut diverger selon le mode.

#### Données disponibles

- `player_match_enrichment.performance_score` — déjà calculé pour chaque match
- `sessions` — groupement temporel existant
- La fenêtre 14j / 90j est configurable via `app_settings.json`

#### Implémentation

1. Nouvelle fonction `compute_form_score(gamertag, anchor_date)` dans `src/analysis/performance_score.py`
2. Colonne `form_score FLOAT` dans la table `sessions` (migration DuckDB)
3. Calculé et stocké à chaque fin de session (appelé depuis le post-sync)
4. Affiché : bloc "Forme actuelle" sur la page d'accueil / profil avec indicateur ↑↓ et sparkline 30j
5. Tests unitaires : fenêtre vide, fenêtre partielle, forme positive/négative

**Complexité estimée** : Faible — calcul purement SQL/Polars sur données existantes

---

### [v6.3] Détection de changement de niveau — tu as progressé ?

**Noté le** : 2026-03-28
**Priorité** : Basse

#### Concept

Détecter algorithmiquement les moments où la performance d'un joueur a **durablement changé** (amélioration ou régression), distinct des variations de forme court terme.

Approche : **moyenne mobile double avec détection de croisement** (simple, no ML dependency) :
- Rolling mean 14j vs rolling mean 90j
- Un croisement ascendant → "pallier de progression"
- Un croisement descendant → "pallier de régression"
- Stocker les breakpoints détectés + direction + date + delta de performance

#### Utilité concrète

- "Tu as franchi un cap le 2026-02-10 (+8 pts perf en moyenne)"
- Page carrière : overlay des breakpoints sur la courbe performance
- Filtres "depuis ma dernière progression" pour les top matches

#### Données disponibles

- `player_match_enrichment.performance_score` + `start_time` par match
- Les rolling means sont calculables en pure Polars (`.rolling_mean(window_size=N)`)

#### Implémentation

1. Fonction `detect_level_breakpoints(df: pl.DataFrame) -> list[Breakpoint]` dans `src/analysis/progression.py` (nouveau module)
2. `Breakpoint` : dataclass `(date, direction: "up"|"down", delta_perf, n_matches_confirmed)`
3. Seuil de confirmation : direction maintenue sur ≥10 matchs consécutifs post-croisement (évite les faux positifs)
4. Table `progression_breakpoints` dans `stats.duckdb` (légère — quelques lignes max)
5. Affichage : overlay "cap franchi" sur les courbes de tendance

**Complexité estimée** : Moyenne — l'algo est simple mais la calibration du seuil de confirmation demande du test empirique sur données réelles

---

### [v6.3] Page Adversaires — Head-to-head, Nemesis, Proie

**Noté le** : 2026-03-28
**Priorité** : Moyenne

#### Concept

Une nouvelle page dédiée aux **adversaires récurrents** : qui tu croises souvent, contre qui tu gagnes/perds, qui te domine et qui tu domines.

#### Données disponibles

Tout est déjà dans `shared_matches_v2.duckdb` :
- `match_participants` : tous les joueurs de chaque match, avec `team_id` → identifier adversaires (team_id ≠ le tien)
- `killer_victim_pairs` : chaque kill → ratio kills/deaths entre deux joueurs spécifiques
- `match_registry` : outcome du match (W/L)
- `xuid_aliases` / `v_gamertag_lookup` : résolution gamertag

#### Métriques cibles

| Métrique | Source | Description |
|----------|--------|-------------|
| `matches_vs` | `match_participants` | Nb de matchs joués contre cet adversaire |
| `win_rate_vs` | `match_registry.outcome` | % de victoires dans ces matchs |
| `kills_on` | `killer_victim_pairs` | Fois où TU as tué CETTE personne |
| `deaths_from` | `killer_victim_pairs` | Fois où CETTE personne t'a tué |
| `nemesis_score` | dérivé | `deaths_from / max(1, kills_on)` pondéré par `matches_vs` |
| `prey_score` | dérivé | `kills_on / max(1, deaths_from)` pondéré par `matches_vs` |

**Nemesis** = adversaire avec le plus haut `nemesis_score` (min. 3 rencontres)
**Proie** = adversaire avec le plus haut `prey_score` (min. 3 rencontres)

#### Architecture cible

1. Nouveau service `src/data/services/rivals_service.py` — `load_rivals_stats(gamertag, min_matches=3, limit=50)`
2. Requête SQL sur `match_participants` (JOIN `killer_victim_pairs` + `match_registry`) — une seule requête agrégée
3. Nouvelle page `src/ui/pages/rivals.py`
4. Filtres : mode_category, fenêtre temporelle (30j/90j/all)
5. 3 sections : Nemesis (top 3), Proie (top 3), tableau complet paginé

#### Points d'attention

- Exclure les bots (`xuid LIKE 'bid(%'`) — comme ailleurs dans le codebase
- Minimum `min_matches` configurable pour éviter les conclusions sur 1 rencontre
- Le head-to-head **win rate** peut être trompeur si les matchs sont en équipe large (BTB 12v12) — noter le contexte

**Complexité estimée** : Moyenne — SQL complexe mais données existantes, pas de nouveau stockage requis (tout calculé à la volée ou mis en vue matérialisée)

---

### [v6.3] Discord — Résumé de session automatique post-sync

**Noté le** : 2026-03-28
**Priorité** : Basse

#### Problème : comment détecter qu'une session est terminée ?

La table `sessions` groupe déjà les matchs en sessions (gap > N min entre deux matchs = nouvelle session). Une session est considérée **terminée** si :

> `now() - last_match_end_time_of_session > SESSION_CLOSE_THRESHOLD` (défaut : 60 min)

Le mécanisme de déclenchement naturel est **la fin du sync** (`sync.py --delta`) : quand le sync s'achève, on vérifie les sessions complètes non encore notifiées.

**Pas de polling.** Pas de process daemon. Déclenché uniquement au sync.

#### Implémentation

**Déclenchement — bouton `📤` dans l'UI**

Pas d'automatisation temporelle. Un petit bouton `📤` placé à côté du bouton "Synchroniser" dans la sidebar/header.

**Condition d'activation** : le bouton est **grisé** tant que `last_match_end_time_of_last_session + SESSION_NOTIFY_DELAY_MINUTES > now()`.
- Valeur par défaut : **5 minutes** (configurable dans `app_settings.json` → `discord_session_notify_delay_minutes`)
- Tooltip quand grisé : "Dernier match terminé il y a 2 min — disponible dans 3 min"
- Tooltip quand actif : "Envoyer le résumé de la session sur Discord"

Pourquoi `last_match_end_time` et pas `last_sync_at` : si tu synces un match terminé il y a 2h, le bouton est immédiatement actif. Si tu synces un match qui vient juste de finir, le délai s'applique pour éviter d'envoyer un résumé partiel.

**Données** :
- Colonne `discord_notified_at TIMESTAMP DEFAULT NULL` dans `sessions` (migration)
- `discord_session_notify_delay_minutes` dans `app_settings.json` (défaut : 5)

**Logique au clic** (`src/utils/discord_notifier.py` existant à étendre) :
1. Identifier la dernière session non encore notifiée (`discord_notified_at IS NULL`)
2. Vérifier que `last_match_end_time + delay < now()` (guard côté serveur, pas seulement UI)
3. `build_session_embed(session)` → POST Discord → `UPDATE sessions SET discord_notified_at = now()`
4. Confirmation inline dans l'UI : "✅ Résumé envoyé sur Discord"

**Contenu de l'embed Discord** :
- Nb matchs / W-L / win rate de la session
- Meilleur match (perf_score max) avec carte + mode + score
- `form_score` delta (si v8 axe 1 implémenté)
- Top médaille de la session
- Badge comeback/dominance si présent (via `DominanceFlag`)
- Composition escouade (coéquipiers présents ≥ 2 matchs)
- **Rôles de soirée** (section légère, max 3 lignes) : réutiliser `compute_impact_scores()` de `src/analysis/friends_impact.py` sur les matchs de la session → extraire 🏆 Champion, 🍌 Maillon Faible, et optionnellement le joueur avec le plus de ⚡ First Blood ou 🎯 Clutch Finisher. Les emojis et libellés i18n existent déjà (`tmi_mvp_label`, `tmi_boulet_label`). Aucune nouvelle logique d'archetype à créer.
- **Héros silencieux 🛡️** (à ajouter si validé en prod) : joueur avec le ratio `assists/(deaths+1)` le plus élevé sur la session. Nécessite un nouveau critère dans `friends_impact.py` + clé i18n `tmi_silent_hero_label`. À brancher dans la matrice d'impact de l'app avant d'intégrer à la notif Discord.

**Opt-in** : bouton visible uniquement si `app_settings.discord_session_notify = true` ET webhook configuré.

**Complexité estimée** : Faible à Moyenne — la logique de déclenchement est simple, l'effort est dans la mise en forme de l'embed et les tests d'idempotence

---

### [v6.3] Clutch moments — détection intelligente des kills décisifs

**Noté le** : 2026-03-28
**Priorité** : Basse

#### Concept

Un kill est "clutch" s'il a été réalisé dans un **contexte de haute valeur** :
- La cible était en pleine série (spree en cours)
- Le match était serré et proche de sa fin
- Le kill a inversé ou préservé un avantage fragile

La difficulté : l'API ne fournit pas l'état du score seconde par seconde. On doit **inférer** le contexte depuis les données disponibles.

#### Définitions retenues (par ordre de fiabilité)

| Type | Définition | Source de données |
|------|-----------|-------------------|
| **Spree-stopper** | Kill sur un joueur qui avait une médaille de série dans ce match (`Killing Spree`, `Rampage`, `Running Riot`, `Demon`) | `medals_earned` × `killer_victim_pairs` |
| **Comeback clutch** | Kill réalisé dans un match tagué `DominanceFlag.COMEBACK` ou `COUNTER_COMEBACK`, où le joueur est dans le top-2 killers de l'équipe | `match_registry.comeback_flag` × `match_participants.kills` |
| **Fin de match sous tension** | Kill dans les dernières 60 secondes d'un match Slayer dont le score final était ≤ 2 pts d'écart | `killer_victim_pairs.timestamp_ms` × `match_registry.duration_ms + team_scores` |

#### Stockage proposé

Pas de nouvelle table — stocker un compteur agrégé par match :

```sql
-- Nouvelle colonne dans player_match_enrichment
clutch_kills INTEGER DEFAULT 0   -- nb de kills clutch dans ce match (tous types confondus)
clutch_type  TEXT DEFAULT NULL   -- type principal : 'spree_stopper' | 'comeback' | 'last_minute'
```

#### Backfill

- Nouveau flag `--clutch-kills` dans `scripts/backfill_data.py`
- Implémentation : `src/analysis/clutch_analysis.py` (nouveau module, 0 accès DB)
- Orchestrateur : `scripts/backfill/orchestrator.py`

#### Affichage UI

- Badge "Clutch" sur la carte de match (page historique) quand `clutch_kills > 0`
- Stat "Clutch kills" dans le détail d'un match
- Filtre "Matchs clutch" dans la page historique

#### Limites connues / honnêteté analytique

- Le **spree-stopper** est approximatif : `medals_earned` dit qu'un joueur a eu une série dans ce match, pas au moment précis du kill. Un joueur peut avoir eu sa série en début de match et être tué en fin.
- Le **last-minute** dépend de `killer_victim_pairs.timestamp_ms` disponible dans le filmshell — seulement pour les matchs avec extraction weapon_kills complète.
- Ces approximations doivent être mentionnées dans l'UI (tooltip ou info icon).

**Complexité estimée** : Moyenne — l'algo spree-stopper est faisable rapidement, le last-minute dépend de la couverture filmshell, le comeback clutch réutilise `DominanceFlag` déjà en place

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
