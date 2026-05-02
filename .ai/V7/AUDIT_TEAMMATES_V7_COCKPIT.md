# Audit exhaustif — Page Coéquipiers (`teammates`) sur `v7/cockpit`

> Document d'audit produit le 2026-04-26 par comparaison entre la branche Python `v7/cockpit` (référence "Cockpit", la plus aboutie) et le portage Go actuel sur `feat/multi-title-adapters-and-mappings`.
>
> Objectif : énumérer **chaque** section, chart, contrôle UI et requête de la page Python pour piloter un portage Go rigoureux. Tout élément absent du portage actuel est marqué **MANQUANT GO**.

---

## 1. Vue d'ensemble

### 1.1 Hiérarchie des modules Python (`src/ui/pages/teammates*`)

| Fichier | Rôle |
|---------|------|
| `teammates.py` | Point d'entrée — KPI cards, en-tête escouade, `multiselect` (max 3 coéquipiers), aiguillage vers `render_multi_teammate_view`. |
| `teammates_views.py` | Compose la page : crée les onglets `Synergies` / `Contributions`, charge `_load_squad_data`, appelle les sous-rendus. |
| `teammates_views_shared.py` | Helpers transverses, notamment `_collect_friend_match_ids` (intersection des matchs en équipe avec chaque coéquipier). |
| `_teammates_trio.py` | Vue Escouade (1 / 2 / 3 coéquipiers) — calcule les `SquadRecordSet`, oriente vers les charts trio, médailles. |
| `_teammates_trio_helpers.py` | `_detect_trio_session`, `_render_per_minute_stats`, merge des DataFrames trio, `_render_trio_performance_charts`, médailles. |
| `teammates_charts.py` | Charts génériques : `render_metric_bar_charts` (killing spree, HS+PK), `render_outcome_bar_chart`, `render_trio_charts`, `render_first_events_chart`. |
| `teammates_map_charts.py` | Lollipop W/L par carte, Bullet winrate vs historique, Perf vs historique, Heatmap escouade par carte, Timeline performance, Cadence trio, Form Score lissé. |
| `teammates_synergy.py` | Radar de complémentarité (axes Combat / Survie / Soutien / Score / Objectifs / Impact), via `compute_participation_profile`. |
| `teammates_intensity.py` | Heatmap match × phase (10 buckets) avec `segmented_control` par joueur. |
| `teammates_impact.py` | Heatmap rôles × joueurs (8 emojis) + tableau ranking MVP/Boulet (8 colonnes, gradient couleur). |
| `teammates_weapons.py` | Tableau armes (top N, slider min kills) + barplot armes top 10 grouped, avec réinjection grenade/mêlée depuis `match_participants`. |
| `teammates_helpers.py` | `render_friends_history_table`, `render_teammate_cards` (Spartan hero cards), `render_top_matches_with_friends`. |
| `teammates_legend.py` | Panneau légende joueurs en `position:fixed` à droite, visible entre sentinelles `#llp-squad-start` et `#llp-medals-start` via `IntersectionObserver`. |

### 1.2 Services / requêtes (Python)

| Fichier | Rôle |
|---------|------|
| `src/data/services/teammates_service.py` | `TeammatesService` — `load_teammate_stats`, `load_all_teammate_stats`, `enrich_series_with_perfect_kills`, `enrich_with_performance_score`, `build_participation_profile`, `get_impact_data`. |
| `src/data/services/_teammates_first_events_queries.py` | Premier kill / première mort par match. |
| `src/data/services/_teammates_history_queries.py` | Historique complet matchs partagés. |
| `src/data/services/_teammates_impact_queries.py` | Chargement matrice impact + outcomes. |
| `src/data/services/_teammates_perf_queries.py` | `load_perf_enrichment_with_session`, `load_team_mmr_by_match`, `load_outcome_by_match`. |
| `src/data/services/_form_score_queries.py` | `load_full_performance_history` (pour Form Score lissé). |

### 1.3 Analyses / algorithmes (Python)

| Fichier | Rôle |
|---------|------|
| `src/analysis/squad_records.py` | `compute_player_pm_records`, `compute_squad_records`, `compute_squad_records_per_map`, `get_dominant_pair_name` — records "fantômes" hachurés sur les charts trio. |
| `src/analysis/friends_impact.py` | `build_impact_matrix`, `identify_silent_hero_multi`, `identify_false_brother_multi`, `identify_top_killer_multi`, `get_all_impact_events` — détection des 8 rôles. |
| `src/analysis/match_intensity.py` | `compute_match_intensity_profiles` (10 buckets normalisés), `compute_squad_cadence_profiles`. |
| `src/analysis/_performance_form.py` | `compute_form_score_history` (LOWESS, seuil `DETAIL_THRESHOLD`). |
| `src/analysis/_performance_squad.py` | `compute_squad_timeseries` — fusion perf + outcome + MMR pour la timeline. |
| `src/analysis/participation_radar.py` | `compute_participation_profile`, `RADAR_THRESHOLDS`, `RADAR_THRESHOLDS_PER_MODE`, `is_objective_mode_from_pair_name`. |

### 1.4 Visualisations Plotly (Python)

| Fichier | Sortie |
|---------|--------|
| `src/visualization/squad_map_heatmap.py` | `plot_squad_map_heatmap` — perf × carte × joueur. |
| `src/visualization/squad_performance_timeline.py` | `plot_squad_performance_timeline` — line chart multi-joueurs avec marker outcome. |
| `src/visualization/squad_cadence_chart.py` | `plot_squad_cadence_profiles` — kills par phase 60 s. |
| `src/visualization/match_intensity_heatmap.py` | `plot_match_intensity_heatmap`. |
| `src/visualization/teammates_hs_pk.py` | `plot_hs_pk_stacked` — Headshot / Perfect / Other empilés. |
| `src/visualization/_form_score.py` | `plot_form_score_history`. |
| `src/visualization/trio.py` | `plot_trio_kills_deaths`, `plot_trio_metric`, `_negative_color`. |
| `src/visualization/map_charts.py` | `plot_map_winrate_bullet`, `plot_map_perf_vs_history`, lollipop W/L. |
| `src/visualization/_chart_series.py` | `ChartData`, `MatchSeries`, `SquadRecordSet`, `add_record_overlays` (records hachurés). |

### 1.5 Tables DuckDB consultées

| Table / vue | Rôle dans la page |
|-------------|--------------------|
| `shared.match_participants` | Stats par joueur par match (kills, deaths, assists, accuracy, ratio, average_life_seconds, headshot_kills, perfect_kills, max_killing_spree, grenade_kills, melee_kills, outcome, team_mmr). |
| `shared.match_registry` (via `v_match_full`) | Map, mode, durée, date. |
| `shared.match_weapon_kills` | Kills par `weapon_id` filmé. |
| `shared.highlight_events` | Premiers kills, premières morts, médailles, finishers, etc. |
| `shared.medal_events` | Médailles agrégées (matrice galerie). |
| `shared.v_gamertag_lookup` | Résolution xuid → gamertag courant. |
| `shared.xuid_aliases` | Aliases historiques. |
| `players/{gt}/stats.duckdb#player_match_enrichment` | `performance_score`, `session_id`, `session_label`. |
| `players/{gt}/stats.duckdb#personal_score_awards` | Profil radar (combat/support/score/objectif/impact/survie). |
| `players/{gt}/stats.duckdb#sessions` | Détection session trio. |

---

## 2. Structure de navigation

La page Coéquipiers (`Mes coéquipiers`) en v7/cockpit comporte :

1. **En-tête personnel "Mes stats sur cette session"** — voir §2.1.
2. **Toolbar contextuelle** : `multiselect("tm_select_teammates")` — max 3 coéquipiers, persistance `st.session_state["teammates_picked_labels"]`.
3. **En-tête escouade — Score d'équipe + scores individuels** — voir §2.2.
4. **Légende joueurs flottante** : panneau `position:fixed` à droite, visible entre les ancres `#llp-squad-start` et `#llp-medals-start`.
5. **Onglets** :
   - **Synergies** (collectif)
   - **Contributions** (individuel comparé)

### 2.1 Bloc "Mes stats sur cette session" (KPI personnels)

Code : `teammates.py::render_teammates_page` → `subheader(t("tm_my_stats_section"))` → `render_kpis_section(dff, df)` (`src/app/kpis_render.py`).

- **Source de données** : `dff` (matchs filtrés courants) + `df` (historique complet, sert de référence pour les flèches de tendance `▲ ▼`).
- **Calcul** : `compute_kpi_stats(dff)` et `compute_kpi_stats(df)` — produit un `KPIStats` avec `total_matches`, `total_play_seconds`, `avg_match_seconds`, `kills_per_game`, `kills_per_minute`, `deaths_per_game`, `deaths_per_minute`, `assists_per_game`, `assists_per_minute`, `avg_accuracy`, `avg_life_seconds`, `wins`, `losses`, `ties`, `no_finish`.
- **Composant** : `render_combined_kpi_cards(_build_kpi_cards(kpis, kpis_all))` (`src/ui/components/kpi.py`).
- **Cartes affichées (8 tuiles)** :

| Card | `main` | `sub` | `trend` (vs all-time) |
|------|--------|-------|-----------------------|
| `kpi_selected_matches` | `total_matches` | `format_duration_hms(avg_match_seconds)` + `/match` | none |
| `kpi_total_duration` | `format_duration_dhm(total_play_seconds)` | — | none |
| `kpi_kills_per_match` | `kills_per_game` (`.2f`) | `kills_per_minute /min` | `_trend(kills_per_game, ref, higher_is_better=True)` |
| `kpi_deaths_per_match` | `deaths_per_game` | `deaths_per_minute /min` | `_trend(..., higher_is_better=False)` |
| `kpi_assists_per_match` | `assists_per_game` | `assists_per_minute /min` | `_trend(..., higher_is_better=True)` |
| `kpi_avg_accuracy` | `avg_accuracy %` | — | trend |
| `kpi_avg_lifespan` | `format_mmss(avg_life_seconds)` | — | trend |
| `mv_results` (wide) | barre empilée | W (`#3DFF9A`), L (`#FF5C5C`), T (`#A855F7`), DNF (`rgba(182,196,214,0.45)`) | none |

- **Logique trend** : `_trend(current, reference, higher_is_better, threshold=0.08)` retourne `'above'` / `'near'` / `'below'` / `'none'` selon `current/reference`. Le seuil de 8 % détermine l'apparition de la flèche colorée sur la carte.
- **Status Go** : **MANQUANT côté page Squad** — un bandeau KPI similaire existe peut-être ailleurs (ex. page Player), mais la page Squad actuelle ne l'affiche pas. À porter avec calcul de référence all-time pour les flèches.

### 2.2 Bloc "Score d'équipe + scores individuels" (en-tête escouade)

Code : `teammates.py::_render_squad_header_if_needed` → `render_squad_session_header(players_data, lang=get_lang())` (`src/ui/components/performance.py:215`).

**Condition d'affichage** : `len(dff) >= 2` AND ≥ 1 coéquipier sélectionné. Le scope du header est :
- **Sans session active** : `dff` (matchs filtrés courants).
- **Avec session active** : `df` filtré par les `match_id` issus de `base_s_ui` correspondant aux `picked_session_labels` (alignement strict avec la page Session, sans recalcul indépendant).

**Composition** :

```
┌──────────────┬──────────────┬──────────────┬──────────────┐
│ Score équipe │ Score moi    │ Score F1     │ Score F2 …   │
│  (compact)   │  (compact)   │  (compact)   │  (compact)   │
└──────────────┴──────────────┴──────────────┴──────────────┘
```

`st.columns(len(perf_list) + 1)` — 1 colonne par joueur + 1 colonne réservée à l'équipe.

**Calcul des scores individuels** : `compute_session_performance_score_v2_ui(df, include_mmr_adjustment=True)` (`src/ui/components/performance.py:40`). Pondération :
- K/D ratio normalisé (30 %)
- Win rate (25 %)
- Précision moyenne (25 %)
- Score moyen par partie normalisé (20 %)
- Ajustement MMR (`team_mmr`) optionnel.

Retourne `{score: float|None, kd_ratio, win_rate, accuracy, kills, components}`.

**Calcul du score d'équipe** : `compute_squad_performance_score(scores)` (`src/analysis/_performance_squad.py:19`).
- `base_avg` = moyenne des `score` individuels.
- **Bonus +5** si win rate moyen équipe > 60 %.
- **Bonus +5** si `min(K/D)` de l'équipe > 1.0 (cohésion : aucun joueur en dessous).
- **Bonus +3** si écart-type des kills entre joueurs < 3.0 (équilibre).
- `final = clamp(base_avg + bonuses, 0, 100)`.
- **Grade** : `resolve_squad_grade(final)` (`src/analysis/performance_config.py`) → string lettre type "S+", "A", "B", "C", "D", "F".
- Components retournés : `{base_avg, team_win_rate, min_kd, kills_std}`.

**Carte équipe (`_render_compact_team_card`)** :
- Label : `t("squad_score_header")` ("Score d'équipe").
- Score affiché : `final` (entier).
- Status : grade (lettre, font 1.6rem, bold).
- Détail bonus : si `final > base_avg` → `t("squad_score_bonus", base, bonus)` ("Base 72 (+8)") ; sinon `t("squad_score_base_only", base)` ("Base 72").
- CSS : `.os-perf-card.os-perf-card--compact`.

**Cartes joueurs (`render_performance_score_card` compact)** :
- Label : nom du joueur.
- Score : `score` (entier).
- Status : `get_score_label(score)` traduit (excellent/good/average/poor/bad).
- Couleur CSS : `get_score_class(score)` qui mappe selon `SCORE_THRESHOLDS` à `text-excellent` / `text-good` / `text-average` / `text-poor` / `text-bad`.
- **Badge comparaison** : `▲` (`text-positive`) si `player_score > avg_score`, `▼` (`text-negative`) si `player_score < avg_score`. Permet de voir d'un coup d'œil qui tire l'équipe vers le haut.

**Seuils & couleurs** : centralisés dans `SCORE_THRESHOLDS` et `get_score_color` (variables CSS `--color-excellent`, `--color-good`, `--color-average`, `--color-poor`, `--color-bad`, `--color-neutral`).

**Status Go** : **MANQUANT (bloc structurant)** — la page Squad Go n'a aucun équivalent du score d'équipe ni des cartes individuelles avec grade et badge ▲/▼. C'est pourtant l'un des éléments les plus visibles et les plus différenciants de la page Python.

---

## 3. Onglet "Synergies"

Code : `teammates_views.py::render_multi_teammate_view` puis `_render_map_breakdown` + `render_squad_heatmap` + `render_squad_timeline` + `render_squad_form_score_section` + `render_impact_taquinerie`.

### 3.1 Lollipop W/L par carte
- **Type** : barres horizontales empilées (lollipop W/L vertical).
- **Données** : `compute_map_breakdown(sub_all)` — 20 dernières cartes jouées en escouade.
- **Axes** : Y = nom de carte (ordre chronologique d'apparition), X = nombre de matchs (Wins vs Losses).
- **Source** : `sub_all` (matchs filtrés escouade) → `bd_all.head(20).reverse()`.
- **Localisé** : `lang=get_lang()`, `map_ui` (traduit) avec fallback `map_name`.
- **Status Go** : **MANQUANT** — pas trouvé dans `apps/web/src/features/squad/`.

### 3.2 Bullet winrate courant vs historique
- **Section** : titre `tm_map_bullet_title`.
- **Type** : 3 barres empilées par carte (winrate session, winrate historique escouade, ligne de référence joueur).
- **Données** : `view` (top 20 cartes) + `bd_history` (`_compute_history_breakdown(full_pl)`).
- **Fonction** : `plot_map_winrate_bullet(view, bd_history, lang, map_order)`.
- **Status Go** : **MANQUANT**.

### 3.3 Perf vs historique par carte
- **Section** : titre `tm_perf_vs_history_title`.
- **Type** : barres horizontales delta — `Δ performance_score (session - historique)` par carte.
- **Fonction** : `plot_map_perf_vs_history(view, bd_history, lang, map_order)`.
- **Status Go** : **MANQUANT**.

### 3.4 Heatmap escouade carte × joueur
- **Section** : titre `tm_map_squad_heatmap_title`.
- **Type** : heatmap 2D (cartes × joueurs), couleur = performance_score normalisé.
- **Condition** : `len(series) >= 2`.
- **Fonction** : `plot_squad_map_heatmap(series, lang)` (`series` = `[(name, df), ...]`).
- **Status Go** : présent mais simpliste — `apps/web/src/features/squad/charts/heatmapChart.ts` (~72 L) construit une heatmap cartes × win rate, **sans dimension joueur séparée**. **PARTIELLEMENT PORTÉ — à enrichir** (passer à un vrai 2D player × map).

### 3.5 Timeline performance escouade
- **Section** : titre implicite via le chart.
- **Type** : line chart multi-joueurs, X = chronologie matchs, Y = `performance_score`, marker = outcome (W/L/T), couleur = joueur (Okabe-Ito).
- **Données** : `compute_squad_timeseries(series)` à partir de `me_df` enrichi (`team_mmr`, `outcome` depuis shared).
- **Fonction** : `plot_squad_performance_timeline(ts, lang)`.
- **Status Go** : présent — `charts/timelineChart.ts` (~79 L) trace `perf` + `winRate` mais **sans dimension multi-joueurs ni marker outcome**. **PARTIELLEMENT PORTÉ**.

### 3.6 Form Score lissé (LOWESS)
- **Section** : `render_squad_form_score_section(series, db_path, sub_all, colors_by_name, xuid)`.
- **Type** : line chart LOWESS — évolution lissée du performance_score escouade vs joueur principal.
- **Données** : `load_full_performance_history(db_path, xuid)` + `compute_form_score_history(history, threshold=DETAIL_THRESHOLD)`.
- **Fonction** : `plot_form_score_history(history, lang)`.
- **Status Go** : **MANQUANT**.

### 3.7 Impact "taquinerie"
- **Section** : `render_impact_taquinerie(db_path, xuid, match_ids, friend_xuids, db_key)` — déclenchée si ≥ 1 coéquipier sélectionné.
- **Composants** :
  - **Heatmap rôles × joueurs** — chaque cellule contient les emojis des événements détectés sur le match : `⚡ first_blood`, `🎯 clutch_finisher`, `💀 last_casualty`, `🐌 last_group_kill`, `🪦 first_group_death`, `🛡️ silent_hero`, `🗡️ false_brother`, `💥 top_killer`. Couleur de fond = outcome (vert win, orange loss, gris tie).
  - **Tableau ranking MVP / Boulet** — 8 colonnes (1 par type d'événement), gradient couleur Okabe-Ito selon le score (`SCORE_SILENT_HERO`, `SCORE_FALSE_BROTHER`, `SCORE_TOP_KILLER`, etc.), inversion pour les rôles "négatifs" (`_IMPACT_INVERTED = {last_casualty, last_group_kill, first_group_death, false_brother}`).
  - **Toggle viz** : `tmi_viz_heatmap` (matrice originale) ou `tmi_viz_scatter` (points/symboles).
  - **Popover légende** : `tm_impact_legend` (8 lignes markdown).
- **Données** : `build_impact_matrix(events_df, match_outcomes)` + `_pivot_matrix_cells` + `_player_agg_counts`.
- **Source** : `shared.highlight_events` + `shared.match_participants` (assists/deaths/kills/outcome).
- **Status Go** : **MANQUANT (gros morceau)** — toute la mécanique des 8 rôles, la heatmap d'emojis et le ranking MVP/Boulet sont absents.

### 3.8 Tableau historique escouade (sous Synergies, hors heatmaps)
- **Section** : `subheader("tm_history")` + caption `tm_history_tz_caption`.
- **Fonction** : `render_friends_history_table(sub_all, db_path, xuid, db_key, waypoint_player, full_df=full_squad_df)` — tableau HTML (`os-table`) listant chaque match commun : carte, mode (`normalize_mode_label`), playlist (`translate_playlist_name`), date locale (`FMT_DATETIME_FR`), résultat, lien Waypoint.
- **Status Go** : **MANQUANT** (la table de l'onglet Squad actuel n'est pas la même : c'est la liste des coéquipiers, pas l'historique des matchs avec eux).

### 3.9 Cadence trio (timeline kills/phase) — uniquement vue escouade
- **Section** : `render_squad_cadence_section(db_path, xuid_name_map, all_match_ids, lang, color_map)`.
- **Type** : barres empilées par phase de 60 s, normalisées sur l'ensemble des matchs — révèle la "cadence" de chaque joueur.
- **Données** : `cached_load_kill_timing_for_matches` + `compute_squad_cadence_profiles`.
- **Fonction** : `plot_squad_cadence_profiles(events_df, opts)`.
- **Note post-graphe** : `tm_note_cadence`.
- **Status Go** : **MANQUANT**.

---

## 4. Onglet "Contributions"

Code : `_teammates_trio.py::render_trio_view` (vue 2-3-4 joueurs) + `_render_bottom_charts` (vue 1+ coéquipiers) + médailles.

### 4.1 Stats par minute (header escouade)
- **Section** : `subheader("tm_per_minute")`.
- **Type** : barres groupées — pour chaque joueur, 3 barres : `frags/min`, `morts/min` (orientées vers le bas en valeur négative, couleur teinte sombre du joueur via `_negative_color`), `assists/min`. Labels absolus.
- **Records overlay** : si `app_settings.show_records=True`, ajout de `pm_records` (records joueur global) → barres fantômes hachurées via `ChartData.add_record_overlays`.
- **Style** : `apply_halo_plot_style`, axe zéro forcé blanc (`zerolinecolor="rgba(255,255,255,0.75)"`, width 2).
- **Source** : `compute_aggregated_stats(df)` par joueur.
- **Status Go** : **MANQUANT**.

### 4.2 Sentinelle légende start (`#llp-squad-start`) + Radar complémentarité escouade
- **Section** : `render_trio_synergy_radar(...)` — radar à 6 axes (Combat, Survie, Soutien, Score, Objectifs, Impact), normalisé via `RADAR_THRESHOLDS_PER_MODE` selon famille de mode (Slayer / CTF / Strongholds / Oddball / Custom).
- **Fonction** : `create_participation_profile_radar(profiles)` + `compute_participation_profile`.
- **Source** : `personal_score_awards` (par DB joueur), pondération par `n_matches`.
- **Note post-graphe** : `tm_note_radar`.
- **Status Go** : présent mais **différent** — `SquadContributionsPage` trace un `scatterpolar` à partir des champs `SQUAD_RADAR_METRICS` (issus de `fields.toml`), normalisation 0–100 simple. Pas de seuils par famille de mode, pas de notion `objective_score` vs `kill_score`. **PORTAGE INCOMPLET — refonte nécessaire**.

### 4.3 Charts performance trio
Section : `_render_trio_performance_charts` → `render_trio_charts` (`teammates_charts.py`).

Ordre des charts :
1. **Frags ↑ / Morts ↓ combinés** (`tm_kills_deaths`) — `plot_trio_kills_deaths(d_self, d_f1, d_f2, names, opts, d_f3, colors_by_name, squad_records)`. Barres groupées kills (positif) + deaths (négatif), records hachurés.
2. **Assists** (`tm_assists`) — `plot_trio_metric(metric="assists")`.
3. **KDA Ratio** (`tm_kda`, format `.3f`) — `plot_trio_metric(metric="ratio")`.
4. **Précision** (`tm_accuracy`, suffixe `%`, format `.2f`) — `plot_trio_metric(metric="accuracy")`.
5. **Vie moyenne** (`tm_avg_life`, format `.1f`, axe `tm_seconds`) — `plot_trio_metric(metric="average_life_seconds")`.
6. **Score de performance** (`tm_performance`, format `.1f`) — `plot_trio_metric(metric="performance")`.

Tous les charts : légende masquée via `_hide_legend` (les noms apparaissent dans le panneau latéral fixe), couleurs `colors_by_name`, records `SquadRecordSet` (overlay hachuré).

- **Status Go** : **MANQUANT** pour la majorité — `SquadSynergiesPage` propose un seul barplot multi-métriques générique (`buildSynergiesChart`) sans la finesse des 6 charts dédiés ni les overlays records.

### 4.4 Killing spree (max) + HS+PK stacked
- **Killing Spree** : `render_metric_bar_charts` → `plot_fn(series, metric_col="max_killing_spree", smooth_window=10, show_smooth_lines=show_smooth, squad_records)`. Subheader `tm_killing_spree`.
- **HS + PK stacked** : `plot_hs_pk_stacked(series, colors, lang, records=hspk_records)` — stacked bar par match : Headshot kills + Perfect kills + Other kills. Subheader `viz_t("hs_pk_combined_title", lang)`.
- **Records** : `compute_squad_records(series, [("max_killing_spree", False)], dominant_pair)` et `compute_squad_records(hspk_dfs, [("hs_pk_total", False)], dominant_pair)`.
- **Status Go** : `charts/hsPkChart.ts` (~79 L) existe — **PARTIELLEMENT PORTÉ** (probablement sans records hachurés ni courbe lissée smoothing). `Killing Spree` : **MANQUANT** côté Go.

### 4.5 Heatmap intensité (match × phase)
- **Section** : `render_squad_intensity_heatmap(db_path, xuid_name_map, match_ids_ordered, lang, me_df)`.
- **Type** : heatmap match × 10 buckets de phase, valeur = profil de kills normalisé.
- **UI** : `segmented_control(t("tm_intensity_player_select"))` — toggle "Tous" / `joueur1` / `joueur2` / `joueur3`.
- **Y labels** : carte (si dispo via `prepare_time_axis`) sinon date.
- **Source** : `cached_load_kill_timing_for_matches` → `pl.DataFrame` → `compute_match_intensity_profiles(player_events, n_buckets=10)` → tri chronologique.
- **Visualisation** : `plot_match_intensity_heatmap(profile, opts, match_labels)`.
- **Conditions** : `len(match_ids_ordered) >= 3` et `xuid_name_map` non vide.
- **Status Go** : **MANQUANT**.

### 4.6 Premier événement (First Events) — optionnel
- **Section** : `render_first_events_chart` — premier kill / première mort par match, position chronologique relative.
- **Branche origine** : `feat/teammates-first-events-chart` mergée dans v7/cockpit.
- **Status Go** : **MANQUANT**.

### 4.7 Tableau armes
- **Section** : `subheader(t("section_weapon_stats"))`.
- **Type** : tableau HTML (`os-table`) — colonnes : Arme | Faction | Me | F1 | F2 | F3 | Total. Tri par kills décroissant.
- **Filtre** : slider min kills + filtre `_FILM_EXCLUDED_IDS` (grenades/mêlée filmés à exclure puis réinjectés via API).
- **Réinjection** : `_append_grenade_melee` lit `shared.match_participants.grenade_kills` + `melee_kills`, ajoute lignes synthétiques `_GRENADE_WEAPON_ID` et `_MELEE_WEAPON_ID`. Cap par `remainder = api_total - film_kills` pour éviter le double comptage.
- **Source** : `repo.load_weapon_kills_aggregated(xuid, match_ids)` + `repo.load_grenade_melee_kills` + `repo.load_total_kills_for_player`.
- **Localisation** : `_resolve_weapon_name` via `resolve_weapon_display(wid, lang)` (table `weapon_labels`).
- **Status Go** : **MANQUANT**.

### 4.8 Barplot armes top 12 grouped
- **Section** : `render_weapon_kills_bar_chart(player_infos, colors_by_name, db_path, key_suffix)`.
- **Type** : barres groupées top 12 armes × N joueurs.
- **Caption** : `tm_weapons_chart_caption`.
- **Status Go** : **MANQUANT**.

### 4.9 Médailles (galerie matchs partagés)
- **Sentinelle légende stop** : `<div id="llp-medals-start">` (cache le panneau légende fixe).
- **Section** : `_render_trio_medals(match_ids, db_path, xuid, f1_xuid, f2_xuid, me_name, f1_name, f2_name, db_key, top_medals_fn, f3_xuid, f3_name)`.
- **Type** : galerie de cartes match — pour chaque match commun (top 20) : carte miniature, date, joueurs de l'escouade, icônes des médailles principales, outcome, lien Waypoint.
- **Helper** : `render_medals_grid` (`src/ui/medals.py`).
- **Status Go** : **MANQUANT**.

---

## 5. Conventions visuelles transverses

| Élément | Référence Python | À reproduire en Go |
|---------|------------------|--------------------|
| Palette joueurs | `OKABE_ITO_PALETTE` (`src/config`) | Utiliser `getSeriesColors(n, ['narrative-dominant', 'perf-tier-3', ...])` ou ajouter un token `okabe-ito-*` dans `apps/web/src/lib/accessibility/palettes/`. |
| Couleurs outcome | `_OUTCOME_BG = {2: rgba(0,158,115,0.30), 3: rgba(213,94,0,0.30)}` ; tie `rgba(100,100,130,0.15)` | Mapper sur tokens sémantiques `outcome.win` / `outcome.loss` / `outcome.tie`. |
| Style Plotly | `apply_halo_plot_style(fig, height)` + `PLOTLY_CLEAN_CONFIG` / `PLOTLY_STATIC_CONFIG` | Wrapper TS équivalent (margins, axes, fond transparent, font Halo). |
| Légende désactivée | `_hide_legend(fig)` (boîte globale + traces) | Toujours `legend.visible = false` quand le panneau latéral fixe affiche les noms. |
| Records hachurés | `SquadRecordSet.add_record_overlays(fig)` — barres fantômes via `ChartData` | Implémenter un overlay similaire (motif `pattern_shape`). |
| Sessions trio | `_detect_trio_session` → caption `tm_trio_session(label=...)` | À reproduire si on porte la vue trio. |
| Légende joueurs flottante | `position:fixed`, `IntersectionObserver` sur `#llp-squad-start` / `#llp-medals-start` | À porter (composant React + observer). |
| Captions info-layer | `hints_visible()` + `tm_*_caption` | Système d'astuces déjà présent côté Go ? À aligner. |

---

## 6. Inventaire des clés i18n (sélection — fichier complet : `src/ui/i18n/pages/teammates.py`)

Section UI / titres :
- `tm_my_stats_section`, `tm_squad_section`, `tm_squad_header`, `tm_select_teammates`, `tm_select_teammate`
- `tab_synergies`, `tab_contributions`
- `tm_history`, `tm_history_tz_caption`, `tm_no_matches_filter`, `tm_not_enough_matches`
- `tm_by_map`, `tm_map_bullet_title`, `tm_perf_vs_history_title`, `tm_map_squad_heatmap_title`
- `tm_per_minute`, `tm_kills_deaths`, `tm_assists`, `tm_kda`, `tm_accuracy`, `tm_avg_life`, `tm_performance`, `tm_score`
- `tm_killing_spree`, `tm_headshots`, `tm_perfect_kills`, `tm_metric_frags_min`, `tm_metric_deaths_min`, `tm_metric_assists_min`
- `tm_intensity_title`, `tm_intensity_caption`, `tm_intensity_player_select`, `tm_intensity_all`, `tm_intensity_no_data`, `tm_intensity_match_count`
- `tm_medals`, `tm_medals_all`, `tm_no_shared_medals`, `tm_no_medals_aggregate`, `tm_computing_medals`, `tm_computing_medals_all`

Impact :
- `tm_impact_header`, `tm_impact_select_two`, `tm_impact_no_matches`, `tm_impact_no_events_matches`, `tm_impact_no_events_players`
- `tm_impact_heatmap`, `tm_impact_ranking`, `tm_impact_legend`, `tm_impact_caption`
- `tmi_viz_heatmap`, `tmi_viz_scatter`

Trio / sessions :
- `tm_trio_session`, `tm_trio_session_unknown`, `tm_trio_warning`, `tm_no_trio_matches`, `tm_trio_header`

Notes / captions :
- `tm_kd_half_caption`, `tm_metrics_caption`, `tm_weapons_chart_caption`, `tm_weapons_no_data`, `tm_note_radar`, `tm_note_cadence`

Spinners :
- `tm_computing_teammate`, `tm_computing_map`, `tm_computing_stats`, `tm_loading_slow`

> Note : les clés `tm_kills`, `tm_deaths`, `tm_kda`, `tm_accuracy`, `tm_avg_life`, `tm_performance` sont des **alias** vers `col_*` de `common.py`.

---

## 7. État actuel du portage Go

### 7.1 Surface du code Go

| Fichier | Lignes |
|---------|--------|
| `apps/web/src/features/squad/SquadSynergiesPage.tsx` | 227 |
| `apps/web/src/features/squad/SquadContributionsPage.tsx` | 143 |
| `apps/web/src/features/squad/SquadLayout.tsx` | (à inspecter) |
| `apps/web/src/features/squad/SquadContext.ts` | (contexte React) |
| `apps/web/src/features/squad/charts/heatmapChart.ts` | 72 |
| `apps/web/src/features/squad/charts/hsPkChart.ts` | 79 |
| `apps/web/src/features/squad/charts/timelineChart.ts` | 79 |
| `apps/web/src/features/squad/queries.ts` | 25 |
| `apps/web/src/features/squad/metrics.ts` | 101 |
| **Total approximatif** | **~931 L** |

À comparer aux **~6 000 L** Python sur les modules teammates_*.

### 7.2 Charts Go actuels

| Chart Go | Équivalent Python | État |
|----------|-------------------|------|
| `buildSynergiesChart` (barplot multi-métriques générique) | aucun équivalent direct — remplace de loin les 6 charts dédiés trio | non-fidèle (1 chart au lieu de 6) |
| `buildHsPkChart` | `plot_hs_pk_stacked` | partiellement porté (à vérifier records + smoothing) |
| `buildTimelineChart` | `plot_squad_performance_timeline` | partiellement porté (perf + winrate, mais pas de multi-joueurs ni marker outcome) |
| `buildHeatmapChart` | `plot_squad_map_heatmap` | partiellement porté (cartes seules, pas la dimension joueur) |
| `buildRadarChart` | `create_participation_profile_radar` | refonte nécessaire (axes génériques au lieu des 6 axes Combat/Survie/Soutien/Score/Objectifs/Impact normalisés par mode) |

---

## 8. Checklist de portage Go (priorisée)

### Phase 0 — En-têtes de page (à porter en priorité, structurels)
- [ ] **Bandeau KPI personnels** "Mes stats sur cette session" (8 cartes : matchs, durée totale, K/match, D/match, A/match, accuracy, vie moyenne, barre W/L/T/DNF) avec flèches de tendance vs all-time (`_trend`, seuil 8 %).
- [ ] **En-tête escouade** : carte "Score d'équipe" (score 0-100 + grade lettre + détail bonus) + cartes individuelles compactes (score + label qualitatif + badge ▲/▼ vs moyenne équipe).
- [ ] Endpoint Go `/players/{gt}/kpi-stats?scope=current|alltime` retournant `KPIStats`.
- [ ] Endpoint Go `/squad/perf-score?xuids=...&matchIds=...` retournant `{players: [{name, score, kd_ratio, win_rate, components}], squad: {score, grade, components: {base_avg, team_win_rate, min_kd, kills_std}}}`.
- [ ] Porter `compute_session_performance_score_v2` (pondération K/D 30 %, WR 25 %, accuracy 25 %, score 20 % + ajustement MMR) en `internal/analysis/performance/`.
- [ ] Porter `compute_squad_performance_score` (base + 3 bonus : winrate >60 %, min K/D >1, kills_std <3).
- [ ] Porter `resolve_squad_grade` (mapping score → lettre).

### Phase 1 — Charts collectifs Synergies (MVP)
- [ ] Lollipop W/L par carte (`plot_map_winrate_bullet` côté Python, vue 20 dernières cartes).
- [ ] Bullet winrate session vs historique escouade.
- [ ] Perf vs historique par carte (delta `performance_score`).
- [ ] Refonte heatmap escouade pour passer à `joueur × carte` (et plus seulement carte × winrate).
- [ ] Timeline performance multi-joueurs avec marker outcome (W/L/T) — étendre `timelineChart.ts`.
- [ ] Tableau historique escouade (`render_friends_history_table`) — date locale, mode normalisé, lien Waypoint.

### Phase 2 — Impact (gros morceau)
- [ ] Détection des 8 rôles (`first_blood`, `clutch_finisher`, `last_casualty`, `last_group_kill`, `first_group_death`, `silent_hero`, `false_brother`, `top_killer`) côté Go (`internal/analysis/impact/`).
- [ ] Endpoint Go `/squad/impact` retournant la matrice (`gamertag → match_id → events[]`) + outcomes.
- [ ] Heatmap rôles × joueurs avec emojis et fond outcome.
- [ ] Tableau ranking 8 colonnes avec gradient Okabe-Ito (inversion pour rôles "négatifs").
- [ ] Toggle viz heatmap / scatter.
- [ ] Popover légende.

### Phase 3 — Charts performance trio (Contributions)
- [ ] Stats par minute groupées (3 barres par joueur, axe zéro blanc, deaths inversés).
- [ ] Frags ↑ / Morts ↓ combinés (`plot_trio_kills_deaths`).
- [ ] Charts dédiés Assists, KDA, Accuracy, Avg Life, Performance Score (5 charts séparés au lieu d'un barplot générique).
- [ ] Killing Spree (max) avec smoothing 10 matchs.
- [ ] Compléter `hsPkChart` avec records overlay et smoothing.

### Phase 4 — Form Score, Cadence, Intensité
- [ ] Form Score lissé (LOWESS) — endpoint + chart.
- [ ] Cadence trio (kills par phase 60 s).
- [ ] Heatmap intensité match × 10 buckets avec `segmented_control` joueur.

### Phase 5 — Radar complémentarité (refonte)
- [ ] Axes : Combat / Survie / Soutien / Score / Objectifs / Impact.
- [ ] Normalisation par famille de mode (`RADAR_THRESHOLDS_PER_MODE`, `is_objective_mode_from_pair_name`).
- [ ] Source : `personal_score_awards` agrégés par scope.
- [ ] Note post-graphe (`tm_note_radar`).

### Phase 6 — Armes
- [ ] Tableau top N armes avec colonnes Me / F1 / F2 / F3 / Total + slider min kills.
- [ ] Réinjection grenade/mêlée capée par `remainder = api_total - film_kills`.
- [ ] Localisation noms d'armes via `weapon_labels` (déjà géré côté Go via `assets.toml` ?).
- [ ] Barplot top 12 grouped.

### Phase 7 — Médailles
- [ ] Galerie médailles (top 20 matchs partagés, carte/date/squad/icônes/outcome/Waypoint link).

### Phase 8 — UX transverse
- [ ] Panneau légende joueurs flottant (`position:fixed`, `IntersectionObserver` sur sentinelles).
- [ ] Sessions trio détectées (caption `tm_trio_session`).
- [ ] Notes post-graphe (`tm_note_radar`, `tm_note_cadence`, `tm_kd_half_caption`).
- [ ] Records overlays hachurés (motif `pattern_shape` Plotly).
- [ ] Style Halo (`apply_halo_plot_style`) — wrapper TS centralisé.

---

## 9. Tables / endpoints à exposer côté Go

| Donnée | Source DuckDB | Endpoint Go suggéré |
|--------|---------------|---------------------|
| Stats par joueur sur matchs communs | `shared.match_participants` filtré par `match_id IN (...)` | `GET /squad/participants?xuids=...&matchIds=...` |
| Highlight events (impact) | `shared.highlight_events` + `v_gamertag_lookup` | `GET /squad/impact?xuids=...&matchIds=...` |
| Outcomes par match pour le joueur principal | `shared.match_participants` filtré `xuid=me` | `GET /squad/outcomes?xuid=me&matchIds=...` |
| Performance score + session enrichies | `players/{gt}/stats.duckdb#player_match_enrichment` | `GET /players/{gt}/perf-enrichment?matchIds=...` |
| Profil participation (radar) | `personal_score_awards` | `GET /players/{gt}/personal-score-awards?matchIds=...` |
| Kill timing (intensité, cadence) | `shared.match_kill_events` (ou `v_highlight_events`) | `GET /squad/kill-timing?xuids=...&matchIds=...` |
| Médailles | `shared.medal_events` | `GET /squad/medals?xuids=...&matchIds=...` |
| Armes | `shared.match_weapon_kills` + `match_participants.grenade_kills/melee_kills` | `GET /squad/weapons?xuid=...&matchIds=...` |

---

## 10. Synthèse

| Indicateur | Python v7/cockpit | Go actuel |
|------------|------------------|-----------|
| Modules | ~15 | 5 (1 layout, 2 pages, 1 metrics, 1 queries) + 3 charts |
| Lignes de code (page) | ~6 000 | ~930 |
| Charts/sections distincts | ~22 | 5 (heatmap, hsPk, timeline, synergies bar, radar) |
| Onglets | 2 (Synergies / Contributions) | 2 (mêmes noms) |
| Bandeau KPI personnels (8 cartes + tendance) | Oui (`render_kpis_section`) | Non |
| Score d'équipe + grade lettre + bonus | Oui (`compute_squad_performance_score`) | Non |
| Cartes scores individuels avec badge ▲/▼ | Oui (`render_performance_score_card` compact) | Non |
| Vue trio dédiée | Oui (`_teammates_trio.py`) | Non |
| Heatmap impact 8 rôles | Oui | Non |
| Form Score lissé | Oui | Non |
| Cadence + Intensité | Oui | Non |
| Tableau historique escouade | Oui | Non |
| Tableau armes + barplot | Oui | Non |
| Galerie médailles | Oui | Non |
| Légende flottante observer | Oui | Non |
| Records hachurés | Oui (`SquadRecordSet`) | Non |

**Conclusion** : la version Go actuelle représente environ **20 à 25 %** de la richesse fonctionnelle de la page Python `v7/cockpit`. Les blocs structurants manquants (Impact, charts trio dédiés, Form Score, Intensité, Armes, Médailles, Tableau historique) doivent être planifiés et portés selon l'ordre de la checklist (section 8). Le radar et la heatmap actuels nécessitent en outre une refonte — ils existent mais ne reflètent pas la sémantique de la version Python.

---

## 11. Faisabilité du portage avec l'état actuel du code

Cette section évalue, pour chaque chart/section, ce qu'on a déjà côté Go (vérifié à la lecture du code) et ce qu'il manque pour atteindre la parité avec la version Python `v7/cockpit`.

### 11.1 Inventaire de l'existant Go (vérifié)

**Analyse** ([apps/go-api/internal/analysis/](apps/go-api/internal/analysis/)) :
- [squad_score.go:21](apps/go-api/internal/analysis/squad_score.go#L21) — `ComputeSquadPerformanceScore(scores []dict)` : porte la logique base + 3 bonus + clamp + grade lettre. **Quasiment complet.**
- [squad_score.go:110](apps/go-api/internal/analysis/squad_score.go#L110) — `resolveSquadGrade(score float64) string`.
- [squad_impact.go:18](apps/go-api/internal/analysis/squad_impact.go#L18) — `ComputeImpactSummary(events, myXUID, friendXUID)` : **4 rôles seulement** (FirstBlood, Clutch, LastKill, FirstDeath), bilatéral 1v1. Le clutch est même approximé via "dernier tiers de la liste de kills" et non vraie fenêtre 30 s finales.
- [performance_score.go](apps/go-api/internal/analysis/performance_score.go) — `ComputeSessionPerformanceScore` (test présent : performance_score_test.go).
- [highlight_event_parser.go](apps/go-api/internal/analysis/highlight_event_parser.go) — parser des événements (utilisable pour étendre l'impact).
- Pas de `LOWESS`, pas de `ComputeMatchIntensityProfiles`, pas de `ComputeSquadCadenceProfiles`, pas de `ComputeMapBreakdown`, pas de `ComputeSquadRecords`, pas de `compute_participation_profile`.

**Repository** ([apps/go-api/internal/platform/duckdb/squad_repo.go](apps/go-api/internal/platform/duckdb/squad_repo.go)) :
- `LoadTopTeammates`, `LookupXUIDByGamertag` ✅
- `LoadSquadMatches(playerXUID, teammateXUID)` ✅ — Q30 retourne déjà `outcome, kills, deaths, assists, kda, accuracy, time_played_seconds, team_mmr, headshot_kills, perfect_kills, performance_score, session_id, map_ui, pair_name, playlist_name, is_firefight, is_ranked` (très riche).
- `LoadTeammateMatches` ✅ — Q31, équivalent pour le coéquipier.
- `LoadImpactEvents(matchIDs)` ✅ — Q32, lit `shared.highlight_events` filtré sur match_ids.
- `LoadSynthesisHeatmap`, `LoadSynthesisMatches` ✅ (utilisé par Synthèse, pas Squad).
- **Manquants** : pas de `LoadKillTimingForMatches`, pas de `LoadWeaponKillsAggregated`, pas de `LoadMedalsForMatches(matchIDs)` (seul `LoadMedalTotals(xuid)` existe via citations_repo), pas de `LoadPersonalScoreAwards`, pas de `LoadFormScoreHistory`.

**Service** ([apps/go-api/internal/service/teammates_service.go](apps/go-api/internal/service/teammates_service.go), [squad_service.go](apps/go-api/internal/service/squad_service.go)) :
- `buildMatchSeries(matches) []SquadMatchSeriesPoint` ✅ — base pour Timeline.
- `computeKPIsFromSquadMatches` ✅ — agrège par teammate.
- `TeammatesPageResponse{Teammates, MatchSeries, MapBreakdown?}` — squelette OK, `MatchSeries` exposé par gamertag.

**Frontend** ([apps/web/src/features/squad/](apps/web/src/features/squad/)) :
- `SquadLayout.tsx`, `SquadContext.ts` ✅ — sélection coéquipiers, scope.
- `SquadSynergiesPage.tsx` (227 L) — 1 barplot générique + heatmap 1D + timeline 2-traces + HS/PK.
- `SquadContributionsPage.tsx` (143 L) — 1 radar 4-6 axes génériques.
- `charts/heatmapChart.ts`, `charts/timelineChart.ts`, `charts/hsPkChart.ts` (~230 L à eux trois).
- `metrics.ts`, `queries.ts` — wiring.
- **Manquants** : pas de `RecordOverlay`, pas de `SegmentedControl` (à confirmer dans shadcn), pas de panneau légende flottante, pas de composant galerie médailles, pas de tableau armes.

### 11.2 Tableau récapitulatif (22 éléments)

Légende : **IMMÉDIAT** = tout existe, juste à wirer / **EFFORT MOYEN** = 1-2 briques manquantes / **EFFORT IMPORTANT** = algo + endpoint + UI à créer / **BLOQUÉ** = données manquantes en DB (au-delà de 3j).

| Section | Chart / Section | Faisabilité | Effort | Bloqueur principal |
|---------|-----------------|-------------|--------|--------------------|
| §2.1 | KPI personnels (8 cartes + tendance) | EFFORT MOYEN | 1.5j | Endpoint `kpi-stats` + référence all-time + barre W/L/T/DNF |
| §2.2 | Score équipe + grade + cartes ▲▼ | EFFORT MOYEN | 1j | Endpoint `/squad/perf-score` + UI 4 cartes (algo Go déjà porté) |
| §3.1 | Lollipop W/L par carte | EFFORT MOYEN | 1j | `ComputeMapBreakdown()` + chart Plotly |
| §3.2 | Bullet winrate session vs historique | EFFORT MOYEN | 1j | Aggregation historique escouade par carte |
| §3.3 | Perf vs historique par carte | EFFORT MOYEN | 1j | Delta perf_score session vs historique |
| §3.4 | Heatmap escouade joueur × carte | EFFORT MOYEN | 1j | Refonte chart 1D → 2D (données déjà chargées) |
| §3.5 | Timeline multi-joueurs + marker outcome | EFFORT MOYEN | 1j | Extension du chart actuel (multi-traces + symbole outcome) |
| §3.6 | Form Score lissé (LOWESS) | EFFORT IMPORTANT | 2j | LOWESS absent en Go (à implémenter ou wrapper `gonum`) |
| §3.7 | Impact 8 rôles (heatmap + ranking) | EFFORT IMPORTANT | 3j | Algo Go ne couvre que 4 rôles (vs 8) ; modèle bilatéral à généraliser N joueurs |
| §3.8 | Tableau historique escouade | EFFORT MOYEN | 1j | Composant React + formatters (date FR, mode, lien Waypoint) |
| §3.9 | Cadence trio (kills/phase 60 s) | EFFORT IMPORTANT | 2j | `LoadKillTimingForMatches` + algo `ComputeSquadCadenceProfiles` |
| §4.1 | Stats par minute groupées | EFFORT MOYEN | 1j | Normalisation par minute + chart 3 barres (deaths inversées) |
| §4.2 | Radar 6 axes normalisés par mode | EFFORT IMPORTANT | 2j | Refonte radar + endpoint `personal_score_awards` + thresholds par mode |
| §4.3 | 6 charts performance trio dédiés | EFFORT IMPORTANT | 3j | 5-6 charts au lieu d'un barplot générique + records overlay |
| §4.4 | Killing Spree + HS/PK enrichis | EFFORT MOYEN | 1j | KS chart absent ; HS/PK existant à étendre (records + smoothing) |
| §4.5 | Heatmap intensité (match × 10 buckets) | EFFORT IMPORTANT | 2j | Idem §3.9 (kill timing) + algo buckets + segmented_control |
| §4.6 | First Events | EFFORT MOYEN | 1j | Données déjà chargeables via `LoadImpactEvents` (FirstBlood/FirstDeath sont déjà parsés) |
| §4.7 | Tableau armes (top N + grenade/mêlée) | EFFORT IMPORTANT | 2j | Repo `LoadWeaponKillsAggregated` + cap remainder grenade/mêlée |
| §4.8 | Barplot armes top 12 grouped | EFFORT MOYEN | 0.5j | Dérivé direct de §4.7 |
| §4.9 | Galerie médailles (top 20) | EFFORT IMPORTANT | 2j | Repo `LoadMedalsForMatches(matchIDs)` (vs `LoadMedalTotals(xuid)` actuel) + composant galerie |

### 11.3 Briques transverses à construire en premier

Ces helpers sont consommés par plusieurs charts — les implémenter avant tout chart évite d'y revenir N fois.

**Algos manquants** ([apps/go-api/internal/analysis/](apps/go-api/internal/analysis/)) :
- `ComputeMapBreakdown(matches []SquadMatchRow) []MapBreakdownRow` — utilisé par §3.1, §3.2, §3.3, §3.4. ~0.5j.
- `ComputeSquadRecords(series, metrics, dominantPair) SquadRecordSet` — utilisé par §4.3, §4.4, §4.1 (stats/min). ~1j.
- `LOWESS(points, alpha)` — utilisé par §3.6 uniquement. **Bloqueur isolé** ~2j (impl from-scratch ou wrapper [gonum/stat/regression](https://pkg.go.dev/gonum.org/v1/gonum/stat) ; pas d'équivalent direct mais polynomial regression locale est faisable).
- `ComputeMatchIntensityProfiles(events, nBuckets=10)` — §4.5. ~0.5j.
- `ComputeSquadCadenceProfiles(events, phaseSeconds=60)` — §3.9. ~0.5j.
- `ComputeParticipationProfile(scores, options)` + `RADAR_THRESHOLDS_PER_MODE` — §4.2. ~1j.
- `ComputeSessionPerformanceScoreV2(matches, includeMMRAdjustment)` — §2.2 (à confirmer si déjà dans `performance_score.go`). ~0.5j.
- Étendre `ComputeImpactSummary` à 8 rôles + N joueurs — §3.7. ~2j (silent_hero, false_brother, top_killer, last_casualty, last_group_kill, first_group_death + fenêtre temporelle réelle au lieu d'une approximation par tiers).

**Méthodes repository manquantes** ([apps/go-api/internal/platform/duckdb/squad_repo.go](apps/go-api/internal/platform/duckdb/squad_repo.go)) :
- `LoadKillTimingForMatches(ctx, matchIDs []string) ([]KillTimingRow, error)` — `shared.match_kill_events` ou parsing des `highlight_events` de type Kill avec `time_ms`. **Bloque §3.9 et §4.5.** ~0.5j.
- `LoadWeaponKillsAggregated(ctx, xuid, matchIDs)` — `shared.match_weapon_kills` + jointure `weapon_labels`. ~0.5j.
- `LoadGrenadeMeleeKills(ctx, xuid, matchIDs)` — `shared.match_participants.{grenade_kills, melee_kills}`. ~0.2j.
- `LoadMedalsForMatchesByXUID(ctx, xuids, matchIDs) ([]MedalRow, error)` — `shared.medals_earned` (existe mais pas exposé filtré par matchIDs côté squad). ~0.3j.
- `LoadPersonalScoreAwards(ctx, xuid, matchIDs)` — table `personal_score_awards` dans la player DB. ~0.3j.
- `LoadFullPerformanceHistory(ctx, xuid)` — pour le seuil `DETAIL_THRESHOLD` du Form Score. ~0.2j.

**Composants UI manquants** ([apps/web/src/](apps/web/src/)) :
- `SegmentedControl` (à confirmer présent dans shadcn — sinon ~0.3j).
- `RecordOverlay` Plotly (motif `pattern_shape` hachuré pour les barres fantômes records). ~0.5j.
- Panneau légende joueurs flottante (`position:fixed` + `IntersectionObserver` sur deux ancres). ~0.5j.
- `MedalsGallery` (grille de cartes match avec icônes médailles + lien Waypoint). ~0.5j.
- `WeaponsTable` générique (tri, slider min kills, colonnes dynamiques par joueur). ~0.5j.

### 11.4 Estimation par phase

| Phase | Sections | Coût | Pré-requis |
|-------|----------|------|------------|
| **Phase 0 — Briques transverses** | helpers algos + repo + UI listés en §11.3 | ~7-8j | À faire avant toute chart |
| **Phase 1 — En-têtes** | §2.1, §2.2 | ~2.5j | Phase 0 (perf score v2) |
| **Phase 2 — Synergies "carte"** | §3.1, §3.2, §3.3, §3.4 | ~4j | `ComputeMapBreakdown` |
| **Phase 3 — Timeline + Form Score** | §3.5, §3.6 | ~3j | LOWESS |
| **Phase 4 — Impact** | §3.7 | ~3j | Extension `ComputeImpactSummary` à 8 rôles |
| **Phase 5 — Tableau historique + First Events** | §3.8, §4.6 | ~2j | — |
| **Phase 6 — Cadence + Intensité** | §3.9, §4.5 | ~3j | `LoadKillTimingForMatches` |
| **Phase 7 — Charts trio** | §4.1, §4.3, §4.4 | ~5j | `ComputeSquadRecords` + `RecordOverlay` UI |
| **Phase 8 — Radar** | §4.2 | ~2j | `ComputeParticipationProfile` + thresholds |
| **Phase 9 — Armes + Médailles** | §4.7, §4.8, §4.9 | ~4.5j | Repo armes + médailles |
| **Total brut** | 22 charts + briques | **~36j** | |

NB : ce chiffrage suppose que tout est fait sérieusement (avec tests Go + i18n FR/EN + tokens couleur respectés). Un MVP "good enough" sans records overlays, sans 8 rôles pour Impact (rester à 4), et sans Form Score, ramène à environ **22 jours** pour environ 80 % de la fidélité visuelle.

### 11.5 Synthèse

| Catégorie | Nombre | Effort cumulé |
|-----------|--------|---------------|
| IMMÉDIAT | 0 | 0j |
| EFFORT MOYEN | 11 | ~10.5j |
| EFFORT IMPORTANT | 9 | ~22j |
| BLOQUÉ (>3j) | 0 (tout est faisable, juste long) | — |
| **Briques transverses** | 14 helpers/composants | ~7.5j |
| **TOTAL réaliste** | 22 charts + briques | **~36j** |

**Trois plus gros bloqueurs identifiés** :

1. **LOWESS** (§3.6) — pas d'implémentation Go standard, à coder ou wrapper `gonum`. Bloqueur isolé de 2j.
2. **Impact 8 rôles + N joueurs** (§3.7) — l'algo Go actuel est limité à 4 rôles et conçu pour 1v1 (`myXUID`/`friendXUID`). Doit être généralisé à N joueurs et étendu aux 4 rôles "comparatifs" (silent_hero, false_brother, top_killer + symétriques) qui nécessitent assists/deaths/outcome par équipe. **Le clutch actuel est aussi à corriger** (vraie fenêtre temporelle au lieu d'une approximation par tiers).
3. **Kill timing endpoint** (§3.9 + §4.5) — `LoadKillTimingForMatches` n'existe pas. Bloque deux charts d'un coup. Implémentation triviale (~0.5j) une fois la table `shared.match_kill_events` confirmée disponible (sinon parsing de `highlight_events` filtré sur Kill avec `time_ms`).

**Bonne nouvelle** : aucun élément n'est strictement BLOQUÉ par l'absence de données — toutes les tables DuckDB nécessaires existent (`shared.highlight_events`, `shared.medals_earned`, `shared.match_weapon_kills` sont attestées dans le code metadata/queries existant). Tout l'effort est donc du dev pur.
