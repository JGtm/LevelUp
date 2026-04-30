# Page Match View — composition

> Source Python : `src/ui/pages/match_view.py` sur `origin/v7/cockpit` (426 lignes).
> Fonction d'entrée : `render_match_view(*, row, match_id, params)` (match_view.py:210).
> 13 fichiers helpers : `match_view_*.py` (charts, citations, encounters, helpers, logic, participation, players*, rank, scoreboard*, tabs, weapon_kills).

## Vue d'ensemble

La page Match View est la **page la plus riche** du dashboard. Composition :

1. **Header** (`_render_match_header`) : 4 KPI cards + bloc carte/perf/rang.
2. **5 onglets** (`st.tabs`) : Summary, Combat, Team, Citations/Médailles, Media.

Le scope **Phase 1** couvre le header + onglet Summary + scoreboard de l'onglet Team (8 visuels).
**Phase 2** couvre le reste (onglet Combat, onglet Citations, onglet Media, encounters).

## Scope Phase 1 (✅ documenté)

| # | Élément | Source | Spec YAML |
|---|---|---|---|
| 1 | Header KPI tiles | `_render_kpi_cards` (match_view.py:101) | `01_header_kpi_cards.yaml` |
| 2 | Map + Perf + Rank block | `_render_map_and_rank` (match_view.py:146) + `_build_match_rank_html` (match_view_rank.py) | `02_map_perf_rank_block.yaml` |
| 3 | Expected vs Actual | `render_expected_vs_actual` (match_view_charts.py:73) | `03_expected_vs_actual.yaml` |
| 4 | Spree + Headshots + Perfect Kills | `_render_spree_headshots` (match_view_charts.py:257) | `04_spree_headshots.yaml` |
| 5 | Pie Kills par arme | `_render_weapon_pie` (match_view_weapon_kills.py:69) | `05_weapon_kills_pie.yaml` |
| 6 | Tableau Kills par arme | `_render_weapon_table` (match_view_weapon_kills.py:113) | `06_weapon_kills_table.yaml` |
| 7 | Radar participation 6 axes | `create_participation_profile_radar` (_radar_participation.py:144) via `_plot_participation_radar_fig` | `07_participation_radar.yaml` |
| 8 | Scoreboard match (2 équipes) | `render_match_scoreboard` → `_render_team_table` (match_view_scoreboard.py:287) | `08_match_scoreboard.yaml` |

## Scope Phase 2 (✅ documenté)

| # | Élément | Source | Spec YAML | chart_kind |
|---|---|---|---|---|
| 09 | Kill/Death timeline du joueur (cumul) | `render_match_impact_section` (match_view_players.py:273) → `plot_match_kill_death_timeline` (match_impact_timeline.py:31) | `09_kd_cumul_timeline.yaml` | line |
| 10 | Tug-of-war dominance + kill feed | `render_team_dominance_section` (match_view_players_timeline.py:39) → `plot_dominance_chart` (team_dominance_timeline.py:209) | `10_team_dominance.yaml` | line |
| 11 | Histogramme cadence kills (bicolore + MA) | `render_match_cadence_section` (match_view_players_timeline.py:162) → `plot_match_cadence_histogram` (_cadence_histogram.py) | `11_match_cadence.yaml` | grouped_bar |
| 12 | Nemesis + Souffre-douleur (KPI cards) | `render_nemesis_section` (match_view_players_nemesis.py:262) | `12_nemesis_cards.yaml` | kpi_row |
| 13 | All players frags timeline | `render_kd_timeline_section` (match_view_players_timeline.py:231) → `plot_all_players_frags_timeline` (match_impact_timeline.py:281) | `13_all_players_frags_timeline.yaml` | line |
| 14 | Encounters du match | `render_encounter_section` (match_view_encounters.py:276) | `14_encounters_match.yaml` | table_html |
| 15 | Citations du match | `render_match_citations_section` (match_view_citations.py:18) | `15_match_citations.yaml` | composite_block |
| 16 | Médailles du match (grille) | `render_medals_tab` (match_view_citations.py:185) | `16_match_medals.yaml` | composite_block |
| 17 | Section Media (captures + vidéos) | `render_media_section` (match_view_helpers.py:295) | `17_media_section.yaml` | composite_block |

> **Note structurelle Phase 2** : pas de nouveau `chart_kind` à créer (tous réutilisent ce qui existe en Phase 1 — `line`, `grouped_bar`, `kpi_row`, `table_html`, `composite_block`).
> Le converter `line` pourra avoir besoin d'évolutions mineures pour gérer les annotations + markers pluriels (chart 09) et les hlines (chart 13).

## Ordre d'affichage Phase 1

```
┌──────────────────────────────────────────────────────────────┐
│  HEADER                                                       │
│  ┌────┬──────────┬──────────┬──────────┐                     │
│  │Date│ Score+   │ Playlist │ Mode/    │  (#1 KPI cards)     │
│  │    │ Badge    │          │ Carte    │                      │
│  └────┴──────────┴──────────┴──────────┘                     │
│  ┌────────────┬──────┬───────────┐                            │
│  │ Map thumb  │ Perf │ Rank      │  (#2 map+perf+rank block) │
│  └────────────┴──────┴───────────┘                            │
└──────────────────────────────────────────────────────────────┘
   [match_id badge popover]
   ┌─Summary─┬─Combat─┬─Team─┬─Citations─┬─Media─┐
   │         │        │      │           │       │
   ▼ Onglet Summary :
     ┌────── KPI cards (MMR, K/D vs expected, AvgLife) ──────┐
     ┌─ Expected vs Actual (#3) ─┬─ Spree+HS+Perfect (#4) ──┐
     │ 3 bars groupées            │  3 bars groupées          │
     │ Réel/Attendu/Hist          │  Réel/Hist                │
     └────────────────────────────┴──────────────────────────┘
     ┌─ Pie kills/arme (#5) ──────┬─ Table kills/arme (#6) ──┐
     │ donut hole=0.35             │  os-sb-table 2 col       │
     └─────────────────────────────┴───────────────────────────┘
     ┌─ Radar participation 6 axes (#7) ─────────────────────┐
     │ ou st.columns([2,1]) si hints_visible() (radar+légende)│
     └────────────────────────────────────────────────────────┘

   ▼ Onglet Team :
     ┌─ Scoreboard équipe alliée (#8a) ──────────────────────┐
     │ os-sb-team--mine, MVP/LVP, rows expandables           │
     └────────────────────────────────────────────────────────┘
     ┌─ Scoreboard équipe ennemie (#8b) ─────────────────────┐
     │ os-sb-team--enemy                                      │
     └────────────────────────────────────────────────────────┘
     [+ Encounters du match — Phase 2]
```

## Pré-conditions

| Donnée | Source | Notes |
|---|---|---|
| `row` (match_stats du joueur) | passé en param | dict — colonnes : kills, deaths, assists, ratio, outcome, my_team_score, enemy_team_score, map_name, pair_name, mode_ui, playlist_name, start_time, time_played_seconds, max_killing_spree, headshot_kills, etc. |
| `pm` (player match enrichi) | `params.load_player_match_result_fn(db_path, match_id, xuid)` | dict — team_mmr, enemy_mmr, kills/deaths/assists avec `expected` + `actual` |
| `medals_last` | `params.load_match_medals_fn(...)` | médailles du match (utilisé Phase 2) |
| `df_full` | passé en param | full polars df pour calculer la moyenne historique par mode_category |
| `_had_bot, _stored_perf, _dominance_flag` | `repo.load_player_match_enrichment(match_id)` | 3 flags du player_match_enrichment |
| `is_abandoned` | `repo.is_abandoned_match(match_id)` | bool — affiche un warning au-dessus si True |
| Awards Personal Score | `_load_participation_awards(db_path, match_id, xuid)` | utilisé pour le radar #7 |
| Weapon kills | `repo.load_match_weapon_kills(match_id, xuid)` | pour pie + table #5/#6 |
| Skill rank du match | `cached_get_match_skill_rank(db_path, match_id, db_key)` | pour rank HTML #2 |
| Scoreboard | `params.load_match_gamertags_fn(...)` + repo | pour #8 |

## Configs Plotly par section

| # | Config | Source |
|---|---|---|
| 3 | `PLOTLY_STATIC_CONFIG` | charts.py via `render_chart_or_info` |
| 4 | `PLOTLY_STATIC_CONFIG` | charts.py |
| 5 | `PLOTLY_CLEAN_CONFIG` | weapon_kills.py:110 (interactif — clic légende toggle) |
| 7 | `PLOTLY_STATIC_CONFIG` | participation.py:138 |

Tableaux 6, 8 et le Map+Perf+Rank block (#2) : pas de Plotly, rendu HTML markdown direct.

## Constantes & énumérations critiques

| Constante | Valeur | Source |
|---|---|---|
| `_DOMINANCE_BADGE_STYLES` | dict[1..5] → (i18n_key, bg_hex, fg_hex) | match_view.py:43-50 |
| `Outcome` enum | `WIN=2, LOSS=3, TIE=1, DNF=4` | refdata.py |
| `TEAM_MAP` | dict team_id → label (Eagle/Cobra/etc.) | scoreboard.py |

`_DOMINANCE_BADGE_STYLES` :
- `1` (DOMINATION)        → vert foncé `#2e7d32` / `#e8f5e9`
- `2` (HUMILIATION)       → violet `#6a1b9a` / `#f3e5f5`
- `3` (REMONTADA)         → bleu `#1565c0` / `#e3f2fd`
- `4` (DÉBÂCLE)           → rouge-brique `#bf360c` / `#fbe9e7`
- `5` (CONTRE-REMONTADA)  → vert-canard `#00695c` / `#e0f2f1`

## Layout et hiérarchie d'appels

```
render_match_view()
├── _render_match_header()
│   ├── _render_kpi_cards()              ← #1
│   └── _render_map_and_rank()           ← #2
│       └── _build_match_rank_html()
├── (warning si is_abandoned)
├── _render_match_id_badge() (popover)
└── _render_match_tabs()
    ├── _render_summary_tab()
    │   ├── render_expected_vs_actual()  ← #3
    │   │   └── _render_spree_headshots() ← #4
    │   ├── render_weapon_kills_section()
    │   │   ├── _render_weapon_pie()     ← #5
    │   │   └── _render_weapon_table()   ← #6
    │   └── render_participation_section() ← #7
    ├── _render_combat_tab()              [Phase 2]
    ├── _render_team_tab()
    │   ├── render_match_scoreboard()    ← #8
    │   └── render_encounter_section()   [Phase 2]
    ├── _render_citations_tab()           [Phase 2]
    └── _render_media_tab()               [Phase 2]
```

## i18n keys de la page (vue d'ensemble — détails par YAML)

Espace `t` :
- Header : `mv_team_label`, `mv_team_n`, `mv_team_unknown`, `mv_match_id_missing`, `mv_match_id_popover`, `mv_match_id_copy_hint`, `mv_loading`, `mv_abandoned_match`, `mv_abandoned_match_desc`, `mv_rating_pending`, `mv_rating_pending_hint`
- Onglets : `mv_tab_summary`, `mv_tab_combat`, `mv_tab_team`, `mv_tab_citations_medals`, `mv_tab_media`
- Outcome : `outcome_domination`, `outcome_humiliation`, `career_top_badge_remontada`, `career_top_badge_debandade`, `career_top_badge_contre_remontada`
- Charts (#3-4) : `mvc_mmr_team`, `mvc_lbl_k`, `mvc_lbl_d`, `mvc_lbl_a`, `mvc_fda_ratio`, `mvc_fda_title`, `mvc_hist_avg`, `lbl_actual`, `lbl_expected`, `mvc_this_match`, `tm_kills`, `tm_deaths`, `tm_killing_spree`, `tm_headshots`, `tm_perfect_kills`, `ts_spree`, `col_avg_life_long`
- Weapons (#5-6) : `mv_weapon_kills_col_weapon`, `mv_weapon_kills_col_frags`, `col_grenade_kills`, `col_melee`
- Participation (#7) : `mvp_participation_title`, `mvp_axes_label`, `radar_objectives`, `radar_combat`, `radar_support`, `col_score`, `radar_impact`, `radar_survival`, `radar_hover_survival_pct`, `radar_hover_deaths`
- Scoreboard (#8) : `mv_team_label`, `mv_team_n`, `mv_team_unknown`

Espace `viz_t` : peu utilisé sur cette page (les charts sont nommés directement avec `t()`).
