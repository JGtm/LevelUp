# Page Win/Loss — composition

> Source Python : `src/ui/pages/win_loss.py` sur `origin/v7/cockpit` (394 lignes).
> Fonction d'entrée : `render_win_loss_page(...)` (win_loss.py:39).
> Page **linéaire** (pas d'onglets) : 7 sections séquentielles.

## Vue d'ensemble

Page dédiée à l'analyse Victoires/Défaites du joueur principal. Toutes les sections
sont des fonctions partagées avec d'autres pages (Synthèse, Timeseries) — la page
Win/Loss est en réalité un **regroupement** axé V/D des charts existants.

## Sections (ordre de rendu réel)

| Pos | Élément | Source | YAML existant | Notes |
|---|---|---|---|---|
| 1 | Outcomes over time | `_render_outcomes_over_time` | [`timeseries.05`](../timeseries/05_outcomes_over_time.yaml) | Bars W↑/L↓ par bucket (semaine/jour/session) |
| 2 | Map / Mode breakdown | `_render_map_mode_breakdown` | [`timeseries.07`](../timeseries/07_map_mode_breakdown.yaml) (cartes) + [`synthesis.01`](../synthesis/01_outcomes_by_map.yaml) / [`synthesis.02`](../synthesis/02_outcomes_by_mode.yaml) | 2 stacked bars côte à côte |
| 3 | Win/Loss heatmap | `_render_heatmap_section` | [`synthesis.03`](../synthesis/03_winrate_heatmap.yaml) / [`timeseries.27`](../timeseries/27_wl_heatmap.yaml) | Heatmap jour × heure |
| 4 | Top matches by week | `_render_top_by_week` | [`synthesis.04`](../synthesis/04_top_matches_by_week.yaml) / [`timeseries.28`](../timeseries/28_top_by_week.yaml) | Bars top vs total + line % |
| 5 | Streak chart | `_render_streak_section` | [`timeseries.06`](../timeseries/06_streak_chart.yaml) | Bars Y signé V↑/D↓ avec compteur cumulé |
| 6 | Personal score | `_render_personal_score_section` | [`timeseries.20`](../timeseries/20_personal_score.yaml) | Bars amber + smoothing rolling 10 (conditionnel) |
| 7 | Winrate & Perf vs Historique | `_render_winrate_perf_vs_history` | [`timeseries.08`](../timeseries/08_winrate_perf_vs_history.yaml) → réutilise [`teammates.02`](../teammates/02_map_winrate_bullet.yaml) + [`teammates.13`](../teammates/13_map_perf_vs_history.yaml) | Bullet + barres groupées par carte |

## Section désactivée

- `_render_ratio_by_map_section` est **commentée** (line 74 — `# DISABLED`). Elle contenait :
  - Lollipop ratio par carte
  - Timeline performance par session
  - Bullet winrate (déjà section 7)
  - Perf vs historique (déjà section 7)
  - `_render_map_ratio_accuracy` : 2 colonnes (ratio + accuracy) via `plot_map_comparison`
  Ce YAML pourrait être ajouté si la section est ré-activée, sinon ignoré.

## Spécificités page Win/Loss vs autres pages

| Aspect | Win/Loss | Timeseries | Synthèse |
|---|---|---|---|
| Structure | linéaire (pas d'onglets) | 5 onglets | linéaire |
| Scope | 1 joueur (xuid) | 1 joueur (xuid) | 1 joueur (xuid) |
| Focus | V/D + outcomes | toutes stats temporelles | overview multi-période |
| Pré-condition | `dff` non vide (sinon `t('no_matches')`) | idem | sélection période |
| Spinner | `st.spinner(t('wl_computing'))` | aucun | aucun |

## Pré-conditions

| Donnée | Source |
|---|---|
| `dff` | DataFrame matchs filtrés (sélection courante) |
| `base` | DataFrame de base (toutes les parties après filtres Firefight) |
| `picked_session_labels` | Liste sessions sélectionnées (pour `is_session_scope`) |
| `db_path, xuid, db_key` | Pour requêtes secondaires |

## Réutilisations entre pages

Toutes les sections WL sont partagées :
- 4 sections sont aussi dans **Timeseries** (tabs Résumé + Avancé) : 05, 06, 20, 27/28
- 3 sections sont aussi dans **Synthèse** : 01-02-03-04
- 1 section (perf_vs_history) réutilise des YAMLs **Teammates** (02, 13)

## Configs Plotly

| Section | Config |
|---|---|
| 1 outcomes_over_time | `PLOTLY_CLEAN_CONFIG` |
| 2 map/mode breakdown | `PLOTLY_STATIC_CONFIG` |
| 3 wl_heatmap | (cf. synthesis.03) |
| 4 top_by_week | `PLOTLY_CLEAN_CONFIG` |
| 5 streak | `PLOTLY_CLEAN_CONFIG` |
| 6 personal_score | `PLOTLY_CLEAN_CONFIG` |
| 7 winrate+perf | `PLOTLY_STATIC_CONFIG` (× 2) |

## Constantes notables

- Section 2 (`_render_map_mode_breakdown`) : `min_matches=1`, `max_categories=12`, `sort_by='total'`.
- Section 7 (`_render_winrate_perf_vs_history`) : fix `map_ui` sur `base` en français pour éviter l'échec du join sur `Cliffhanger / Fortress / Nemesis / The Pit` (voir `timeseries.08` notes).
