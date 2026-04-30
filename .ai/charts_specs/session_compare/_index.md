# Page Session Comparison — composition

> Source Python : `src/ui/pages/session_compare.py` sur `origin/v7/cockpit` (534 lignes).
> Fonction d'entrée : `render_session_comparison_page(...)` (session_compare.py:468).
> Page **linéaire** (pas d'onglets sauf le dernier match_history split en 2 tabs).

## Vue d'ensemble

Page de comparaison côte-à-côte de 2 sessions du joueur. Toutes les sections affichent A
(Session A, rouge corail `#E74C3C`) vs B (Session B, bleu vif `#3498DB`). Optionnellement,
une **moyenne historique** (violet `#9B59B6`) sert de référence sur le radar (07) et le bar
chart (08). 14 sections séquentielles.

## Couleurs canoniques

```yaml
session_a: "#E74C3C"     # rouge corail
session_a_fill: "rgba(231, 76, 60, 0.3)"
session_b: "#3498DB"     # bleu vif
session_b_fill: "rgba(52, 152, 219, 0.3)"
historical: "#9B59B6"    # violet (moyenne historique)
historical_fill: "rgba(155, 89, 182, 0.2)"
```

## Sections (ordre de rendu réel)

| # | Élément | Source | chart_kind |
|---|---|---|---|
| 01 | Temporal header (date+nombre parties par session) | `render_session_temporal_header` (`_session_compare_viz.py:99`) | `composite_block` (HTML inline) |
| 02 | Score cards perf (Session A / B) | `_render_score_cards` → 2× `render_performance_score_card` (`components/performance.py:105`) | `kpi_row` |
| 03 | Outcomes distribution (2 donuts W/L/T/DNF) | `render_outcomes_distribution` (`_session_compare_viz.py:192`) | `pie` (× 2 en colonnes) |
| 04 | Match highlights (best/worst par session) | `render_match_highlights` (`_session_compare_extra.py:158`) | `composite_block` (markdown) |
| 05 | Detailed metrics rows (kills/deaths/F-D ratio/win rate/...) | `_render_detailed_metrics` → multiples `render_metric_comparison_row` | `table_html` |
| 06 | MMR comparison rows | `_render_mmr_comparison` → `render_metric_comparison_row` | `table_html` |
| 07 | Radar comparison chart (3 axes : F/D, Win%, Accuracy) | `render_comparison_radar_chart` (`session_compare_charts.py:73`) | `radar` |
| 08 | Bar chart comparison (axes gauche+droit Y1/Y2 win rate) | `render_comparison_bar_chart` (`session_compare_charts.py:356`) | `grouped_bar` (avec Y2) |
| 09 | Cumulative net score comparison | `_render_cumulative_section` → `plot_cumulative_comparison` (`_perf_session.py:216`) | `line` (avec hline 0) |
| 10 | K/D progression par partie + précision Y2 | `render_kd_progression` (`_session_compare_viz.py:357`) | `line` (avec Y2) |
| 11 | Modes breakdown horizontal | `render_modes_breakdown` (`_session_compare_extra.py:79`) | `grouped_bar` (horizontal) |
| 12 | Map table comparatif (parties + V/D/E par carte) | `render_map_table` (`_session_compare_extra.py:208`) | `table_html` |
| 13 | Participation trend (6 axes en bars horizontales A vs B) | `render_participation_trend_section` (`session_compare_charts.py:453`) | `grouped_bar` (horizontal, ex-radar) |
| 14 | Match history tables (2 onglets Session A / B) | `_render_match_history_tabs` → 2× `render_session_history_table` | `table_html` (× 2) |

## Pré-conditions

| Donnée | Source |
|---|---|
| `df_session_a, df_session_b` | DataFrames matchs des 2 sessions sélectionnées |
| `df_full` | DataFrame de base (toutes les parties — pour history_table) |
| `db_path, xuid` | Pour `_load_friends_mapping_from_db`, `_load_participation_profiles`, etc. |
| `perf_a, perf_b` | dicts métriques calculées par `compute_session_performance_score_v2_ui` |
| `hist_avg` | dict optionnel — moyenne historique de sessions similaires (sert de référence) |

## Sections "info-only" (sans Plotly)

- 01 (temporal_header) : `st.markdown(...)` HTML inline avec dates colorées
- 04 (match_highlights) : `st.markdown(...)` avec best/worst en text formaté
- 05 / 06 (detailed_metrics, mmr) : `render_metric_comparison_row` produit des lignes HTML stylées
- 12 (map_table) : table HTML construite manuellement avec `_TH/_TD` styles
- 14 (history_tables) : `render_session_history_table` produit une table HTML

## Configs Plotly

Toutes les sections Plotly utilisent **`PLOTLY_STATIC_CONFIG`** (cohérent avec l'usage page-comparison).

## Constantes notables

- `_COLOR_A`, `_COLOR_B` : couleurs sessions (en réalité `SESSION_COLORS["session_a"]` / `["session_b"]`)
- Radar normalisation (07) : `kd * 50 → 100` (F/M 2.0 = 100), `wr` direct, `acc` direct (déjà en %)
- KD layout (10) : `dtick=1` (1 tick par match), Y2 range fixe `[0, 100]` pour accuracy
- Mode breakdown (11) : `height = max(180, len(modes) * 48)` — adaptatif au nombre de modes
- Hist avg badge ⚠️ si `session_count < 3` (radar 07 + bar 08)

## Fragments

- `render_outcomes_distribution`, `render_kd_progression`, `render_comparison_radar_chart`,
  `render_comparison_bar_chart` ont `@fragment_if_available` — re-rendu rapide.

## Réutilisations

Aucune section n'est partagée avec d'autres pages — toutes spécifiques à Session Comparison.
