# Page Synthèse — composition

> Source Python : `src/ui/pages/synthesis.py` sur `origin/v7/cockpit` (320 lignes au moment de la spec).
> Fonction d'entrée : `render_synthesis_page(dff, base, db_path, xuid, db_key)` (synthesis.py:265).

## Vue d'ensemble

La page Synthèse est une vue agrégée stratégique. Elle :
- Applique un filtre de **période local** (segmented control en haut) qui régit toutes les sections.
- Compose **3 visualisations importées** de la page Win/Loss (déléguées en l'état).
- Ajoute **1 visualisation locale** : duel Solo vs Escouade.

Toutes les sub-renders sauf la première sont décorées `@fragment_if_available` (`src/ui/streamlit_modern.py`) — chaque fragment se recalcule indépendamment quand un widget interne change, sans re-render de la page entière.

## Ordre d'affichage

| # | Élément | Type | Source | Spec YAML |
|---|---|---|---|---|
| 0 | Sélecteur de période | `st.segmented_control` | `_render_period_selector` (synthesis.py:50) | _ce fichier_, section "Contrôles de page" |
| 1 | Résultats par carte | Barres empilées **verticales** (catégories sur X, count sur Y, tickangle 45°) | `plot_stacked_outcomes_by_category` via `_render_map_mode_breakdown` (win_loss.py:93) — colonne gauche | `01_outcomes_by_map.yaml` |
| 2 | Résultats par mode | Barres empilées verticales (idem, sur `_mode_short` extrait après `" : "`) | idem, colonne droite | `02_outcomes_by_mode.yaml` |
| 3 | Heatmap Win Rate jour × heure | Heatmap 2D (24h × 7j, Lun en haut via `autorange="reversed"`) | `plot_win_ratio_heatmap` via `_render_heatmap_section` (win_loss.py:175) | `03_winrate_heatmap.yaml` |
| 4 | Matchs Top par semaine | Barres empilées (top vs others) + ligne ratio sur Y2 | `plot_matches_at_top_by_week` via `_render_top_by_week` (win_loss.py:193) | `04_top_matches_by_week.yaml` |
| 5 | Duel Solo vs Escouade | Barres horizontales bidirectionnelles (Solo négatif, Squad positif) | `_build_duel_chart` (synthesis.py:218) via `_render_solo_squad_compare` | `05_solo_squad_duel.yaml` |

Sections 1 et 2 vivent dans le **même `st.divider() + st.subheader + st.columns(2)`** (un seul subheader `wl_results_by_map_mode`, deux colonnes `wl_by_map` / `wl_by_mode`). Toutes les autres ont leur propre `st.divider()` + `st.subheader`.

## Contrôles de page

La page Synthèse a **un seul contrôle de niveau page** (le sélecteur de période). Les sous-renders importés de Win/Loss n'introduisent aucun contrôle interne supplémentaire — chaque section affiche directement son chart, sans selectbox/toggle au-dessus ou en dessous.

> Convention pour les autres pages : si un widget est positionné au-dessus ou en dessous d'un chart pour modifier son dataset, ses axes ou ses traces, il doit figurer dans la section `controls:` du YAML du chart concerné (voir `_schema.yaml`). Si le widget affecte tous les charts de la page, le documenter ici (page-level) et les YAML chart le référencent par `id`.

### Sélecteur de période (page-level)

Code source : `_render_period_selector` (synthesis.py:50–58).

| Champ | Valeur |
|---|---|
| Widget | `st.segmented_control` |
| Clé `session_state` | `synthesis_period` |
| Position | Au-dessus de toutes les sections, sous le titre de page |
| Scope | `page` — affecte les 5 charts |
| Options | `["all", "2y", "1y", "1m", "1w"]` (constante `_PERIOD_KEYS` synthesis.py:30) |
| Default | `"all"` |
| Format display | `t(f"encounters_period_{key}")` (clés i18n existantes) |
| Label | `t("encounters_period_label")` |
| Effet | `filter_dataset` |
| Refetch API requis ? | Côté Go : selon stratégie. Soit param `?period=` envoyé à chaque appel, soit refetch explicite côté client. |

**Effet sur le DataFrame** : `_filter_by_period(df, period)` — soustrait `_PERIOD_OFFSETS[period]` jours à `now(UTC)` et filtre `start_time >=` (synthesis.py:61–69). Offsets : `{2y: 730, 1y: 365, 1m: 30, 1w: 7}`. La valeur `all` ne filtre pas.

## Pré-traitement commun

Avant le rendu des sections, `render_synthesis_page` :
1. Convertit `base` en Polars (`ensure_polars`).
2. Filtre par période (`_filter_by_period`).
3. Attache `is_with_friends` si absent via `load_is_with_friends(db_path, xuid, match_ids)` (`explorer_data.py`). Cette colonne est requise pour la section 5.
4. Si DF vide après filtre → `st.info(t("no_matches"))` et sortie.

## États vides

| Section | Condition | Affichage |
|---|---|---|
| Toutes | `base` vide | `st.warning(t("no_matches"))` |
| Toutes | DF vide après filtre période | `st.info(t("no_matches"))` |
| 1, 2 | Colonnes manquantes (`map_name`/`outcome` ou `mode_*`/`outcome`) | `st.info(t("insufficient_data_chart"))` |
| 1, 2 | `plot_stacked_outcomes_by_category` retourne `None` | idem |
| 3 | `start_time` ou `outcome` absent | `st.info(t("missing_time_data"))` |
| 3 | `plot_win_ratio_heatmap` retourne `None` | `st.info(t("insufficient_data_chart"))` |
| 4 | `start_time` absent | `st.info(t("missing_time_data"))` |
| 4 | `plot_matches_at_top_by_week` retourne `None` | `st.info(t("insufficient_data_chart"))` |
| 5 | `is_with_friends` absent ou DF vide | `st.info(t("syn_no_data"))` |
| 5 | Solo OU Squad vide | `st.info(t("syn_no_data"))` |
| 5 | Aucune métrique calculable | `st.info(t("syn_no_data"))` |

## Config Plotly par section

| # | Config | Source |
|---|---|---|
| 1, 2 | `PLOTLY_STATIC_CONFIG` | win_loss.py:122, 163 |
| 3 | `PLOTLY_STATIC_CONFIG` | win_loss.py:184 |
| 4 | `PLOTLY_CLEAN_CONFIG` | win_loss.py:207 |
| 5 | `PLOTLY_STATIC_CONFIG` | synthesis.py:293 |

Définitions : `src/ui/streamlit_modern.py`.

## Clés i18n utilisées par la page (hors charts)

Espace de noms `t` :
- `encounters_period_label`, `encounters_period_all`, `encounters_period_2y`, `encounters_period_1y`, `encounters_period_1m`, `encounters_period_1w`
- `no_matches`
- `wl_results_by_map_mode`, `wl_by_map`, `wl_by_mode`, `wl_heatmap_title`, `wl_heatmap_caption`, `wl_top_by_week`, `wl_top_by_week_caption`
- `syn_solo_squad_title`, `syn_solo_squad_caption`, `syn_no_data`, `syn_sample_split`, `syn_solo`, `syn_squad`
- `col_win_rate`, `col_accuracy`, `col_kpm`, `col_avg_life`, `sc_performance_score`
- `insufficient_data_chart`, `missing_time_data`

Les clés par chart figurent dans le YAML correspondant.

## Pré-condition côté DataFrame

`base` est attendu issu du flux principal `data/players/{gamertag}/stats.duckdb` (v5) ou `data/titles/halo_infinite/players/{gamertag}/stats.duckdb` (v7). Colonnes minimales attendues : `match_id`, `start_time`, `outcome`, `kills`, `deaths`, `accuracy`, `time_played_seconds`, `performance_score`, `map_name`/`map_ui`, `mode_ui`/`mode_category`/`pair_name`. La colonne `is_with_friends` est ajoutée par `_attach_is_with_friends` si absente.
