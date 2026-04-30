# Page Timeseries — composition

> Source Python : `src/ui/pages/timeseries.py` sur `origin/v7/cockpit` (430 lignes).
> Fonction d'entrée : `render_timeseries_page(...)` (timeseries.py:351).
> 5 onglets + helpers répartis dans 6 sous-modules + `win_loss.py` (sections partagées).

## Vue d'ensemble

Page d'analyse des séries temporelles de stats du joueur principal. 5 onglets :
1. **Résumé K/D/A** (`ts_tab_kda`) — KPIs + form score + KDA + outcomes + streaks
2. **Cartes & Modes** (`ts_tab_maps`) — breakdown par carte/mode + perf vs historique
3. **Distributions** (`ts_tab_distributions`) — 6 histogrammes + scatter matrix
4. **Progression** (`ts_tab_advanced`) — performance/assists/per-min/spree/shots/damage/rank/personal_score
5. **Avancé** (`ts_tab_progression`) — intensity heatmap + skill rank + cumul perf + WL heatmap + top by week

⚠️ **Ordre tabs ≠ ordre des `with`** : les variables `_tab_advanced` et `_tab_progression` sont
inversées par rapport aux labels i18n (artefact du code source — confirmé par lecture du `st.tabs()`
ligne 396). Le tableau ci-dessous suit l'ordre RÉEL d'affichage utilisateur.

## Onglet 1 — "Résumé K/D/A" (`ts_tab_kda` → `_render_summary_tab`)

| # | Élément | Source | chart_kind |
|---|---|---|---|
| 00 | KPI Section — 8 cards stylées avec trends vs all-time | `render_kpis_section` (`app/kpis_render.py:116`) | `kpi_row` |
| 01 | Form score (évolution session vs historique) | `_render_form_score_section` (`_timeseries_form.py:94`) | `line` (areaStyle) — *réutilise teammates.05* |
| 02 | K/D/A timeseries | `plot_timeseries` (`timeseries_combat.py`) | `line` (3 traces) |
| 03 | K/D/A distribution (boxplot ou hist) | `plot_kda_distribution` (`distributions.py`) | `histogram` |
| 04 | Top weapons by kills | `_render_weapon_kills_chart` → `plot_top_weapons` | `grouped_bar` (horizontal) |
| 05 | Outcomes over time (W/L/T par session) | `_render_outcomes_over_time` (`win_loss.py:77`) | `stacked_bar` |
| 06 | Streak section (séries consécutives) | `_render_streak_section` (`win_loss.py:217`) | `line` ou `bar` |

## Onglet 2 — "Cartes & Modes" (`ts_tab_maps`)

| # | Élément | Source | chart_kind |
|---|---|---|---|
| 07 | Map/mode breakdown | `_render_map_mode_breakdown` (`win_loss.py:93`) | `stacked_bar` — *réutilise synthesis.01-02* |
| 08 | Winrate & perf vs historique (par carte) | `_render_winrate_perf_vs_history` (`win_loss.py:259`) | `grouped_bar` (similar teammates.13 mais self) |

## Onglet 3 — "Distributions" (`ts_tab_distributions`)

| # | Élément | Source | chart_kind |
|---|---|---|---|
| 09 | 6 histogrammes (kills, deaths, accuracy, perf, score/min, winrate) | `render_distributions` (`_timeseries_distributions.py:29`) | `histogram` × 6 |
| 10 | Scatter matrix corrélations | `render_correlations` | `scatter_matrix` (nouveau kind) |

## Onglet 4 — "Progression" (`ts_tab_advanced` → `_tab_prog`)

| # | Élément | Source | chart_kind |
|---|---|---|---|
| 11 | Premier événement distribution | `plot_first_event_distribution` (`distributions.py`) | `histogram` |
| 12 | Performance timeseries | `plot_performance_timeseries` (`timeseries_combat.py`) | `line` |
| 13 | Assists timeseries | `plot_assists_timeseries` | `line` |
| 14 | Stats per minute timeseries | `plot_per_minute_timeseries` | `line` (multi metrics) |
| 15 | Average life timeseries | `plot_average_life` | `line` |
| 16 | Spree + headshots + accuracy | `plot_spree_headshots_accuracy` | `line` (multi-axis) |
| 17 | Shots accuracy | `plot_shots_accuracy` (Sprint 7) | `line` |
| 18 | Damage dealt vs taken | `plot_damage_dealt_taken` | `line` (2 traces) |
| 19 | Rank score evolution | `plot_rank_score` | `line` |
| 20 | Personal score awards | `_render_personal_score_section` (`win_loss.py:234`) | `bar` |

## Onglet 5 — "Avancé" (`ts_tab_progression` → `_tab_adv`)

| # | Élément | Source | chart_kind |
|---|---|---|---|
| 21 | Match intensity heatmap (self) | `_render_intensity_heatmap` (`_timeseries_intensity.py:68`) | `heatmap` — *réutilise teammates.15 logic, mode self only* |
| 22 | Skill rank progression (LUSR/CSR) | `render_skill_rank_progression` (`timeseries_skill_rank.py`) | `line` |
| 23 | Net score per hour | `plot_net_score_per_hour` (`progression.py`) | `line` |
| 24 | Cumulative K-D avec CI 95% | `plot_cumulative_kd_with_ci` | `line` (avec areaStyle CI) |
| 25 | EWMA K-D | `plot_ewma_kd` | `line` |
| 26 | Regression trend | `plot_regression_trend` | `line` |
| 27 | Win/Loss heatmap (jour×heure) | `_render_heatmap_section` (`win_loss.py:175`) | `heatmap` — *réutilise synthesis.03* |
| 28 | Top matches by week | `_render_top_by_week` (`win_loss.py:193`) | `grouped_bar` — *réutilise synthesis.04* |

## Pré-conditions

| Donnée | Source |
|---|---|
| `df, dff, base_df, df_full` | DataFrames matchs joueur principal (filtrés/non) |
| `db_path, xuid` | Pour requêtes secondaires (weapons, intensity, skill rank, personal_score) |
| `is_session_scope` | Bool — change le rendu de outcomes_over_time |

## Réutilisations entre pages

| Chart | Réutilisations |
|---|---|
| 01 form_score | teammates.05 (mais contexte session vs hist différent) |
| 07 map_mode_breakdown | synthesis.01, 02 |
| 21 intensity_heatmap | teammates.15 (sans toggle joueur, mode self only) |
| 27 wl_heatmap | synthesis.03 |
| 28 top_by_week | synthesis.04 |

## Configs Plotly

Toutes les `st.plotly_chart` utilisent **`PLOTLY_CLEAN_CONFIG`** (mode interactif léger) sauf
`plot_form_score_history` et `plot_intensity_heatmap` qui utilisent `PLOTLY_STATIC_CONFIG`.

## Constantes notables

- `_BIN_SIZE_S = 15` (first_event distribution, partagé teammates.17)
- `EWMA_ALPHA` : facteur de lissage (à confirmer dans progression.py)
- CI level = 95% (cumulative_kd)
- `KDA_HISTOGRAM_BINS = 20` (à confirmer)
