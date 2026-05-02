# Migration Gap — Python `v7/cockpit` → Go (`apps/go-api/`)

> Audit de la couverture des stats du dashboard Halo Infinite sur la branche Python
> de référence (`v7/cockpit`) vs la réécriture Go en cours (`apps/go-api/`).
>
> Objectif : recenser pour **chaque graphe / tableau / KPI** la source de données
> exacte (table.colonne DuckDB, JOIN, formule), puis pointer ce qui est manquant ou
> erroné côté Go.
>
> ## ✅ Validation BDD (2026-04-26)
>
> Probe Go ad-hoc exécutée sur `C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/` (production Python active). **Tous les inputs nécessaires aux 3 chantiers prioritaires (scoreboard + expander, citations, impact) sont présents et peuplés** :
>
> | Élément | État | Volumétrie |
> |---|---|---|
> | `shared.match_participants` (28 cols, kills_expected/team_mmr inclus) | ✅ | 24 395 lignes |
> | `shared.v_gamertag_lookup` (VIEW) | ✅ | 15 370 |
> | `shared.v_weapon_kills` (VIEW, `effective_weapon_id`) | ✅ | 100 343 / 85 110 hors sentinelles |
> | `shared.v_killer_victim_full` (VIEW) | ✅ | OK |
> | `shared.medals_earned` (Perfect=590, Steaktacular=204) | ✅ | 22 023 |
> | `shared.highlight_events` (kill=100 778 tous joints à team_id) | ✅ | 263 414 |
> | `meta.medal_definitions` + `medal_translations` (14 langues BCP-47) | ✅ | 167 + 2 145 |
> | `meta.weapon_labels` (UBIGINT) | ✅ | 42 |
> | `meta.citation_mappings` (88, dont 84 actifs) | ✅ | weapon_stat=24, medal=15, custom=12, stat=11, award=9, composite=7, pve_stat=6 |
> | 12 `custom_function` distinctes en BDD | ✅ | compute_{annexion_forcee, bulldozer, flag_em_down, hijack, mongoose/warthog/wraith_destroyer, vandalism, wins_ctf/slayer/strongholds} |
> | `personal_score_awards` (kill/assist/objective/penalty/vehicle) | ✅ | 1 064 |
> | `match_skill_rank`, `player_match_enrichment` (friends_xuids, had_bot_teammate, dominance_flag) | ✅ | 364 |
> | `match_citations` (déjà calculées par Python) | ✅ | 5 137 |
> | **`_SCOREBOARD_SQL` end-to-end sur 1 match réel** | ✅ | 8 lignes retournées avec tout résolu |
>
> **Petits trous hors scope** : `pve_match_stats` n'a pas 5 colonnes Firefight (`wave_completed, crawler/soldier/knight/warden_kills`) ; `compute_wins_firefight` absent de `citation_mappings`. N'affecte ni le scoreboard ni l'impact ; pour les citations, 6/10 mappings `pve_stat` sont câblés.
>
> **Conclusion** : aucun blocage côté schéma ou sync. Les 3 chantiers sont purement applicatifs (logique Go à porter). Les VIEWs shared sont par ailleurs définies dans les migrations Go (`steps_shared.go:applyResolutionViews`), donc le Go peut produire le même schéma.

---

> Conventions de bases :
>
> - **shared** = `data/warehouse/shared_matches_v2.duckdb` (ATTACH AS `shared`)
> - **meta** = `data/warehouse/metadata.duckdb` (ATTACH AS `meta`)
> - **stats** = `data/players/{gamertag}/stats.duckdb` (DB locale)
> - **pve** = `data/warehouse/shared_pve.duckdb`
> - Encodage outcome : **WIN=2, LOSS=3, TIE=1, DNF=4** (constant tout le long).

---

## Table des matières

1. [Match View — Scoreboard (corpus principal)](#1-match-view--scoreboard)
2. [Citations (definitions, mapping_type, custom rules, composite)](#2-citations)
3. [Médailles (definitions, traductions, IDs spéciaux, Perfect Kill)](#3-médailles)
4. [Match View — Tabs / Charts](#4-match-view--tabs--charts)
5. [Badges d'impact (Premier sang, Touriste, Boulet, Héros silencieux, Faux-frère, Top Killer, Top Gun, …)](#5-badges-dimpact-tourist-boulet)
6. [Teammates (perfect_kills, history, impact, synergy)](#6-teammates)
7. [Timeseries](#7-timeseries)
8. [Career & Encounters](#8-career--encounters)
9. [Synthesis & Squad](#9-synthesis--squad)
10. [Performance Score / Skill Rating / MMR](#10-performance-score--skill-rating--mmr)
11. [Killer/Victim & Weapon Kills](#11-killervictim--weapon-kills)
12. [Spawn / Comeback / Win Streaks / Cadence / First Events / Friends Impact](#12-spawn--comeback--win-streaks--cadence--first-events--friends-impact)
13. [Awards (personal_score_awards)](#13-awards)
14. [Citations Backfill](#14-citations-backfill)
15. [Tables / vues croisées de référence](#15-tables-vues-de-référence)
16. [Plan de portage compatible multi-titres](#16bis-plan-de-portage-compatible-multi-titres)
17. [Synthèse des écarts critiques Python → Go](#16-synthèse-des-écarts-critiques-python--go)

---

## 1. Match View — Scoreboard

> 🟢 **Priorité 1** : 80 % du corpus côté UI passe par cette requête.

### 1.1 Source unique côté Python — `_SCOREBOARD_SQL`

`src/data/repositories/_roster_loader.py:124-160`. Loader :
`RosterLoaderMixin.load_match_scoreboard(match_id)` (~ligne 390), utilisé par
`src/ui/pages/match_view_players_data.py::load_match_scoreboard`. Renderer :
`src/ui/pages/match_view_scoreboard.py::render_match_scoreboard` (~ligne 365).

```sql
SELECT
  p.xuid,
  COALESCE(vg.gamertag, p.gamertag, p.xuid) AS gamertag,
  p.team_id, p.rank, p.score,
  p.kills, p.deaths, p.assists, p.kda,
  p.max_killing_spree, p.headshot_kills,
  p.shots_fired, p.shots_hit, p.accuracy,
  p.melee_kills, p.power_weapon_kills,
  p.damage_dealt, p.damage_taken, p.avg_life_seconds,
  COALESCE(pk.perfect_kills, 0)            AS perfect_kills,
  wk.weapon_id                             AS top_weapon_id
FROM shared.match_participants p
LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = p.xuid
LEFT JOIN (
  SELECT xuid, SUM(count) AS perfect_kills
  FROM shared.medals_earned
  WHERE match_id = ? AND medal_name_id = 1512363953
  GROUP BY xuid
) pk ON pk.xuid = p.xuid
LEFT JOIN (
  SELECT xuid, effective_weapon_id AS weapon_id
  FROM (
    SELECT xuid, effective_weapon_id,
           ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn
    FROM shared.v_weapon_kills
    WHERE match_id = ? AND effective_weapon_id NOT IN (0, 1, 2)
    GROUP BY xuid, effective_weapon_id
  ) WHERE rn = 1
) wk ON wk.xuid = p.xuid
WHERE p.match_id = ?
  AND NOT (
    COALESCE(p.kills,0)=0 AND COALESCE(p.deaths,0)=0
    AND COALESCE(p.assists,0)=0 AND COALESCE(p.score,0)=0
    AND (p.kills IS NOT NULL OR p.deaths IS NOT NULL
         OR p.assists IS NOT NULL OR p.score IS NOT NULL)
  )
ORDER BY p.team_id ASC NULLS LAST, p.rank ASC NULLS LAST
```

3 placeholders : `[match_id, match_id, match_id]`.

### 1.2 Mapping colonne par colonne

| Colonne | Source | Formule / notes |
|---|---|---|
| `xuid` | `shared.match_participants.xuid` | str-cast |
| `gamertag` | `COALESCE(shared.v_gamertag_lookup.gamertag, mp.gamertag, mp.xuid)` | gamertags fantômes (chiffres seuls, `xuid(...)`, vide, `?`) → forcés `None` dans `_resolve_scoreboard_gamertags` ; bots résolus via `get_bot_name(bid(...))` |
| `team_id` | `mp.team_id` | int |
| `rank` | `mp.rank` | int, fallback row-index+1 |
| `score` | `mp.score` | int |
| `kills` / `deaths` / `assists` | `mp.{kills,deaths,assists}` | int |
| `kda` | `mp.kda` (déjà calculé en shared) | float, affiché `.2f` |
| `max_killing_spree` | `mp.max_killing_spree` | int |
| `headshot_kills` | `mp.headshot_kills` | int |
| `shots_fired` / `shots_hit` | `mp.{shots_fired, shots_hit}` | int |
| `accuracy` | `mp.accuracy` | float ; UI multiplie par 100 si `<=1.0`, affiche `%.1f %` |
| `melee_kills` | `mp.melee_kills` | int |
| `power_weapon_kills` | `mp.power_weapon_kills` | int |
| `damage_dealt` / `damage_taken` | `mp.{damage_dealt, damage_taken}` | float, affiché `int(round(...))` |
| `avg_life_seconds` | `mp.avg_life_seconds` | float, affiché `m:ss` |
| `perfect_kills` | `SUM(shared.medals_earned.count) WHERE medal_name_id = 1512363953` par (match_id, xuid) | constant `1512363953` (« Perfect » / « À table ») |
| `top_weapon_id` | `shared.v_weapon_kills.effective_weapon_id`, top-1 par COUNT par xuid, sentinelles `(0,1,2)` exclues | résolu côté UI via `resolve_weapon_display(int(weapon_id))` |

### 1.3 Filtre « ghost player » (réutilisé partout)

`_SQL_NOT_GHOST` exclut une ligne **uniquement si** `kills, deaths, assists, score`
sont tous à 0/NULL **mais** au moins un est `IS NOT NULL`. Une ligne avec tout
NULL est conservée.

### 1.4 MVP / LVP / surlignage des cellules

`src/ui/pages/match_view_scoreboard.py:_compute_scoreboard_extremes`,
`_sb_cell_class`, `_compute_mvp_lvp` :

- pour chaque colonne (sauf `gamertag`, `rank`, `top_weapon_id`), calcule (min, max).
- cellule classée `--best` si à l'extrême ; les colonnes inversées (`deaths`, `damage_taken`) inversent best/worst ;
- MVP = humain avec le plus de cellules « best » (≥ 2 best mini) ; LVP = symétrique sur worst (≥ 2 worst). Bots (xuid commençant par `bid(`) exclus.

### 1.5 Panneau de détail joueur

`src/ui/pages/match_view_scoreboard_detail.py:load_scoreboard_player_extra_data`
agrège par joueur :

- **Armes** : `repo.load_weapon_kills_for_player(xuid, [match_id])` = `shared.v_weapon_kills`, filtrer `kills>0`, exclure `EXCLUDED_WEAPON_IDS={0,1,2}`, fusion via `WEAPON_FUSION_MAP_ID`. Enrichi avec grenades/melee depuis `shared.match_participants.{grenade_kills, melee_kills}`, plafonné à `api_total_kills - film_kills`.
- **Médailles** : `repo.load_match_medals(match_id)` = `SELECT medal_name_id, count FROM shared.medals_earned WHERE match_id=? AND xuid=?` → top 5 ; nom via `resolve_medal_name(id, lang)` (cf. §3.2).
- **Citations** : `CitationEngine(player_db, xuid).aggregate_for_display(match_ids=[match_id])`. Citations déjà masterisées avant ce match exclues. Top 4.
- **Expected vs Actual** : `repo.load_match_skill_data(match_id)` → `{kills:{count,expected}, deaths:{...}, assists:{...}}` depuis `match_participants.{kills, kills_expected, …}`.
- **Antagonistes** : `compute_personal_antagonists_from_pairs_polars` sur `shared.v_killer_victim_full`.
- **Skill rank** : `SELECT rating_type, rating_value, tier_label, rating_delta FROM match_skill_rank WHERE match_id=?` (DB joueur).
- **Performance score / had_bot_teammate** : `repo.load_player_match_enrichment(match_id)` = `stats.player_match_enrichment`.

### 1.6 ⚠️ Écarts Go — Match View Scoreboard

Source Go : `apps/go-api/internal/platform/duckdb/queries_match.go:Q12MatchScoreboard`.

| Écart | Détail | Impact |
|---|---|---|
| **`v_gamertag_lookup` non utilisée** | Go fait `LEFT JOIN shared.xuid_aliases xa` au lieu de la vue `shared.v_gamertag_lookup` (qui est l'union `xuid_aliases ∪ match_participants`). | Gamertags manquants quand un joueur n'est pas dans `xuid_aliases` mais présent dans `match_participants`. |
| **COALESCE incomplet** | Go : `COALESCE(xa.gamertag, p.xuid)` ; Python : `COALESCE(vg.gamertag, p.gamertag, p.xuid)`. | Pareil — perd les gamertags portés par `match_participants.gamertag` directement. |
| **Filtre ghost manquant** | Go n'applique PAS le `_SQL_NOT_GHOST`. | Affichage de joueurs « tous à zéro » qui devraient être filtrés. |
| **`ORDER BY` sans `NULLS LAST` sur team_id** | Python a `team_id ASC NULLS LAST, rank ASC NULLS LAST`. | Tri instable quand team_id est NULL. |
| **MVP / LVP non implémenté** | `_compute_mvp_lvp` + `_compute_scoreboard_extremes` absents côté Go. | KPI de surlignage scoreboard manquant. |
| **Filtre bots `bid(...)`** | Bots non détectés / non exclus côté Go. | MVP/LVP fausses, gamertag bot affiché en xuid brut. |

---

## 2. Citations

### 2.1 Source — `meta.citation_mappings`

`src/data/citation_definitions.py`, `src/analysis/citations/engine.py:load_mappings`.

Colonnes : `citation_name_norm, citation_name_display, mapping_type, medal_id,
medal_ids, stat_name, award_name, award_category, custom_function,
composite_children, confidence, notes, image_path, category, description,
tier_targets, subcategory, enabled`.

Filtre : `enabled IS NOT FALSE`.

### 2.2 Dispatch par `mapping_type` (`compute_citation_for_match`)

| `mapping_type` | Logique |
|---|---|
| `medal` | Somme de `match_medals[id]` sur les IDs de `medal_ids` (CSV) ou `medal_id` simple. Source : `shared.medals_earned` (xuid courant). |
| `stat` / `pve_stat` / `weapon_stat` | `int(match_stats[stat_name])` depuis le join `shared.match_participants p ⨝ shared.v_match_full r`. `pve_stat` ajoute les colonnes `shared_pve.pve_match_stats`. `weapon_stat` lit `weapon_kills:{name}` (nom canonique EN résolu depuis `shared.v_weapon_kills`). |
| `award` | `match_awards[award_name]` agrégé depuis `personal_score_awards` (DB joueur), `SUM(award_count)`. |
| `custom` | Dispatch via `CUSTOM_FUNCTIONS` registry. |
| `composite` | Non calculé par-match ; `_apply_composite_citations` compte combien d'enfants ont atteint la dernière valeur de `tier_targets`. |

### 2.3 Custom functions (`src/analysis/citations/custom_rules.py`)

| Fonction | Logique | Données |
|---|---|---|
| `compute_bulldozer` | Slayer/Assassin avec KDA > 8, exclut firefight/BTB | `match_participants` (playlist_name, game_variant_name, kda) |
| `compute_wins_ctf` | Wins où `playlist_name` matche `ctf|capture.*drapeau|drapeau.*neutre|neutral.*flag` ET `outcome=2` | match_stats |
| `compute_wins_firefight` | `firefight\|baptême\|bapteme` + outcome=2 | match_stats |
| `compute_wins_slayer` | `slayer\|assassin` + outcome=2 | match_stats |
| `compute_wins_strongholds` | `stronghold\|bases` + outcome=2 | match_stats |
| `compute_annexion_forcee` | Walk de `highlight_events` (mode/death) — séries de 3 events « mode » sans « death » entre. Fallback : `awards["zone_captured"] // 3`. | `shared.highlight_events` |
| `compute_flag_em_down` | `awards["runner_stopped"] + ["Porteur arrêté"] + ["Flag Carrier Kill"] + ["Flag Carrier Killed"]` | awards |
| `compute_hijack` | Awards commençant par `hijacked_` ou contenant `hijack/skyjack` | awards |
| `compute_vandalism` | Awards `destroyed_*` ou contenant `destroyed/destruction` | awards |
| `compute_wraith_destroyer` | `awards["destroyed_wraith"] + fallbacks legacy` | awards |
| `compute_mongoose_destroyer` | `awards["destroyed_mongoose"] + fallbacks` | awards |
| `compute_warthog_destroyer` | `destroyed_warthog + destroyed_rocket_warthog + fallbacks` | awards |

### 2.4 Citations composites (`src/analysis/citations/composite.py`)

Pour chaque `mapping_type='composite'`, parse `composite_children` (JSON list).
Pour chaque enfant :

- si `tier_targets` vide → masterisé si count > 0 ;
- sinon : masterisé si `count >= max(tier_targets)`.

Résultat = nombre d'enfants masterisés, écrit dans `result[norm_name]` si `> 0`.

### 2.5 Stockage

Calcul par-match via `compute_and_store_for_match` →
`match_citations(match_id, citation_name_norm, value)` PK `(match_id, citation_name_norm)`.
Marqueur `_processed=1` par match.
Agrégation : `SELECT citation_name_norm, SUM(value) FROM match_citations [WHERE …] GROUP BY 1`.

### 2.6 ⚠️ Écarts Go — Citations

| Écart | Détail | Impact |
|---|---|---|
| **Citation engine absent** | Go a un flag `SyncScope.Citations` + `--citations`, mais **aucune implémentation** ne calcule/écrit dans `match_citations`. Seule la lecture `Q35 SELECT FROM match_citations` existe (`citations_repo.go:LoadCitationTotals`). | Page Citations vide ou désynchronisée. |
| **Custom functions** | Aucune des 12 fonctions custom n'est portée. | Citations Bulldozer, Annexion forcée, Flag em down, Hijack, Vandalism, Wraith/Mongoose/Warthog destroyer, Wins par mode → toutes manquantes. |
| **Composites** | Logique composite (max(`tier_targets`) atteint) absente. | Citations composites jamais débloquées. |
| **`mapping_type` partiel** | `queries_home_citations.go:409` filtre `WHERE mapping_type = 'medal'` — seuls les médailles sont câblées. | `stat`, `pve_stat`, `weapon_stat`, `award`, `custom`, `composite` non calculés. |
| **`tier_targets` parsing** | Stocké en VARCHAR mais non parsé pour la logique masterisation (`_parse_tier_targets`, `_compute_mastery_display`). | Affichage tiers cassé. |

---

## 3. Médailles

### 3.1 Tables

| Table | DB | Colonnes |
|---|---|---|
| `medal_definitions` | meta | `medal_name_id, name_fr, name_en, description_fr, description_en, is_custom` |
| `medal_translations` | meta | `medal_name_id, lang (BCP-47, ex: "fr-FR"), name, description` (priorité sur definitions) |
| `medals_earned` | shared | `match_id, xuid, medal_name_id, count` |

### 3.2 Chaîne de résolution (`src/data/medal_definitions.py`)

`resolve_medal_name(id, lang)` →
`medal_translations` (BCP-47) →
`medal_translations` (en-US) →
`medal_definitions.{name_fr|name_en}`.
Description analogue (`resolve_medal_description`). Bulk : `load_medal_name_maps()`
retourne `(fr_map, en_map)` ; `load_medal_description_map(lang)` idem.

### 3.3 IDs spéciaux / constantes

| ID | Sens | Constante Python |
|---|---|---|
| `1512363953` | « Perfect » / « À table » → colonne `perfect_kills` du scoreboard | hardcodé dans `_SCOREBOARD_SQL` ; `MedalsMixin.count_perfect_kills_by_match` (= `count_medal_by_match(ids, 1512363953)`) |
| `1169390319` | « Steaktacular » → déclenche Domination/Humiliation (`DominanceFlag.DOMINATION=1`, `HUMILIATION=2`) | `src/analysis/_medal_verdicts.py:MEDAL_STEAKTACULAR_ID` |

### 3.4 API `MedalsMixin` (`src/data/repositories/_medals_repo.py`)

- `load_top_medals(match_ids, top_n=25)` → `SELECT medal_name_id, SUM(count) ... GROUP BY 1 ORDER BY total DESC LIMIT ?`.
- `load_match_medals(match_id)` → `SELECT medal_name_id, count FROM shared.medals_earned WHERE match_id=? AND xuid=?`.
- `count_medal_by_match(match_ids, medal_name_id)` → scalaire par-match, filtré xuid.
- `count_perfect_kills_by_match(match_ids)` → idem avec `medal_name_id=1512363953`.
- `load_medal_definitions()` → DataFrame Polars.
- `get_medal_label(id, lang)`.

### 3.5 `DominanceFlag` IntEnum (`src/analysis/_medal_verdicts.py`)

Stocké dans `stats.player_match_enrichment.dominance_flag` (TINYINT) :

- `0` NONE
- `1` DOMINATION (Steaktacular gagné par mon équipe)
- `2` HUMILIATION (adversaire a gagné Steaktacular)
- `3` REMONTADA (était mené, gagné)
- `4` DEBANDADE (était mené devant, perdu)
- `5` CONTRE_REMONTADA (était devant, adversaire revenu à égalité, j'ai tenu)

Comeback : `COMEBACK_DEFICIT_PCT=0.40`, `COMEBACK_DEFICIT_FALLBACK=20`,
`COMEBACK_MAX_SLAYER_WIN_SCORE=100`. `SLAYER_WIN_SCORES =
{arena_slayer:50, btb_slayer:100, escalation_slayer:11}`.

### 3.6 ⚠️ Écarts Go — Médailles

| Écart | Détail | Impact |
|---|---|---|
| **Pas de chaîne `medal_translations` BCP-47** | Go résout depuis `medal_definitions.{name_fr,name_en}` sans passer par `medal_translations`. | Les traductions BCP-47 (fr-FR, fr-CA, en-GB…) non honorées ; locales custom Halo non prises. |
| **Constante `MEDAL_STEAKTACULAR_ID` absente** | Go calcule DOMINATION/HUMILIATION depuis la courbe de score (cf. §12.2), pas depuis `medals_earned[1169390319]`. | Sémantique du dominance_flag différente (cf. §10/§12 et §16). |
| **`load_top_medals`** | À vérifier ; pas de couverture dans la requête scoreboard (médailles uniquement dans `Q14MatchMedals`). | OK pour vue match, mais pas de top global cross-match. |

---

## 4. Match View — Tabs / Charts

Orchestré dans `src/ui/pages/match_view_tabs.py` (Summary, Combat, Team,
Citations, Media).

### 4.1 Expected vs Actual (`render_expected_vs_actual` — `match_view_charts.py`)

- KPI : `team_mmr` vs `enemy_mmr`, `kills`/expected, `deaths`/expected, `assists`/expected, `avg_life_seconds`.
- Toutes lues depuis la ligne courante de `shared.match_participants` (`team_mmr, enemy_mmr, kills, kills_expected, deaths, deaths_expected, assists, assists_expected, avg_life_seconds`).
- Moyennes catégorie historique via `compute_mode_category_averages(df_full, mode_category)` (`pair_name`).

### 4.2 Weapon Kills (`match_view_weapon_kills.py`)

- Pie + table.
- Source : `shared.v_weapon_kills` filtrée par xuid+match, GROUP BY `effective_weapon_id`. Filtre `kills>0`, exclut `{0,1,2}`, fusion `WEAPON_FUSION_MAP_ID`.
- Noms via `resolve_weapon_display(weapon_id, lang)` (`meta.weapon_labels` UBIGINT v5.4).
- Enrichi grenade/melee depuis `match_participants.{grenade_kills, melee_kills}`, plafonné à `api_total - film_kills`.

### 4.3 Participation radar (`match_view_participation.py`)

- Source : `repo.load_personal_score_awards_as_polars(match_id=match_id)` = `personal_score_awards` (DB joueur), colonnes `award_name, award_category, award_count, award_score`.
- + `match_row` (pair_name, deaths, time_played_seconds) pour Impact / Survie.
- 6 axes : Objectifs, Combat, Support, Score, Impact, Survie. Seuils par famille de mode dans `RADAR_THRESHOLDS_PER_MODE` (`src/analysis/participation_radar.py`).

### 4.4 Match Impact + Timeline (`match_view_players.py:render_match_impact_section`)

- Charge `shared.highlight_events` du match.
- `compute_single_match_impact(events, me_xuid, outcome, team_xuids, participants_stats, lang)` → premier sang, finisseur, dernière victime, etc. (cf. §5).
- Rend les badges + `plot_match_kill_death_timeline`.

### 4.5 Team Dominance Timeline (`match_view_players_timeline.py`)

- PvP only (`is_firefight=False`).
- `compute_dominance_buckets(he, xuid_to_team, my_team_id, duration_s)` — tug-of-war 30 s.
- `detect_streaks(he, xuid_to_team, xuid_to_gamertag)` — séries annotées.
- `playable_duration_seconds` depuis `match_registry` ; `xuid_to_team` depuis `match_participants.team_id`.

### 4.6 Match Cadence (`render_match_cadence_section`)

`compute_cadence_buckets(events, xuid_to_team, my_team_id, duration_s, bucket_s=30)` — kills par bucket de 30 s par équipe. Source `shared.highlight_events` event_type='kill'.

### 4.7 Nemesis (`match_view_players_nemesis.py:render_nemesis_section`)

- `repo.load_killer_victim_pairs_as_polars(match_id=match_id)` = `shared.v_killer_victim_full` (cols : `match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms`).
- `compute_personal_antagonists_from_pairs_polars(pairs_df, me_xuid)` → top nemesis (kills contre moi) + top victim (kills par moi).

### 4.8 KD Timeline

`plot_all_players_frags_timeline(he, ...)` à partir de `shared.highlight_events` event_type='kill'.

### 4.9 Encounters (`match_view_encounters.py`)

`load_encounter_stats(self_xuid, target_xuids, db_path, match_start_time, current_match_id)` (`_encounter_loader.py`).

CTE :

- `my_matches` (`mp ⨝ mr`) → matchs du joueur ;
- `encounters` : count, ally_count, enemy_count, winrate_as_ally (`outcome=2`), winrate_vs_enemy (`outcome IN (3,4)`) ;
- `kvp_agg` : K/D croisé depuis `shared.killer_victim_pairs` ;
- option `start_time < match_start_time` (passé seulement) ;
- badges (allié+, coriace, tough nut) via `compute_encounter_badges`.

### 4.10 Rank section (`match_view_rank.py`)

Lit `match_skill_rank` (DB joueur) du match courant : `rating_type, rating_value,
rating_deviation, tier_label, sub_tier, tier_name, tier_fr, rating_delta,
playlist_group`. Image rank + delta + caveat bot-teammate.

### 4.11 Citations panel (`match_view_citations.py`)

`CitationEngine(db_path, xuid).aggregate_for_display(match_ids=[match_id])` (delta) +
`aggregate_for_display(None)` (full). Mastery affichage : masque celles déjà
masterisées avant ce match.

### 4.12 Médailles tab (`render_medals_tab`)

Rend le dict `medals_last` (= `repo.load_match_medals`) via `render_medals_grid`.
Noms : `load_medal_name_maps` ; descriptions : `load_medal_description_map`.

### 4.13 ⚠️ Écarts Go — Tabs / Charts

| Section | État Go | Écart |
|---|---|---|
| Expected vs Actual | partiel | `compute_mode_category_averages` (moyennes catégorie historique) absent. |
| Weapon Kills | partiel | Logique `WEAPON_FUSION_MAP_ID` non appliquée → variantes (Bandit Evo / M392, Shock Rifle Ranked / Shock Rifle, etc.) comptées séparément. Plafond grenade/melee par soustraction non fait. |
| Participation radar | absent | `RADAR_THRESHOLDS_PER_MODE` non porté. |
| Match Impact (badges) | **fortement incomplet** | cf. §5. |
| Team Dominance | absent | Pas de tug-of-war 30 s. |
| Match Cadence | partiel | À vérifier — `compute_cadence_buckets` dans `analysis/`. |
| Nemesis | partiel | À vérifier (présent dans `analysis/killer_victim.go`). |
| Encounters | partiel | Logique badges (allié+, coriace) à confirmer. |
| Rank section | partiel | `match_skill_rank` lu, mais delta + caveat bot à valider. |
| Citations panel | absent | cf. §2.6. |
| Médailles tab | partiel | Pas de chaîne `medal_translations` BCP-47 (cf. §3.6). |

---

## 5. Badges d'impact (Touriste, Boulet, …)

> 🟢 **Priorité utilisateur** : pris en flagrant délit côté Go, c'est ici que le delta est le plus voyant.

### 5.1 Catalogue Python (français → en)

| event_type | Libellé FR | Icône | Logique (Python) | Source |
|---|---|---|---|---|
| `first_blood` | Premier sang | ⚡ | premier kill du match (toutes équipes confondues) | `highlight_events` event_type='kill', min(time_ms) |
| `clutch_finisher` | Finisseur | 🎯 | dernier kill d'une **victoire** | kills filtrés par équipe alliée, max(time_ms), `outcome==WIN` |
| `last_casualty` | **Boulet** | 💀 | dernière mort d'une **défaite** | deaths filtrés par équipe alliée, max(time_ms), `outcome==LOSS` |
| `last_group_kill` | **Touriste** | 🐌 | joueur le plus lent à obtenir son **1er kill** (≥ 2 joueurs avec ≥ 1 kill) | `kills` filtrés par équipe alliée → groupBy xuid → min(time_ms) par joueur → argmax |
| `first_group_death` | Première victime | 🪦 | première mort du match | deaths, min(time_ms) |
| `top_gun` | Top Gun | 🔫 | premier joueur à atteindre `TOP_GUN_KILL_THRESHOLD=10` kills | kills triés ASC, premier xuid à passer 10 |
| `top_killer` | Bourreau | 💥 | joueur de l'équipe alliée avec le plus de kills (≥ 1, équipe ≥ 2) | `participants_stats` filtré team_xuids |
| `silent_hero` | Héros silencieux | 🛡️ | **Victoire** : joueur de l'équipe avec MAX(assists) ET MIN(deaths) (même joueur), exclut le top killer | `participants_stats` (kills, deaths, assists) |
| `false_brother` | Faux-frère | 🗡️ | **Défaite** : joueur de l'équipe avec MAX(deaths) ET MIN(assists) (même joueur), exclut le top killer | idem |

Fichier source : `src/visualization/_match_impact_events.py:compute_single_match_impact`
(toutes les fonctions internes `_find_silent_hero_event`, `_find_false_brother_event`,
`_find_top_killer_event`, `_find_top_gun_event`).

Scores associés (`src/analysis/_impact_types.py`) :

```
SCORE_FIRST_BLOOD       = 2
SCORE_CLUTCH_FINISHER   = 2
SCORE_LAST_CASUALTY     = -2     # Boulet
SCORE_SILENT_HERO       = 1.5
SCORE_FALSE_BROTHER     = -1.5
SCORE_LAST_GROUP_KILL   = ...    # Touriste (cf. _impact_types.py)
SCORE_FIRST_GROUP_DEATH = ...
SCORE_TOP_KILLER        = 1.0
```

### 5.2 État côté Go — `apps/go-api/internal/analysis/match_impact.go`

```go
if input.MyKills == 0 {
    badges = append(badges, ImpactBadge{BadgeKey: "tourist", BadgeFR: "Touriste", ...})
}
// ... first_blood, finisher, first_victim
```

### 5.3 ⚠️ Écarts Go — Badges d'impact

| Badge | Côté Go | Bug / écart |
|---|---|---|
| **Touriste** | `tourist` déclenché si `MyKills == 0` | **❌ FAUX** : Python le déclenche pour le joueur de l'équipe avec le **1er kill le plus tardif**, pas pour qui a 0 kills. Si tout le monde a ≥ 1 kill, l'algo Python attribue toujours le badge ; Go ne le donne jamais. À l'inverse, Go le donne en double si plusieurs joueurs sont à 0. |
| **Boulet** | absent | `last_casualty` non implémenté. |
| **Héros silencieux** | absent | `silent_hero` non implémenté. |
| **Faux-frère** | absent | `false_brother` non implémenté. |
| **Top Killer** (« Bourreau ») | absent | non implémenté. |
| **Top Gun** | absent | seuil `TOP_GUN_KILL_THRESHOLD=10` à porter. |
| **Première victime** | partiellement | Go détecte `first_victim` uniquement si la victime est le joueur courant ; Python attribue le badge à n'importe quel joueur (et l'affiche pour tous). |
| **Premier sang / Finisseur** | partiellement | Go ne détecte que sur le joueur courant ; Python liste pour tous. Le filtre « alliés seulement » pour Finisseur n'est pas appliqué côté Go. |
| **Filtre équipe alliée** | absent | Python filtre `kills/deaths` à `team_xuids` pour Finisseur/Boulet/Touriste/Première victime/Héros/Faux-frère ; Go n'a pas la notion. |
| **Scores numériques** | absents | `SCORE_FIRST_BLOOD=2`, etc., utilisés dans `friends_impact.py` pour aggréger un score d'impact teammates. |

Conséquence : la page Match View Impact + la heatmap teammates « Impact » sont
soit vides soit faussées côté Go.

---

## 6. Teammates

### 6.1 `TeammatesService` (`src/data/services/teammates_service.py`)

- `_resolve_xuid_from_shared(conn, gamertag)` : `shared.v_gamertag_lookup` → `shared.xuid_aliases` → `shared.match_participants` (case-insensitive).
- `load_teammate_stats(gamertag, match_ids, ref_db)` → `_query_teammate_shared_stats(conn, xuid, match_ids)` joint `shared.match_participants p ⨝ shared.v_match_full r`. Colonnes : `match_id, start_time, map_id, map_name, map_name_fr, map_ui, playlist_id, playlist_name, pair_id, pair_name, game_variant_id, game_variant_name, outcome, team_id, kda AS ratio, max_killing_spree, headshot_kills, avg_life_seconds AS average_life_seconds, duration_seconds AS time_played_seconds, kills, deaths, assists, accuracy = 100*hit/fired, my_team_score, enemy_team_score, personal_score, is_firefight, is_ranked, xuid, team_mmr, enemy_mmr, kills_per_min`.

  Normalisation des scores : `NORM_MY_TEAM_SCORE_SQL` / `NORM_ENEMY_TEAM_SCORE_SQL` (`src/data/_score_sql.py`).

- `load_all_teammate_stats(gamertag, ref_db)` → `query_teammate_full_history(conn, xuid)` (sans match-id filter, exclut Firefight).
- `enrich_with_performance_score(df, gamertag, ref_db_path, is_main)` lit `player_match_enrichment.{performance_score, session_id, session_label}` depuis la DB du teammate (ou main).
- **`enrich_series_with_perfect_kills(series, db_path)`** (~ligne 339) : pour chaque (gamertag, df), résout xuid (df col → sync_meta → DB teammate), `repo.count_perfect_kills_by_match(match_ids)` (= `medals_earned medal_name_id=1512363953` filtré xuid). Ajoute la colonne `perfect_kills`. Fallback DB joueur si DB teammate absente.
- `compute_participation_profiles` : 6 axes radar par joueur (`personal_score_awards`).

### 6.2 Sub-modules

| Fichier | Rôle |
|---|---|
| `_teammates_perf_queries.py` | `load_perf_enrichment_with_session(db, match_ids)`, `load_team_mmr_by_match`, `load_outcome_by_match` (joint `xuid_aliases` pour gamertag). |
| `_teammates_history_queries.py` | `query_teammate_full_history(conn, xuid)` — full df Polars `shared.match_participants ⨝ v_match_full` filtré xuid, exclut Firefight. |
| `_teammates_impact_queries.py` | `_collect_impact_data` joint `shared.highlight_events ⨝ v_gamertag_lookup` + `match_participants.outcome`. Appelle `get_all_impact_events(events_df, matches_df, friend_xuids)` (cf. §12.6). |
| `_teammates_first_events_queries.py` | timestamps premier kill / mort. |

### 6.3 ⚠️ Écarts Go — Teammates

`apps/go-api/internal/service/teammates_service.go` + `domain/teammates.go`.

| Écart | Détail | Impact |
|---|---|---|
| **`perfect_kills_per_game`** | Champ `*float64` exposé (`PerfectKillsPerGame`), source `shared.medals_earned medal_name_id=1512363953`. | OK structurellement, mais à vérifier sur le filtre xuid + match_ids — `queries_squad.go:60+` enquête. |
| **Vue gamertag v6** | Côté Python = `v_gamertag_lookup` (union) ; Go = `xuid_aliases` direct. | Gamertags manquants pour des teammates récemment indexés. |
| **Filtre Firefight** | À valider que `is_firefight=FALSE` est appliqué partout côté Go pour la page Teammates standard. | Si non, contamine les moyennes. |
| **Normalisation `my_team_score` / `enemy_team_score`** | Les normalisateurs SQL `NORM_MY_TEAM_SCORE_SQL` / `NORM_ENEMY_TEAM_SCORE_SQL` (gestion des modes négatifs / firefight) ne sont probablement pas portés en Go. | Affichage de scores incohérents. |
| **6-axes participation profiles** | Probablement absent (lié à §4.3). | Heatmaps teammates « Impact » dégradées. |
| **Impact heatmap (badges friends_impact)** | Boucle `get_all_impact_events` → cf. §5.3 + §12.6. | Manque les multi-badges silent_hero / false_brother / top_killer. |

---

## 7. Timeseries

`src/data/services/timeseries_service.py`.

| Méthode | Source / formule |
|---|---|
| `enrich_performance_score(dff, df_full)` | `compute_performance_series(dff, history)` (cf. §10). |
| `compute_cumulative_metrics(dff)` | `compute_cumulative_net_score_series_polars`, `compute_cumulative_kd_series_polars`, `compute_rolling_kd_polars(window=5)`. Requiert `CORE_STAT_COLUMNS`. |
| `compute_score_per_minute(dff)` | `personal_score / (time_played_seconds/60)`, drop nulls et `time_played<=0`. |
| `compute_rolling_win_rate(dff)` | rolling_mean de `(outcome==WIN)` window=5 ×100. |
| `load_first_event_times(db, xuid, match_ids)` | `repo.get_first_kill_death_times(match_ids)` depuis `shared.highlight_events`. |
| `load_perfect_kills(db, xuid, match_ids)` | `repo.count_perfect_kills_by_match(match_ids)` (medal_id=1512363953). |

`compute_first_events_rolling(df, window=10)` (`src/analysis/first_events.py`) :
rolling mean de `first_kill_s`, `first_death_s` par xuid, sorted by `start_time`,
min_samples=3.

UI `src/ui/pages/timeseries.py` orchestre summary/distribution/correlation ;
sub-renderers `_timeseries_distributions.py`, `_timeseries_form.py`,
`_timeseries_intensity.py`, `_timeseries_weapons.py`. Skill rank progression :
`timeseries_skill_rank.py` sur historique `match_skill_rank`.

### 7.1 ⚠️ Écarts Go — Timeseries

`apps/go-api/internal/service/timeseries_service.go`.

| Écart | Détail | Impact |
|---|---|---|
| **`compute_rolling_kd_polars(window=5)`** | Vérifier la fenêtre exacte côté Go. | KD glissant peut différer. |
| **`compute_first_events_rolling(window=10, min_samples=3)`** | Constantes à reproduire. | Courbe « 1er kill / 1ère mort » faussée. |
| **`load_perfect_kills`** | Présent (queries Q35-like). | OK si filtre xuid bien appliqué. |
| **Skill rank progression** | À vérifier que `timeseries_skill_rank` rend `rating_delta`, `tier`, `sub_tier`. | Sinon graphe LUSR/CSR incomplet. |
| **`_timeseries_intensity.py`** | À voir s'il existe en Go (`match_intensity.go` semble présent côté Go). | OK potentiellement. |

---

## 8. Career & Encounters

### 8.1 `CareerMixin` (`_career_repo.py`)

| Méthode | Source |
|---|---|
| `load_career_data()` | `career_progression ORDER BY recorded_at DESC LIMIT 1` (rank, rank_name, rank_tier, current_xp, xp_for_next_rank, xp_total, is_max_rank, adornment_path, recorded_at, spartan_id) |
| `load_career_history(limit)` | idem ASC |
| `load_pre_sync_match_dates(first_sync_at)` | `shared.match_registry ⨝ match_participants WHERE start_time < first_sync_at` |
| `load_lusr_snapshot()` | `match_skill_rank` window `ROW_NUMBER() OVER (PARTITION BY playlist_group ORDER BY COALESCE(start_time, updated_at) DESC)` + `rating_delta` via `LAG` |
| `load_lusr_history(playlist_group)` | rows ASC sur `start_time/created_at`, filtre optionnel |
| `load_is_with_friends_batch(match_ids)` | `player_match_enrichment.is_with_friends` |
| `load_post_sync_match_count(first_sync_at)` | count `match_registry ⨝ match_participants WHERE start_time >= first_sync_at` |
| `load_friends_xuids_csv(match_id)` | `player_match_enrichment.friends_xuids` |
| `load_skill_ratings_batch(match_ids)` | `match_skill_rank` (rating_value, rating_type) batch |

### 8.2 `EncounterCareerMixin` (`_career_encounters_repo.py`)

- `_TOP_ENCOUNTERED_SQL` — encounter career-wide ; `shared.v_killer_victim_full`, exclut self + bots (`xuid LIKE 'bid(%`).
- `_ANTAGONISTS_SQL` — top nemesis OU top victims, period-filtered (`since`). Retourne `opponent_xuid, opponent_gamertag, times_killed, times_killed_by, matches_against, net_kills`. Source `shared.v_killer_victim_full + v_gamertag_lookup`.
- `_TOP_MATCHES_SQL` — Top 10 best/worst :
  - `shared.mv_player_matches mv ⨝ player_match_enrichment pme` ;
  - filtres : `xuid=?`, `outcome IN (2,3)`, `time_played_seconds >= MIN_MATCH_DURATION_SECONDS=180`, `had_bot_teammate=FALSE`, `is_firefight=FALSE`, deux team scores non null, option `mode_category != 'BTB'` ;
  - tri : `_BADGE_PRIORITY_EXPR[best]` (best : CONTRE_REMONTADA=3 > REMONTADA=2 > DOMINATION=1 ; worst : DEBANDADE=2 > HUMILIATION=1) DESC, puis `time_played_seconds ASC`, puis score-spread/max(scores) DESC.

### 8.3 UI Career

- `career.py` : rank + LUSR + top matches + encounters.
- `career_logic.py` : `_compute_active_xp_per_day`, `_compute_estimated_xp_curve`, `_compute_hero_projections`, `CAREER_XP_LAUNCH_DATE`.
- `career_charts.py` : `_create_xp_history_chart`.
- `career_progress_circle` : `XP_HERO_TOTAL`, `RANK_MAX`.
- `career_lusr.py` : LUSR/CSR par playlist group.
- `career_top_matches_*.py` : `EncounterCareerMixin.load_top_match_list`.
- `career_encounters_*.py` : `load_top_encountered`, `load_antagonists`.

### 8.4 ⚠️ Écarts Go — Career

`apps/go-api/internal/platform/duckdb/queries_career.go` + `service/career_service.go`.

| Écart | Détail |
|---|---|
| **`mv_player_matches`** | Présent côté shared ; vérifier que la requête Go top-matches l'utilise et applique tous les filtres (`time_played≥180`, `had_bot_teammate=FALSE`, `is_firefight=FALSE`, scores non null, `mode_category != 'BTB'`). |
| **`_BADGE_PRIORITY_EXPR`** | Tri par badge (CONTRE_REMONTADA > REMONTADA > DOMINATION) à reproduire — sinon top matches ne reflète pas l'ordre Python. |
| **`load_lusr_snapshot`** | `ROW_NUMBER() PARTITION BY playlist_group ORDER BY COALESCE(start_time, updated_at) DESC` + `LAG` pour `rating_delta` à porter exactement. |
| **XP hero projections** | `_compute_estimated_xp_curve`, `XP_HERO_TOTAL`, `RANK_MAX`, `CAREER_XP_LAUNCH_DATE` à vérifier côté Go. |
| **Encounters career-wide** | Filtre `xuid LIKE 'bid(%'` (exclusion bots) à valider. |

---

## 9. Synthesis & Squad

### 9.1 Synthesis (`src/ui/pages/synthesis.py`)

- `_filter_by_period` (offsets : 2y=730, 1y=365, 1m=30, 1w=7).
- `_attach_is_with_friends` backfill via `load_is_with_friends(db_path, xuid, match_ids)` (`player_match_enrichment.is_with_friends`).
- Solo vs Squad : filtre `is_with_friends`, KPI win rate (`Outcome.WIN=2`), K/D, K/min, etc.
- Réutilise `_render_heatmap_section`, `_render_map_mode_breakdown`, `_render_top_by_week` de `win_loss.py`.

### 9.2 Squad / Sessions / Records

- `src/ui/pages/sessions.py` : page sessions (`stats.sessions`).
- `src/analysis/squad_records.py` : record par metric par pair_name (`compute_player_record`, `compute_player_pm_records`).
- `src/analysis/_performance_squad.py:compute_squad_performance_score(scores)` : moyenne de scores v2 + bonus :
  - WR > 60 % → +5
  - min(K/D) > 1 → +5
  - std(kills) < 3 → +3

### 9.3 ⚠️ Écarts Go — Synthesis & Squad

`apps/go-api/internal/analysis/squad_*.go` + `service/synthesis_service.go`.

| Écart | Détail |
|---|---|
| `_compute_squad_performance_score` | Vérifier le bonus +5/+5/+3 (WR, K/D, std(kills)). |
| `_filter_by_period` offsets | À vérifier (730/365/30/7 jours). |
| `is_with_friends` solo/squad split | Vérifier que le flag est bien lu depuis `player_match_enrichment.is_with_friends`. |
| Heatmap / map_mode_breakdown / top_by_week | À auditer côté Go. |

---

## 10. Performance Score / Skill Rating / MMR

### 10.1 Performance Score v5-relative (Python)

`src/analysis/_performance_relative.py:compute_relative_performance_score(row, df_history, had_bot_teammate)`.

10 métriques : `kpm, dpm_deaths, apm, kda, accuracy, pspm, dpm_damage, rank_perf,
kills_vs_expected, deaths_vs_expected`. Chaque métrique = percentile du joueur dans
son historique. `dpm_deaths` et `deaths_vs_expected` inversés.

Poids `RELATIVE_WEIGHTS` (`performance_config.py`) :

```
kpm:                 0.17
dpm_deaths:          0.13
apm:                 0.08
kda:                 0.13
accuracy:            0.06
pspm:                0.12
dpm_damage:          0.09
rank_perf:           0.04
kills_vs_expected:   0.10
deaths_vs_expected:  0.08
                ─────
                     1.00
```

`MIN_MATCHES_FOR_RELATIVE=10`. Si insuffisant → `_fallback_kda_percentile`.
`had_bot_teammate=True` ET défaite → `_apply_bot_bonus(score, row)` (+5).
Seuils affichage : excellent ≥ 75, good ≥ 60, average ≥ 45, below ≥ 30, sinon bad.
Durée par défaut : 600 s. `_compute_rank_performance(rank, team_mmr, enemy_mmr,
history)` → `rank_perf`.

### 10.2 Skill Rating LUSR (TrueSkill 2 adapté, Python)

`src/analysis/skill_rating.py` + `_trueskill_math.py` + `_composite.py`. Config `skill_rating_config.py`.

```
INITIAL_MU                = 1500
INITIAL_SIGMA             = 350
MIN_SIGMA                 = 60
MAX_SIGMA                 = 350
BETA                      = 200
TAU                       = 25
DRAW_PROBABILITY          = 0.06
K_ELO                     = 32         # update Elo-style continu
MIN_MATCHES_FOR_RATING    = 10
MIN_RATING                = 200
INACTIVITY_SIGMA_PER_DAY  = 1.0
MAX_INACTIVITY_DAYS       = 14
INACTIVITY_THRESHOLD_DAYS = 1.0
INDIVIDUAL_MU_ALPHA       = 150        # z-score scaling
DEFAULT_OPPONENT_SIGMA    = 150
ACCURACY_HISTORY_SIZE     = 50
MIN_MATCHES_FOR_ACCURACY_DELTA = 5

COMPOSITE_WEIGHTS = {
    kills_vs_expected:    0.31,
    deaths_vs_expected:   0.28,
    win_factor:           0.05,
    damage_efficiency:    0.23,
    accuracy_delta:       0.13,
}

# outcome → win_factor
{2: 1.0, 1: 0.5, 3: 0.0, 4: 0.15}

# damage_efficiency = damage_dealt / (damage_dealt + damage_taken)
# puis sigmoid_ratio vs avg_damage_eff
# estimate_individual_mu(ke, avg_ke, std_ke, base_mu) = base_mu + 150 × (ke-avg)/std
```

Tiers (`SKILL_TIERS`) : Bronze 1000-1200, Silver 1200-1400, Gold 1400-1600,
Platinum 1600-1800, Diamond 1800-2000, Onyx 2000+. 6 sub-tiers chacun (Onyx=1).

Calibration : `skill_rating_calibration.py`, `_calibration_*.py` — calibre les
poids contre la CSR ground-truth.

### 10.3 Stockage

`match_skill_rank` (DB joueur) : `match_id PK, rating_type ∈ {LUSR,CSR},
rating_value, rating_deviation, tier_label, sub_tier, tier, tier_fr,
playlist_group, rating_delta, start_time, updated_at, created_at`. CSR pour les
playlists ranked ; LUSR calculé localement pour les unranked. **Mutuellement
exclusifs par match.**

### 10.4 ⚠️ Écarts Go — Performance Score / Skill Rating

#### Performance Score — `apps/go-api/internal/analysis/performance_score.go`

| Élément | Python | Go | Écart |
|---|---|---|---|
| Métriques | 10 | **13** | Go ajoute `medal_exploit`, `offensive_conversion`, `defensive_resistance` ; Python ne les a pas. |
| Σ poids | 1.00 | 1.01 (renormalisé) | OK conceptuellement, mais les valeurs absolues divergent. |
| `kpm` | 0.17 | **0.14** | -3 pts |
| `dpm_deaths` | 0.13 | **0.10** | -3 pts |
| `apm` | 0.08 | **0.06** | -2 pts |
| `kda` | 0.13 | **0.11** | -2 pts |
| `accuracy` | 0.06 | **0.04** | -2 pts |
| `pspm` | 0.12 | **0.10** | -2 pts |
| `dpm_damage` | 0.09 | **0.06** | -3 pts |
| `rank_perf` | 0.04 | 0.04 | OK |
| `kills_vs_expected` | 0.10 | **0.09** | -1 pt |
| `deaths_vs_expected` | 0.08 | **0.07** | -1 pt |
| `medal_exploit` | n/a | **0.06** | absent côté Python |
| `offensive_conversion` | n/a | **0.09** | absent côté Python |
| `defensive_resistance` | n/a | **0.05** | absent côté Python |
| Bonus bot teammate | +5 si défaite + bot | idem (à confirmer dans `_apply_bot_bonus`) | OK conceptuel |
| `MIN_MATCHES_FOR_RELATIVE` | 10 | 10 | OK |
| Durée fallback | 600 s | 600 | OK |

**Conséquence** : les scores Python et Go ne seront jamais alignés tant que la
liste de métriques et les poids ne sont pas réconciliés. À choisir : aligner Go
sur Python, ou documenter que `apps/go-api` divergera officiellement (et
synchroniser les seuils d'affichage en conséquence).

#### Skill Rating — `apps/go-api/internal/analysis/skill_rating.go`

À auditer ligne à ligne contre `skill_rating_config.py` :

- `INITIAL_MU=1500`, `BETA=200`, `TAU=25`, `K_ELO=32`, `INDIVIDUAL_MU_ALPHA=150` ;
- `COMPOSITE_WEIGHTS` (5 facteurs) ;
- `SKILL_TIERS` (6 paliers + sub_tiers) ;
- gestion inactivité (`INACTIVITY_SIGMA_PER_DAY=1`, max 14 jours) ;
- exclusivité LUSR/CSR par match ;
- accuracy history rolling (size 50, min 5).

Le fichier Go fait 452 L donc l'algo a bien été porté ; les tests
`skill_rating_test.go` doivent valider les constantes.

---

## 11. Killer/Victim & Weapon Kills

### 11.1 `KillerVictimMixin` (`_killer_victim_repo.py`)

- `load_killer_victim_pairs_as_polars(match_id, match_ids, limit)` ← `shared.v_killer_victim_full` (cols : match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms).
- `get_antagonists_summary_polars(top_n=20)` → top nemeses/victims pour xuid courant.
- `has_killer_victim_pairs()`.

### 11.2 `WeaponKillsMixin` (`_weapon_kills_repo.py`)

Lit `shared.v_weapon_kills` (per-kill schéma v5.7+ : `match_id, xuid, time_ms,
weapon_id UBIGINT, delta_ms, confidence, swap_detected, delayed_damage,
reconciled_as, attribution_path, player_index`, avec `effective_weapon_id`).

- `load_weapon_kills_for_match(match_id)` → (xuid, weapon_id, kills) GROUP BY.
- `load_top_weapon_per_player(match_id)` → `{xuid: (weapon_id, kills)}`, ROW_NUMBER PARTITION BY xuid.
- `load_weapon_kills_for_player(xuid, match_ids?)` → (match_id, weapon_id, kills).
- `load_weapon_kills_aggregated(xuid, match_ids?)` → (weapon_id, total_kills).
- `load_total_kills_for_player(xuid, match_ids)` → SUM(`match_participants.kills`).
- `load_grenade_melee_kills(xuid, match_ids?)` → SUM `grenade_kills, melee_kills` depuis `match_participants`.

`WeaponKillsReconcileMixin` (`_weapon_kills_reconcile.py`) : `insert_weapon_kill_rows`,
`insert_weapon_kill_rows_v2` (avec `reconciled_as`, `attribution_path`, `player_index`).
Bits sentinel : `_WEAPON_KILLS_BIT=1<<21`, `_WEAPON_KILLS_NO_FILM_BIT=1<<22`.

### 11.3 Sentinelles & fusion (`src/analysis/_weapon_data.py`)

- IDs sentinelles : `MELEE_WEAPON_ID=1`, `GRENADE_WEAPON_ID=0`, `VEHICLE_WEAPON_ID=2`. `EXCLUDED_WEAPON_IDS = frozenset({0,1,2})`.
- `WEAPON_ID_MAP` (8-byte filmshell → name) : 30+ armes (Bandit Evo, BR75, Cindershot, CQS48 Bulldog, Disruptor, Fuel Rod SPNKr, Gravity Hammer, Heatwave, M41 SPNKr, M392 Bandit, MA40 AR, MA5K Avenger, Mangler, MLRS-2 Hydra, Mk51 Sidekick, Mutilator, Mythic Sandwich, Needler, Plasma Pistol, Pulse Carbine, Ravager, S7 Sniper, Sandwich, Sentinel Beam, Shock Rifle, Shock Rifle Ranked, Skewer, Stalker Rifle, Vestige Carbine, VK78 Commando, Energy Sword + cosmetics, Gravity Hammer cosmetics, Frag/Plasma/Dynamo grenades).
- `WEAPON_FUSION_MAP_ID` : M392 Bandit→Bandit Evo ; Fuel Rod SPNKr→M41 SPNKr ; Shock Rifle Ranked→Shock Rifle ; Vestige Carbine→Pulse Carbine ; sword/hammer cosmetics→base.

### 11.4 ⚠️ Écarts Go — Killer/Victim & Weapons

- `weapon_parser.go`, `weapon_scanner.go`, `weapon_correlation.go`, `weapon_reconciliation.go`, `kill_attribution.go` semblent bien présents côté Go — à valider que `WEAPON_FUSION_MAP_ID` est appliquée dans toutes les requêtes UI (scoreboard détail, weapons matchview, teammates weapons).
- Vérifier l'exclusion `EXCLUDED_WEAPON_IDS={0,1,2}` partout (pas seulement scoreboard).
- `mv_player_matches` doit refléter les fusions.

---

## 12. Spawn / Comeback / Win Streaks / Cadence / First Events / Friends Impact

### 12.1 Spawn detection (`src/analysis/spawn_detection.py`)

Pure function sur les chunks REPLICATION_DATA filmshell.

1. Scan des changements de signature `bytes[9:16]` par `player_index`.
2. Fenêtre glissante `[t, t+2s]` (`_SPAWN_CLUSTER_WINDOW_MS=2000`) où le plus de joueurs sont actifs simultanément.
3. Estimation = début de la fenêtre.
4. Cap API : si estimation > premier kill/death, recule à `api_first_event_ms - 5s`.

Performance : 55 % à ±5 s, 91 % à ±30 s. Constantes : `_AFK_THRESHOLD_MS=10000`,
`_VALID_BASE_TYPES={0x08,0x09,0x0A,0x0B,0x28,0x29}`, `_MIN_FRAME_LEN=14`.

### 12.2 Comeback (`src/analysis/comeback_analysis.py`)

`detect_comeback_flag(events, participants, my_xuid, meta: MatchMeta)` :

- Construit le différentiel kill (enemy_kills − my_kills) à partir de `highlight_events` event_type='kill', xuid → team.
- `_resolve_threshold(meta)` : skip si non-Slayer (variant ne contient pas `slayer`). Threshold = `int(win_score × COMEBACK_DEFICIT_PCT)` (fallback 20). N'applique que si `win_score ≤ COMEBACK_MAX_SLAYER_WIN_SCORE=100`.
- Test max-deficit / max-lead vs threshold + outcome final → REMONTADA / DEBANDADE / CONTRE_REMONTADA.

`src/data/comeback_backfill.py` écrit le flag dans `player_match_enrichment.dominance_flag` (valeurs 3-5).

### 12.3 Win streaks (`src/analysis/win_streaks.py`)

`OUTCOME_WIN=2, OUTCOME_LOSS=3`. `StreakRecord(streak_type, length,
start/end_index, start/end_time)`. `StreakSummary(current_streak,
longest_win/loss, avg_win/loss, total_win/loss_streaks≥2)`.

### 12.4 Match cadence (`src/analysis/match_cadence.py`)

`compute_cadence_buckets(events, xuid_to_team, my_team_id, duration_s,
bucket_s=30)` → `CadenceBucket(t_start_s, t_end_s, my_kills, enemy_kills)`.
Filtre event_type='kill'.

### 12.5 First events (`src/analysis/first_events.py`)

`compute_first_events_rolling(df, window=10)` — rolling mean de `first_kill_s` et
`first_death_s` par xuid, sorted by start_time, `_ROLL_MIN_SAMPLES=3`.

### 12.6 Friends impact (`src/analysis/friends_impact.py`)

`get_all_impact_events(events_df, matches_df, friend_xuids)` → `ImpactEventSets(
first_bloods, clutch_finishers, last_casualties, last_group_kills,
first_group_deaths, silent_heroes, false_brothers, top_killers)`.

Identifiers (`_impact_event_badges.py`) :

- `identify_first_blood` — premier kill du match.
- `identify_clutch_finisher` — dernier kill d'une victoire.
- `identify_last_casualty` (**Boulet**) — dernière mort d'une défaite.
- `identify_first_group_death` — première mort du groupe d'amis dans une défaite.
- `identify_last_group_kill` (**Touriste**) — joueur le plus lent de l'équipe alliée à obtenir son 1er kill.
- `identify_silent_hero_multi(participants_df, matches_df, friend_xuids)` — par victoire : max(assists) ET min(deaths).
- `identify_top_killer_multi`, `identify_false_brother_multi`.

### 12.7 ⚠️ Écarts Go — Comeback / Dominance Flag

`apps/go-api/internal/analysis/comeback.go`.

| Écart | Détail | Impact |
|---|---|---|
| **DOMINATION/HUMILIATION** | Python : déclenchés par la **médaille Steaktacular** (`medal_id=1169390319`). Go : calculés via `sensitivityThresholds.standard = {0.40, 0.35}` sur la courbe de score reconstruite par kill events. | **Sémantique différente** : Go peut donner DOMINATION sur des matchs qui n'ont pas Steaktacular, et inversement. |
| **Filtre Slayer-only pour comeback** | Python ne calcule REMONTADA/DEBACLE/CONTRE_REMONTADA que pour Slayer (`game_variant_name LIKE '%slayer%'` + `win_score ≤ 100`). Go : pas ce garde-fou (les seuils s'appliquent à tous les modes). | Faux positifs sur CTF/Strongholds/Oddball. |
| **Source de la courbe** | Python : `compute_score_curve_from_events(events, ...)` mais aussi via `participants` (kills/deaths). Go : reconstruit depuis `KillEvent.TeamID`. | Si `team_id` n'est pas correctement résolu pour chaque kill (notamment sur les modes objectifs), la courbe est fausse. |
| **`SLAYER_WIN_SCORES`** | `{arena_slayer:50, btb_slayer:100, escalation_slayer:11}` non porté côté Go. | Threshold de comeback non adapté au sous-mode. |

---

## 13. Awards (personal_score_awards)

`stats.personal_score_awards(match_id, xuid, award_name, award_category,
award_count, award_score, created_at)`.

API `_awards_repo.py` :

- `load_personal_score_awards_as_polars(match_id?, match_ids?, category?, limit?)` → DataFrame ordonné par `award_score DESC`.
- `has_personal_score_awards()`.
- `insert_citation(match_id, citation_name_norm, value)` écrit dans `match_citations`.

Utilisé par : Match View Participation radar (§4.3), Citations engine `award`/custom rules (§2.2-2.3).

### 13.1 ⚠️ Écarts Go — Awards

- À vérifier que `personal_score_awards` est bien rempli au sync côté Go (achievements.go/transforms.go).
- Citations consumant les awards (Hijack, Vandalism, Wraith/Mongoose/Warthog Destroyer, Flag em down) impossibles tant que §2.6 n'est pas levé.

---

## 14. Citations Backfill

`src/data/citations_backfill.py` — backfill incrémental post-sync, DB-only :

1. `_get_uncited_match_ids(shared_ro, player_conn, xuid)` : matchs dans `shared.match_participants` pour xuid mais non encore dans `match_citations`.
2. Chargeurs bulk (1 SQL par type pour N matchs) :
   - `_bulk_medals` (`shared.medals_earned`)
   - `_bulk_stats` (`shared.match_participants ⨝ v_match_full`)
   - bulk awards (`personal_score_awards`)
   - bulk pve (`shared_pve.pve_match_stats`)
   - bulk weapon_kills (`shared.v_weapon_kills`)
   - bulk highlight_events (`shared.highlight_events`)
3. `CitationEngine.compute_all_for_match(...)` par match → écriture `match_citations` + marker `_processed=1`.

### 14.1 ⚠️ Écarts Go

Tout ce mécanisme **est absent** côté Go (cf. §2.6). Le flag
`SyncScope.Citations` existe mais n'est consommé nulle part dans la pipeline de
sync.

---

## 15. Tables / vues de référence

| Table / Vue | DB | Rôle |
|---|---|---|
| `match_registry` | shared | Métadata par-match : map_name, playlist_name, game_variant_name, pair_name, mode_category, start_time, duration_seconds, is_firefight, is_ranked. |
| `match_participants` | shared | Participants × matchs ; 31+ colonnes incluant kills/deaths/assists/score/kda, kills_expected, deaths_expected, assists_expected, team_mmr, enemy_mmr, max_killing_spree, headshot_kills, shots_fired, shots_hit, accuracy, melee_kills, grenade_kills, power_weapon_kills, damage_dealt, damage_taken, avg_life_seconds, outcome, team_id, rank, gamertag, xuid. |
| `medals_earned` | shared | (match_id, xuid, medal_name_id, count). |
| `highlight_events` | shared | (match_id, xuid, event_type, time_ms). event_types : `kill`, `death`, `mode`, … |
| `killer_victim_pairs` / `v_killer_victim_full` | shared | (match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms). |
| `weapon_kills` / `v_weapon_kills` | shared | per-kill (`effective_weapon_id`). |
| `xuid_aliases` / `v_gamertag_lookup` | shared | xuid → gamertag (centralisé v6). |
| `mv_player_matches` | shared | Vue matérialisée denorm (xuid, match_id) — top_matches. |
| `v_match_full` | shared | match_registry étendue (translations). |
| `medal_definitions`, `medal_translations` | meta | Labels FR/EN/BCP-47. |
| `citation_mappings` | meta | Définitions des règles (+ tiers, image_path, composite_children). |
| `weapon_labels` | meta | weapon_id (UBIGINT) → labels EN/FR (v5.4). |
| `career_ranks` | meta | Définitions tiers carrière. |
| `career_progression` | stats | Snapshots XP/rank par-joueur. |
| `match_skill_rank` | stats | LUSR/CSR par match (PK = match_id). |
| `player_match_enrichment` | stats | (match_id, performance_score, session_id, session_label, dominance_flag, is_with_friends, friends_xuids, had_bot_teammate). |
| `personal_score_awards` | stats | Awards (objectif participation). |
| `match_citations` | stats | Citations post-calculées. |
| `sessions`, `sync_meta` | stats | Sessions + métadata sync (xuid stocké dans `sync_meta WHERE key='xuid'`). |
| `pve_match_stats` | pve | Stats Firefight par-joueur (waves, boss, kills par type Grunt/Elite/Jackal/Brute/Hunter/Skimmer/Crawler/Soldier/Knight/Warden). |

---

## 16bis. Plan de portage compatible multi-titres

> Cette section formalise comment porter les 3 chantiers prioritaires (scoreboard
> + expander, citation engine, tableau d'impact) **en respectant l'arch
> multi-titres** (`internal/games/`, `TitleDataAdapter`, `CapabilityKey`,
> `canonical.*`).
>
> Règle générale issue de `arch-rules` :
> *« Aucune colonne DuckDB title-specific dans un service (tout via adapter). Brancher
> sur `HasCapability()`, jamais sur `slug == "halo_infinite"`. Dégrader gracieusement
> sur `ErrCapabilityNotSupported`. »*

### 16bis.1 Patterns à respecter

| Couche | Rôle | Règle |
|---|---|---|
| `service/` | Orchestration | Reçoit `games.Resolver`, appelle `data.LoadX(...)`, renvoie `domain.*`. **0 SQL Halo, 0 column name title-specific.** |
| `analysis/` | Algos purs | Opère sur `canonical.*` ou `domain.*` neutres. Aucun import de `platform/duckdb` ou `games/halo_infinite`. |
| `games/halo_infinite/adapter_data.go` | Implémentation Halo | Lit DuckDB, traduit colonnes title-specific → `canonical.*`. Là où vivent les SQL. |
| `games/canonical/` | Contrat stable | Étendre les structs (Participant, HighlightEvent, CitationTotal, …) sans casser les autres adapters. |
| `internal/games/adapter.go` | `TitleDataAdapter` interface | Ajouter les nouvelles méthodes `Load*` ici. |

### 16bis.2 Chantier 1 — Scoreboard + expander

#### Capability requise

`games.CapMatchDetailCore` (existante) — couvre les 21 colonnes du scoreboard. Pour
l'expander, on s'appuie en plus sur :

- `games.CapMatchSkillSnapshot` (existante) — pour Expected vs Actual.
- Une nouvelle `CapMatchScoreboardExtra` à ajouter dans `adapter.go` pour : top weapon
  par joueur, perfect_kills, gamertag resolver v6, melee/grenade/power_weapon kills.
  Si le titre ne supporte pas, le service dégrade et l'expander affiche une partie réduite.

#### Extension du canonical

Ajouter dans `internal/games/canonical/match.go` (extension de `MatchParticipant`,
**non-breaking** pour les autres titres : tous nouveaux champs `*pointer`) :

```go
type MatchParticipant struct {
    // ... champs existants
    KDA                *float64
    MeleeKills         *int
    GrenadeKills       *int
    PowerWeaponKills   *int
    AvgLifeSeconds     *float64
    PerfectKills       *int          // 0 si shared.medals_earned absent
    TopWeapon          *AssetReference
    IsBot              bool          // résolu via "bid(..." côté adapter
}

// Et un nouveau type pour le contexte « match courant » du joueur principal :
type ScoreboardContext struct {
    MatchDetail        // embed
    PlayerPerfScore    *float64
    HadBotTeammate     bool
    SkillRank          *MatchSkillRank   // tier_label, sub_tier, rating_delta
}
```

#### Méthode adapter à ajouter

```go
// internal/games/adapter.go
type TitleDataAdapter interface {
    // ... existant
    LoadMatchScoreboard(ctx context.Context, matchID, viewerXUID string) (*canonical.ScoreboardContext, error)
}
```

Implémentation Halo : `internal/games/halo_infinite/adapter_data_scoreboard.go`
porte le `_SCOREBOARD_SQL` Python en utilisant `v_gamertag_lookup`, le filtre
`_SQL_NOT_GHOST`, l'exclusion sentinelle `(0,1,2)`, la résolution bot `bid(...)`.

#### Service côté `service/match_view_service.go`

```go
func (s *MatchViewService) GetScoreboard(ctx context.Context, matchID, xuid string) (*domain.MatchScoreboard, error) {
    data, err := s.resolver.Data(ctxkeys.TitleSlug(ctx))
    if err != nil { return nil, err }

    raw, err := data.LoadMatchScoreboard(ctx, matchID, xuid)
    if errors.Is(err, games.ErrCapabilityNotSupported) {
        return s.degradedScoreboard(ctx, matchID, xuid)
    }
    if err != nil { return nil, err }

    // Algos purs neutres : MVP/LVP, surlignage extremes, exclusion bots
    extremes := analysis.ComputeScoreboardExtremes(raw.MatchDetail.Participants)
    mvpLvp := analysis.ComputeMVPLVP(raw.MatchDetail.Participants, extremes)

    return assemble(raw, extremes, mvpLvp), nil
}
```

#### Modules / découpage

| Fichier | Rôle | LOC estimées |
|---|---|---|
| `analysis/scoreboard_extremes.go` | `ComputeScoreboardExtremes`, `ComputeMVPLVP`, `IsBot(xuid)` — purs | ~120 |
| `games/halo_infinite/adapter_data_scoreboard.go` | port `_SCOREBOARD_SQL` + assemblage `ScoreboardContext` | ~200 |
| `service/match_view_service.go` | orchestration + dégradation | +60 (existant ~400, surveiller seuil 500) |
| `domain/match_view.go` | enrichir le DTO `MatchScoreboard` | +40 |

#### Tests

- `analysis/scoreboard_extremes_test.go` : table-driven sur les cas Python (équipe < 2, tous à 0, bot exclu, MVP avec 2 best mini).
- `games/halo_infinite/adapter_data_scoreboard_test.go` : DuckDB `:memory:` avec fixture du match `0014603f-...` ; vérifier que les 8 lignes attendues sortent avec les mêmes K/D/A/perfect/top_weapon que la sortie Python observée en BDD.
- `service/match_view_service_test.go` : mock `Resolver` qui renvoie `ErrCapabilityNotSupported` → vérifier la dégradation sans 500.

---

### 16bis.3 Chantier 2 — Citation engine

#### Capability requise

**Nouvelle** `games.CapCitationsEngine` à ajouter (les citations sont une surface
produit de plein droit, avec mappings et custom functions title-specific).

Le titre `synthetic_title_b` (présent dans le repo) déclarera `CapCitationsEngine =
CapUnsupported` → la page Citations affichera un placeholder plutôt qu'une erreur.

#### Extension du canonical

```go
// internal/games/canonical/citations.go (nouveau)
type CitationMapping struct {
    NameNorm           string
    NameDisplay        string
    MappingType        string   // medal | stat | award | weapon_stat | pve_stat | custom | composite
    MedalIDs           []int64
    StatName           string
    AwardName          string
    AwardCategory      string
    CustomFunction     string
    CompositeChildren  []string // JSON parsé
    TierTargets        []int
    Category           string
    Subcategory        string
    Description        string
    ImagePath          string
    Enabled            bool
}

type CitationTotal struct {
    NameNorm string
    Total    int
}

type CitationMatchInputs struct {
    MatchID      string
    Stats        map[string]any   // colonnes match_participants ⨝ v_match_full
    Medals       map[int64]int
    Awards       map[string]int   // award_name → SUM(award_count)
    WeaponKills  map[string]int   // canonical EN name → kills
    PveStats     map[string]any   // colonnes pve_match_stats
    HighlightEvents []HighlightEvent
}
```

#### Méthodes adapter à ajouter

```go
type TitleDataAdapter interface {
    // ... existant
    LoadCitationMappings(ctx context.Context) ([]canonical.CitationMapping, error)
    LoadCitationInputsBulk(ctx context.Context, xuid string, matchIDs []string) (map[string]*canonical.CitationMatchInputs, error)
    LoadCitationTotals(ctx context.Context, xuid string, scope canonical.CitationScope) ([]canonical.CitationTotal, error)
    WriteCitations(ctx context.Context, xuid string, perMatch map[string]map[string]int) error
}
```

Note : `WriteCitations` est sur l'adapter (et non sur le sync) parce que la table
`match_citations` vit dans la DB joueur title-specific. Le sync orchestrera l'appel.

#### Algo pur dans `analysis/`

Le **moteur** est neutre — il ne dépend ni de DuckDB ni de Halo :

```go
// internal/analysis/citations_engine.go (~150 LOC)
// Dispatch par mapping_type → délègue à un Resolver pour les custom functions.
type CitationCustomResolver interface {
    Compute(name string, inputs canonical.CitationMatchInputs) (int, bool)
}

func ComputeCitationsForMatch(
    inputs canonical.CitationMatchInputs,
    mappings []canonical.CitationMapping,
    custom CitationCustomResolver,
) map[string]int { ... }
```

Les **12 custom functions Halo** (Bulldozer, Annexion forcée, Flag em down, etc.)
vivent dans `games/halo_infinite/citations_custom.go` et implémentent
`CitationCustomResolver`. Si le titre n'en a pas, on injecte un resolver no-op.

La **logique composite + tier_targets** reste pure dans `analysis/citations_composite.go`.

#### Modules / découpage

| Fichier | Rôle | LOC |
|---|---|---|
| `games/canonical/citations.go` | structs canoniques | ~80 |
| `games/halo_infinite/adapter_data_citations.go` | bulk loaders SQL (medals_earned, match_participants ⨝ v_match_full, personal_score_awards, pve_match_stats, v_weapon_kills) | ~250 |
| `games/halo_infinite/citations_custom.go` | les 12 fonctions custom Halo | ~250 |
| `analysis/citations_engine.go` | dispatch par mapping_type (algo pur) | ~150 |
| `analysis/citations_composite.go` | logique composite + tier_targets | ~80 |
| `service/citations_service.go` | orchestration lecture | ~120 |
| `sync/citations.go` | backfill incrémental (utilise `WriteCitations`) | ~200 |
| `platform/duckdb/citations_repo.go` | déjà existant — passer en pure lecture | nettoyer |

#### Tests

- `analysis/citations_engine_test.go` : un cas par mapping_type, mock du `CitationCustomResolver`.
- `analysis/citations_composite_test.go` : tier_targets vide / non-vide, enfants partiels.
- `games/halo_infinite/citations_custom_test.go` : un test par custom function (12 tests, table-driven).
- `sync/citations_test.go` : backfill idempotent, ne recompte pas un match déjà marqué `_processed`.
- **Test de parité Python** : sur les 5 137 lignes de `match_citations` déjà calculées par Python en BDD prod, lancer le moteur Go sur le même corpus et vérifier l'égalité ligne à ligne. Référentiel disponible (cf. validation BDD §2026-04-26).

---

### 16bis.4 Chantier 3 — Tableau d'impact (Boulet, Touriste, …)

#### Capability requise

Pas de nouvelle capability nécessaire si on s'appuie sur des `HighlightEvent`
canoniques, déjà alignés avec le concept abstrait « événement temporel d'un match ».

Une `CapabilityGap{Severity:"info"}` peut être ajoutée si un titre n'a pas
d'événements timestampés (ex. titre purement asynchrone) → le service renvoie
un tableau d'impact vide sans erreur.

#### Extension du canonical

```go
// internal/games/canonical/match.go (extension)
type HighlightEvent struct {
    MatchID   string
    XUID      string
    EventType string  // "kill" | "death" | "medal" | "mode" | …
    TimeMS    int64
    TypeHint  string  // optionnel, sous-type natif
}

// Et un type pour les badges :
type ImpactBadge struct {
    EventType   string  // first_blood | clutch_finisher | last_casualty | last_group_kill | first_group_death | top_gun | top_killer | silent_hero | false_brother
    XUID        string
    Gamertag    string
    TimeMS      int64   // -1 si badge basé sur stats
    ExtraLabel  string  // ex: "12 assists · 2 morts"
    IsViewer    bool    // remplace l'is_me Python
}
```

#### Méthodes adapter à ajouter

```go
type TitleDataAdapter interface {
    // ... existant
    LoadHighlightEvents(ctx context.Context, matchID string) ([]canonical.HighlightEvent, error)
    LoadFriendsXUIDs(ctx context.Context, viewerXUID, matchID string) ([]string, error)
}
```

Pour le tableau d'impact escouade (multi-matchs), on s'appuie sur des bulk loaders
similaires à `LoadHighlightEvents` mais pluralisés.

#### Algos purs

```go
// internal/analysis/match_impact.go — réécriture
// Single-match : 9 identifiers (premier sang, finisseur, Boulet, Touriste, première
// victime, Top Gun, Top Killer, Héros silencieux, Faux-frère) avec filtre équipe alliée.
func ComputeSingleMatchImpact(
    events []canonical.HighlightEvent,
    participants []canonical.MatchParticipant,
    viewerXUID string,
    teamXUIDs map[string]bool,
    outcome canonical.Outcome,
) []canonical.ImpactBadge { ... }

// internal/analysis/friends_impact.go — nouveau
// Multi-matchs : agrège les badges par friend_xuid sur N matchs.
func ComputeFriendsImpact(
    eventsByMatch map[string][]canonical.HighlightEvent,
    participantsByMatch map[string][]canonical.MatchParticipant,
    outcomeByMatch map[string]canonical.Outcome,
    friendXUIDs []string,
) canonical.FriendsImpactSets { ... }
```

#### Modules / découpage

| Fichier | Rôle | LOC |
|---|---|---|
| `games/canonical/match.go` | étendre avec `HighlightEvent`, `ImpactBadge`, `FriendsImpactSets` | +60 |
| `games/halo_infinite/adapter_data_events.go` | port lecture `shared.highlight_events` | ~100 |
| `analysis/match_impact.go` | réécriture (les 9 identifiers single-match) | ~250 |
| `analysis/friends_impact.go` | aggregator multi-matchs (8 sets) | ~250 |
| `service/squad_service.go` | orchestration tableau d'impact | +80 |
| `service/match_view_service.go` | orchestration single-match | +40 |

Tous les fichiers <500L. La fonction `ComputeSingleMatchImpact` Python fait 100L —
on découpera en `_findSilentHero`, `_findFalseBrother`, `_findTopKiller`, `_findTopGun`
pour rester sous 80L par fonction.

#### Tests

- `analysis/match_impact_test.go` : un cas par badge (premier sang sur match vide → no-op, Boulet en victoire → no-op, Touriste avec un seul joueur ayant des kills → no-op, Héros silencieux avec top killer égal → exclu, …).
- `analysis/friends_impact_test.go` : table-driven multi-matchs.
- `games/halo_infinite/adapter_data_events_test.go` : DuckDB `:memory:` avec fixture highlight_events.
- **Test de parité Python** : sur 10-20 matchs réels, comparer les badges Python (calculés via `compute_single_match_impact`) vs Go.

---

### 16bis.5 Ordre de portage recommandé (révisé)

| Phase | Chantier | Préreq | Effort | Livrable |
|---|---|---|---|---|
| **A** | Étendre `canonical.MatchParticipant` + ajouter `HighlightEvent`, `ImpactBadge`, `FriendsImpactSets` (pas d'algo, juste le contrat) | — | 0.5 j | PR de typage seul, non-breaking |
| **B** | Chantier 3 — Impact (Boulet/Touriste) | A | 2-3 j | Algos purs + adapter `LoadHighlightEvents` + service squad et match view |
| **C** | Étendre `canonical.MatchParticipant` (KDA, melee, grenade, perfect_kills, top_weapon) + nouvelle cap `CapMatchScoreboardExtra` | A | 0.5 j | PR de typage seul |
| **D** | Chantier 1 — Scoreboard + expander (sans citations top 4) | C | 3-4 j | Scoreboard à parité Python, expander partiel |
| **E** | Ajouter `canonical.CitationMapping`, `CitationMatchInputs`, `CitationTotal` + cap `CapCitationsEngine` | A | 0.5 j | PR typage |
| **F** | Chantier 2 — Citation engine | E | 5-7 j | Moteur complet, 12 custom functions, composite, backfill, parité 5 137 lignes Python |
| **G** | Chantier 1 finition — câbler citations top 4 dans l'expander | D + F | 0.5 j | Expander 100 % parité |

**Total** : 12-16 j-h sur ~3 sprints, livrable phase par phase, **0 régression
multi-titres** parce que tout passe par adapter + capability dégradée gracieusement
pour les titres qui n'ont pas la fonctionnalité.

### 16bis.6 Critères go/no-go par chantier (`delivery-checklist`)

Chaque PR doit passer :

- [ ] `go test ./...` vert (incl. tests de parité Python listés ci-dessus)
- [ ] `go vet ./...` sans warning
- [ ] Aucune fonction > 80 L, aucun fichier > 500 L
- [ ] Aucune colonne DuckDB title-specific dans `service/` ou `analysis/`
- [ ] `slog.*Context` partout, clés `"err"`, `"match_id"`, `"player"`, `"titleSlug"`
- [ ] `synthetic_title_b` adapter n'implémente PAS les nouvelles capabilities → vérifier la dégradation
- [ ] i18n FR + EN pour les nouveaux libellés UI (badges, citations names)
- [ ] Couleurs via `tokenCssVar` / `getSeriesColors` côté frontend
- [ ] Entrée thought_log par chantier
- [ ] Pas de `SyncScope.Citations` mort une fois F livré

---

## 16. Synthèse des écarts critiques Python → Go

Classés par impact perçu sur l'expérience utilisateur :

### 🔴 Bloquants — corpus de données erroné

1. **Citation engine absent côté Go** (§2.6)
   - `mapping_type ∈ {stat, pve_stat, weapon_stat, award, custom, composite}` non câblés.
   - 12 fonctions custom (Bulldozer, Annexion forcée, Flag em down, Hijack, Vandalism, Wraith/Mongoose/Warthog Destroyer, Wins par mode) absentes.
   - Composites jamais débloquées.
   - Page Citations + détails scoreboard vides ou figés.

2. **Badges d'impact incorrects/incomplets** (§5.3)
   - **Touriste** détecté à tort (`MyKills==0` au lieu du « 1er kill le plus tardif »).
   - **Boulet** absent.
   - **Héros silencieux**, **Faux-frère**, **Top Killer**, **Top Gun** absents.
   - Filtre équipe alliée non appliqué.
   - Match View Impact + heatmap teammates Impact dégradés.

3. **DOMINATION/HUMILIATION sémantiquement différents** (§12.7)
   - Python = médaille Steaktacular (`1169390319`).
   - Go = courbe de score + seuil 0.40/0.35.
   - Go peut afficher des badges sur des matchs qui n'en ont pas en Python, et inversement.

4. **Comeback s'applique à tous les modes côté Go** (§12.7)
   - Python le restreint au Slayer (variant `%slayer%`, win_score ≤ 100).
   - Faux positifs CTF/Strongholds/Oddball côté Go.

### 🟠 Majeurs — écarts visibles

5. **Performance Score divergent** (§10.4)
   - Go ajoute `medal_exploit`, `offensive_conversion`, `defensive_resistance` (3 métriques inexistantes en Python).
   - Tous les autres poids sont décalés (-1 à -3 pts chacun).
   - Les scores ne seront jamais alignés — décider : aligner Go sur Python ou documenter la divergence.

6. **Scoreboard Match View — gamertag resolver** (§1.6)
   - Go utilise `xuid_aliases` au lieu de `v_gamertag_lookup` → joueurs récents sans gamertag.
   - `COALESCE(p.gamertag, p.xuid)` manquant.
   - Filtre `_SQL_NOT_GHOST` absent.
   - MVP/LVP non calculé.

7. **Médailles — chaîne BCP-47 absente** (§3.6)
   - Pas de fallback `medal_translations[lang] → en-US → medal_definitions`.
   - Locales custom non honorées.

8. **Fusion d'armes (`WEAPON_FUSION_MAP_ID`)** (§11.4)
   - À auditer côté Go pour scoreboard détail / Weapons / Teammates Weapons.
   - M392 Bandit ≠ Bandit Evo, Shock Rifle Ranked ≠ Shock Rifle, etc.

### 🟡 Mineurs / à valider

9. Top matches career : `_BADGE_PRIORITY_EXPR`, filtre `had_bot_teammate=FALSE`, `MIN_MATCH_DURATION_SECONDS=180` (§8.4).
10. LUSR snapshot : `ROW_NUMBER() PARTITION BY playlist_group ORDER BY ...` + `LAG` pour `rating_delta` (§8.4).
11. Squad performance score : bonus +5/+5/+3 sur WR/min(K/D)/std(kills) (§9.3).
12. Timeseries : fenêtres `compute_rolling_kd_polars(window=5)`, `compute_first_events_rolling(window=10, min_samples=3)` (§7.1).
13. Score normalization SQL (`NORM_MY_TEAM_SCORE_SQL`) absent côté Go (§6.3).

### Ancres de comparaison à utiliser pour la review Go

Quand on touche au code Go, vérifier d'abord ces points :

- **Scoreboard SQL** (§1.1) — golden SQL à reproduire à la lettre, 3 placeholders identiques, sentinelle `(0,1,2)`, constante `1512363953`, ghost filter.
- **Outcome encoding** : WIN=2 / LOSS=3 / TIE=1 / DNF=4.
- **Performance score weights** (§10.1) — décider de la convergence.
- **Skill rating constants** (§10.2) — INITIAL_MU=1500, K_ELO=32, BETA=200, TAU=25, INDIVIDUAL_MU_ALPHA=150, COMPOSITE_WEIGHTS, paliers de tier.
- **Citation `mapping_type`** (§2.2-2.4) — dispatch et 12 fonctions custom.
- **DominanceFlag** sémantique (§3.5) + algorithme comeback (§12.2).
- **Encounter SQL** (§4.9) — `winrate_as_ally` / `winrate_vs_enemy` (outcome 2 vs 3,4).
- **Top matches SQL** (§8.2) — priorité de badge, MIN_MATCH_DURATION_SECONDS=180.
- **Weapon fusion / sentinelles** (§11.3).
- **Ghost player filter** `_SQL_NOT_GHOST` (§1.3).
- **Medal name resolution** : BCP-47 → en-US → definitions (§3.2).

---

*Document généré le 2026-04-26 sur la branche `feat/multi-title-adapters-and-mappings` à partir de `v7/cockpit` (ref `db638c09`) vs `apps/go-api` (HEAD courant).*
