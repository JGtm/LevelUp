# Page Teammates — composition

> Source Python : `src/ui/pages/teammates.py` sur `origin/v7/cockpit` (343 lignes).
> Fonction d'entrée : `render_teammates_page(...)` (teammates.py:201).
> 13 fichiers helpers : `teammates_*.py` + `_teammates_trio*.py`.

## Vue d'ensemble

Page d'analyse des coéquipiers fréquents. Vue **multi** quand au moins 1 ami est sélectionné via le sidebar. Si 0 ami → message "Sélectionner un coéquipier".

**Structure** :
1. Header : cards Spartan des amis sélectionnés (`render_teammate_cards`)
2. Panel légende joueurs flottant à droite (`render_player_legend_panel` — couleurs assignées)
3. **2 onglets** : Synergies (collectif) | Contributions (individuel comparé)

## Onglet "Synergies" (collectif) — `tab_syn` dans `teammates_views.py:90`

Ordre de rendu réel dans la page :

| Pos | # | Élément | Source | chart_kind |
|----|---|---|---|---|
| 1  | 02 | Bullet chart : winrate session vs historique par carte | `plot_map_winrate_bullet` | **`bullet`** |
| 2  | 13 | Performance par carte — Session vs Historique (barres groupées) | `plot_map_perf_vs_history` | `grouped_bar` |
| 3  | 11 | Historique des matchs avec coéquipiers (250 derniers) | `render_friends_history_table` | `table_html` |
| 4  | 03 | Heatmap performance par joueur × carte (top 15) | `plot_squad_map_heatmap` | `heatmap` |
| 5  | 04 | Timeline performance d'escouade par session | `plot_squad_performance_timeline` | `line` (subplots Y2) |
| 6  | 05 | Form score history (fill positif/négatif) | `plot_form_score_history` | `line` (areaStyle) |
| 7  | 07 | Impact des coéquipiers (tableau scoreboard) | `render_impact_taquinerie` | `composite_block` |

## Onglet "Contributions" (individuel comparé) — `tab_con` dans `teammates_views.py:122`

Ordre de rendu réel dans la page (cf. `_teammates_trio.py::render_trio_view` + `teammates_views.py::_render_bottom_charts`) :

| Pos | # | Élément | Source | chart_kind |
|----|---|---|---|---|
| 1  | 14 | Stats par minute — Frags/Morts/Assists | `_render_per_minute_stats` (`_teammates_trio_helpers.py:91`) | `grouped_bar` (3 cat × N joueurs) |
| 2  | 06 | Radar synergie 6 axes (moi + 1-3 amis) | `render_trio_synergy_radar` → `create_participation_profile_radar` | `radar` |
| 3  | 15 | Heatmap d'intensité — kills par phase | `render_squad_intensity_heatmap` (`teammates_intensity.py:102`) | `heatmap` (matchs × 10 phases) |
| 4  | 16 | Charts de performance escouade (6 sous-charts) | `_render_trio_performance_charts` → `render_trio_charts` (`teammates_charts.py:194`) | `line` (KD combined + 5 metrics) |
| 5  | 09 | Weapon kills bar chart (multi-joueurs horizontal) | `render_weapon_kills_bar_chart` | `grouped_bar` (horizontal) |
| 6  | 08 | Metric bar charts (killing spree + HS/PK) | `render_metric_bar_charts` | `line` (bars+line multi-joueurs) |
| 7  | 17 | Premier frag / première mort (butterfly bins de 15s) | `render_first_events_chart` (`teammates_charts.py:507`) | `grouped_bar` (butterfly chronologique) |
| 8  | 10 | Weapon kills table (joueur principal) | `render_weapon_kills_table` | `table_html` |
| 9  | 12 | Trio medals (grilles N joueurs) | `_render_trio_medals` | `composite_block` |

**Note** : 10 (weapon table) est rendu via `_render_bottom_charts` quand `rendered_bottom_charts == False` (fallback non-trio), pas dans `render_trio_view` direct. Sinon ce chart est dans le flux principal.

## Header (toujours rendu en haut de page)

| Pos | # | Élément | Source | chart_kind |
|----|---|---|---|---|
| 1  | 00 | KPI Section — mes stats sur le scope (8 cards avec trends) | `render_kpis_section` (`app/kpis_render.py:116`) | `kpi_row` (réutilise [`timeseries.00`](../timeseries/00_kpis_section.yaml)) |
| 2  | 01 | Squad session header — Team card grade + N cards joueurs | `render_squad_session_header` (`components/performance.py:215`) | `kpi_row` |

## Pré-conditions

| Donnée | Source |
|---|---|
| `picked_xuids` | sélection sidebar (1+ amis) |
| `df, dff, base` | DataFrames matchs (filtrés / complets / brut) |
| `series` | `[(name, df), ...]` — 1 entrée par membre escouade (moi + amis) |
| `colors_by_name` | dict {gamertag → hex} via `assign_player_colors_fn` |
| `sub_all` | sous-ensemble des matchs où tous les amis sélectionnés sont présents |

## Configs Plotly

| # | Config | Source |
|---|---|---|
| 02 | `PLOTLY_STATIC_CONFIG` | teammates_map_charts.py:80 |
| 03 | `PLOTLY_STATIC_CONFIG` | (à confirmer) |
| 04 | `PLOTLY_STATIC_CONFIG` | (à confirmer) |
| 05 | `PLOTLY_STATIC_CONFIG` | (à confirmer) |
| 06 | `PLOTLY_STATIC_CONFIG` | _render_radar_display |
| 13 | `PLOTLY_STATIC_CONFIG` | teammates_map_charts.py:87 |
| 14 | `PLOTLY_STATIC_CONFIG` | _teammates_trio_helpers.py:172 |
| 15 | `PLOTLY_CLEAN_CONFIG` | teammates_intensity.py:177 |
| 16 | `PLOTLY_CLEAN_CONFIG` | teammates_charts.py:218,256 |
| 17 | `PLOTLY_CLEAN_CONFIG` | teammates_charts.py:530 |

## Constantes notables

- `_BULLET_DELTA_THRESHOLD` (visualization/_maps_outcome_bullet.py) — seuil ±5% pour colorer barres session vert/ambre/rouge
- `_OKABE_ROSE = "#CC79A7"` — couleur rose Okabe pour barre historique
- Top maps = 15 par fréquence (heatmap)
- Form score : seuils visuels via `_FORM_THRESHOLDS`
