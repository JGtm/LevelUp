# Plan de portage de la page Carrière — Python `v7/cockpit` → Go + Next.js

> Audit comparatif rigoureux et plan de portage par phases.
> Branche source : `v7/cockpit` (Streamlit/Python).
> Branche cible : `feat/multi-title-adapters-and-mappings` (Go + Next.js + recharts).
> Date d'audit : 2026-04-26.
> Co-document : `.ai/PLAN_MATCH_VIEW_GO_PORTAGE.md`, `docs/MIGRATION_GAP_PYTHON_TO_GO.md` § 8.

> **Note d'amendement — 2026-04-27** : ce plan est **partiellement supersedé** par
> [`PLAN_META_FOUNDATIONS_GO.md`](./PLAN_META_FOUNDATIONS_GO.md). Avant toute
> implémentation à partir de ce plan, consulter le méta-plan pour les fondations
> communes : `LoadPlayerMatches(filters)` au lieu de Q9/Q10/Q11 ad hoc, helpers
> `analysis/{breakdown,narrative}` (encounter badges, dominance), stack chart
> **ECharts** (recharts retiré), DTOs alignés sur `ChartSeries[T]`, manifest
> i18n centralisé. Réécriture complète prévue en **Phase 2** du méta-plan.

### Statut des sections de ce plan vis-à-vis du méta-plan

| Section / Phase | Statut | Action |
|---|---|---|
| Phase A — Refondation `CareerPageResponse` | À refactorer | Aligner sur `ChartSeries[T]` (méta-plan § 5.2.1) ; supprimer les `Charts.X interface{}=nil`. |
| Phase B — Q6 (rank+adornment) | À conserver | Lecture `meta.career_ranks` spécifique. |
| Phase B — Q8 LUSR cards (LAG par playlist_group) | À conserver | Spécifique LUSR/CSR. |
| Phase B — Q9 Top matches (filtres durs + tri badge) | À refactorer | `LoadPlayerMatches(filters: BTB, dur≥180, outcomeIn{2,3})` + tri narrative + `analysis/narrative.ResolveDominanceBadge`. |
| Phase B — Q10 Top encountered | À refactorer | `LoadPlayerMatches` + agrégation in-memory ; badges via `analysis/narrative.ComputeEncounterBadges`. |
| Phase B — Q11 Antagonists (Némésis / Souffre-douleurs) | À refactorer | Source partagée via `LoadPlayerMatches` ou repo dédié `v_killer_victim_full` ; badges narrative. |
| Phase C — Logic XP curve / projections / Hero | À conserver | Spécifique Career (constantes `XP_HERO_TOTAL`, `WEEKLY_CHALLENGE_XP`). |
| Phase D — Service refactor | À refactorer | Adopter `LoadPlayerMatches` + `analysis/temporal.Period`. |
| Phase E — Frontend (recharts/Plotly mix) | À refactorer | Wrappers ECharts (`<Gauge>`, `<TimeseriesLine>`, `<Lollipop>`, `<Bullet>`). |
| Phase F — Tests & golden parity | À refactorer | Aligner sur stratégie tests méta-plan § 3 (golden parity Go-only en `cmd/foundations_golden_parity`). |
| Phase G — Cleanup & doc | À conserver | Spécifique Career. |
| Friends file JSON obsolète | À refactorer | Utiliser `friends_xuids` agrégé via `LoadPlayerMatches`. |

---

## 0. Synthèse exécutive

La page Carrière Python v7/cockpit est articulée en **5 sections principales** rendues dans `src/ui/pages/career.py`. Le portage Go/Next.js actuel n'en couvre qu'une partie réduite, avec :

- des **sections architecturales déplacées** vers Synthèse / Palmarès (Top matches → `SynthesisHighlightsSection`, Encounters → `palmares/queries.ts`) — mais la version transplantée est **dégradée** (pas de badges narratifs DOMINATION / REMONTADA / DEBANDADE / HUMILIATION / CONTRE-REMONTADA, pas de tri par badge, pas de filtre BTB, pas d'exclusion `had_bot_teammate`, pas de filtre durée minimale 180 s, pas de cellule date relative ni légende inline, pas de wins/losses par allié vs ennemi) ;
- des **sections jamais portées** : antagonistes (Némésis / Souffre-douleurs), filtres période (`all/2y/1y/1m/1w`), comparaison multi-joueurs sur le graphe XP, courbe XP estimée pré-sync, projections optimistes Hero, table snapshot XP ;
- des **données présentes côté Go mais non exposées au front** : `Charts.RankProgressGauge`, `Charts.HeroProgressGauge`, `Charts.XPHistoryFigure`, `Charts.LUSRRatingFigure` sont **figés à `interface{}` nil** dans `domain.CareerPageCharts` (ligne 113 de `domain/career.go`) — le service ne les construit jamais ;
- des **builders d'agrégat absents en Go** : `_BADGE_PRIORITY_EXPR`, `_TOP_MATCHES_SQL` (Q9 actuel ignore tous les filtres), `_TOP_ENCOUNTERED_SQL`, `_ANTAGONISTS_SQL`, `_compute_estimated_xp_curve`, `_compute_hero_projections`, `compute_encounter_badges`.

**Ratio de couverture global** : sur **24 unités de visualisation/donnée** identifiées dans la version Python, **8** sont présentes côté Go (parfois en payload mais non rendues), **16** sont manquantes ou dégradées. Conformément à la demande de l'utilisateur, **les sections déplacées sont incomplètes** : le portage doit donc également combler les régressions dans `synthesis/` et `palmares/` ou ré-internaliser ces blocs dans la page Carrière.

### Ventilation par section

| Section | Viz Python | Présent Go/web | Manquant / dégradé |
|---|---:|---:|---:|
| Header rang & jauges | 4 (icône adornment, métriques, gauge XP, gauge Hero) | 3 (RankProgressGauge SVG, HeroProgressGauge SVG, badge label) | Icône adornment URL→data-URI, métrique `recorded_at`, métriques `xp_for_next_rank`/`xp_total`, snapshot table XP collapsible |
| Historique XP | 5 traces (réelle, estimée pré-sync, projection Hero, projection optimiste, autres joueurs) + ligne seuil | 0 (figure nil dans `Charts.XPHistoryFigure`) | Tout : 5 traces, ligne seuil, hover translaté, légende horizontale |
| LUSR / CSR | 2 (cartes par playlist_group avec tier image + delta + badge type ; courbe filtrable période × groupe) | 1 KPI textuel "Actuel: …" + courbe figée nil | 6 cartes par playlist_group (ranked/arena/btb/tactical/social/fun) avec image rang, delta arrow, badge LUSR/CSR ; segmented control période ; selectbox playlist_group ; courbe `plot_lusr_timeseries` |
| Top matches | 1 tableau Best/Worst, badges 5 types, filtre BTB optionnel, légende inline | 1 tableau dégradé via `SynthesisHighlightsPreview` (top_by_kills / worst_by_deaths) | Filtres `time_played≥180`, `had_bot_teammate=FALSE`, `is_firefight=FALSE`, `mode_category!='BTB'`, `outcome IN (2,3)`, scores non-null ; tri `_BADGE_PRIORITY_EXPR` ; tri secondaire `time_played ASC` puis score-spread DESC ; 5 badges narratifs (DOMINATION, HUMILIATION, REMONTADA, DEBANDADE, CONTRE_REMONTADA) ; durée mm:ss ; score `my—enemy` ; ratio K/D coloré ; popover Légende |
| Encounters | 3 sous-blocs (Top encountered avec badges allié/ennemi/coriace, Némésis, Souffre-douleurs) + segmented control période + exclusion friends | 1 tableau "Rencontres fréquentes" via `useCareerEncounters` + tri `AsTeammate>=AsEnemy` côté service Go | Top encountered avec colonnes ally/enemy split, winrate_as_ally, winrate_vs_enemy, kills_dealt, deaths_suffered, last_seen relatif ; Némésis (vs `times_killed_by`) ; Souffre-douleurs (vs `times_killed`) ; segmented period (2y/1y/1m/1w) ; exclusion `friends_xuids` ; badges `compute_encounter_badges` (Allié+, Coriace, Tough nut, Ordinal) |

---

## 1. Cartographie source — `v7/cockpit`

### 1.1 Fichiers Python (12 fichiers, ~2 050 lignes)

```
src/ui/pages/career.py                      355 L  entry, fragment_if_available
  +- career_data.py                         142 L  loaders DuckDBRepository + autres profils
  +- career_logic.py                        231 L  estimations, projections, constantes
  +- career_charts.py                       331 L  _create_xp_history_chart (5 types de traces)
  +- career_lusr.py                         206 L  cartes LUSR/CSR + courbe filtrable
  +- career_top_matches_data.py              63 L  proxy vers EncounterCareerMixin
  +- career_top_matches_render.py           303 L  tableau best/worst + 5 badges + légende
  +- career_encounters_data.py               57 L  proxy load_top_encountered/_nemeses/_victims
  +- career_encounters_html.py              179 L  tables HTML encounters & antagonists
  +- career_encounters_render.py             86 L  segmented period + 3 sous-tables

src/data/repositories/_career_repo.py      ~250 L  load_career_data, load_career_history,
                                                  load_pre_sync_match_dates, load_lusr_*
src/data/repositories/_career_encounters_repo.py
                                          ~360 L  EncounterCareerMixin :
                                                  - load_top_encountered (_TOP_ENCOUNTERED_SQL)
                                                  - load_antagonists (_ANTAGONISTS_SQL)
                                                  - load_top_match_list (_TOP_MATCHES_SQL)
src/ui/components/career_progress_circle.py ~150 L XP_HERO_TOTAL=9_319_350, RANK_MAX=272,
                                                   create_career_progress_gauge,
                                                   create_hero_progress_gauge,
                                                   compute_hero_progress
src/ui/career_ranks.py                       —    get_rank_info (metadata.duckdb),
                                                   get_rank_icon_path,
                                                   format_career_rank_label_fr
src/ui/pages/match_view_encounters.py        —    badge_html, kd_cell_html,
                                                   ordinal_badge_html, role_cell_html,
                                                   wr_cell_html, build_badge_legend_html
src/ui/pages/match_view_encounters_logic.py  —    EncounterStats dataclass,
                                                   compute_encounter_badges,
                                                   _relative_date, ordinal,
                                                   _load_friends_from_json
src/visualization/timeseries_combat.py       —    plot_lusr_timeseries
```

### 1.2 Pipeline de données

- **DB centrales** : `shared_matches_v2.duckdb` (`match_participants`, `match_registry`, `mv_player_matches`, `v_killer_victim_full`, `v_gamertag_lookup`)
- **DB par joueur** : `data/players/<gt>/stats.duckdb` (`career_progression`, `match_skill_rank`, `player_match_enrichment`)
- **Référentiels** : `metadata.duckdb` (`career_ranks` → `get_rank_info` retourne `full_label_fr`, `tier`, `title`, `grade`, `image_path`, `adornment_path`)
- **Profils** : `db_profiles.json` (lecture des historiques XP des autres joueurs locaux)
- **Friends** : fichier JSON par joueur lu par `_load_friends_from_json(xuid)`

### 1.3 Algorithmes (logic)

| Fonction | Constantes | Sortie |
|---|---|---|
| `_compute_estimated_xp_curve(history, pre_sync_dates)` | `CAREER_XP_LAUNCH_DATE = 2023-06-20` | Liste `(date, xp_estimé)` ; XP du 1er snapshot réparti uniformément sur les matchs `>= 2023-06-20`, point de raccord inclus. |
| `_compute_active_xp_per_day(history)` | `INACTIVITY_GAP_DAYS = 14` | XP/jour actif. Gaps `> 14j` comptés comme `7j` (indulgence). |
| `_compute_fallback_xp_per_day(xp_total, first_date)` | — | XP / jours depuis `first_date` (debloque projections quand peu de snapshots). |
| `_compute_hero_projections(xp_total, last_date, xp_per_active_day)` | `XP_HERO_TOTAL = 9_319_350`, `WEEKLY_CHALLENGE_XP = 950`, `DAILY_CHALLENGE_XP = 500`, `XP_BOOST_MULTIPLIER = 2.0`, cap à 10 ans, points hebdomadaires | `(normal_curve, optimistic_curve)` : normale = `xp_per_active_day` ; optimiste = `(xp_per_active_day + WEEKLY_CHALLENGE_XP/7 + DAILY_CHALLENGE_XP) * 2.0`. |
| `compute_hero_progress(xp_total, rank, is_max_rank)` | `XP_HERO_TOTAL`, `RANK_MAX = 272` | `{percentage, xp_remaining, xp_total, xp_hero_total}`. |

### 1.4 SQL critiques

#### 1.4.1 `_TOP_MATCHES_SQL` (top 10 best/worst)

Source : `_career_encounters_repo.py:165-203`. Filtres durs :

```sql
FROM shared.mv_player_matches mv
LEFT JOIN player_match_enrichment pme ON pme.match_id = mv.match_id
WHERE mv.xuid = ?
  AND mv.outcome IN (2, 3)                    -- WIN ou LOSS uniquement
  AND COALESCE(mv.time_played_seconds, 0) >= 180
  AND COALESCE(pme.had_bot_teammate, FALSE) = FALSE
  AND COALESCE(mv.is_firefight, FALSE) = FALSE
  AND mv.my_team_score IS NOT NULL
  AND mv.enemy_team_score IS NOT NULL
  {btb_filter}                                 -- AND mv.match_id NOT IN (SELECT match_id FROM shared.match_registry WHERE mode_category='BTB')
ORDER BY
  {badge_priority} DESC,                       -- voir 1.4.2
  time_played_seconds ASC,                     -- match plus court = plus dominant
  ABS(my-enemy)/GREATEST(my,enemy) DESC        -- score-spread relatif
LIMIT 10
```

#### 1.4.2 `_BADGE_PRIORITY_EXPR`

```python
True  (best):  CASE dominance_flag
                 WHEN 5 (CONTRE_REMONTADA) THEN 3
                 WHEN 3 (REMONTADA)        THEN 2
                 WHEN 1 (DOMINATION)       THEN 1
                 ELSE 0 END
False (worst): CASE dominance_flag
                 WHEN 4 (DEBANDADE)        THEN 2
                 WHEN 2 (HUMILIATION)      THEN 1
                 ELSE 0 END
```

`dominance_flag` est calculé en amont par le sync (cf. `src/analysis/_medal_verdicts.py` `DominanceFlag`) et stocké dans `player_match_enrichment.dominance_flag`. Les 5 badges narratifs :

| Flag | Nom | Critère métier |
|---|---|---|
| 1 | DOMINATION | Victoire écrasante (gros écart de score, peu de morts subies) |
| 2 | HUMILIATION | Défaite écrasante (faible score subi, grand écart) |
| 3 | REMONTADA | Victoire après être mené (score live inversé) |
| 4 | DEBANDADE | Défaite après avoir mené (score live perdu) |
| 5 | CONTRE_REMONTADA | Victoire après remontée adverse partielle (rare) |

Affichage : `1, 3, 5` dans l'onglet Best ; `2, 4` dans Worst.

#### 1.4.3 `_TOP_ENCOUNTERED_SQL` (Top encountered career-wide)

Source : `_career_encounters_repo.py:21-90`. Filtres :

```sql
WITH my_matches AS (
  SELECT mp.match_id, mp.team_id, mp.outcome
  FROM shared.match_participants mp
  JOIN shared.match_registry r2 ON r2.match_id = mp.match_id
  WHERE mp.xuid = ?
    AND (? IS NULL OR r2.start_time >= ?)        -- filtre période
),
encounters AS (
  SELECT p.xuid,
    MAX(COALESCE(vg.gamertag, p.gamertag)) AS gamertag,
    COUNT(*)                               AS total_encounters,
    SUM(CASE WHEN p.team_id = m.team_id  THEN 1 ELSE 0 END) AS ally_count,
    SUM(CASE WHEN p.team_id != m.team_id THEN 1 ELSE 0 END) AS enemy_count,
    AVG(CASE WHEN p.team_id = m.team_id AND m.outcome = 2 THEN 1.0
             WHEN p.team_id = m.team_id AND m.outcome IN (3, 4) THEN 0.0
             ELSE NULL END)                AS winrate_as_ally,
    AVG(CASE WHEN p.team_id != m.team_id AND m.outcome = 2 THEN 1.0
             WHEN p.team_id != m.team_id AND m.outcome IN (3, 4) THEN 0.0
             ELSE NULL END)                AS winrate_vs_enemy,
    MAX(r.start_time)                       AS last_seen
  FROM shared.match_participants p
  INNER JOIN my_matches m  ON m.match_id = p.match_id
  LEFT JOIN  shared.v_gamertag_lookup vg ON vg.xuid = p.xuid
  LEFT JOIN  shared.match_registry r ON r.match_id = p.match_id
  WHERE p.xuid != ?
    AND <ghost_filter>                          -- src/data/repositories/_roster_loader.py:_SQL_NOT_GHOST
    AND NOT LOWER(CAST(p.xuid AS VARCHAR)) LIKE 'bid(%'   -- exclusion bots
  GROUP BY p.xuid
),
kvp_agg AS (
  SELECT CASE WHEN k.killer_xuid = ? THEN k.victim_xuid ELSE k.killer_xuid END AS opp,
         SUM(CASE WHEN k.killer_xuid = ? THEN k.kill_count ELSE 0 END) AS kills_dealt,
         SUM(CASE WHEN k.victim_xuid = ? THEN k.kill_count ELSE 0 END) AS deaths_suffered
  FROM shared.v_killer_victim_full k
  WHERE k.killer_xuid = ? OR k.victim_xuid = ?
  GROUP BY 1
)
SELECT e.xuid, e.gamertag, e.total_encounters, e.ally_count, e.enemy_count,
       e.winrate_as_ally, e.winrate_vs_enemy,
       COALESCE(kvp.kills_dealt, 0)     AS kills_dealt,
       COALESCE(kvp.deaths_suffered, 0) AS deaths_suffered,
       e.last_seen
FROM encounters e LEFT JOIN kvp_agg kvp ON kvp.opp = e.xuid
ORDER BY e.total_encounters DESC
LIMIT ?
```

À comparer avec `Q10Encounters` (Go) qui :

- N'utilise **pas** `shared.` (lit `match_participants` — base joueur, alors qu'il faudrait `shared.match_participants`),
- N'agrège **pas** `winrate_as_ally`, `winrate_vs_enemy`, `kills_dealt`, `deaths_suffered`, `last_seen`,
- N'exclut **pas** les bots (`xuid LIKE 'bid(%'`),
- N'applique **pas** de filtre `since`,
- Limite à `LIMIT 50` au lieu de paramétrer.

#### 1.4.4 `_ANTAGONISTS_SQL` (Némésis & Souffre-douleurs)

Source : `_career_encounters_repo.py:92-130`. Le mode est sélectionné par `order_col` :

- `nemesis` → `ORDER BY times_killed_by DESC`
- `victim`  → `ORDER BY times_killed DESC`

Retourne `opponent_xuid, opponent_gamertag, times_killed, times_killed_by, matches_against, net_kills`.

Source de vérité : `shared.v_killer_victim_full + shared.v_gamertag_lookup` ; filtre `since` injecté via `period_matches`.

### 1.5 Rendu UI (référence)

#### 1.5.1 Header rang (`_render_rank_header`)

3 colonnes `[1, 2, 2]` :

1. **Icône adornment** (140 px, data-URI base64) ; fallback `get_rank_icon_path(rank)`. Caption `"Données du JJ/MM/AAAA HH:MM"` (TZ locale).
2. **Métriques** : `subheader(rank_label_fr)` + `metric("Rang", "{n} / 272")`, `metric("XP totale", xp_total)`, soit (si max) `metric("Rang max", "Rang max")`, soit `metric("XP actuel", current_xp) + metric("XP rang suivant", xp_for_next)`.
3. **Gauge Plotly** `create_career_progress_gauge` (270 px, 4 paliers de couleur 25/50/75 %, threshold blanc).

#### 1.5.2 Section Hero (`_render_hero_section`)

`subheader("Progression vers Héros")` + 4 métriques `[XP earned, XP remaining, XP required = 9_319_350, Rank n/272]` + gauge `create_hero_progress_gauge`.

#### 1.5.3 Historique XP (`_create_xp_history_chart`)

Figure Plotly à 5 catégories de traces :

1. **XP réel courant** — `mode='lines+markers'`, couleur `THEME_COLORS.accent`, hover localisé (`career_rank_hover`).
2. **XP estimé pré-sync** — `dash='dot'`, couleur `#CE93D8`, depuis `_compute_estimated_xp_curve`.
3. **Autres joueurs** — itération sur `db_profiles.json` ; couleurs distinctes `["#EF5350","#29B6F6","#FFCA28","#26C6DA","#FF7043","#AB47BC"]`, `visible='legendonly'`. Pour chaque joueur : trace réelle, trace estimée, trace projection Hero, trace projection optimiste.
4. **Projection Hero** — `dash='dash'`, couleur `#FFA726`, `visible='legendonly'`.
5. **Projection optimiste** — `dash='dashdot'`, couleur `#66BB6A`, `visible='legendonly'`.
6. **Ligne seuil Hero** — `add_hline(XP_HERO_TOTAL, dash='dot')`, annotation `"career_hero_threshold"` haut-gauche.

Layout : `height=400`, légende horizontale en bas, `apply_halo_plot_style`.

#### 1.5.4 Snapshots XP (table)

`expander("Historique des snapshots", expanded=False)` → 10 dernières lignes de `history` formatées :
`{date} | Rang {n}: {label} | XP: {xp_total}`.

#### 1.5.5 Section LUSR (`_render_lusr_section`)

- **Cartes** : grille 3 par ligne, ordonnée par `_PG_ORDER = ["ranked","arena","btb","tactical","social","fun"]`. Chaque carte affiche : icône playlist (`🏆 ⚔️ 💥 🎯 🎮 🎉`), label playlist FR, image rang (`get_rank_image_path(rating_value)`, 90×90), tier_label, badge type (`LUSR` cyan / `CSR` doré), valeur arrondie, delta avec arrow `▲ ▼ =` colorée.
- **Courbe** : `segmented_control` période `[all,2y,1y,1m,1w]` + `selectbox` playlist_group (`Tous` ou un subset présent). Granularité `"1d" / "1w"` selon période. Trace via `plot_lusr_timeseries(df, playlist_group)`.

#### 1.5.6 Top matches (`render_top_matches_section`)

- En-tête : `"Top 10 meilleurs / pires matchs"` ; suffixe `" — BTB exclu"` si `settings.career_top_exclude_btb`.
- 2 onglets `Tabs` : `Meilleurs` / `Pires`.
- Tableau HTML par onglet : colonnes `[match_id (lien Explorer), date FR, mode (pair_name_fr), map (cell HTML carte), score (my—enemy), KDA (k/d/a), K/D ratio coloré, durée mm:ss, badge]`.
- Badges affichés : `1, 3, 5` côté Best ; `2, 4` côté Worst ; couleurs précises (`#2e7d32 / #6a1b9a / #1565c0 / #bf360c / #00695c`).
- Popover `"ℹ️ Légende"` listant tous les badges actifs avec le label long traduit.

#### 1.5.7 Encounters (`render_encounters_section`)

- `segmented_control` période.
- Exclusion friends via `_load_friends_from_json(xuid)` injecté en `exclude_xuids`.
- Section 1 : Top encountered (10 lignes) — colonnes `gamertag + ordinal_badge + badges, role(side), total(A:n|E:n), winrate_ally, winrate_vs_enemy, KD(kills_dealt - deaths_suffered), last_seen relatif`.
- Section 2 : 2 colonnes côte à côte — Némésis (`load_top_nemeses`) + Souffre-douleurs (`load_top_victims`).
- Popover Légende `build_badge_legend_html`.

---

## 2. Cartographie cible — Go + Next.js

### 2.1 Frontend (8 fichiers, ~860 L)

```
apps/web/src/features/career/
  +- CareerHubPage.tsx              60 L  hub Tabs Progression / Citations
  +- CareerProgressionTab.tsx      131 L  rang + gauges + LUSR text + leaderboard
  +- CareerCitationsTab.tsx        198 L  citations
  +- CareerSummaryCard.tsx          70 L  carte rang + gauges arc SVG
  +- CareerChartsSection.tsx        60 L  4 PlotCards reliés à `data.charts.*`
  +- CareerTopMatchesTable.tsx     136 L  tableau best/worst (utilisé par CareerPage legacy)
  +- CareerEncountersSection.tsx   113 L  table flat sans antagonistes
  +- queries.ts                     35 L  3 hooks (page / top-matches / encounters)
  +- CareerPage.tsx                188 L  legacy (route différente, encore utilisé ?)
```

Routing TanStack : `/players/$playerSlug/career?tab=progression|citations` (cf. `CareerHubPage`). L'ancien `CareerPage.tsx` n'est plus monté dans le hub mais est toujours référencé par `apps/web/src/router.tsx` (à vérifier en B0).

> ⚠️ Sprint 55 a déplacé top_matches + encounters hors de la page Carrière vers `synthesis/SynthesisHighlightsSection.tsx` et `palmares/PalmaresRelationsPage.tsx`. La demande explicite de l'utilisateur est que ce déplacement n'a pas été conservé fidèlement et que le contenu manque côté Go ; la suite du plan suppose un **retour intégré dans la page Carrière** (option A), avec une mention parallèle (option B) pour conserver les blocs Synthèse/Palmarès tout en complétant leur contenu.

### 2.2 Backend Go (5 fichiers, ~1 200 L)

```
apps/go-api/internal/api/handlers/career.go            77 L  3 routes
apps/go-api/internal/service/career_service.go        542 L  builders + projections
apps/go-api/internal/platform/duckdb/career_repo.go   140 L  4 méthodes
apps/go-api/internal/platform/duckdb/queries_career.go 264 L  Q6/Q7/Q8/Q9
apps/go-api/internal/domain/career.go                 178 L  DTOs
```

Routes :

```
GET /players/{slug}/pages/career             → CareerPageResponse
GET /players/{slug}/pages/career/top-matches → CareerTopMatchesResponse
GET /players/{slug}/pages/career/encounters  → CareerEncountersResponse
```

### 2.3 Décalages structurels

| Plan Python | État Go | Action requise |
|---|---|---|
| `domain.CareerPageCharts.*` champs Plotly figures (data + layout) | `*interface{}` toujours nil | Le front Go n'utilise pas Plotly côté Carrière (`CareerSummaryCard` SVG arc + texte) → décision : (a) rendre `Charts` côté front en recharts/Plotly natif ou (b) construire un payload `figure` côté Go. Préférence : **front recharts + payload structuré** (cf. § 4) car aucune autre page de la migration Go ne pousse de `figure` Plotly serveur. |
| `xp_history` retourné en `[]XPHistoryPoint` | Présent | OK, juste non consommé en front |
| Estimated XP curve, hero projections | Calcul présent dans `service` (estimated_hero_date) mais **pas** la courbe estimée pré-sync ni la courbe optimiste | Ajouter `estimated_curve`, `hero_projection_curve`, `optimistic_projection_curve` au DTO ou les calculer côté front (préférence : **côté Go**, déterministe et testable) |
| Top matches : badge dominance, filtres durs | Q9 retourne `LIMIT 20 ORDER BY performance_score DESC`, aucun filtre BTB / had_bot / is_firefight / time_played | Réécrire Q9 → `Q9TopMatches` complet (cf. § 4.4) |
| Encounters Top + Némésis + Souffre-douleurs | Un seul `Q10Encounters` agrégeant teammates/enemies sans split allié vs ennemi, sans antagonists | Ajouter `Q10TopEncountered`, `Q11Antagonists` ; mettre à jour le DTO |
| LUSR snapshot par playlist_group avec rating_delta (LAG) | `buildLUSRSummary` ne retourne **qu'un** rating courant (max-rating) | Ajouter Q sur `match_skill_rank` partitionnant par `playlist_group` avec `LAG(rating_value)` |
| Filtre période `all/2y/1y/1m/1w` | Pas exposé | Ajouter `?period=` aux endpoints encounters & lusr |
| Friends exclusion | Pas géré | Ajouter chargement `friends_xuids` (`player_match_enrichment.friends_xuids` ou file JSON) |

---

## 3. Inventaire détaillé des écarts

Référence pour le ticketing. Chaque ligne = une unité atomique de portage.

### 3.1 Header rang & gauges

| # | Item | Source Python | État Go | État web | Sévérité |
|---|---|---|---|---|---|
| H1 | Icône adornment ou fallback rank icon (data-URI base64, 140 px) | `_render_rank_header:111-152` | DTO ne porte pas `adornment_path` ni icône | `<RankProgressGauge>` arc SVG sans icône joueur | Moyenne |
| H2 | Métrique `recorded_at` localisé `JJ/MM/AAAA HH:MM` | `_render_rank_header:154-159` | `Summary.RecordedAt` exposé | non rendu en `CareerSummaryCard` | Faible |
| H3 | Métriques 2×2 `[Rang n/272, XP totale, XP actuel, XP rang suivant]` | `_render_rank_header:165-176` | DTO complet | `CareerSummaryCard` n'affiche que `rank_label`, `progress_pct`, `is_max_rank`, `xp_total/xp_for_next_rank` (sous-titre) | Faible |
| H4 | Gauge Plotly `create_career_progress_gauge` avec 4 steps + threshold | `career_progress_circle.py:34-83` | DTO `Charts.RankProgressGauge` nil | `RankProgressGauge` SVG (équivalent fonctionnel) | OK fonctionnel |
| H5 | Section "Progression vers Héros" : 4 métriques `[xp_earned, xp_remaining, XP required=9_319_350, Rank n/RANK_MAX=272]` | `_render_hero_section` | DTO complet | `CareerSummaryCard` n'affiche que `current_rank` + `xp_remaining` (sous-titre) | Faible |
| H6 | Gauge Hero Plotly | `create_hero_progress_gauge` | DTO `Charts.HeroProgressGauge` nil | `RankProgressGauge` SVG (équivalent) | OK fonctionnel |

### 3.2 Historique XP (chart)

| # | Item | Python | Go | Sévérité |
|---|---|---|---|---|
| X1 | Trace XP réelle (lines+markers, hover localisé) | `_add_main_xp_trace` | DTO contient `XPHistory` brut | front ne le consomme pas | **Critique** |
| X2 | Trace XP estimée pré-sync (dash dot violet) | `_compute_estimated_xp_curve` + `_add_estimated_xp_trace` | Aucun calcul, aucun DTO | absent | **Critique** |
| X3 | Trace projection Hero (dash orange) | `_compute_hero_projections.normal` + `_add_projection_traces` | `EstimatedHeroDate` (date seule, pas la courbe) | absent | **Critique** |
| X4 | Trace projection optimiste (dashdot vert) | `_compute_hero_projections.optimistic` | absent | absent | **Critique** |
| X5 | Traces autres joueurs (réelle + estimée + 2 projections, 6 couleurs, legendonly) | `_load_other_players_histories` + `_add_single_other_player_*` | absent (pas de notion de profils multiples côté Go) | absent | Moyenne |
| X6 | Ligne seuil Hero `add_hline(XP_HERO_TOTAL)` | `_add_projection_traces` | absent | absent | Moyenne |
| X7 | Hover translaté `career_rank_hover`, `career_xp_estimated_hover`, etc. | `career_charts.py` | absent | absent | Faible |
| X8 | Layout : légende horizontale en bas, height=400, palette `THEME_COLORS.bg_plot` | `_create_xp_history_chart` | absent | absent | Faible |

### 3.3 Snapshots XP

| # | Item | Sévérité |
|---|---|---|
| S1 | Expander `"career_rank_history_title"` listant 10 derniers snapshots `{date \| Rang {n}: {label} \| XP: {xp_total}` | Faible (info redondante avec graphe) |

### 3.4 Section LUSR

| # | Item | Python | Go | Sévérité |
|---|---|---|---|---|
| L1 | Cartes par playlist_group (6 cartes : ranked, arena, btb, tactical, social, fun) | `_render_lusr_rank_cards` | `LUSR.Current*` retourne **un seul** rating (max), pas de split par group | **Critique** |
| L2 | `rating_delta` via `LAG` par playlist_group | `load_lusr_snapshot` | `buildLUSRSummary` calcule un trend global `last - prev` | **Critique** |
| L3 | Image rang (`get_rank_image_path(rating_value)`) | `skill_rating_config.get_rank_image_path` | absent | Moyenne (déjà câblé pour Halo Infinite côté assets) |
| L4 | Badge type `LUSR` cyan / `CSR` doré | `_render_lusr_rank_cards` | absent | Faible |
| L5 | Courbe `plot_lusr_timeseries(df, playlist_group)` filtrable période + group | `_render_lusr_rating_chart` | DTO `Charts.LUSRRatingFigure` nil + pas de filtre serveur | **Critique** |
| L6 | `segmented_control` période `[all,2y,1y,1m,1w]` | `_render_lusr_rating_chart` | absent | **Critique** |
| L7 | `selectbox` playlist_group | `_render_lusr_rating_chart` | absent | **Critique** |
| L8 | Granularité (`1d` / `1w`) selon période | `_PERIOD_GRANULARITY` | absent | Moyenne |

### 3.5 Top matches (Best/Worst)

| # | Item | Python | Go | Sévérité |
|---|---|---|---|---|
| T1 | Source `shared.mv_player_matches LEFT JOIN player_match_enrichment` | `_TOP_MATCHES_SQL` | Q9 utilise `player_match_enrichment LEFT JOIN shared.match_registry` (mauvaise base) | **Critique** |
| T2 | Filtres `outcome IN (2,3)`, `time_played≥180`, `had_bot_teammate=FALSE`, `is_firefight=FALSE`, scores non-null | `_TOP_MATCHES_SQL` | aucun filtre | **Critique** |
| T3 | Filtre BTB optionnel (`mode_category != 'BTB'`) avec setting persisté `career_top_exclude_btb` | `_TOP_MATCHES_SQL` + `AppSettings.career_top_exclude_btb` | absent | **Critique** |
| T4 | Tri par `_BADGE_PRIORITY_EXPR` (best : 5,3,1 / worst : 4,2) | `_TOP_MATCHES_SQL` | tri `performance_score DESC` (différent) | **Critique** |
| T5 | Tri secondaire `time_played ASC` puis `score-spread relatif DESC` | idem | absent | **Critique** |
| T6 | Split best/worst par `outcome=WIN/LOSS` (au lieu d'un split sur N=20 matchs comme actuellement Go) | idem | `splitTopRows` coupe à mi-liste — incorrect | **Critique** |
| T7 | DTO `dominance_flag` exposé | `_TOP_MATCHES_SQL` | absent (TopMatchDTO ne porte pas le flag) | **Critique** |
| T8 | DTO `time_played_seconds, my_team_score, enemy_team_score, kda` | idem | TopMatchDTO porte `kills,deaths,kda,outcome,map_ui,mode_ui,performance_score` | **Critique** |
| T9 | Render badge avec couleurs spécifiques par flag (`#2e7d32`, `#6a1b9a`, `#1565c0`, `#bf360c`, `#00695c`) | `career_top_matches_render.py:_BADGE_CONFIGS` | `MatchBadge` utilise `getMatchNarrativeBadgeMeta` → vérifier ces 5 couleurs | Moyenne (vérifier) |
| T10 | Affichage durée mm:ss | `_format_duration` | `CareerTopMatchesTable` n'affiche pas la durée | Moyenne |
| T11 | Affichage score `my — enemy` | `_format_score` | absent (col Score affiche `score_label` non typé) | Moyenne |
| T12 | Légende inline via popover, listant uniquement les badges présents | `_build_match_badge_legend_html` | absent | Faible |
| T13 | Settings persistés `career_top_exclude_btb` | `AppSettings` | absent (pas de notion de settings côté Go) | Moyenne |

### 3.6 Encounters (Top encountered + Némésis + Souffre-douleurs)

| # | Item | Python | Go | Sévérité |
|---|---|---|---|---|
| E1 | Source `shared.match_participants ⨝ shared.match_registry ⨝ shared.v_gamertag_lookup` | `_TOP_ENCOUNTERED_SQL` | `Q10Encounters` utilise base joueur + `xuid_aliases` | **Critique** |
| E2 | Champs `winrate_as_ally, winrate_vs_enemy, kills_dealt, deaths_suffered, last_seen` | idem | absents (`avg_kda` seul agrégat) | **Critique** |
| E3 | Exclusion bots (`xuid LIKE 'bid(%'`) | idem | absent | **Critique** |
| E4 | Exclusion ghost (`<_SQL_NOT_GHOST>`) | idem | présence à confirmer | Moyenne |
| E5 | Filtre période (`since`) | idem | absent | **Critique** |
| E6 | Exclusion friends_xuids | `_load_friends_from_json` | absent | Moyenne |
| E7 | Sous-section Némésis (`order_col=times_killed_by`) | `_ANTAGONISTS_SQL` mode=`nemesis` | absent | **Critique** |
| E8 | Sous-section Souffre-douleurs (`order_col=times_killed`) | `_ANTAGONISTS_SQL` mode=`victim` | absent | **Critique** |
| E9 | Champ `matches_against` | `_ANTAGONISTS_SQL` | absent | Moyenne |
| E10 | Badges encounter (Allié+, Coriace, Tough nut, Ordinal) | `compute_encounter_badges` + `match_view_encounters` | absent | Moyenne |
| E11 | Date relative (`_relative_date`) | `match_view_encounters_logic` | absent (`toLocaleDateString` simple) | Faible |
| E12 | Légende popover `build_badge_legend_html` | `match_view_encounters` | absent | Faible |
| E13 | DTO encounters split `Teammates / Enemies` côté Go | `loadEncounterRows` Go | OK structurellement mais vide en sémantique (split sur `AsTeammate >= AsEnemy` au lieu d'avoir 3 listes Top/Némésis/Souffre-douleurs) | **Critique** |

---

## 4. Plan de portage par phases

Les phases sont conçues pour pouvoir être livrées indépendamment. Les phases A et B sont des prérequis ; les phases C → F sont parallélisables une fois A & B en place.

### Phase A — Refondation du modèle de données

**Objectif** : aligner `domain.CareerPageResponse` sur la version Python sans casser les consommateurs.

1. Étendre `domain/career.go` :
   - `XPHistoryPoint` : ajouter `Rank int`, `RankLabel *string`, `RankTier *string` (déjà présents partiellement).
   - Ajouter `EstimatedXPPoint{ Date time.Time; XPTotal int }`.
   - Ajouter `HeroProjectionPoint{ Date time.Time; XPTotal int; Optimistic bool }`.
   - `CareerProjections` : ajouter `Curve []HeroProjectionPoint`, `OptimisticCurve []HeroProjectionPoint`, `EstimatedPreSyncCurve []EstimatedXPPoint`.
   - `CareerRankSummary` : ajouter `AdornmentPath *string`, `RankIconPath *string`.
   - `LUSRSummary` : remplacer le scalar par `Cards []LUSRCardDTO` où chaque carte porte `playlist_group, tier_label, rating_value, rating_type ('LUSR'|'CSR'), rating_delta *float64, rank_image_path *string`.
   - `LUSRCheckpointDTO` : ajouter `RatingType string` (la requête doit dériver `'CSR'` quand `playlist_group='ranked'`, sinon `'LUSR'`).
   - Nouveau `TopMatchDTO` exhaustif : `match_id, start_time, map_id, map_ui, mode_ui, playlist_name, outcome, kills, deaths, assists, kda, time_played_seconds, my_team_score, enemy_team_score, dominance_flag, kd_ratio, score_spread`.
   - Nouveau `EncounterRow` (Top encountered) : `xuid, gamertag, total_encounters, ally_count, enemy_count, winrate_as_ally, winrate_vs_enemy, kills_dealt, deaths_suffered, last_seen, badges []EncounterBadge`.
   - Nouveau `AntagonistRow` : `opponent_xuid, opponent_gamertag, times_killed, times_killed_by, matches_against, net_kills`.
   - `CareerEncountersResponse` : remplacer `Teammates/Enemies/Total` par `TopEncountered []EncounterRow`, `Nemeses []AntagonistRow`, `Victims []AntagonistRow`, `Period string`.
2. Couper `Charts CareerPageCharts{}` (champ vidé) — le rendu se fait côté front (recharts SVG).
3. Mettre à jour `port.CareerService` et `port.CareerRepository`.
4. Tests unitaires : adapter `career_service_test.go` et fixtures.

### Phase B — Repo & SQL

1. **Q6/Q7 inchangés** mais ajouter à la projection `adornment_path` (joindre `meta.career_ranks` sur `rank_number` → champ `adornment_path`).
2. **Q8 LUSR** : réécrire pour partitionner par `playlist_group` :

   ```sql
   WITH ranked AS (
     SELECT msr.match_id, msr.rating_value, msr.tier_label, msr.playlist_group,
            COALESCE(r.start_time, msr.updated_at) AS recorded_at,
            CASE WHEN msr.playlist_group='ranked' THEN 'CSR' ELSE 'LUSR' END AS rating_type,
            ROW_NUMBER() OVER (PARTITION BY msr.playlist_group
                               ORDER BY COALESCE(r.start_time, msr.updated_at) DESC) AS rn,
            LAG(msr.rating_value) OVER (PARTITION BY msr.playlist_group
                                        ORDER BY COALESCE(r.start_time, msr.updated_at) ASC) AS prev_rating
     FROM match_skill_rank msr
     LEFT JOIN shared.match_registry r ON msr.match_id = r.match_id
   )
   SELECT match_id, rating_value, tier_label, playlist_group, recorded_at, rating_type,
          rating_value - prev_rating AS rating_delta
   FROM ranked
   WHERE rn = 1
   ```

   Et conserver une **seconde** requête `Q8History(?)` pour la courbe (tous les checkpoints, filtre période + group optionnel via paramètres).
3. **Q9TopMatches** : remplacer par le SQL § 1.4.1, paramétrer `(xuid, win=2, loss=3, win_or_loss=2 or 3, MIN_DURATION=180, target_outcome, exclude_btb_bool)`. Préserver le placeholder `{badge_priority}` et `{btb_filter}` injectés en Go via `text/template` ou `strings.Replace`. Retourner les 10 lignes triées.
4. **Q10TopEncountered** : porter `_TOP_ENCOUNTERED_SQL` (cf. § 1.4.3). Paramètres `(xuid, since_iso, since_iso, xuid, xuid, xuid, xuid, xuid, xuid, limit)` ; clé : utiliser **`shared.match_participants`** et **`shared.v_killer_victim_full`**. Rester sur la même implémentation que pour la match view.
5. **Q11Antagonists** : porter `_ANTAGONISTS_SQL` (mode `nemesis|victim`). Paramètres `(since, since, xuid, xuid, xuid, xuid, xuid, xuid, limit)`.
6. **CareerRepo** : ajouter `GetLUSRCards`, `GetTopMatchesV2`, `GetTopEncountered`, `GetAntagonists` (3 méthodes : `nemeses`, `victims`).
7. **Pré-sync match dates** : ajouter `GetPreSyncMatchDates(ctx, firstSync time.Time)` (`shared.match_participants ⨝ shared.match_registry WHERE start_time < firstSync`). Source : `_load_pre_sync_match_dates`.
8. Tests : `career_repo_test.go` avec fixtures DuckDB (`testdata/`) couvrant chaque SQL.

### Phase C — Logic Go

Porter dans `service/career_logic.go` (nouveau fichier) :

| Python | Go équivalent | Notes |
|---|---|---|
| `CAREER_XP_LAUNCH_DATE = 2023-06-20` | `var careerXPLaunchDate = time.Date(2023, 6, 20, 0, 0, 0, 0, time.UTC)` | Constante |
| `WEEKLY_CHALLENGE_XP=950, DAILY_CHALLENGE_XP=500, XP_BOOST_MULTIPLIER=2.0, INACTIVITY_GAP_DAYS=14` | Constantes typées | `const (...)` |
| `_compute_estimated_xp_curve(history, pre_sync_dates)` | `computeEstimatedXPCurve` | Ports identiques avec `[]EstimatedXPPoint` |
| `_compute_active_xp_per_day` | présent (`computeActiveXPPerDay`) | déjà OK, juste à factoriser |
| `_compute_fallback_xp_per_day` | présent (`computeFallbackXPPerDay`) | déjà OK |
| `_compute_hero_projections` | `computeHeroProjections(xpTotal, lastDate, xpPerDay) (normal, optimistic []HeroProjectionPoint)` | Inclure cap 10 ans + points hebdomadaires |

Exposer ces résultats via `buildProjections` et nouveaux champs DTO Phase A.

Tests : `career_logic_test.go` porte les golden tests Python (`tests/test_career_xp_projection.py`) sous forme de table-driven Go.

### Phase D — Service refactor

1. `GetCareerPage` (existant) :
   - charger `latestRank` + `xpHistory` + `lusrSnapshot (cards) + lusrHistory + preSyncMatchDates + otherPlayersHistories (optionnel)`,
   - construire `summary, hero, projections (avec curve + optimistic + estimated curve), lusr (cards + checkpoints), xp_history`,
   - réponse complète sans `Charts`.
2. `GetTopMatches(ctx, exclude_btb bool)` :
   - lire la requête Phase B Q9 deux fois (best & worst),
   - retourner `BestMatches` + `WorstMatches` (10 chacun).
   - exposer `?exclude_btb=true|false` au handler ; valeur par défaut `false`.
3. `GetEncounters(ctx, period string)` :
   - parser `period` ∈ `{all,2y,1y,1m,1w}` → `since *time.Time`,
   - charger `friends_xuids` (via `player_match_enrichment.friends_xuids` agrégé ou file JSON `data/players/<gt>/friends.json` ; choisir une source unique — la file JSON du repo Python est obsolète, préférer `friends_xuids` DB),
   - 3 appels parallèles : `GetTopEncountered(since, exclude_friends)`, `GetAntagonists(nemesis, since, exclude_friends)`, `GetAntagonists(victim, since, exclude_friends)`.
4. Construction des badges encounters (`computeEncounterBadges`) : porter `compute_encounter_badges` côté Go (logique pure, déterministe). Tests unitaires.

### Phase E — Frontend (web)

1. **Décisions architecturales** :
   - **Pas de Plotly côté serveur** : la page Carrière utilise recharts/SVG comme le reste du portage. Conséquence : `CareerChartsSection.tsx` est supprimé, remplacé par des composants spécialisés.
   - **Charts via recharts** : `CareerXPHistoryChart.tsx` (LineChart multi-trace) ; `CareerLUSRTimeseriesChart.tsx` (LineChart par playlist_group).
   - **Réintégration top_matches & encounters dans la page Carrière** : conformément à la demande utilisateur. Les déplacements vers `synthesis/` et `palmares/` sont conservés mais pointent désormais vers les mêmes hooks (`useCareerTopMatches`, `useCareerEncounters`) avec les nouveaux DTO complets, en preview.
2. **Refactor `CareerProgressionTab.tsx`** :
   - Header rang : `<RankProgressGauge>` + métriques `[Rang n/272, XP totale, XP actuel, XP rang suivant]` + caption `recorded_at`.
   - Section Hero : `<RankProgressGauge variant="hero">` + 4 métriques `[xp_earned, xp_remaining, xp_required, current_rank/RANK_MAX]`.
   - Section XP history : nouveau composant `CareerXPHistoryChart` avec 5 traces et toggles legendonly.
   - Section snapshot table : `Collapsible` + 10 dernières lignes formatées.
   - Section LUSR : grille de cards `<LUSRPlaylistCard>` (6 cartes ordonnées) + segmented control période + select playlist_group + courbe.
   - Section Top matches : tableau best + tableau worst, badges 5 types, popover légende, switch `exclude_btb` (persisté localStorage en attendant settings serveur).
   - Section Encounters : segmented control période, tableau Top encountered enrichi (cols : ordinal+badges, role, total A:n|E:n, WR ally, WR vs enemy, K/D dealt-suffered, last_seen relatif), 2 tableaux côte à côte Némésis & Souffre-douleurs, popover légende.
3. **Composants nouveaux** :
   - `CareerXPHistoryChart.tsx` (recharts) — palette `_OTHER_PLAYERS_COLORS` reproductible, hover translaté, ligne seuil Hero.
   - `CareerLUSRPlaylistCards.tsx` (grille 3×2) — image rang via `getRankImagePath(rating_value, rating_type)` (à porter dans `apps/web/src/lib/halo/skill-rating.ts`).
   - `CareerLUSRTimeseriesChart.tsx` (recharts) — segmented period + select group, granularité dérivée.
   - `CareerTopMatchesTableV2.tsx` (remplace l'existant) — colonnes `[match_id link, date, mode, map, score, KDA, K/D, durée, badge]`, badge couleurs spécifiques `BADGE_BG`, popover légende.
   - `EncounterBadgeRow.tsx` (porte `compute_encounter_badges` rendu inline) — réutilisé dans match-view.
4. **Hooks** : étendre `queries.ts` :
   - `useCareerTopMatches(slug, { excludeBtb })` (paramètre query).
   - `useCareerEncounters(slug, { period, exclude_friends })`.
5. **Routes** : conserver `/players/$slug/career?tab=progression`. Accepter `?tab=progression&period=1y&exclude_btb=true`.
6. **i18n** : ajouter clés Carrière dans `apps/web/src/lib/i18n/` (équivalents `career_top_legend_*`, `career_encounters_*`, `career_rank_hover`, etc.).
7. **Cleanup** : `CareerPage.tsx` (legacy) supprimé après vérification qu'il n'est plus monté nulle part.
8. **Régressions Synthèse / Palmarès** : adapter les composants Synthesis & Palmares pour consommer le DTO complet (sinon ils restent dégradés). Cible minimale : afficher au moins le badge dominance sur les highlights et exposer un lien vers la page Carrière.

### Phase F — Tests & golden parity

1. **Unitaires Go** :
   - `career_logic_test.go` : pour les 4 fonctions de projection (porter `tests/test_career_xp_projection.py`).
   - `career_service_test.go` : étendre avec un golden test sur `GetCareerPage` (fixture DuckDB minimale, comparer JSON canonique avec un snapshot Python).
   - `career_repo_test.go` : tester chaque SQL (Q8 cards, Q9 V2, Q10 top encountered, Q11 antagonists).
2. **Unitaires React** : `CareerXPHistoryChart.test.tsx`, `CareerLUSRPlaylistCards.test.tsx`, `CareerTopMatchesTableV2.test.tsx`.
3. **Golden parity** (script Go-only conformément à `feedback_no_python`) : nouveau `cmd/career_golden_parity` qui :
   - prend une DB joueur de référence,
   - exécute `GetCareerPage` + `GetTopMatches` + `GetEncounters`,
   - produit un JSON canonique trié,
   - compare avec un snapshot pré-généré à partir de la version Python (utiliser un export gelé une fois pour toutes en `testdata/career_golden.json`).
4. **E2E playwright** : scénario `tests/e2e/career.spec.ts` qui visite `/players/:slug/career`, vérifie présence des sections, déclenche le toggle BTB, change la période encounters, vérifie le re-render.

### Phase G — Cleanup & doc

1. Mettre à jour `docs/MIGRATION_GAP_PYTHON_TO_GO.md` § 8 pour refléter l'état post-portage.
2. Ajouter un schéma `docs/MATCH_AND_CAREER_PAGES.md` (ou intégrer dans le doc existant) avec les flux de données.
3. Mettre à jour `.ai/project_map.md` pour signaler les nouveaux fichiers.
4. Entrée `thought_log.md` à chaque phase livrée.
5. Supprimer `CareerPage.tsx` legacy + `CareerEncountersSection.tsx` legacy + tests associés.

---

## 5. Risques & dépendances

| Risque | Probabilité | Mitigation |
|---|---|---|
| `mv_player_matches` n'existe pas dans toutes les bases joueurs synchronisées en Go | Haute | Le sync Go doit garantir la création/maintenance de la vue. Vérifier `apps/go-api/internal/sync/` ; sinon ajouter un fallback `shared.match_participants ⨝ match_registry`. |
| `dominance_flag` non peuplé sur les anciens matchs | Haute | Le sync Python le populait via `_medal_verdicts.py`. Le sync Go doit reproduire ce calcul. À auditer en Phase B avant Q9. |
| Pas de notion de profils multiples côté Go (X5 — autres joueurs sur le graphe XP) | Moyenne | Reporter X5 à un sprint dédié ; livrer la page Carrière sans cette trace en Phase D, ajouter en Phase F.5. |
| Settings persistés `career_top_exclude_btb` | Moyenne | Stocker en `localStorage` côté front en attendant la couche `app_settings` Go (cf. Sprint 56 prévu). |
| Friends file JSON obsolète | Moyenne | Lire `friends_xuids` depuis `player_match_enrichment` en agrégeant — préférable, déjà stocké. |
| Recharts ne reproduit pas le visuel Plotly Halo | Faible | Réutiliser tokens couleur de `apps/web/src/lib/accessibility/palettes/` (cf. règle CLAUDE.md §20). Pas de hex en dur sauf pour les 5 couleurs de badge dominance déjà tolérées comme « rareté/narratif ». |
| Régression DTO `CareerEncountersResponse` casse `palmares/queries.ts` | Haute | Versionner : conserver `Teammates/Enemies/Total` + ajouter les nouveaux champs en parallèle dans la même payload pendant 1 sprint, puis retirer. |

---

## 6. Estimation par phase

| Phase | Effort (j-h) | Bloque |
|---:|---:|---|
| A — Modèle | 1.5 | B, C, D |
| B — Repo & SQL | 2 | D |
| C — Logic | 1 | D |
| D — Service | 1.5 | E, F |
| E — Frontend | 4 | F |
| F — Tests & golden | 2 | G |
| G — Cleanup & doc | 0.5 | — |
| **Total** | **12.5 j-h** | |

---

## 7. Checklist de livraison

- [ ] DTO `CareerPageResponse` conforme § 4.A et tests passent
- [ ] Q6 jointure `meta.career_ranks` pour `adornment_path`
- [ ] Q8 partitionne LUSR par `playlist_group` + `LAG` rating_delta + `rating_type`
- [ ] Q9 réécrit avec filtres durs + tri badge + tri secondaire ; split WIN/LOSS et non N/2
- [ ] Q10 + Q11 portés depuis `_TOP_ENCOUNTERED_SQL` et `_ANTAGONISTS_SQL`
- [ ] Préfixe `shared.` partout dans Q10/Q11
- [ ] Exclusion `xuid LIKE 'bid(%'` présente
- [ ] `friends_xuids` chargés et exclus
- [ ] `careerXPLaunchDate=2023-06-20` constant Go
- [ ] `XP_HERO_TOTAL=9_319_350`, `RANK_MAX=272` constants Go (déjà présents → vérifier)
- [ ] Courbe estimée pré-sync calculée
- [ ] Courbe projection Hero normale + optimiste calculées
- [ ] Cap projection 10 ans appliqué
- [ ] LUSR : 6 cartes par playlist_group avec rating_delta
- [ ] CareerXPHistoryChart : 5 traces + ligne seuil + hover translaté
- [ ] CareerLUSRTimeseriesChart : segmented period + select group + granularité
- [ ] CareerTopMatchesTableV2 : 5 badges, popover légende, toggle BTB, tri badge respecté
- [ ] Encounters : 3 sous-tableaux (Top, Némésis, Souffre-douleurs) + segmented period + badges
- [ ] Golden parity test passe
- [ ] Playwright E2E passe
- [ ] `CareerPage.tsx` legacy supprimé
- [ ] `docs/MIGRATION_GAP_PYTHON_TO_GO.md` § 8 mis à jour
- [ ] Synthèse/Palmarès mis à jour pour consommer les nouveaux DTO sans régression
