# Plan de portage de la page Synthèse — Python v7/cockpit -> Go + React

> Audit comparatif rigoureux et plan de portage par phases.
> Branche source : `v7/cockpit` (Streamlit/Python), worktree `C:/Users/Guillaume/Downloads/Scripts/LevelUp/`.
> Branche cible Git : `feat/synthesis-iso-v7` (à créer depuis `feat/multi-title-adapters-and-mappings`).
> Date d'audit : 2026-04-26 (révisé 2026-04-27 : ajout Phase 0 migration canonical, alignement multi-titre).
> Plans jumeaux : [PLAN_TIMESERIES_GO_PORTAGE.md](PLAN_TIMESERIES_GO_PORTAGE.md), [PLAN_MATCH_VIEW_GO_PORTAGE.md](PLAN_MATCH_VIEW_GO_PORTAGE.md), [PLAN_CAREER_GO_PORTAGE.md](PLAN_CAREER_GO_PORTAGE.md), [PLAN_CITATIONS_GO_PORTAGE.md](PLAN_CITATIONS_GO_PORTAGE.md).

> **Note d'amendement — 2026-04-27** : ce plan est **partiellement supersedé** par
> [`PLAN_META_FOUNDATIONS_GO.md`](./PLAN_META_FOUNDATIONS_GO.md). La **Phase 0
> migration `canonical.PlayerMatchRow`** disparaît : le méta-plan formalise
> directement ce contrat (méta-plan § 5.3) et les autres pages le consomment.
> Les 4 charts canoniques (outcomes by map/mode, heatmap WR jour×heure, top by week,
> bipolaire Solo/Escouade) passent par les wrappers ECharts du méta-plan. Le
> nouveau plan Synthesis (en cours de rédaction côté équipe) doit être écrit
> **directement sur les fondations**, sans amendement intermédiaire.

### Statut des sections de ce plan vis-à-vis du méta-plan

| Section / Phase | Statut | Action |
|---|---|---|
| Phase 0 — Migration `canonical.PlayerMatchRow` (12 sous-phases) | Obsolète (absorbée) | Le contrat est formalisé en Phase 0 méta-plan (§ 5.3). |
| Phase ≥1 — Outcomes by map/mode (stacked bars W/L/T/Left) | À refactorer | `analysis/breakdown.ByMap/ByMode` + `<BarStacked>` ECharts (méta-plan § 5.2). |
| Phase ≥1 — Heatmap WR jour×heure | À refactorer | `<Heatmap2D>` ECharts ; palette divergente via tokens narrative. |
| Phase ≥1 — Top by week (stacked + line top_rate) | À refactorer | `analysis/temporal.BucketByGranularity(GranWeek)` + `<BarStacked>` + `<TimeseriesLine>`. |
| Phase ≥1 — Bipolaire Solo/Escouade | À conserver | Spécifique Synthesis. |
| Sections excédentaires Go (Scope, Overview, KPI cards, Highlights, Relations) | À décider | Décision côté nouveau plan : réduire vers les 4 charts canoniques ou conserver. |
| `outcome=4 → OutcomeDNF` mapping | À conserver | Décision déjà tranchée. |
| `is_with_friends` / `friends_xuids` | À refactorer | Utiliser `port.PlayerMatchFilters.ExcludeFriendsXUIDs`. |

---

## 0. Synthèse exécutive

La page **Synthèse Python v7/cockpit** est une page **mono-bloc** très ciblée : 1 sélecteur de période + **4 visualisations Plotly canoniques** (et pas une de plus). L'esprit en est explicité dans la docstring de `synthesis.py:1-5` : « Agrège les graphes existants (map/mode, heatmap temporelle, activité hebdo) et ajoute une comparaison Solo vs Escouade ». Trois des quatre charts sont importés depuis `win_loss.py` ; seul le quatrième (Solo vs Escouade) est natif à la page.

La page **Synthesis** côté Go/React a divergé sans étape intermédiaire de cadrage : elle compte **11 sections** dont seulement **1 sur 4 charts** correspond visuellement au Python (le bipolaire Solo/Escouade). Les trois autres charts canoniques sont :
1. **Outcomes par carte/mode** (stacked bars W/L/T/Left) → remplacé en Go par deux **tableaux** triés par win rate (D7 Breakdowns).
2. **Heatmap Win Rate jour×heure** (palette divergente rouge→ambre→vert, zmin=0 zmax=1, count en overlay texte) → remplacé en Go par une heatmap **d'activité** (palette `Blues`, z=count) qui n'évoque ni la performance ni la dimension W/L.
3. **« Matchs au top par semaine »** (stacked bars top/other + courbe top_rate sur axe Y₂) → remplacé en Go par un **tableau** des 5 meilleures semaines triées par win rate (sémantique différente).

Côté ajouts Go non présents en Python : Scope bar (D3), Vue d'ensemble (D4), KPI Solo/Escouade (4 cartes chacun), Comparaison détaillée (table redondante avec le bipolaire), Highlights (D5, top kills / top KDA / top deaths), Relations (D6, teammates / enemies). Ces blocs sont fonctionnels mais ils noient les 4 charts canoniques attendus et placent sur une seule page un mélange de sections « tableau de bord » qui dilue le rôle initial de la page Synthèse.

| Bloc Python                          | Type Python                              | Statut Go                                       | Verdict          |
|--------------------------------------|------------------------------------------|-------------------------------------------------|------------------|
| Sélecteur période                    | segmented control 5 options              | boutons identiques (5 options)                  | OK               |
| `_render_map_mode_breakdown`         | 2 stacked bars W/L/T/Left (max 12 / 10)  | tableaux top maps + top modes triés win rate    | DIVERGE          |
| `_render_heatmap_section`            | heatmap **win rate** divergent rouge→vert| heatmap **count** palette `Blues`               | DIVERGE          |
| `_render_top_by_week`                | stacked bars top/other + line top_rate y₂| tableau top 5 semaines par win rate             | DIVERGE          |
| `_render_solo_squad_compare`         | bipolaire 6 métriques (K/D, WR, Acc, K/min, Life, Perf) | bipolaire 6 métriques équivalent       | OK (mineurs)     |
| —                                    | —                                        | Scope bar D3                                    | EXCÉDENT         |
| —                                    | —                                        | Overview D4 (6 KPI cards)                       | EXCÉDENT         |
| —                                    | —                                        | KPI Solo + KPI Escouade (4 cartes chaque)       | EXCÉDENT         |
| —                                    | —                                        | Comparaison détaillée table                     | EXCÉDENT (redondant) |
| —                                    | —                                        | Highlights D5 (best/worst matches)              | EXCÉDENT         |
| —                                    | —                                        | Relations D6 (teammates/enemies)                | EXCÉDENT         |

**Constats structurels** :

1. Le schéma `domain.SynthesisMatchRow` (`apps/go-api/internal/domain/squad.go:257-269`) ne porte **ni `map_name`, ni `mode_name`, ni `rank`**. Les charts canoniques #2, #3, #4 du Python (map/mode breakdown, heatmap win rate, top by week) ne peuvent donc pas être calculés à partir de ce flux sans extension du modèle de données.
2. `domain.TemporalHeatmapCell` (`squad.go:222-226`) ne porte **que `Count`**, pas de `Wins`. Aucun pivotage ne pourra produire un win rate par cellule sans modifier ce type.
3. `domain.TopWeekEntry` (`squad.go:191-198`) porte un agrégat hebdo orienté win rate / KDA, pas un agrégat « top matches » à la manière de Python (`plot_matches_at_top_by_week`).
4. La fonction `analysis.ComputeTemporalHeatmap` (`squad_breakdown.go:412-431`) compte les matchs par cellule mais ignore l'outcome.
5. `domain.SynthesisHeatmapRow{MapName, ModeName, MatchCount, Wins}` (`squad.go:81-86`) est déjà chargée par `SynthesisRepo.LoadSynthesisHeatmap`, mais elle agrège toutes les paires (map, mode) ensemble et ne distingue que `wins` (pas tie / left). Elle est insuffisante pour reproduire la sémantique W/L/T/Left de `plot_stacked_outcomes_by_category`.
6. La page React a pris le parti d'enrichir Synthesis avec D3/D4/D5/D6/D7 — mais ces blocs n'appartiennent pas au cadrage v7. Ils dupliquent partiellement Career (Vue d'ensemble), Squad (Relations) et Performances marquantes (Highlights). Ce mélange n'est pas un simple « plus de fonctionnalités », c'est un changement implicite de définition de la page.

**Priorités de portage** (du plus urgent au plus secondaire) :

0. **Migration canonical + TitleDataAdapter** (Phase 0) : pré-requis archi multi-titre — basculer `SynthesisService` du flux `domain.SynthesisMatchRow` vers `[]canonical.PlayerMatchRow` via `dataAdapter.LoadPlayerMatches`. C'est l'étape la plus dense (12 sous-phases), mais sans change visuel ; elle débloque toutes les phases suivantes et aligne sur l'archi cible déjà appliquée à Career.
1. **Heatmap Win Rate** : c'est la divergence la plus visible (couleurs et message inversés : la heatmap actuelle n'apprend rien sur la performance, juste sur les habitudes de jeu).
2. **Outcomes par carte / par mode** : remplacer les deux tableaux statiques par les deux stacked bars W/L/T/DNF attendues.
3. **Matches at Top by Week** : remplacer le tableau par le combo stacked-bars + line top_rate sur axe Y₂.
4. **Bipolaire Solo/Escouade** : 1 ajustement mineur (ordre des métriques + couleurs sémantiques) + correctif `AvgLifeSeconds`.
5. **Réordonnancement et déclassement des excédents** : placer les 4 charts canoniques en tête, déplacer Highlights/Relations vers leur page dédiée ou les reléguer en fin de page sous un séparateur clair, supprimer la table de comparaison détaillée (le bipolaire suffit).
6. **Filtres cascade L1** (Phase 7) : appliquer enfin les filtres NavL1 ignorés aujourd'hui par le backend.

**Conformité archi (post-révision 2026-04-27)** : ce plan respecte les 8 sections de la grille `plan-review` (architecture 3-couches Go, multi-titres via `TitleDataAdapter` + `canonical.*` + TOML mappings, capability gates avec `ErrCapabilityNotSupported`, tests à chaque couche, logging `slog`, frontend via tokens et i18n FR+EN, livraison commit-par-phase). La seule entorse documentée est le **fallback repo legacy** (`SynthesisRepo.LoadSynthesisMatches`) conservé pendant la transition — daté et destiné à disparaître quand tous les titres futurs exposeront `CapPlayerMatchHistory`.

---

## 1. Cartographie source (v7/cockpit)

### 1.1 Fichiers Python (4 fichiers actifs, ~1 100 L)

| Fichier                                                    | L   | Rôle                                                                           |
|------------------------------------------------------------|----:|--------------------------------------------------------------------------------|
| `src/ui/pages/synthesis.py`                                | 320 | Orchestrateur — sélecteur période, append `is_with_friends`, 4 sections        |
| `src/ui/pages/win_loss.py`                                 | 394 | Source des 3 sections importées (`_render_map_mode_breakdown`, `_render_heatmap_section`, `_render_top_by_week`) |
| `src/visualization/distributions_outcomes.py`              | 404 | `plot_stacked_outcomes_by_category`, `plot_win_ratio_heatmap`, `plot_matches_at_top_by_week` |
| `src/ui/pages/v7_sections.py`                              | 232 | Routeur de sections — dispatch vers `render_synthesis_page` (clé `synthesis`)  |

### 1.2 Structure de la page Synthèse (5 blocs séquentiels)

#### Bloc 1 — Sélecteur de période (`synthesis.py:44-53`)

`st.segmented_control` avec 5 options `_PERIOD_KEYS = ["all", "2y", "1y", "1m", "1w"]`. Mappage en jours : `{"2y": 730, "1y": 365, "1m": 30, "1w": 7}`. La sélection est stockée dans `st.session_state["synthesis_period"]`. Le filtrage est effectué par `_filter_by_period()` (`synthesis.py:56-66`) : `pl.col("start_time") >= now_utc - timedelta(days=...)` — filtrage UTC naïf.

#### Bloc 2 — `_render_map_mode_breakdown` (`win_loss.py:92-171`)

Layout : `st.columns(2)` (gauche carte, droite mode), précédé de `st.divider()` + `st.subheader(t("wl_results_by_map_mode"))`.

**Chart 2a — Outcomes par carte** :
- Source : `plot_stacked_outcomes_by_category(dff, _wl_map_col, min_matches=1, sort_by="total", max_categories=12, opts=PlotOptions(lang=get_lang()))`
- Colonne pivot : `map_ui` (FR si dispo, sinon `map_name`)
- Pivot : 4 séries empilées via `add_outcome_traces` → WIN / LOSS / TIE / LEFT (clés `OUTCOME_CODES` = `Outcome` IntEnum)
- Couleurs : `HALO_COLORS` (vert/rouge/ambre/slate)
- Tri : `sort_by="total"` → cartes les plus jouées en premier, troncature à 12
- Légende horizontale en haut, marges `{l:40, r:20, t:30, b:80}`
- Axes : `tickangle=45` sur X, titre Y = `viz_t("trace_matches", lang)`

**Chart 2b — Outcomes par mode** :
- Pré-traitement : extraction du suffixe après `" : "` pour raccourcir (`Arène : Assassin → Assassin`) via `map_elements`
- Source : idem `plot_stacked_outcomes_by_category` avec `_mode_short`, `max_categories=10`
- Colonne pivot : préférence `mode_ui` → fallback `mode_category` → fallback `pair_name`
- Sinon paramètres identiques au 2a

#### Bloc 3 — `_render_heatmap_section` (`win_loss.py:174-189`)

`st.divider()` + `st.subheader(t("wl_heatmap_title"))` + caption optionnel `wl_heatmap_caption{tz=get_tz_name()}` (gating `hints_visible()`).

Source : `plot_win_ratio_heatmap(dff, min_matches=1, lang=get_lang())` (`distributions_outcomes.py:175-279`).

Sémantique précise (à reproduire à la lettre côté Go) :
- Grille **complète** 7×24 (lundi=0 … dimanche=6) — cellules vides remplies par `full_grid.join(agg, how="left")`.
- Pour chaque cellule (dow, hour) :
  - `wins = sum(outcome == WIN)`
  - `total = count(match_id)`
  - `win_rate = wins / total` masqué à `None` si `total < min_matches` (1 dans Synthèse, 2 dans WL native)
- Encodage Plotly : `go.Heatmap(z=win_rate_vals, zmin=0, zmax=1, colorscale=[[0.0, red], [0.5, amber], [1.0, green]])`
- Texte overlay : count entier par cellule (`text_matrix[count == 0] = ""`)
- Hover : `"%{y} %{x}<br>Win Rate: %{z:.1%}<br>Matches: %{text}"`
- Axes : Y inversé (`autorange="reversed"`) pour avoir Lundi en haut, X labels `00h..23h`
- Colorbar : `tickformat=".0%"`, titre `viz_t("hover_win_rate", lang)`
- Retourne **`None`** si aucune cellule ne dépasse `min_matches` ou si `np.isnan` partout après reshape.

#### Bloc 4 — `_render_top_by_week` (`win_loss.py:192-213`)

`st.divider()` + `st.subheader(t("wl_top_by_week"))` + caption `wl_top_by_week_caption`.

Source : `plot_matches_at_top_by_week(dff, rank_col="rank" if "rank" in dff.columns else "outcome", top_n_ranks=1, lang=get_lang())` (`distributions_outcomes.py:287-403`).

Sémantique :
- `period_label, d = determine_top_period(d, lang)` — bucket adaptatif : semaine si peu de matchs, mois sinon (cf. helper interne).
- Définition de « top » :
  - Si la colonne `rank` existe : `is_top = (rank ≤ top_n_ranks)` avec `fill_null(99)` → un match est « top » s'il est rank 1 (palier 1).
  - Sinon (cas Synthèse aujourd'hui — `rank` non chargé) : `is_top = (outcome == WIN)` → fallback victoire.
  - Sinon : `is_top = False` (chart vide).
- Agrégation par bucket : `total = count`, `top_count = sum(is_top)`, `other_count = total - top_count`, `top_rate = top_count / total * 100`.
- Composition Plotly :
  - Trace 1 (`go.Bar`) : `top_count` empilé, vert opacité 0.85, texte `inside`.
  - Trace 2 (`go.Bar`) : `other_count` empilé, slate opacité 0.55, texte `inside` masqué si 0.
  - Trace 3 (`go.Scatter`) : `top_rate` sur axe `y2` (range 0-100), ambre, mode `lines+markers`.
- `barmode="stack"`, `yaxis2={overlaying:"y", side:"right", range:[0,100]}`.
- `tickangle=45` sur X.

#### Bloc 5 — `_render_solo_squad_compare` (`synthesis.py:264-293`)

Préalable côté `render_synthesis_page` : `period_df = _attach_is_with_friends(period_df, db_path, xuid)` charge le flag depuis DuckDB par batch des `match_id` (`load_is_with_friends`). Colonne ajoutée : `is_with_friends: pl.Boolean`.

Sémantique :
- `solo_df = dff.filter(is_with_friends == False)` ; `squad_df = dff.filter(is_with_friends == True)`.
- Court-circuit si `solo_df.is_empty() or squad_df.is_empty()` → `st.info("syn_no_data")`.
- Métriques (ordre exact, formatage exact, `_build_comparison_metrics` `synthesis.py:157-206`) :
  1. **K/D** = `kills_sum / deaths_sum`, format `{value:.2f}`.
  2. **Win Rate** = `wins / total * 100`, format `{value:.1f}%`.
  3. **Accuracy** = `mean(accuracy)`, format `{value:.1f}%`.
  4. **K/min** = `kills_sum / (time_played_seconds_sum / 60)`, format `{value:.2f}`.
  5. **Avg Life** = `mean(avg_life_seconds)`, format `{value:.0f}s`.
  6. **Performance Score** = `mean(performance_score)`, format `{value:.1f}`.
  - Si l'une des deux valeurs (solo ou squad) est `None`, la métrique est **omise** (cf. `_append_metric`).
- Chart `_build_duel_chart` (`synthesis.py:209-261`) :
  - Ordre : `list(reversed(metrics))` (la première métrique est en bas du graphe, la dernière en haut — convention Plotly horizontal bar).
  - **Normalisation** : pour chaque métrique, `scale = max(solo, squad, 1.0)`, puis `solo_x = -solo/scale * 100`, `squad_x = squad/scale * 100`.
  - 2 traces `go.Bar(orientation="h")` : Solo (cyan, x négatifs) à gauche, Escouade (vert, x positifs) à droite.
  - `barmode="overlay"`, `add_vline(x=0, line_color=slate, opacity=0.8)`.
  - `xaxis: {showgrid:false, showticklabels:false, range:[-120, 120]}` ; `yaxis: {automargin:true}`.
  - `text=metric.solo_text|squad_text`, `textposition="outside"`, `cliponaxis=false`.
  - Hauteur dynamique : `max(320, 70 * len(metrics))`.
  - Caption sous chart : `t("syn_sample_split", solo=solo_df.height, squad=squad_df.height)`.

### 1.3 Clés i18n attendues (utilisées par v7)

`encounters_period_label`, `encounters_period_{all|2y|1y|1m|1w}`, `wl_results_by_map_mode`, `wl_by_map`, `wl_by_mode`, `wl_heatmap_title`, `wl_heatmap_caption`, `wl_top_by_week`, `wl_top_by_week_caption`, `syn_solo_squad_title`, `syn_solo_squad_caption`, `syn_no_data`, `syn_sample_split`, `syn_solo`, `syn_squad`, `col_win_rate`, `col_accuracy`, `col_kpm`, `col_avg_life`, `sc_performance_score`, `no_matches`, `insufficient_data_chart`, `missing_time_data`, `trace_matches`, `trace_others`, `trace_top_rate`, `axis_hour_label`, `axis_day_label`, `hover_win_rate`, `axis_rate_pct`.

---

## 2. Cartographie cible (Go + React)

### 2.1 Fichiers actuels (10 fichiers, ~990 L côté React + ~750 L côté Go)

| Fichier                                                                                  | L    | Rôle                                                                  |
|------------------------------------------------------------------------------------------|-----:|-----------------------------------------------------------------------|
| `apps/web/src/routes/players/$playerSlug/synthesis.tsx`                                  | 10   | Mount TanStack Router                                                 |
| `apps/web/src/features/synthesis/SynthesisPage.tsx`                                      | 464  | Orchestration + 11 sections + 2 builders Plotly (bipolaire, heatmap)  |
| `apps/web/src/features/synthesis/SynthesisHighlightsSection.tsx`                         | 102  | Bloc D5 (best/worst matches)                                          |
| `apps/web/src/features/synthesis/SynthesisRelationsPreview.tsx`                          | 86   | Bloc D6 (teammates/enemies)                                           |
| `apps/web/src/features/synthesis/queries.ts`                                             | 24   | TanStack Query `useSynthesisPage`                                     |
| `apps/web/src/components/shell/shellNavigation.ts`                                       | ~111 | Entrée L1 « Synthèse » (Recap)                                        |
| `apps/go-api/internal/api/handlers/synthesis.go`                                         | 68   | Handler HTTP `POST /api/v1/players/{slug}/pages/synthesis`            |
| `apps/go-api/internal/service/synthesis_service.go`                                      | 437  | Service — orchestration KPIs / comparison / heatmap / top weeks / breakdowns / highlights / rivalries |
| `apps/go-api/internal/platform/duckdb/synthesis_repo.go`                                 | 62   | Repo — délègue à `SquadRepo` pour `LoadSynthesisMatches` et `LoadSynthesisHeatmap`, charge `Q10Encounters` |
| `apps/go-api/internal/analysis/squad_breakdown.go`                                       | 456  | `ComputeSynthesisKPIs`, `ComputeComparisonMetrics`, `ComputeSynthesisTopWeeks`, `ComputeTemporalHeatmap` |
| `apps/go-api/internal/domain/squad.go` + `synthesis.go`                                  | ~430 | Types `SynthesisMatchRow`, `SynthesisHeatmapRow`, `SynthesisKPIs`, `TemporalHeatmapCell`, `TopWeekEntry`, `SynthesisBreakdowns`, `SynthesisOverview`, `SynthesisHighlightsPreview`, `SynthesisRivalriesPreview` |

### 2.2 Ordre actuel des sections rendues (`SynthesisPage.tsx:322-462`)

1. Sélecteur de période (5 boutons)
2. `<ScopeBar scope={data.scope} />` (D3) — non v7
3. `<SynthesisOverviewSection overview={data.overview} />` (D4) — non v7, 6 KPI cards
4. Cartes KPI Solo + KPI Escouade (4 KPI chacun) — non v7, redondant avec le bipolaire
5. Bipolaire Solo/Escouade (`buildBipolaireChart`) — équivalent v7
6. Comparaison détaillée (table HTML) — redondant avec #5
7. `<SynthesisHighlightsSection />` (D5) — non v7
8. Heatmap activité (`buildHeatmapChart` palette `Blues`) — DIVERGE de v7
9. Top semaines (table HTML triée par win rate) — DIVERGE de v7
10. `<SynthesisRelationsPreview />` (D6) — non v7
11. `<SynthesisBreakdownsSection breakdowns={data.breakdowns} />` (D7) — DIVERGE (tableaux vs stacked bars)

### 2.3 Charts effectifs aujourd'hui

| # | Bloc                            | Type                | Lib       | Source données                              |
|---|---------------------------------|---------------------|-----------|---------------------------------------------|
| 1 | Bipolaire Solo/Escouade         | `bar` orientation=h | Plotly.js | `comparison_metrics: ComparisonMetricItem[]`|
| 2 | Heatmap activité                | `heatmap` Blues     | Plotly.js | `heatmap_data: HeatmapCell[]` (count seul)  |

Tout le reste est rendu en HTML brut (`<table>`, `<div>` cards). Pour une page « Synthèse » côté v7 avec 4 charts canoniques, le ratio chart/section est aujourd'hui de **2/11** côté Go contre **4/5** côté Python — d'où l'impression « brut » signalée.

### 2.4 Ce que le payload Go ne fournit pas (et qui bloque le portage)

| Besoin v7                                         | Champ requis (manquant)                                  | Cause racine                                                    |
|---------------------------------------------------|----------------------------------------------------------|-----------------------------------------------------------------|
| Stacked W/L/T/Left par carte (max 12)             | série pivot `{category, win, loss, tie, left}` par carte | `SynthesisMatchRow` n'a pas `MapName` ; `SynthesisHeatmapRow` n'a que `wins/match_count` |
| Stacked W/L/T/Left par mode (max 10, suffixe coupé)| idem par mode + label court                              | idem (pas de `ModeName` granulaire ; pas de séparation T / Left) |
| Heatmap Win Rate 7×24                             | `dow, hour, count, wins`                                 | `TemporalHeatmapCell` ne porte pas `Wins`                       |
| Top by week (chart)                               | `period_label, top_count, other_count, top_rate`         | `TopWeekEntry` orienté win rate / KDA, pas top/other            |
| Définition de « top » via `rank`                  | `RankPosition int` sur `SynthesisMatchRow`               | colonne rank non chargée dans la query (fallback outcome=WIN possible mais sémantique différente) |

---

## 3. Diff visuel point par point

### 3.1 Sections du Python à porter

| Section v7                          | Python (référence)         | Go/React (état)                | Action                                                        |
|-------------------------------------|----------------------------|--------------------------------|---------------------------------------------------------------|
| Sélecteur période                   | `synthesis.py:44-53`       | `SynthesisPage.tsx:324-336`    | Conserver tel quel (équivalent fidèle).                       |
| Outcomes par carte (stacked)        | `win_loss.py:99-127`       | tableau `top_maps` (D7)        | **Réécrire** : remplacer par stacked bars W/L/T/Left max 12.  |
| Outcomes par mode (stacked)         | `win_loss.py:129-171`      | tableau `top_modes` (D7)       | **Réécrire** : stacked bars W/L/T/Left max 10 + suffixe coupé.|
| Heatmap Win Rate 7×24               | `win_loss.py:174-189`      | heatmap activité Blues         | **Réécrire** : palette divergente, z=win_rate, text=count.    |
| Top by Week (combo bars + line)     | `win_loss.py:192-213`      | tableau top weeks              | **Réécrire** : 3 traces Plotly (top, other, top_rate y₂).     |
| Solo vs Escouade (bipolaire 6m)     | `synthesis.py:264-293`     | bipolaire 6m                   | Ajustements mineurs (ordre, couleurs, formats).               |

### 3.2 Sections Go excédentaires (non v7)

| Section Go               | Verdict produit              | Décision proposée                                                                |
|--------------------------|------------------------------|----------------------------------------------------------------------------------|
| Scope bar D3             | Utile cross-page             | À conserver mais déplacer en sous-titre compact sous le sélecteur (1 ligne).     |
| Overview D4              | Recoupement avec Career      | À déplacer vers la page Career (où elle a sa place naturelle), retirer ici.      |
| KPI Solo/Escouade        | Redondant avec bipolaire     | À retirer (le bipolaire encode déjà ces 4 valeurs en plus de 2 autres).          |
| Comparaison détaillée    | Redondant avec bipolaire     | À retirer.                                                                       |
| Highlights D5            | Hors scope « Synthèse v7 »   | À retirer ici, garder l'implémentation pour la page Performances marquantes.     |
| Relations D6             | Recoupement avec Squad       | À retirer ici (déjà couvert par `palmares/relations`).                           |
| Breakdowns D7 (tableaux) | Remplacé par #2 et #3 ci-dessus | À retirer une fois les stacked bars en place.                                 |

> Note : les API et services Go correspondants (`buildHighlightsPreview`, `buildRivalriesPreview`, `buildBreakdowns`) ne sont **pas** à supprimer — ils alimentent d'autres pages. Seule la consommation côté `SynthesisPage.tsx` doit s'arrêter.

### 3.3 Différences de paramètres à connaître

| Élément                  | Python                          | Go actuel                           | Cible                          |
|--------------------------|---------------------------------|-------------------------------------|--------------------------------|
| Map breakdown — tri      | `sort_by="total"`, max 12       | tri par win rate, max non spécifié | `sort_by="total"`, max 12      |
| Mode breakdown — tri     | `sort_by="total"`, max 10       | idem                                | `sort_by="total"`, max 10      |
| Heatmap — encoding       | win rate (0..1), text=count     | count (0..N), pas d'overlay         | win rate, palette divergente, text=count |
| Heatmap — `min_matches`  | 1 dans Synthèse                 | aucun seuil                         | seuil 1 → cellule masquée si total=0 (laisser vide, pas zéro) |
| Top by Week — bucket     | adaptatif via `determine_top_period` (semaine ou mois) | semaine fixe              | adaptatif (semaine si ≤ 26 buckets, sinon mois) |
| Top by Week — top_n      | `top_n_ranks=1` (rank 1 only)   | non applicable                      | reproduire avec fallback `outcome=WIN` jusqu'à ce que `rank` soit chargé |
| Bipolaire — ordre        | K/D, Win%, Acc, K/min, Life, Perf | ordre via service                 | aligner sur Python (l'ordre des labels affecte la lecture)   |
| Bipolaire — caption      | `solo: N matchs · squad: N matchs` | identique                        | OK                             |

---

## 4. Plan de portage par phases

### Avant-propos — alignement multi-titre

L'audit du 2026-04-27 a montré que l'archi cible du repo (canonical types + `TitleDataAdapter` + TOML mappings + capability gates) **n'est pas** appliquée à la page Synthèse aujourd'hui :
- `SynthesisService` injecte `games.TitleDataAdapter` mais ne fait que **logger** la capability ; il lit toujours via `port.SynthesisRepository.LoadSynthesisMatches` (`synthesis_service.go:23-69`).
- `domain.SynthesisMatchRow` est un type Halo-specific dans `internal/domain/squad.go`, pas un type canonique.
- Aucun consommateur réel d'un type canonical match-level n'existe encore (career_service consomme `canonical.CareerSnapshot`/`EncounterRow`, pas un slice de matchs).
- `canonical.MatchParticipant` (`internal/games/canonical/match.go:48-65`) couvre déjà la majorité des stats per-player (Kills, Deaths, Assists, RankInMatch, Accuracy, ShotsFired, …) mais **ne porte pas** `TimePlayedSeconds`, `AvgLifeSeconds`, `KDA`, `KillsPerMin` — pourtant Halo-natifs.
- `canonical.PlayerStats` (`canonical/identity.go`) est conçu agrégé-périodique (`MatchesPlayed`, `Wins`, `WinRate`, …), incompatible avec un flux match-level.
- Aucun champ « enrichissement LevelUp » (`is_with_friends`, `session_label`, `performance_score` calculé) n'a sa place dans le canonique strict — ce sont des sur-couches LevelUp non garanties par les autres titres.

Le portage ISO impose donc, **avant** toute phase métier, une migration `domain.SynthesisMatchRow` → flux canonique + couche d'enrichissement LevelUp. C'est l'objet de la Phase 0.

### Phase 0 — Migration canonical + TitleDataAdapter (multi-titre)

**Objectif** : exposer un nouveau flux canonique « matchs du joueur enrichis » consommé par `SynthesisService`, avec dégradation gracieuse via `ErrCapabilityNotSupported`.

#### 0.1 — Étendre `canonical.MatchParticipant` (Halo-natif)

Fichier : `apps/go-api/internal/games/canonical/match.go`. Ajouter 4 champs aux 16 existants :
```go
type MatchParticipant struct {
    // ... champs existants ...
    KDA              *float64  // dérivé Halo-standard
    TimePlayedSeconds *int     // exposé par Halo API
    AvgLifeSeconds   *float64  // exposé par Halo API
    KillsPerMin      *float64  // dérivé canonique
}
```
Ces champs sont sémantiquement Halo-agnostiques (existent dans la plupart des FPS) ; ils ont leur place dans le canonique.

#### 0.2 — Créer `canonical.PlayerMatchEnrichment` (LevelUp-specific)

Nouveau fichier : `apps/go-api/internal/games/canonical/player_match.go`. Ce type isole les sur-couches LevelUp non garanties par les autres titres :
```go
// PlayerMatchEnrichment porte les enrichissements calculés ou agrégés par
// LevelUp pour un match donné, en dehors des stats natives du titre.
type PlayerMatchEnrichment struct {
    IsWithFriends    *bool    // calculé via xuid_aliases ∩ participants
    SessionLabel     *string  // calculé via session detection
    PerformanceScore *float64 // calculé localement (composite kills/assists/objectives/dégâts)
}

// PlayerMatchRow est le n-uplet (résumé match, stats du joueur courant, enrichissement)
// consommé par les pages bilan/synthèse.
type PlayerMatchRow struct {
    Summary    MatchSummary
    Self       MatchParticipant
    Enrichment *PlayerMatchEnrichment // optionnel, nil si non disponible
}
```

#### 0.3 — Étendre `canonical.Outcome` enum

Fichier : `apps/go-api/internal/games/canonical/enums.go`. Le code source Halo `outcome=4` est aujourd'hui mappé sur `OutcomeDNF = "dnf"`. **Décision** : conserver `OutcomeDNF` comme code canonique (la sémantique « DNF = quitté » couvre exactement le « Left » Python). Ne **pas** créer `OutcomeLeft` séparé. À documenter dans la projection : `match_participants.outcome == 4 → canonical.OutcomeDNF`.

#### 0.4 — Étendre `TitleDataAdapter`

Fichier : `apps/go-api/internal/games/adapter.go`. Ajouter une méthode + une capability :
```go
const (
    // ... existantes ...
    CapPlayerMatchHistory CapabilityKey = "player.match_history" // stats per-match du joueur
)

type TitleDataAdapter interface {
    // ... existantes ...
    LoadPlayerMatches(
        ctx context.Context,
        xuid string,
        scope canonical.StatsScope,
    ) ([]canonical.PlayerMatchRow, error)
}
```
La méthode renvoie les matchs d'un joueur sur le scope demandé (période + filtres), avec stats per-match enrichies. C'est l'équivalent canonique de l'actuel `LoadSynthesisMatches`. La `StatsScope` existante (`canonical/scopes.go`) porte déjà la notion de plage temporelle.

#### 0.5 — Implémenter dans `halo_infinite/adapter_data.go`

- Méthode `LoadPlayerMatches(ctx, xuid, scope) ([]PlayerMatchRow, error)`. Délègue à un nouvel `internal/games/halo_infinite/source.go::PlayerMatchesSource` interface (parallèle à `CareerSource`) qui encapsule l'accès DuckDB.
- `Capabilities()` : ajouter `games.CapPlayerMatchHistory: games.CapSupported`.
- Si `playerMatchesSource == nil` (cas où l'adapter est instancié sans CareerSource au boot, comme aujourd'hui) → retourner `games.ErrCapabilityNotSupported`. Pattern identique à `LoadCareerSnapshot` (lignes 101-124).

#### 0.6 — Repo DuckDB → enrichir Q33b et projeter

Fichier : `apps/go-api/internal/platform/duckdb/queries_squad.go`. Modifier `Q33bSynthesisMatches` :
```sql
SELECT
    r.match_id,
    r.start_time,
    r.duration_seconds,                                     -- NEW
    COALESCE(r.map_name_fr, r.map_name)        AS map_name, -- NEW
    COALESCE(r.pair_name_fr, r.pair_name)      AS mode_name,-- NEW
    p.outcome,
    p.kills,
    p.deaths,
    p.assists,                                              -- NEW (canonical natif)
    p.kda,
    p.accuracy,
    p.time_played_seconds,
    p.avg_life_seconds,                                     -- NEW
    p.rank                                     AS rank_in_match, -- NEW
    pme.performance_score,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE)       AS is_with_friends
FROM shared.match_participants p
JOIN shared.match_registry r ON r.match_id = p.match_id
LEFT JOIN player_match_enrichment pme ON r.match_id = pme.match_id
WHERE p.xuid = ?
ORDER BY r.start_time DESC
```
Créer `apps/go-api/internal/platform/duckdb/player_matches_repo.go` qui implémente `halo_infinite.PlayerMatchesSource` :
```go
type PlayerMatchesRepo struct{ pdb *PlayerDB }

func (r *PlayerMatchesRepo) LoadPlayerMatchesRaw(ctx context.Context, xuid string) ([]RawPlayerMatchRow, error) {
    // exécute Q33b enrichi, retourne un type intermédiaire halo-specific
}
```
Et créer la **projection** dans `internal/games/halo_infinite/projections.go` :
```go
func projectPlayerMatchRow(raw RawPlayerMatchRow) canonical.PlayerMatchRow {
    return canonical.PlayerMatchRow{
        Summary: canonical.MatchSummary{
            MatchID:         raw.MatchID,
            StartedAtUTC:    raw.StartTime,
            DurationSeconds: raw.DurationSeconds,
            Map:             projectMapAsset(raw.MapName),
            GameVariant:     projectGameVariantAsset(raw.ModeName),
            Outcome:         projectOutcome(raw.Outcome), // 4 → OutcomeDNF
        },
        Self: canonical.MatchParticipant{
            Kills:             intPtr(raw.Kills),
            Deaths:            intPtr(raw.Deaths),
            Assists:           intPtrOrNil(raw.Assists),
            KDA:               raw.KDA,
            Accuracy:          raw.Accuracy,
            TimePlayedSeconds: raw.TimePlayedSecs,
            AvgLifeSeconds:    raw.AvgLifeSeconds,
            RankInMatch:       raw.RankInMatch,
            // KillsPerMin calculé en aval si TimePlayedSeconds > 0
        },
        Enrichment: &canonical.PlayerMatchEnrichment{
            IsWithFriends:    boolPtrIfPresent(raw.IsWithFriends),
            SessionLabel:     raw.SessionLabel,
            PerformanceScore: raw.PerformanceScore,
        },
    }
}
```

#### 0.7 — Mettre à jour les TOML mappings

Fichier : `config/titles/halo_infinite/mappings/fields.toml`. Ajouter les déclarations manquantes (à vérifier avant — l'audit a confirmé `kills`, `deaths`, `accuracy`, `kda`, `duration_seconds`, `started_at_utc`, `outcome` ; à compléter si absents) :
```toml
[fields.time_played_seconds]
labels = { en = "Time Played", fr = "Temps de jeu" }
storage_unit = "seconds"
display_unit = "seconds"
format = "duration_hms"

[fields.avg_life_seconds]
labels = { en = "Avg Life", fr = "Espérance de vie" }
storage_unit = "seconds"
display_unit = "seconds"
format = "duration_ms"

[fields.rank_in_match]
labels = { en = "Rank", fr = "Palier" }
storage_unit = "rank"
display_unit = "rank"
format = "integer"

[fields.kills_per_minute]
labels = { en = "Kills / Min", fr = "Frags / Min" }
storage_unit = "per_minute"
display_unit = "per_minute"
format = "decimal_2"

[fields.performance_score]
labels = { en = "Perf. Score", fr = "Score de perf." }
storage_unit = "score"
display_unit = "score"
format = "decimal_1"
```
Et ajouter aux `canonical.FieldKey` (`canonical/fields.go`) les clés correspondantes : `FieldTimePlayedSeconds`, `FieldAvgLifeSeconds`, `FieldRankInMatch`, `FieldKillsPerMinute`, `FieldPerformanceScore`. Mettre à jour `AllFieldKeys()`.

`outcomes.toml` est déjà conforme (4 codes : win/loss/tie/dnf — aucun ajout requis).

#### 0.8 — Réécrire `SynthesisService`

Fichier : `apps/go-api/internal/service/synthesis_service.go`.

```go
func (s *SynthesisService) GetSynthesisPage(ctx context.Context, ...) (..., error) {
    var rows []canonical.PlayerMatchRow
    var err error

    // Pattern career_service : adapter d'abord, repo en fallback gracieux
    if s.dataAdapter != nil {
        caps := s.dataAdapter.Capabilities()
        if caps.Has(games.CapPlayerMatchHistory) {
            rows, err = s.dataAdapter.LoadPlayerMatches(ctx, playerXUID, scope)
            if err != nil && !errors.Is(err, games.ErrCapabilityNotSupported) {
                slog.ErrorContext(ctx, "load_player_matches_failed",
                    "title_slug", s.dataAdapter.TitleSlug(),
                    "err", err)
                return nil, err
            }
        } else {
            slog.WarnContext(ctx, "capability_not_supported",
                "title_slug", s.dataAdapter.TitleSlug(),
                "capability", string(games.CapPlayerMatchHistory))
        }
    }

    // Fallback : si adapter absent ou capability non exposée, lecture repo legacy.
    // À supprimer une fois tous les titres ont CapPlayerMatchHistory exposée.
    if rows == nil {
        legacyRows, err := s.repo.LoadSynthesisMatches(ctx, playerXUID)
        if err != nil { return nil, err }
        rows = projectLegacyToCanonical(legacyRows) // helper local de transition
    }

    // À partir d'ici, toute la suite (filterByPeriod, ComputeKPIs, etc.)
    // opère sur []canonical.PlayerMatchRow, pas sur []SynthesisMatchRow.
    ...
}
```

#### 0.9 — Mettre à jour `internal/analysis/squad_breakdown.go`

Toutes les fonctions `Compute*` qui prennent `[]domain.SynthesisMatchRow` doivent accepter `[]canonical.PlayerMatchRow`. Pour chaque champ :
- `r.Outcome == domain.OutcomeWin` → `r.Summary.Outcome == canonical.OutcomeWin`
- `r.IsWithFriends` → `derefBoolOr(r.Enrichment != nil ? r.Enrichment.IsWithFriends : nil, false)`
- `r.Kills` → `derefIntOr(r.Self.Kills, 0)`
- `r.Deaths` → idem
- `r.Accuracy` → `r.Self.Accuracy`
- `r.PerformanceScore` → `r.Enrichment != nil ? r.Enrichment.PerformanceScore : nil`
- `r.AvgLifeSeconds` → `r.Self.AvgLifeSeconds` (nouveau champ canonique)
- `r.RankPosition` → `r.Self.RankInMatch`
- `r.MapName` → `r.Summary.Map.DefaultLabel` (avec nil-check)
- `r.ModeCategory` → `r.Summary.GameVariant.DefaultLabel` (avec nil-check + post-split `" : "`)

#### 0.10 — Tests

- `internal/games/canonical/match_test.go` : vérifier la rétro-compat (les anciens consommateurs de `MatchParticipant` ne sont pas cassés par les 4 nouveaux champs).
- `internal/games/halo_infinite/adapter_data_test.go` : tester `LoadPlayerMatches` avec un `PlayerMatchesSource` mocké (capability supportée et non supportée).
- `internal/games/halo_infinite/projections_test.go` : tester `projectPlayerMatchRow` avec différents codes outcome (incluant 4 → DNF), avg_life nil, etc.
- `internal/platform/duckdb/player_matches_repo_test.go` : DB `:memory:`, créer `match_participants` + `match_registry` + `player_match_enrichment`, exécuter Q33b enrichi, vérifier scan de toutes les colonnes.
- `internal/service/synthesis_service_test.go` : ajouter cas « adapter retourne ErrCapabilityNotSupported → fallback repo ». Refactorer les cas existants pour utiliser `[]canonical.PlayerMatchRow`.

#### 0.11 — Logging

Ajouter `slog` aux endroits-clés :
- `slog.InfoContext(ctx, "player_matches_loaded", "title_slug", ..., "xuid", xuid, "count", len(rows))` dans `LoadPlayerMatches`.
- `slog.ErrorContext(ctx, "load_player_matches_failed", "err", err)` en cas d'échec.
- `slog.WarnContext(ctx, "capability_not_supported", "capability", "player.match_history", "title_slug", ...)` au call site quand on bascule sur le repo legacy.

#### 0.12 — Critère de complétion Phase 0

- `dataAdapter.LoadPlayerMatches(ctx, xuid, scope)` retourne effectivement les matchs Halo (vérification manuelle sur 1 joueur de test).
- Le service Synthesis ne référence plus `domain.SynthesisMatchRow` (supprimer le type) **sauf** dans le helper de transition `projectLegacyToCanonical` (gardé tant qu'un fallback repo est nécessaire — à dater explicitement, par exemple « à supprimer dès que tous les titres exposent `CapPlayerMatchHistory` »).
- `go test ./...` passe sans erreur.
- `go vet ./...` sans warning.
- Pas d'autre page régressée (les autres pages utilisent leurs propres flux canoniques ou repos).

---

### Phase 1 — Types résultat d'analyse (Go)

> Dépend de Phase 0.

**Objectif** : ajouter les types « résultats de calcul » consommés par le service et l'API HTTP. Ces types vivent dans `internal/domain/` (ce sont des DTOs de sortie API), **pas** dans `canonical/` (qui contient uniquement les données d'entrée multi-titre).

1. Étendre `domain.TemporalHeatmapCell` (`internal/domain/squad.go`) :
   ```go
   type TemporalHeatmapCell struct {
       DOW   int `json:"dow"`
       Hour  int `json:"hour"`
       Count int `json:"count"`
       Wins  int `json:"wins"` // NEW — pour calculer win_rate côté frontend ou côté analysis
   }
   ```
2. Créer `domain.SynthesisTopByPeriodEntry` (nouveau type, ne pas écraser `TopWeekEntry` qui est consommé ailleurs) :
   ```go
   type SynthesisTopByPeriodEntry struct {
       PeriodLabel string  `json:"period_label"` // "12/03" ou "2026-03"
       PeriodKey   string  `json:"period_key"`   // ISO week start ou YYYY-MM
       BucketKind  string  `json:"bucket_kind"`  // "match" | "day" | "week" | "month"
       Total       int     `json:"total"`
       TopCount    int     `json:"top_count"`
       OtherCount  int     `json:"other_count"`
       TopRate     float64 `json:"top_rate"` // 0..100
   }
   ```
3. Créer `domain.OutcomeBreakdown` :
   ```go
   type OutcomeBreakdownRow struct {
       Category string `json:"category"` // map name FR ou mode label court
       Win      int    `json:"win"`
       Loss     int    `json:"loss"`
       Tie      int    `json:"tie"`
       DNF      int    `json:"dnf"` // aligné canonical.OutcomeDNF (≡ "Left" Python)
       Total    int    `json:"total"`
   }
   type OutcomeBreakdown struct {
       ByMap  []OutcomeBreakdownRow `json:"by_map"`
       ByMode []OutcomeBreakdownRow `json:"by_mode"`
   }
   ```
4. Tests `internal/domain/squad_test.go` : sérialisation JSON des 3 nouveaux types (snapshot ou champs).

**Validation** : `go test ./internal/domain/...` passe ; aucun consommateur existant régressé.

### Phase 2 — Analysis (Go)

**Objectif** : implémenter les 3 calculs manquants à la sémantique exacte du Python.

> Toutes les fonctions ci-dessous prennent en entrée `[]canonical.PlayerMatchRow` (issu de `dataAdapter.LoadPlayerMatches`), pas `[]domain.SynthesisMatchRow`. Les helpers d'accès — `outcomeOf(r)`, `mapLabelOf(r)`, `modeLabelOf(r)`, `isWithFriendsOf(r)`, `accuracyOf(r)`, etc. — sont centralisés dans `internal/analysis/_canonical_accessors.go` pour éviter la duplication des nil-checks.

1. **`ComputeOutcomeBreakdown(rows []canonical.PlayerMatchRow, kind OutcomeBreakdownKind, maxCategories int) []domain.OutcomeBreakdownRow`** :
   - `kind` est un type énuméré `OutcomeBreakdownKind` (`KindMap` | `KindMode`) défini dans `analysis/`.
   - `categoryKey = mapLabelOf(r)` (helper qui lit `r.Summary.Map.DefaultLabel` avec nil-check) ou `modeLabelOf(r)` (qui lit `r.Summary.GameVariant.DefaultLabel` puis applique `split(" : ")` si présent).
   - Compteurs `{win, loss, tie, dnf}` par catégorie (mapping `canonical.OutcomeWin/Loss/Tie/DNF`).
   - Filtrage `min_matches=1`.
   - Tri par `total DESC`, troncature à `maxCategories` (12 pour map, 10 pour mode).
2. **`ComputeWinRateHeatmap(rows []canonical.PlayerMatchRow, minMatches int) []domain.TemporalHeatmapCell`** :
   - Remplace `ComputeTemporalHeatmap` (`squad_breakdown.go:412-431`).
   - Émettre la grille **complète** 7×24 (cellules avec `count=0` incluses) — laisser le frontend décider du masquage `null` vs `0`.
   - Pour chaque cellule : `count = matches dans (dow, hour)`, `wins = count(outcome == OutcomeWin)`.
   - DOW : convention Lundi=0 (la conversion `(goDow + 6) % 7` reste valide ; `r.Summary.StartedAtUTC.Weekday()`).
3. **`ComputeMatchesAtTopByPeriod(rows []canonical.PlayerMatchRow, lang string) []domain.SynthesisTopByPeriodEntry`** :
   - Implémenter `determineTopPeriod(rows)` à l'identique du helper Python `determine_top_period` (`src/visualization/_distributions_outcomes_helpers.py:207-259`). Constantes à recopier :
     - `topPeriodWeekMaxDays = 548` (~18 mois — au-delà, on bascule en bucket mensuel)
     - `topPeriodMinMatchesPerYear = 12`
     - `topPeriodMinActiveWeeksPerYear = 4`
   - Algorithme exact (4 paliers selon `days = max(start_time) - min(start_time)`) :
     1. Si `days > 548` : pré-filtre des années « actives » (`>= 12` matchs ET `>= 4` semaines distinctes), recalcule `days` après filtre.
     2. Si `days < 2` : bucket = `match` (index ordinal stringifié, label i18n `bucket_cap_match`).
     3. Si `days < 7` : bucket = `day` (`YYYY-MM-DD`, label `bucket_cap_day`).
     4. Si `days <= 548` : bucket = `week` (`dt.truncate("1w")`, label `bucket_cap_week`).
     5. Sinon : bucket = `month` (`YYYY-MM`, label `bucket_cap_month`).
   - `isTop = (rankInMatch != nil && *rankInMatch <= 1) || (rankInMatch == nil && outcome == OutcomeWin)` — `r.Self.RankInMatch` est `*int` au canonical, donc fallback documenté en commentaire.
   - Agrégation `total / top_count / other_count / top_rate`.
   - Tri chronologique ascendant (les charts demandent l'axe X temporel ordonné).
4. **Correctif denom Win Rate** (alignement Python) : modifier `ComputeSynthesisKPIs` (à refactorer pour prendre `[]canonical.PlayerMatchRow`) pour que le dénominateur du win rate soit `MatchCount` (toutes lignes du sous-ensemble solo/squad), pas `totalWL = wins + losses`. Aligne sur `_compute_group_stats` (`synthesis.py:97-99`). Les ties (`canonical.OutcomeTie`) et DNF (`canonical.OutcomeDNF`) entrent au dénominateur — chiffre plus pessimiste mais conforme à v7.
5. **Correctif `AvgLifeSeconds`** : `ComputeSynthesisKPIs` doit lire `r.Self.AvgLifeSeconds` (champ canonique ajouté en Phase 0.1) et plus déduire `sumTimePlayed/nTime` (qui est en réalité la durée moyenne de match). Bug actuel : `squad_breakdown.go:373`.
6. Tests `analysis/squad_breakdown_test.go` :
   - Cas heatmap : 1 match win lundi 14h → `wins=1, count=1` ; ajouter une perte → `wins=1, count=2`.
   - Cas top by period : 10 matchs sur 4 semaines, 3 wins répartis, `RankInMatch=nil` partout → fallback `outcome==Win` → vérifier `top_count` et `top_rate`. Doublon avec `RankInMatch=1` sur 5 matchs → vérifier que `top_count` correspond bien aux rank-1, pas aux wins.
   - Cas outcome breakdown : modes avec préfixes `"Arène : Slayer"` → catégorie agrégée `"Slayer"` ; cas Map nil → la ligne est filtrée hors agrégat (ou bucketée en `"Unknown"` selon la convention canonique adoptée — à fixer dans le helper `mapLabelOf`).
   - Cas WR denom : 10 matchs, 5 wins, 1 tie, 1 DNF → win rate = 5/10 = 50 % (pas 5/8 = 62.5 %).

### Phase 3 — Service & DTO HTTP (Go)

> Dépend de Phase 0 (LoadPlayerMatches livré) + Phase 1 (DTOs) + Phase 2 (Compute*).

**Objectif** : exposer les nouvelles données dans `SynthesisPageResponse` (DTO API), avec dégradation gracieuse via `ErrCapabilityNotSupported`.

1. Étendre `domain.SynthesisPageResponse` (`squad.go:230+`) avec :
   ```go
   OutcomeBreakdown     *OutcomeBreakdown          `json:"outcome_breakdown,omitempty"`
   MatchesAtTopByPeriod []SynthesisTopByPeriodEntry `json:"matches_at_top_by_period,omitempty"`
   ```
   Le champ `HeatmapData` continue d'exister mais transporte désormais aussi `Wins` (Phase 1).
2. Mettre à jour `synthesis_service.go` (suite Phase 0.8) :
   - Travailler sur `[]canonical.PlayerMatchRow` (nom local : `rows`).
   - Appeler `ComputeOutcomeBreakdown(rows, KindMap, 12)` et `ComputeOutcomeBreakdown(rows, KindMode, 10)`.
   - Appeler `ComputeWinRateHeatmap(rows, 1)`.
   - Appeler `ComputeMatchesAtTopByPeriod(rows, lang)`.
   - **Décision produit** : laisser temporairement `Overview`, `Highlights`, `Rivalries`, `Breakdowns` peuplés pour permettre un retrait React progressif (déprécation soft). Marquer dans le code par `// LEGACY: à retirer une fois SynthesisPage.tsx réécrit, voir PLAN_SYNTHESIS_GO_PORTAGE.md`.
   - **Capability gates** : si l'adapter retourne `ErrCapabilityNotSupported` pour `LoadPlayerMatches`, ne PAS retourner 500. Soit fallback repo (Phase 0 prévoit), soit renvoyer une réponse partielle avec `CapabilityGap{ReasonCode: "player.match_history.unsupported", Severity: "blocking"}` dans un nouveau champ `Limitations []canonical.CapabilityGap` du DTO. Décider lequel à l'implémentation — pour Halo, le fallback repo est garanti, donc `Limitations` reste vide.
3. Logging :
   - `slog.InfoContext(ctx, "synthesis_page_built", "xuid", playerXUID, "rows_count", len(rows), "period", scope.Period, "filters_applied", len(applied), "duration_ms", elapsedMs)` à la fin de `GetSynthesisPage`.
   - `slog.ErrorContext(ctx, "compute_failed", "fn", "ComputeMatchesAtTopByPeriod", "err", err)` dans chaque branche d'erreur.
4. Tests :
   - `synthesis_service_test.go` : refactorer pour mocker un `games.TitleDataAdapter` (test double `stubAdapter`) qui retourne `[]canonical.PlayerMatchRow`. Couvrir : cas nominal, cas `ErrCapabilityNotSupported` → fallback repo, cas erreur autre → propagation.
   - `api/handlers/synthesis_handler_test.go` : `httptest` qui valide la sérialisation JSON des 3 nouveaux champs (`outcome_breakdown`, `matches_at_top_by_period`, `heatmap_data[].wins`).

### Phase 4 — Frontend (React)

**Objectif** : reconstituer la page v7 à 5 sections, retirer les excédents.

1. Créer 3 nouveaux composants dans `apps/web/src/features/synthesis/` :
   - `SynthesisOutcomesByMapMode.tsx` — props : `breakdown: OutcomeBreakdown`. Layout 2 colonnes (`grid grid-cols-1 lg:grid-cols-2`). Deux `PlotlyChart` :
     - `buildStackedOutcomesFigure(rows, {labelTitle, maxCategories: 12})` — 4 traces `bar` empilées (`barmode="stack"`) avec couleurs sémantiques :
       - Win → `resolveToken('outcome-win')`
       - Loss → `resolveToken('outcome-loss')`
       - Tie → `resolveToken('outcome-tie')`
       - Left → `resolveToken('outcome-left')` (à créer si manquant — voir `apps/web/src/lib/accessibility/palettes/`).
     - `tickangle=-45` sur X, légende horizontale en haut.
   - `SynthesisWinRateHeatmap.tsx` — props : `cells: HeatmapCell[]` (avec `wins`). Construction matrice 7×24 :
     - `winRate[dow][hour] = total > 0 ? wins/total : null` (`null` pour Plotly = trou colormap).
     - `text[dow][hour] = total > 0 ? String(total) : ""`.
     - Colorscale divergente : `[[0, perf-tier-0/red], [0.5, perf-tier-2/amber], [1, divergent-pos/green]]` — utiliser des tokens existants ou créer `heatmap-divergent-{lo,mid,hi}`.
     - `zmin=0, zmax=1`. Colorbar `tickformat=".0%"`.
     - Y axis `autorange='reversed'` pour avoir Lundi en haut.
   - `SynthesisTopByPeriod.tsx` — props : `entries: SynthesisTopByPeriodEntry[]`. 3 traces :
     - Trace 1 `bar` `top_count` (vert).
     - Trace 2 `bar` `other_count` (slate, opacité 0.55).
     - Trace 3 `scatter` `top_rate` `mode='lines+markers'` `yaxis='y2'`.
     - `barmode='stack'`, `yaxis2={overlaying:'y', side:'right', range:[0,100], showgrid:false}`.
2. Réécrire `SynthesisPage.tsx` :
   - Conserver : sélecteur de période, `<ScopeBar>` (compact, 1 ligne).
   - Section 1 (nouvelle) : `<SynthesisOutcomesByMapMode>`.
   - Section 2 (nouvelle) : `<SynthesisWinRateHeatmap>`.
   - Section 3 (nouvelle) : `<SynthesisTopByPeriod>`.
   - Section 4 (existante, ajustée) : bipolaire Solo/Escouade — supprimer la table « Comparaison détaillée » qui suit ; renommer le titre en `Solo vs Escouade` (sans flèches typographiques HTML qui posent problème en lecture/screenreader).
   - **Retirer** : `SynthesisOverviewSection`, double `KPISection` Solo/Escouade, `SynthesisHighlightsSection`, `SynthesisRelationsPreview`, `SynthesisBreakdownsSection`, table `Comparaison détaillée`, table « Top semaines ».
3. Mettre à jour `apps/web/src/lib/api/types.ts` pour refléter le nouveau payload (ajouter `outcome_breakdown`, `matches_at_top_by_period`, `wins` sur `HeatmapCell`).
4. Mettre à jour le handler MSW (`apps/web/src/lib/api/msw/`) avec des fixtures crédibles pour les 3 nouveaux champs.
5. Tests :
   - `SynthesisOutcomesByMapMode.test.tsx` : monte avec 12 cartes + 10 modes + cas vide.
   - `SynthesisWinRateHeatmap.test.tsx` : monte avec grille partielle, vérifie le masquage des cellules `total=0` (text vide, z=null).
   - `SynthesisTopByPeriod.test.tsx` : monte avec 3 buckets, vérifie présence des 3 traces.
   - `SynthesisPage.test.tsx` : retirer les assertions sur les blocs excédentaires (overview, highlights, relations, breakdowns).

### Phase 5 — i18n & couleurs

1. Ajouter clés FR/EN nécessaires dans `apps/web/src/lib/i18n/...` (les v7 utilisent : `wl_results_by_map_mode`, `wl_by_map`, `wl_by_mode`, `wl_heatmap_title`, `wl_heatmap_caption`, `wl_top_by_week`, `wl_top_by_week_caption`, `syn_solo_squad_title`, `syn_solo`, `syn_squad`, `syn_sample_split`, `trace_matches`, `trace_others`, `trace_top_rate`, `axis_hour_label`, `axis_day_label`, `hover_win_rate`, `axis_rate_pct`, `insufficient_data_chart`).
2. Tokens de couleur (cf. `color-tokens` skill, et CLAUDE.md règle 20) :
   - Pas de hex en dur, pas de `text-{couleur}-{n}` Tailwind. Tout via `tokenCssVar(...)` ou `resolveToken(...)`.
   - Outcomes : `outcome-win`, `outcome-loss`, `outcome-tie`, `outcome-left` (vérifier l'existant ; sinon ajouter dans `accessibility/palettes/`).
   - Heatmap divergente : si pas déjà existante, ajouter une palette dédiée à 3 stops dans `palettes/heatmap-divergent.ts`.
   - Bipolaire : déjà alignée sur `perf-tier-2` (Solo) et `divergent-pos` (Squad) — conserver.

### Phase 6 — Nettoyage & déprécation

1. Une fois la nouvelle page validée par l'utilisateur en navigateur :
   - Retirer du payload Go les champs uniquement consommés par les blocs supprimés (`Highlights`, `Rivalries`, `Breakdowns` legacy basés tableaux). Ne pas supprimer les services sous-jacents.
   - Retirer les composants `SynthesisHighlightsSection.tsx` et `SynthesisRelationsPreview.tsx` du dossier `features/synthesis/` s'ils n'ont pas d'autre consommateur (sinon les déplacer vers leur page d'origine).
2. Vérifier qu'aucune route/page tierce ne consomme le payload `SynthesisPageResponse` au-delà de cette page.
3. Mettre à jour `.ai/project_map.md` et `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` si Synthèse y est mentionnée.

### Phase 7 — Filtres cascade L1 (parité ISO totale)

> Dépend de Phase 0 (les champs Map/GameVariant sont sur `canonical.MatchSummary`) + Phase 3 (signature service).

**Objectif** : appliquer les filtres globaux NavL1 (`experience_types`, `playlists`, `modes`, `maps`, `picked_session_labels`) au scope de la page Synthèse, comme `dff` les reçoit côté Python v7.

1. Créer `filterPlayerMatchRows(rows []canonical.PlayerMatchRow, f domain.FilterContextInput) (filtered []canonical.PlayerMatchRow, applied []string, ignored []string)` dans `apps/go-api/internal/service/synthesis_filters.go`. Modèle de référence : `service/stats_filters.go::filterStatsMatchRows` et `service/match_history_service.go::applyAllFilters`. Champs filtrés :
   - `f.Maps` → match si `r.Summary.Map` non nil et `r.Summary.Map.DefaultLabel` (ou `Labels["en"]`) appartient au set ; normalisation insensible à la casse comme `applyAllFilters`.
   - `f.Modes` → match si `r.Summary.GameVariant` non nil ; comparaison sur le label court (post-split `" : "` si applicable).
   - `f.Playlists` → match si `r.Summary.Playlist` non nil ; sinon marquer `ignored` avec log explicite.
   - `f.ExperienceTypes` → match sur `r.Summary.MatchType` (canonical.MatchType porte la distinction PvP/PvE/Custom) ; aligné avec la valeur Firefight de `IsPvE`.
   - `f.PickedSessionLabels` → match si `r.Enrichment != nil && r.Enrichment.SessionLabel` ∈ set.
   - `f.FilterMode` → drapeau `Sessions` vs `Filters` (cf. `FilterContextInput.FilterMode` enum) — utilisé pour activer/désactiver le filtrage par session.
2. Modifier `synthesis_service.go` pour appliquer ce filtrage **après** le cutoff période et **avant** les `Compute*`. Supprimer le commentaire `_ domain.FilterContextInput` (ligne 130 actuelle).
3. Renseigner `SynthesisScope.FiltersApplied` et `SynthesisScope.FiltersIgnored` à partir des retours de `filterPlayerMatchRows` pour que le `<ScopeBar>` côté React reflète ce qui a effectivement été pris en compte.
4. Logging :
   - `slog.DebugContext(ctx, "synthesis_filters_resolved", "applied", applied, "ignored", ignored, "rows_before", before, "rows_after", after)`.
5. Tests :
   - `synthesis_filters_test.go` : cas isolé sur `filterPlayerMatchRows`. Au moins 6 cas : Maps, Modes, Playlists, ExperienceTypes, PickedSessionLabels, et un cas combiné.
   - `synthesis_service_test.go` : cas E2E `Maps=["Live Fire"]` → vérifier que la heatmap, les outcomes et le top by period reflètent le sous-ensemble.
   - Cas `FiltersIgnored` : passer un filtre non implémenté → vérifier la présence dans `SynthesisScope.FiltersIgnored` et l'absence d'effet.
6. Côté React : `useGlobalFilterStore().filterContext` est déjà transmis dans `SynthesisQueryRequest.filters` (`SynthesisPage.tsx:294`). Vérifier que la **query key** TanStack Query inclut un hash du `filterContext` (sinon TanStack ne re-fetch pas quand un filtre NavL1 change). Fichier : `apps/web/src/features/synthesis/queries.ts`.

**Critère de complétion Phase 7** : avec un filtre NavL1 actif (par ex. mode = Slayer), la heatmap, les outcomes par carte, le top by period et le bipolaire reflètent **tous** le sous-ensemble filtré. Le `ScopeBar` affiche `Filtres : Modes (Slayer)`. Aucun filtre ne doit silencieusement passer en `ignored` — si un filtre n'est pas câblable aujourd'hui (manque de colonne en base), ouvrir un ticket dédié et documenter le `ignored` avec la raison.

---

## 5. Checklist de validation finale

### Multi-titre & canonical (Phase 0)
- [ ] `canonical.MatchParticipant` étendu avec `KDA`, `TimePlayedSeconds`, `AvgLifeSeconds`, `KillsPerMin` (champs nullable).
- [ ] `canonical.PlayerMatchEnrichment` et `canonical.PlayerMatchRow` créés dans `canonical/player_match.go`.
- [ ] `TitleDataAdapter.LoadPlayerMatches` ajoutée à l'interface, capability `CapPlayerMatchHistory` exposée.
- [ ] `halo_infinite/adapter_data.go::LoadPlayerMatches` implémentée, dégrade en `ErrCapabilityNotSupported` si la source DuckDB est absente.
- [ ] `SynthesisService` consomme `dataAdapter.LoadPlayerMatches` ; le repo legacy `SynthesisRepo.LoadSynthesisMatches` reste en fallback gracieux documenté avec date de péremption.
- [ ] `domain.SynthesisMatchRow` n'est plus référencé hors du helper `projectLegacyToCanonical`.
- [ ] `fields.toml` enrichi avec `time_played_seconds`, `avg_life_seconds`, `rank_in_match`, `kills_per_minute`, `performance_score`.
- [ ] `canonical/fields.go` a les nouveaux `FieldKey` correspondants.
- [ ] La projection `match_participants.outcome=4 → canonical.OutcomeDNF` est documentée et testée.

### Visuel & sémantique (Phases 1→6)
- [ ] La page Synthèse Go affiche **exactement 5 sections** : sélecteur, outcomes map+mode, heatmap win rate, matches at top by period, bipolaire Solo/Escouade.
- [ ] La heatmap utilise une palette divergente rouge→ambre→vert et affiche le **count** en overlay texte.
- [ ] Les outcomes par carte/mode rendent **4 séries empilées** (W/L/T/DNF) avec les couleurs sémantiques.
- [ ] Le chart « top by period » rend bien **3 traces** (bars empilées + line top_rate sur axe Y₂).
- [ ] Le bipolaire Solo/Escouade rend exactement **6 métriques** dans l'ordre Python (K/D, Win%, Acc, K/min, Avg Life, Perf Score).
- [ ] La page utilise uniquement des tokens de couleur (aucun hex, aucun `text-red-*`).
- [ ] Aucune mention de `Overview`, `Highlights`, `Rivalries`, `Breakdowns` dans `SynthesisPage.tsx` (Phase 6 finale).
- [ ] `domain.TemporalHeatmapCell` porte `Wins`.
- [ ] Le dénominateur du Win Rate utilise `MatchCount` (toutes lignes), pas `Wins+Losses` (alignement Python `_compute_group_stats`).
- [ ] `determineTopPeriod` Go reproduit le helper Python à la valeur près (`548 / 12 / 4`, paliers `<2 / <7 / <=548 / >548`).
- [ ] `ComputeSynthesisKPIs.AvgLifeSeconds` lit `r.Self.AvgLifeSeconds` (champ canonique), pas `sumTimePlayed/nTime`.

### Filtres cascade (Phase 7)
- [ ] Le sélecteur de période propage bien le filtre côté backend (cas `1w`, `1m`, `1y`, `2y`, `all`).
- [ ] Filtres cascade NavL1 (`maps`, `modes`, `playlists`, `experience_types`, `picked_session_labels`) appliqués au scope Synthèse ; `SynthesisScope.FiltersApplied` et `FiltersIgnored` peuplés.
- [ ] La query key TanStack Query (`apps/web/src/features/synthesis/queries.ts`) inclut le hash du `filterContext`.

### Tests
- [ ] `internal/games/canonical/match_test.go` : rétro-compat `MatchParticipant` après extension.
- [ ] `internal/games/halo_infinite/adapter_data_test.go` : `LoadPlayerMatches` capability supportée + non supportée.
- [ ] `internal/games/halo_infinite/projections_test.go` : couverture des codes outcome (incluant 4 → DNF) + champs nullable.
- [ ] `internal/platform/duckdb/player_matches_repo_test.go` : DB `:memory:` + Q33b enrichi.
- [ ] `internal/analysis/...` : 4 cas (heatmap, outcome breakdown, top by period, denom WR).
- [ ] `internal/service/synthesis_service_test.go` : cas adapter+fallback+filtres.
- [ ] `internal/service/synthesis_filters_test.go` : 6 cas filtres isolés.
- [ ] `internal/api/handlers/synthesis_handler_test.go` : sérialisation JSON des nouveaux champs.
- [ ] `apps/web/src/features/synthesis/...test.tsx` : 3 nouveaux composants + cas vides.
- [ ] `go test ./... && go vet ./...` passent.
- [ ] `npm run typecheck && npm run lint` passent.

### Logging & livraison
- [ ] `slog.InfoContext` sur l'entrée `LoadPlayerMatches` et la sortie `GetSynthesisPage` (count, duration, filters_applied).
- [ ] `slog.WarnContext` sur les capability non supportées (avec `title_slug`, `capability`, `caller`).
- [ ] `slog.ErrorContext` sur toutes les erreurs non-triviales avec clé `"err"`.
- [ ] Aucun `fmt.Println` ou `log.Printf` introduit.
- [ ] Branche Git `feat/synthesis-iso-v7` créée depuis `feat/multi-title-adapters-and-mappings`, commits par phase (P0→P7).
- [ ] Smoke test navigateur : aucun warning React, le sélecteur de période et les filtres NavL1 rechargent bien le payload.
- [ ] Entrée `.ai/thought_log.md` ajoutée (règle CLAUDE.md).

---

## 6. Notes complémentaires

### 6.1 Sur le fallback `rank → outcome=WIN`

Le Python tolère deux interprétations de « top match » selon la disponibilité de la colonne `rank` (`win_loss.py:202`). Côté Go, le payload actuel ne charge pas `rank` ; on retombe donc sur `outcome=WIN`. La sémantique réelle attendue par les utilisateurs est « rang 1 dans le match » (premier au scoreboard), pas « victoire de l'équipe ». Tant que `rank` n'est pas câblé dans la query, le label visible doit être adapté : afficher « Wins par semaine » plutôt que « Top par semaine » et basculer le label quand `rank` est branché — sinon le chart affiche un message qui ne correspond pas à ce qu'il calcule.

### 6.2 Sur l'overlap avec d'autres pages

Le mélange D3..D7 actuel reflète une décision de produit non documentée : faire de Synthèse une « home dashboard ». Si cette intention est confirmée par l'utilisateur, le présent plan reste valide pour le contenu canonique v7, mais il faudra ouvrir une seconde itération qui :
- soit intègre Synthèse comme tab dans une nouvelle page « Cockpit » (v7 cockpit shell), avec les autres tabs comme Career, Squad, etc. ;
- soit assume Synthèse comme page « bilan » et déplace D4/D5/D6 vers les pages dédiées (Career / Performances marquantes / Squad).

À défaut de directive explicite, le présent plan vise la fidélité au cadrage v7 minimaliste (« 4 charts canoniques »).

### 6.3 Sur le filtrage `is_with_friends` en Go

Le Python attache `is_with_friends` après le filtrage par période via un appel à `load_is_with_friends(db_path, xuid, match_ids)`. Le Go le portait dans `SynthesisMatchRow.IsWithFriends` ; après Phase 0, il vit dans `r.Enrichment.IsWithFriends` (`*bool` pour permettre `null` quand l'enrichissement est absent — par exemple pour un titre qui n'a pas la notion de session sociale).

### 6.4 Sur `min_matches`

Synthèse Python passe `min_matches=1` partout (heatmap, map breakdown, mode breakdown). Win/Loss page utilise `min_matches=2` pour la heatmap. Conserver la valeur Synthèse à 1 dans le service Go (paramètre par défaut) ; ne pas dupliquer la divergence WL.

### 6.5 Sur la colonne `rank`

Confirmé dans l'audit : `rank` (palier scoreboard du match) existe dans `shared.match_participants` et est déjà sélectionnée par d'autres queries (`queries_career.go:213`, `queries_home_citations.go:80`). Phase 0.6 l'ajoute à Q33b et la projette vers `canonical.MatchParticipant.RankInMatch` (champ `*int` déjà présent au canonical). Le fallback `outcome=Win` reste documenté pour les titres futurs qui n'auraient pas la notion de rang scoreboard.

### 6.6 Sur la dette laissée par Phase 0

Phase 0 ne couvre que le flux **lecture** (`LoadPlayerMatches`). Elle n'introduit pas :
- une migration des autres pages qui consomment `SynthesisMatchRow` (à ce jour aucune, mais à vérifier avant la suppression du type — `grep -rn SynthesisMatchRow apps/go-api/`).
- une bascule du `SynthesisRepo` lui-même vers une source canonique : il continue d'exister comme fallback, indispensable tant que tous les titres futurs n'exposent pas `CapPlayerMatchHistory`.
- une généralisation de `LoadPlayerMatches` à d'autres pages (Squad, Career history). C'est la prochaine étape naturelle, mais hors scope de ce plan Synthèse.

À chaque ajout de titre futur, le critère d'acceptation est : « le titre expose `CapPlayerMatchHistory` sans dégradation, ou la page Synthèse renvoie un `CapabilityGap` lisible côté UI ».

### 6.7 Sur l'ordre des phases et les commits

Ordre de livraison recommandé (1 commit par sous-phase Phase 0, 1 commit par phase Phases 1-7) :
- P0.1 → P0.7 : commits Go uniquement, pas de change frontend, pas de visuel modifié. Cette série peut être merge-able indépendamment du reste : elle introduit la nouvelle plomberie sans changer le comportement utilisateur (le service consomme `LoadPlayerMatches` qui retourne strictement les mêmes données que `LoadSynthesisMatches`).
- P0.8 → P0.12 : bascule du service + tests + nettoyage. Toujours pas de change visuel.
- P1 → P3 : DTOs + analysis + service. Le payload API change ; le frontend continue d'afficher les anciens blocs (compat soft).
- P4 → P5 : réécriture frontend des 5 sections + tokens couleur.
- P6 : suppression des blocs excédentaires côté React + payload Go.
- P7 : filtres cascade L1.

Cette progression garantit que **chaque commit reste vert** (`go test ./...` + `npm run typecheck`) et que la page reste fonctionnelle à tout moment, même partiellement migrée.

---

## 8. Spécifications ECharts — charts Synthesis-specific

> Les wrappers génériques `<BarStacked>` (outcomes par carte/mode) et `<Heatmap2D>` (heatmap WR jour×heure) sont spécifiés **dans le méta-plan** (`PLAN_META_FOUNDATIONS_GO.md` §3.2.4) avec leur `buildOption` exact. Les deux charts ci-dessous sont **page-only** (pas de promotion au catalogue) — chacun est un fichier dédié dans `apps/web/src/features/synthesis/charts/`.

### 8.1 Combo top-by-period (`buildTopByPeriodOption.ts`)

**Décision** : composition côté page via `<ChartCard>` + `buildOption` custom. Pas de wrapper générique. Raison : un seul consommateur (Synthesis), 3 séries hétérogènes (2 bars empilés + 1 line sur Y secondaire) — le coût d'abstraction surpasserait le bénéfice tant qu'il n'y a pas de 2e usage.

**Fichier** : `apps/web/src/features/synthesis/charts/buildTopByPeriodOption.ts`

**Données attendues** (depuis `domain.SynthesisTopByPeriodEntry[]`) :
```ts
type TopByPeriodEntry = {
  period_label: string   // "12/03" ou "2026-03"
  period_key: string
  bucket_kind: 'match' | 'day' | 'week' | 'month'
  total: number
  top_count: number
  other_count: number
  top_rate: number       // 0..100
}
```

**Signature** :
```ts
function buildTopByPeriodOption(
  entries: TopByPeriodEntry[],
  ctx: {
    tokens: Record<SemanticToken, string>
    labels: {
      topLegend: string      // "Top" ou "Wins" selon présence de rank — voir §6.1
      otherLegend: string    // "Autres"
      rateLegend: string     // "Taux Top (%)"
      yLeft: string          // "Matchs"
      yRight: string         // "Taux (%)"
    }
  },
): echarts.EChartsCoreOption {
  const periods = entries.map((e) => e.period_label)
  return {
    animation: false,
    grid: { left: 56, right: 56, top: 56, bottom: 80, containLabel: true },
    legend: { type: 'plain', orient: 'horizontal', top: 0, left: 'center', itemGap: 16 },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      data: periods,
      axisLabel: { rotate: -45, interval: 0, hideOverlap: false },
    },
    yAxis: [
      {
        type: 'value',
        name: ctx.labels.yLeft,
        nameLocation: 'middle',
        nameGap: 36,
        minInterval: 1,
      },
      {
        type: 'value',
        name: ctx.labels.yRight,
        nameLocation: 'middle',
        nameGap: 40,
        min: 0,
        max: 100,
        splitLine: { show: false },  // pas de double quadrillage avec yAxis[0]
        axisLabel: { formatter: '{value}%' },
      },
    ],
    series: [
      {
        type: 'bar',
        stack: 'matches',
        name: ctx.labels.topLegend,
        data: entries.map((e) => e.top_count),
        itemStyle: { color: ctx.tokens['outcome-win'] },
        emphasis: { focus: 'series' },
        barMaxWidth: 40,
      },
      {
        type: 'bar',
        stack: 'matches',
        name: ctx.labels.otherLegend,
        data: entries.map((e) => e.other_count),
        itemStyle: { color: ctx.tokens['perf-tier-3'], opacity: 0.55 },
        emphasis: { focus: 'series' },
        barMaxWidth: 40,
      },
      {
        type: 'line',
        yAxisIndex: 1,
        name: ctx.labels.rateLegend,
        data: entries.map((e) => Math.round(e.top_rate * 10) / 10),
        smooth: false,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { color: ctx.tokens['warning'], width: 2 },
        itemStyle: { color: ctx.tokens['warning'] },
        z: 10,  // ligne au-dessus des bars
      },
    ],
  }
}
```

**Tokens** : `outcome-win` pour la barre « Top » (vert), `perf-tier-3` pour « Autres » (neutre, opacité réduite à 0.55), `warning` (ambre) pour la line `top_rate`.

**Tests Vitest** : snapshot avec 4 entrées ; cas `total=0` partout (chart empty) ; cas `top_rate=100` (vérifier que la line touche le top du Y₂).

**Adaptation `bucket_kind`** : le wrapper ne sait pas s'adapter — c'est le service Go qui choisit `match | day | week | month` via `determineTopPeriod`. Le label de l'axe X est juste passé via `xAxis.name` (à ajouter au `ctx.labels` si on veut afficher « Période : Semaine » par exemple).

### 8.2 Bipolaire Solo/Escouade (`buildBipolarOption.ts`)

**Décision** : composition côté page. Raison : le seul autre lieu théorique d'usage serait Career (compare contexte Solo/Squad), mais Career a déjà ses propres patterns (Bullet, Lollipop). Pas de promotion au catalogue tant qu'un 2e usage ne se présente pas.

**Fichier** : `apps/web/src/features/synthesis/charts/buildBipolarOption.ts`

**Données attendues** (depuis `domain.ComparisonMetricItem[]`) :
```ts
type ComparisonMetric = {
  label: string         // "K/D", "Win Rate", "Précision", …
  solo_value: number
  squad_value: number
  solo_text: string     // déjà formaté (ex: "2.15", "54.3 %")
  squad_text: string
}
```

**Signature** :
```ts
function buildBipolarOption(
  metrics: ComparisonMetric[],
  ctx: {
    tokens: Record<SemanticToken, string>
    labels: { solo: string; squad: string }
  },
): echarts.EChartsCoreOption {
  // Conventions Python (synthesis.py:209-261) :
  // - Ordre inversé pour que la 1ʳᵉ métrique soit en bas du chart
  // - Normalisation : scale = max(solo, squad, 1) → x_solo = -solo/scale*100, x_squad = squad/scale*100
  // - Range fixe [-120, +120] pour laisser de la marge au texte "outside"
  const ordered = [...metrics].reverse()
  const scales = ordered.map((m) => Math.max(m.solo_value, m.squad_value, 1))
  const soloX = ordered.map((m, i) => -(m.solo_value / scales[i]) * 100)
  const squadX = ordered.map((m, i) => (m.squad_value / scales[i]) * 100)
  const labels = ordered.map((m) => m.label)
  const soloTexts = ordered.map((m) => m.solo_text)
  const squadTexts = ordered.map((m) => m.squad_text)
  const height = Math.max(320, 70 * metrics.length)

  return {
    animation: false,
    grid: { left: 110, right: 80, top: 40, bottom: 24, containLabel: true },
    legend: {
      type: 'plain', orient: 'horizontal', top: 0, left: 'center', itemGap: 24,
    },
    tooltip: {
      trigger: 'item',
      formatter: (p: any) => {
        const idx = p.dataIndex as number
        const text = p.seriesName === ctx.labels.solo ? soloTexts[idx] : squadTexts[idx]
        return `<b>${p.seriesName}</b><br/>${labels[idx]} : ${text}`
      },
    },
    xAxis: {
      type: 'value',
      min: -120,
      max: 120,
      show: false,             // pas de labels ni de quadrillage X (cf. spec Python)
    },
    yAxis: {
      type: 'category',
      data: labels,
      axisTick: { show: false },
      axisLine: { show: false },
    },
    series: [
      {
        type: 'bar',
        name: ctx.labels.solo,
        data: soloX,
        itemStyle: { color: ctx.tokens['compare-a'] },
        label: {
          show: true,
          position: 'left',                     // texte à gauche de la barre négative
          formatter: (p: any) => soloTexts[p.dataIndex],
          color: ctx.tokens['compare-a'],
          fontSize: 12,
          fontWeight: 600,
        },
        emphasis: { focus: 'self' },
        barMaxWidth: 28,
        markLine: {
          symbol: 'none',
          silent: true,
          lineStyle: { color: ctx.tokens['perf-tier-3'], width: 1, type: 'solid' },
          data: [{ xAxis: 0 }],                 // ligne verticale x=0 (équivalent Plotly add_vline)
        },
      },
      {
        type: 'bar',
        name: ctx.labels.squad,
        data: squadX,
        itemStyle: { color: ctx.tokens['compare-b'] },
        label: {
          show: true,
          position: 'right',                    // texte à droite de la barre positive
          formatter: (p: any) => squadTexts[p.dataIndex],
          color: ctx.tokens['compare-b'],
          fontSize: 12,
          fontWeight: 600,
        },
        emphasis: { focus: 'self' },
        barMaxWidth: 28,
      },
    ],
    // hauteur dynamique : Synthesis passe height à <ChartCard>
    // (le buildOption ne porte pas la hauteur — c'est une prop séparée)
  }
}
```

**Hauteur dynamique** : `height = Math.max(320, 70 * metrics.length)` à passer en prop `<ChartCard height={...}>`, pas dans `buildOption`.

**Tokens** : `compare-a` pour Solo (gauche, x négatifs), `compare-b` pour Escouade (droite, x positifs). Ligne médiane via `markLine.data: [{ xAxis: 0 }]` colorée en `perf-tier-3` (neutre slate).

**Tests Vitest** : snapshot 6 métriques (cas nominal Synthesis) ; snapshot 1 métrique (vérifier `height = 320`) ; cas `solo_value=0` partout (vérifier que `scale=max(0, x, 1)=1` → pas de division par zéro).

### 8.3 Couplage avec le payload backend

Pour que ces deux builders fonctionnent sans glue code page-side, le `SynthesisService` (Phase 3 du plan) renvoie déjà :
- `MatchesAtTopByPeriod []SynthesisTopByPeriodEntry` → directement consommable par `buildTopByPeriodOption`.
- `ComparisonMetrics []ComparisonMetricItem` → directement consommable par `buildBipolarOption`.

Le frontend n'a donc qu'à `resolveToken` les couleurs et passer le payload tel quel au builder. Aucune transformation intermédiaire.

### 8.4 Action requise hors plan Synthesis

Le token `heatmap-divergent-mid` est requis par `<Heatmap2D>` pour la palette divergente 3 stops (cf. méta-plan §3.2.4). À ajouter en Phase 0 du méta-plan, **avant** la Phase 4 frontend Synthesis :
- `apps/web/src/lib/accessibility/semantic-tokens.ts:69` — ajouter `'heatmap-divergent-mid'` à l'enum `SemanticToken` et à `ALL_TOKENS`.
- `apps/web/src/lib/accessibility/palettes/default.ts` — ajouter `'heatmap-divergent-mid': '#F59E0B'` (amber-500).
- `apps/web/src/lib/accessibility/palettes/okabe-ito.ts` — ajouter `'heatmap-divergent-mid': '#F0E442'` (Yellow Okabe-Ito).
- Tests `apps/web/src/lib/accessibility/__tests__/palettes.test.ts` — vérifier que toutes les palettes définissent ce token.
