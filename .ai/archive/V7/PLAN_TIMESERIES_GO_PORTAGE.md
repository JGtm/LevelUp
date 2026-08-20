# Plan de portage de la page Timeseries — Python v7/cockpit -> Go + Next.js

> Audit comparatif rigoureux et plan de portage par phases.
> Branche source : `v7/cockpit` (Streamlit/Python).
> Branche cible : `feat/multi-title-adapters-and-mappings` (Go + Next.js + Plotly.js).
> Date d'audit : 2026-04-26.
> Plan jumeau : [PLAN_MATCH_VIEW_GO_PORTAGE.md](PLAN_MATCH_VIEW_GO_PORTAGE.md).

> **Note d'amendement — 2026-04-27** : ce plan est **partiellement supersedé** par
> [`PLAN_META_FOUNDATIONS_GO.md`](./PLAN_META_FOUNDATIONS_GO.md). Avant toute
> implémentation à partir de ce plan, consulter le méta-plan pour les fondations
> communes : `LoadPlayerMatches(filters)` au lieu de Q33b enrichi custom,
> helpers `analysis/{temporal,breakdown}` au lieu de calculs ad hoc, stack chart
> **ECharts** (Plotly.js retiré). Tous les stubs `*_chart: PlotlyFigurePayload`
> sont supprimés en Phase 0 du méta-plan. Réécriture complète prévue en
> **Phase 2** du méta-plan.

### Statut des sections de ce plan vis-à-vis du méta-plan

| Section / Phase | Statut | Action |
|---|---|---|
| Phase 1 — Onglet Résumé (5 KPI manquants) | À refactorer | `LoadPlayerMatches` + `analysis/breakdown` (méta-plan § 5.3). |
| Phase 2 — Onglet Cartes & Modes (totalement absent en Go) | À refactorer | `analysis/breakdown.ByMap/ByMode` + `<Lollipop>` `<BarStacked>` ECharts. |
| Phase 3 — Distributions (5 manquants) | À refactorer | `<Histogram>` ECharts. |
| Phase 4 — Onglet Progression (10 timelines) | À refactorer | `<TimeseriesLine>` ECharts ; `analysis/temporal.BucketByGranularity`. |
| Phase 5 — Avancé (LUSR, intensity, K/D IC, net score) | À refactorer | LOWESS / IC restent spécifiques ; rendu via wrappers ECharts. |
| Stubs `*_chart: PlotlyFigurePayload` (16 occurrences) | Obsolète | Supprimés en Phase 0 méta-plan (§ 6.0.1). |
| §7.1 `first_events_rolling` — bloqué par absence de loader `highlight_events` | Débloqué via méta-plan | `LoadHighlightEvents` (§ 5.3.6) + `temporal.RollingMeanAdaptive` + `narrative.ComputeFirstEventsPerMatch`. Câblé en Phase 2 méta-plan. |
| Intensity heatmap match × phases — bloqué pour la même raison | Débloqué via méta-plan | `LoadHighlightEvents` + `narrative.ComputeMatchIntensityProfiles` (10 buckets). Réutilise l'algo Squad porté en Phase 1. |
| Cadence intra-match — bloqué pour la même raison | Débloqué via méta-plan | `LoadHighlightEvents` + bucket 60s + wrapper `<Cadence>` (réutilisé Squad/MatchView). |
| Phase 7 — Parité numérique EWMA (`adjust=True` Python vs `adjust=False` Go) | À conserver | Spécifique numérique. |
| Window adaptatif `max(3, n*10/100)` | À conserver | Spécifique. |
| Phase 8 — Combat Yield (S56, ajout Go hors v7) | À conserver | Ajout spécifique Go. |
| `WithDataAdapter` câblé mais jamais appelé | Obsolète | Nettoyé via adoption `LoadPlayerMatches`. |

---

## 0. Synthèse exécutive

La page Timeseries en Go expose **6 onglets** (`KPIs / Cumul / Forme / Intensité / Distributions / Combat`) alors que la version Python v7/cockpit en propose **5** (`Résumé / Cartes & Modes / Distributions / Progression / Avancé`). Le découpage n'est pas seulement renommé : **l'onglet entier "Cartes & Modes" et la quasi-totalité de l'onglet "Progression" Python n'ont aucune contrepartie Go**, et plusieurs charts critiques de l'onglet "Avancé" Python (skill rank LUSR/CSR, intensity heatmap par match × phases, cumul K/D avec IC 90 %, net score/heure, first frag/mort) sont soit absents soit représentés par un payload `null` non rempli.

Mesuré côté visualisations : la version Python expose **~39 visualisations distinctes** (KPI cards, charts, tableaux, heatmaps), la version Go n'en rend que **~13** (dont 1 inédite — Combat Yield S56 — qui n'existe pas en Python). C'est un MVP très partiel, plus une page « points cumulés + distributions » qu'une page de séries temporelles riches.

| Onglet Python                | Viz Python | Mapping Go                  | Viz Go effectives | Manquantes |
|------------------------------|-----------:|-----------------------------|------------------:|-----------:|
| Résumé                       |          7 | KPIs (partiel)              | 2                 |          5 |
| Cartes & Modes               |          3 | (aucun)                     | 0                 |          3 |
| Distributions                |         11 | Distributions (partiel)     | 6                 |          5 |
| Progression                  |         10 | (éclaté Cumul/Forme partiel)| 1                 |          9 |
| Avancé                       |          8 | Cumul + Forme + Intensité   | 4                 |          4 |
| —                            |          — | Combat (S56, hors v7)       | 1                 |          0 |

**Constats structurels** :
1. Tous les champs `*_chart: PlotlyFigurePayload` sont systématiquement `null` côté Go par décision d'architecture (le frontend est censé reconstruire les charts depuis les data points bruts), mais cette décision n'est appliquée correctement que pour ~1/3 des charts attendus. Pour `first_kill_dist`, `correlations`, `regression_chart`, `net_score_per_hour_chart`, etc., **aucun data point équivalent n'est calculé**. Ces champs sont donc des stubs morts.
2. Le service Go (`timeseries_service.go`, 646L) couvre **moins de la moitié** des méthodes du `TimeseriesService` Python (12 méthodes publiques vs 6 calculs Go).
3. Plusieurs paramètres divergent silencieusement de Python :
   - Rolling K/D : `window=20` fixe en Go vs **adaptatif** en Python (`max(3, n × 10 / 100)`).
   - `minForTrend` : Go=20 vs Python=4 → la régression est presque toujours masquée.
   - Heatmap intensité : Go calcule `jour × heure (count + avg_kd)` (ce qui correspond à la WL heatmap Python, pas à l'intensity heatmap Python). **L'intensity heatmap Python = match × 10 phases** (timeline de kills intra-match) n'a aucune contrepartie Go.
4. Aucune section ne consomme `damage_dealt`, `damage_taken`, `perf_score`, `rank` qui sont pourtant présents dans `TimeseriesMatchRow`.
5. Le `WithDataAdapter()` est câblé mais non appelé : la bascule multi-titre via `canonical.MetricSeries` est restée à l'état d'amorce.

**Priorités de portage** (du plus urgent au plus secondaire) :
1. Onglet Progression complet (Performance, Assists, Per-min, Lifespan, Spree/HS/Perfect, Shots, Damage, Rank/Score) — c'est le cœur d'une page « timeseries ».
2. Skill Rank LUSR/CSR (chart majeur, attendu par les utilisateurs ranked).
3. Onglet Cartes & Modes (résultats par carte / par mode / perf vs historique).
4. Form Score (forme glissante 14 vs 90, distinct de l'EWMA).
5. Intensity heatmap par match × phases (renommer l'actuel en "WL heatmap" pour libérer la nomenclature).
6. Cumul K/D + IC 90 %, Net score/heure, First kill/death, Streaks, Top by week, Outcomes over time, KPIs détaillés.
7. Top Weapons (réutilise `WeaponsService` existant).

**Objectif de qualité** : la Phase 7 (Polish) intègre désormais une cible **« parité numérique vérifiée »** (fixtures Python ↔ Go, tolérance `< 1e-3`, alignement EWMA `adjust=True`) et un volet **i18n FR/EN complet** (~90 clés à porter, labels de stats via `useFieldLabel`/`useOutcomeLabel`, tooltips Plotly.js i18nés). Cible finale : ~95 % de parité fonctionnelle + ~99 % de parité numérique vérifiée par test automatisé.

---

## 1. Cartographie source (v7/cockpit)

### 1.1 Fichiers Python (12 fichiers, ~3 200 L)

| Fichier                                        | L    | Rôle                                                              |
|------------------------------------------------|-----:|-------------------------------------------------------------------|
| `src/ui/pages/timeseries.py`                   | 430  | Orchestrateur — 5 onglets `st.tabs`, fonction `render_timeseries_page` |
| `src/ui/pages/_timeseries_form.py`             | 152  | Section Forme récente (Form Score 14 vs 90 + KPI delta + plot)    |
| `src/ui/pages/_timeseries_distributions.py`    | 281  | 6 histogrammes (KDE) + 5 scatter corrélations                     |
| `src/ui/pages/_timeseries_intensity.py`        | 119  | Heatmap d'intensité par match × phases (10 buckets, filtre outcome)|
| `src/ui/pages/_timeseries_weapons.py`          |  90  | Bar horizontal Top weapons + grenades + mêlée                     |
| `src/ui/pages/timeseries_skill_rank.py`        |  78  | Évolution rating LUSR/CSR avec zones de tier + bande IC + lissage |
| `src/visualization/timeseries.py`              | 477  | KDA dual-axis, Assists, Per-min (KPM/DPM/APM symétrique)          |
| `src/visualization/timeseries_combat.py`       | 474  | Lifespan, Spree/HS/PK, Shots/Acc, Damage dealt/taken, Streaks     |
| `src/visualization/_timeseries_progression.py` | 396  | Performance scoré coloré, Rank/Score dual-axis, LUSR plot         |
| `src/visualization/_timeseries_helpers.py`     | 100  | `prepare_time_axis`, `apply_chrono_xaxis`, `_rolling_mean`, `_normalize_df` |
| `src/data/services/timeseries_service.py`      | 379  | Service métier (12 méthodes statiques + 6 dataclasses retour)     |
| `src/ui/i18n/pages/timeseries.py`              | 301  | Labels FR/EN (~90 clés)                                           |

### 1.2 Structure des 5 onglets

#### Onglet 1 — « Résumé »
1. **KPI grid** (8 cartes, `src/app/kpis_render.py`) : matchs sélectionnés (+ durée moyenne), durée totale (DHM), frags/match (+ K/min), morts/match (+ D/min), assistances/match (+ A/min), précision moyenne, durée de vie moyenne (mm:ss), barre stacked résultats (V/D/T/NF). Chaque carte porte un trend ±8 % vs all-time (`above/near/below/none`).
2. **Form Score** (`_timeseries_form.py:168-252`) : courbe `compute_form_score_history(df_full)` (rolling mean 14 vs rolling mean 90) + KPI métrique (form score moyen sur la sélection + delta vs 14 derniers matchs hors sélection). Caption explicite « Positif → en forme, négatif → creux ». Hauteur 320px.
3. **KDA principal** (`timeseries.py:101-189` `plot_timeseries`) : barres groupées Kills (cyan) / Deaths (rouge) sur axe Y₁, courbe ratio K/D (vert) sur axe Y₂. `customdata=(K,D,A,Acc,Ratio)`, hover unifié `x unified`. Downsampling LTTB si > 200 points (gain ~35 %). Annotations max/min via `add_extreme_annotations`.
4. **Distribution KDA** : histogramme + KDE + rug plot (violet).
5. **Top Weapons** (si `db_path` + `xuid`) : bar horizontal `repo.load_weapon_kills_aggregated()` + grenades + mêlée séparés (`repo.load_grenade_melee_kills()`), résolution noms via `resolve_weapon_display`, exclusions `EXCLUDED_WEAPON_IDS`.
6. **Outcomes over time** : barres stacked (ou scatter en mode Sessions) V/D/T/NF par bucket temporel adaptatif.
7. **Streaks** : barres signées alternées (vert pour séries de victoires +N, rouge pour défaites −N).

#### Onglet 2 — « Cartes & Modes »
1. **Map breakdown** : barres stacked horizontales V/D/T par carte, `min_matches=1`, `max_categories=12`, source `map_ui` → `map_name`.
2. **Mode breakdown** : idem par mode, parsing `Arène : Assassin → Assassin`.
3. **Perf vs History** : `_render_winrate_perf_vs_history(dff, base_df)` — comparaison sélection courante vs `base` (hors Firefight).

#### Onglet 3 — « Distributions »
1. **6 histogrammes** (2 colonnes × 3 lignes) : Accuracy (cyan), Kills (vert), Lifespan (or), Performance score (violet), Score/min (calculé via `compute_score_per_minute`, ≥ 6 points), Rolling Win Rate 5-match (calculé via `compute_rolling_win_rate`, ≥ 10 matchs). Chaque histogramme porte une courbe KDE.
2. **5 scatter corrélations** (2+2+1 plein largeur) : Lifespan vs Kills, Accuracy vs KDA, Lifespan vs Deaths, Kills vs Deaths, Team MMR vs Enemy MMR. Coloration par `outcome`, trendline activée, min 6 points par scatter.

#### Onglet 4 — « Progression »
1. **First event distribution** : `load_first_event_times(db_path, xuid, match_ids)` → 2 histogrammes superposés avec KDE (1er kill vs 1ère mort) sur axe en secondes.
2. **Performance** (`plot_performance_timeseries`) : barres colorées par seuil (vert/cyan/amber/orange/rouge selon `SCORE_THRESHOLDS`) + courbe lissée rolling 10 (violet). Range Y [0, 100]. Customdata `date`.
3. **Assists** : barres violettes (opacity 0.7) + courbe lissée verte (rolling 10).
4. **Per-min** : barres groupées KPM (cyan, +Y), DPM (rouge, simulé en −Y), APM (violet, +Y) ; 3 courbes lissées correspondantes ; ticks symétriques `build_symmetric_abs_ticks`.
5. **Lifespan** : barres vertes + courbe lissée cyan (sec).
6. **Spree / HS / Perfect kills** : barres groupées 3 séries (`max_killing_spree`, `headshot_kills`, perfect kills via `load_perfect_kills`).
7. **Shots & accuracy** (conditionnel `shots_fired`/`shots_hit`) : 2 séries de barres + ligne accuracy sur axe Y₂.
8. **Damage dealt / taken** (conditionnel) : barres + smooth pour chaque série (cyan/rouge).
9. **Rank & Personal score** (conditionnel) : barres `personal_score` (Y₁) + ligne `rank` (Y₂ inversé).
10. **Personal score section** (depuis `win_loss.py`) : breakdown bonus / objectifs.

#### Onglet 5 — « Avancé »
1. **Match intensity heatmap** (`_timeseries_intensity.py`) : rows = matchs, cols = 10 phases égales, valeur = nombre de kills par phase. `compute_match_intensity_profiles(events_df, n_buckets=10)`. Filtre `st.segmented_control` (Tous / Victoires / Défaites). Tri par date croissante (`_reorder_profile_by_date`). Min 3 matchs avec données. Source DB : `cached_load_kill_timing_for_matches(db_path, xuid, match_ids, xuids=(xuid,))`.
2. **Skill Rank LUSR/CSR** (`timeseries_skill_rank.py` + `_timeseries_progression.py:195-310`) : ligne + markers + zones de tier en fond (Bronze/Silver/Gold/Platinum/Diamond/Onyx, rgba semi-translucide) + bande de confiance (± `rating_deviation`, `fill='tonexty'`) + courbe lissée optionnelle (rolling 20, dashdot violet). Sélecteur radio LUSR/CSR si les deux types existent. Si > 50 matchs, regroupement par semaine (`dt.truncate('1w')`). Source : `_load_lusr_history(db_path, xuid)`.
3. **Cumulative K/D + IC 90 %** (`compute_cumulative_kd_with_ci`, z=1.645) : courbe `cumulative_kd` + bande translucide `[ci_lower, ci_upper]`, marqueurs outcome.
4. **Net score / heure** (`compute_rolling_net_score_per_hour`) : aire stacked verte (zone positive) / orange (zone négative) + moyenne mobile + marqueurs V/D.
5. **EWMA K/D + régression** (`compute_ewma_kd(alpha=0.20)` + `compute_linear_regression_kd`) : courbe EWMA + ligne de régression pointillée (vert si improving, rouge si declining) + marqueurs outcome.
6. **Confirmation tendance** (`plot_regression_trend`) : barre 0–100 % du R² en pourcentage de certitude (vert improving / rouge declining).
7. **WL heatmap jour × heure** (`win_loss.py` `_render_wl_heatmap_section`) : 7 × 24 cellules colorées par win rate %, hover détaillé (matchs, wins, losses).
8. **Top by week** : barres stacked par semaine (top matchs vs total).

### 1.3 Service Python `TimeseriesService` (12 méthodes publiques)

| Méthode                                        | Rôle                                                                |
|------------------------------------------------|---------------------------------------------------------------------|
| `enrich_performance_score(dff, df_full=None)`  | Ajoute la colonne `performance_score` (calcul relatif au all-time)  |
| `compute_cumulative_metrics(dff)`              | Net cumulé + K/D cumulé + rolling K/D (window adaptatif)            |
| `compute_score_per_minute(dff)`                | Série `personal_score / (time_played_seconds / 60)` (filtre > 0)    |
| `compute_rolling_win_rate(dff)`                | Rolling mean 5 de `(outcome == WIN)` × 100 (min 5 matchs)           |
| `load_first_event_times(db, xuid, match_ids)`  | Mappings `match_id → first_kill_ms` et `first_death_ms`             |
| `load_perfect_kills(db, xuid, match_ids)`      | Mapping `match_id → count` perfect kills                            |
| `compute_ewma_kd(dff, alpha=0.2)`              | EWMA K/D via `pl.col('kd').ewm_mean(alpha=0.2, adjust=True)`        |
| `compute_cumulative_kd_with_ci(dff, z=1.645)`  | K/D cumulé + bande IC 90 % (z paramétrable)                         |
| `compute_linear_regression_kd(ewma_df)`        | slope, R², trend, p_value, win_rate_slope, is_significant           |
| `compute_rolling_net_score_per_hour(dff)`      | `(kills - deaths) / (sec / 3600)`, fenêtre adaptative               |

### 1.4 Conventions de visualisation

- Couleurs `HALO_COLORS` : `cyan #00B7EB`, `green #3DFF9A`, `red #FF5C5C`, `amber #FFBF00`, `violet #8B5CF6`, `orange #FF8C00`, `gray #888888`.
- Outcomes : V=#3DFF9A, D=#FF5C5C, T=#A855F7, NF=gris translucide.
- Hauteurs : `PLOT_CONFIG.{short, default, tall, progression}`.
- Hover commun : `hovermode='x unified'`, légende `get_legend_horizontal_bottom`.
- Annotations extrêmes via `add_extreme_annotations`.
- Downsampling LTTB > 200 points (gain ~35 %).
- Préparation X : `prepare_time_axis` → labels `#N<br>MapName` ou date `FMT_TICK_DATETIME`, step adaptatif `max(1, len // 10)`.

---

## 2. Cartographie cible (Go + Web actuel)

### 2.1 Backend Go

| Fichier                                                | L   | Rôle                                                              |
|--------------------------------------------------------|----:|-------------------------------------------------------------------|
| `apps/go-api/internal/api/handlers/timeseries.go`      |  50 | Handler `POST /api/v1/players/{slug}/pages/timeseries`            |
| `apps/go-api/internal/service/timeseries_service.go`   | 646 | Service : 1 méthode publique `GetPage` + 6 builders d'onglets     |
| `apps/go-api/internal/domain/timeseries.go`            | 170 | Types `TimeseriesPageResponse`, 5 sous-onglets, `MatchRow`        |
| `apps/go-api/internal/games/canonical/timeseries.go`   |  19 | Type `MetricSeries` canonique (non utilisé)                       |
| `apps/go-api/internal/games/halo_infinite/adapter_data.go` |  — | Implémente `LoadTimeseries` du `TitleDataAdapter` (non utilisé)  |
| `apps/go-api/internal/api/registry.go`                 |   — | `reg.Timeseries(ctx, slug)` factory (`WithDataAdapter` câblé)     |

#### Calculs Go effectifs

| Calcul                                     | Présent | Détails / divergences vs Python                                       |
|--------------------------------------------|:-------:|-----------------------------------------------------------------------|
| KPI Cards (5)                              |   Oui   | total_matches, win_rate, kd_ratio, kills_per_game, accuracy. Aucun trend. |
| Cumulative K/D                             |   Oui   | Sans IC 90 %.                                                         |
| Cumulative Net (kills − deaths)            |   Oui   | OK.                                                                   |
| Rolling K/D                                |   Oui   | **Window=20 fixe** (Python : adaptatif `max(3, n*10/100)`).           |
| EWMA K/D                                   |   Oui   | α=0.20 OK. Manque la sortie `outcome` côté point.                     |
| Régression linéaire K/D                    |   Oui   | `kd_slope`, `r_squared`, `trend`. **Pas de `winrate_slope`** (toujours `null`). **`minForTrend=20`** (Python : 4). |
| Heatmap jour × heure (count, avg_kd)       |   Oui   | Mauvais nom : c'est la WL heatmap Python, **pas** l'intensity heatmap. Manque `win_rate %`. |
| Score per minute (timeline)                |   Oui   | OK.                                                                   |
| Distribution K/D, Kills, Accuracy, Score/min, Rolling WR | Oui | 5 buckets (Python en a 6 : il manque la **distribution Lifespan** et la **distribution Performance score**). |
| Corrélations scatter                       |   Oui   | 6 paires (Python : 5 paires). Go ajoute `kills_vs_kd` qui n'existe pas en Python. Le reste est aligné. |
| Form Score (14 vs 90)                      | **Non** | Concept Python distinct de l'EWMA — totalement absent.                |
| Intensity heatmap (match × 10 phases)      | **Non** | Aucun équivalent Go. Filtre outcome non implémenté.                   |
| Skill Rank LUSR/CSR                        | **Non** | `_load_lusr_history` / tier zones / IC : tout absent.                 |
| First kill / first death distribution      | **Non** | Champ JSON `first_kill_dist: null`, calcul absent.                    |
| Net score per hour                         | **Non** | Champ JSON `net_score_per_hour_chart: null`, calcul absent.           |
| Cumulative K/D + IC 90 %                   | **Non** | Cumul présent, IC absent.                                             |
| Outcomes over time                         | **Non** | Aucun calcul.                                                         |
| Streaks (séries V/D)                       | **Non** | Aucun calcul.                                                         |
| Map breakdown                              | **Non** | Aucun calcul.                                                         |
| Mode breakdown                             | **Non** | Aucun calcul.                                                         |
| Perf vs history                            | **Non** | Aucun calcul.                                                         |
| Top Weapons (timeline)                     | **Non** | Existe dans `WeaponsService` mais non câblé dans timeseries.          |
| Top by week                                | **Non** | Aucun calcul.                                                         |
| Performance score timeline (coloré)        | **Non** | `perf_score` présent dans `MatchRow` mais non agrégé / coloré.        |
| Assists timeline                           | **Non** | Données dans `MatchRow` mais aucun chart dédié.                       |
| Per-minute (KPM/DPM/APM)                   | **Non** | Aucun calcul.                                                         |
| Lifespan timeline                          | **Non** | Pas de chart, et `time_played_seconds` n'est pas exposé en lifespan. |
| Spree / Headshots / Perfect kills timeline | **Non** | `max_killing_spree`, `headshot_kills`, `perfect_kills` absents de `MatchRow`. |
| Shots / Accuracy timeline                  | **Non** | `shots_fired`, `shots_hit` absents de `MatchRow`.                     |
| Damage dealt / taken timeline              | **Non** | `damage_dealt`/`damage_taken` exposés dans `MatchRow` mais aucun chart. |
| Rank / Personal score timeline             | **Non** | `rank`/`personal_score` exposés mais aucun chart.                     |

#### Champs `PlotlyFigurePayload` en stub (toujours `null`)

`win_rate_chart`, `score_chart`, `kda_dist_chart`, `cumul_net_chart`, `cumul_kd_chart`, `rolling_kd_chart`, `ewma_kd_chart`, `regression_chart`, `net_score_per_hour_chart`, `intensity_heatmap`, `score_per_minute_chart`, `kda_distribution`, `first_kill_dist`, `correlations[]`. Décision documentée (`domain/timeseries.go:7-10`) : « le frontend reconstruit les charts depuis les data points bruts ». À nettoyer en fin de portage : ces champs ne servent à rien.

### 2.2 Frontend Web

| Fichier                                                              | L   | Rôle                                              |
|----------------------------------------------------------------------|----:|---------------------------------------------------|
| `apps/web/src/features/timeseries/TimeseriesPage.tsx`                | 360 | Page principale (6 onglets `useState<TabId>`)     |
| `apps/web/src/features/timeseries/queries.ts`                        |  44 | Hooks TanStack Query (`useTimeseriesPage`, `useCombatYieldHistory`) |
| `apps/web/src/routes/players/$playerSlug/stats/timeseries.tsx`       |   — | Route                                             |
| `apps/web/src/components/ui/timeseries-line-chart.tsx`               | 129 | Chart Plotly.js (line + référence Y)              |
| `apps/web/src/components/ui/timeseries-histogram.tsx`                | 103 | Chart Plotly.js (bar)                             |
| `apps/web/src/components/ui/timeseries-scatter.tsx`                  | 180 | Chart Plotly.js (scatter, sélecteur 6 onglets)    |
| `apps/web/src/components/ui/timeseries-heatmap.tsx`                  | 124 | Chart Plotly.js (heatmap 7×24)                    |
| `apps/web/src/components/ui/timeseries-kda-bars.tsx`                 | 141 | Chart Plotly.js (barres K/D + ligne ratio)        |
| `apps/web/src/components/ui/combat-yield-timeseries.tsx`             | 142 | Chart Plotly.js (Combat S56)                      |

#### Onglets actuels

1. **summary** : KPI grid (cards `card.value` pré-formaté côté Go) + `<TimeseriesKdaBars rows={data.match_rows}>`.
2. **cumul** : 3 line charts (`cumulative_kd`, `cumulative_net` avec `fill=tozeroy`, `rolling_kd` window=20).
3. **form** : 3 `DeltaCard` (Pente K/D, Pente Win Rate `[null]`, R²) + 1 line chart EWMA. `EmptyStateNotice` si `has_enough_for_trend === false`.
4. **intensity** : `<TimeseriesHeatmap colorBy="count">` + line chart `score_per_min_data`.
5. **distributions** : 5 histograms en grille 2×2+1 + 1 scatter avec sélecteur 6 paires.
6. **combat** : `<CombatYieldTimeseries>` (S56) — 2 courbes OC + DR + lignes p80.

### 2.3 Test E2E `slice-3b-timeseries.spec.ts`

Vérifie uniquement : HTTP 200, `total_matches` numérique, présence des 5 onglets dans la réponse, `summary_tab.kpi_cards.length > 0`. **Aucune vérification de contenu chart** (ni de cardinalité de séries, ni de bornes, ni d'intégrité des points cumul/EWMA).

### 2.4 Modules Go déjà disponibles (réutilisables)

Le portage ne part pas d'une page blanche : le package `apps/go-api/internal/analysis/` (~280 KB) contient déjà des briques réutilisables qui réduisent significativement la charge réelle des phases ci-dessous.

| Module                                              | Lignes | Utilisable pour                                                       |
|-----------------------------------------------------|-------:|-----------------------------------------------------------------------|
| `analysis/performance_score.go`                     |    485 | Phase 3 — coloration de la timeline Performance (seuils déjà définis) |
| `analysis/skill_rating.go`                          |    430 | Phase 4 — calculs LUSR/CSR (delta, regroupement, tier resolution)     |
| `analysis/weapon_data.go` + `weapon_parser.go`      |    420 | Phase 1.6 — Top weapons (parsing déjà industrialisé)                  |
| `analysis/squad_timeseries.go`                      |    180 | Phase 3 — patterns de timelines (rolling, grouped bars)               |
| `analysis/sessions.go`                              |    300 | Phase 1.4 — bucketing temporel (jour/semaine/mois)                    |
| `analysis/kd_timeline.go`                           |     60 | Phase 1.3 — base K/D timeline                                         |
| `analysis/combat_yield.go`                          |     80 | Onglet Combat (déjà branché S56)                                      |
| `analysis/highlight_event_parser.go`                |    270 | Phase 3a — extraction first kill / first death (déjà parser opérationnel) |
| `analysis/comeback.go` + `tug_of_war.go`            |    250 | Phase 5 — patterns d'agrégation par phase de match                    |
| `analysis/mode_category.go` + `mode_label.go`       |    330 | Phase 2 — normalisation modes pour breakdown                          |
| `analysis/match_history_avg.go`                     |     70 | Phase 1.1 — calcul moyennes pour KPI trends                           |
| `analysis/match_impact.go`                          |     70 | Phase 5 — annotations d'impact sur timelines                          |
| `service/career_service.go` (CareerEncounters)      |      — | Patterns DTO + CSR/LUSR DB queries                                    |
| `service/synthesis_service.go`                      |      — | Patterns top-by-week, agrégations temporelles                         |
| `service/squad_service.go` (squad timeseries)       |      — | Patterns de buckets glissants / multi-séries                          |

**Impact sur la charge** : la Phase 4 (Skill Rank) passe de 1.5 j → ~0.8 j (calculs déjà faits dans `skill_rating.go`), la Phase 3a (enrichir `MatchRow` first_kill_ms / first_death_ms) passe de 1 j → ~0.5 j (parser déjà opérationnel), la Phase 1.6 (Top weapons) passe de 0.5 j → ~0.3 j (parser + résolution déjà faits).

**Charge réelle ajustée : ~8–9 j-h** au lieu des 11 j-h estimés en absence de cet inventaire.

---

## 3. Tables comparatives par section

### 3.1 Cartographie « ce que voit l'utilisateur »

| # | Section Python                              | Onglet Python | Présent en Go ? | Onglet Go     | Gap principal                                                                 |
|--:|---------------------------------------------|---------------|:---------------:|---------------|-------------------------------------------------------------------------------|
| 1 | KPI grid 8 cartes + trends ±8 %             | Résumé        | Partiel         | summary       | Trends, durée totale DHM, K/D/A par minute, barre stacked outcomes, durée vie |
| 2 | Form Score (14 vs 90) + KPI delta           | Résumé        | **Non**         | —             | À implémenter (calcul + viz + KPI)                                             |
| 3 | KDA dual-axis (barres K/D + ligne ratio)    | Résumé        | Partiel         | summary       | Pas de ligne ratio K/D dual-axis explicite, pas d'annotations max/min, pas de hover unifié custom |
| 4 | Distribution KDA                            | Résumé        | Oui             | distributions | OK (Distribution K/D)                                                          |
| 5 | Top Weapons + grenades + mêlée              | Résumé        | **Non**         | —             | Réutiliser `WeaponsService` (déjà existant) et brancher                        |
| 6 | Outcomes over time                          | Résumé        | **Non**         | —             | Calcul + viz (mode session vs normal)                                          |
| 7 | Streaks V/D                                 | Résumé        | **Non**         | —             | Calcul cumul consécutif signé                                                  |
| 8 | Map breakdown V/D/T                         | Cartes&Modes  | **Non**         | —             | À implémenter (peut réutiliser repo facets)                                    |
| 9 | Mode breakdown V/D/T                        | Cartes&Modes  | **Non**         | —             | Idem + parsing `pair_name`                                                     |
|10 | Perf vs history                             | Cartes&Modes  | **Non**         | —             | Calcul comparatif sélection vs base (df hors Firefight)                        |
|11 | Distribution Accuracy (KDE)                 | Distributions | Oui             | distributions | KDE absente (histogram nu), libellés moins fins                                |
|12 | Distribution Kills                          | Distributions | Oui             | distributions | KDE absente                                                                    |
|13 | Distribution Lifespan                       | Distributions | **Non**         | —             | Bucket lifespan absent côté Go                                                 |
|14 | Distribution Performance score              | Distributions | **Non**         | —             | Bucket performance absent côté Go                                              |
|15 | Distribution Score/min                      | Distributions | Oui             | distributions | OK                                                                              |
|16 | Distribution Rolling Win Rate               | Distributions | Oui             | distributions | Window=14 en Go, vs 5 en Python : à aligner ou justifier                       |
|17 | Scatter Lifespan vs Kills                   | Distributions | Oui             | distributions | OK                                                                              |
|18 | Scatter Accuracy vs KDA                     | Distributions | Oui             | distributions | OK                                                                              |
|19 | Scatter Lifespan vs Deaths                  | Distributions | Oui             | distributions | OK                                                                              |
|20 | Scatter Kills vs Deaths                     | Distributions | Oui             | distributions | OK                                                                              |
|21 | Scatter Team MMR vs Enemy MMR               | Distributions | Oui             | distributions | OK (conditionnel `mmr` dispo)                                                  |
|22 | First kill / first death distribution       | Progression   | **Non**         | —             | Champ JSON stub, calcul absent (besoin DB → à câbler)                          |
|23 | Performance timeline (colorée par seuil)    | Progression   | **Non**         | —             | Calcul + viz à porter                                                          |
|24 | Assists timeline                            | Progression   | **Non**         | —             | Idem                                                                            |
|25 | Per-minute (KPM/DPM/APM symétrique)         | Progression   | **Non**         | —             | Idem                                                                            |
|26 | Lifespan timeline                           | Progression   | **Non**         | —             | Idem                                                                            |
|27 | Spree / Headshots / Perfect kills           | Progression   | **Non**         | —             | Nécessite enrichir `MatchRow` (3 colonnes manquantes)                          |
|28 | Shots & accuracy dual-axis                  | Progression   | **Non**         | —             | Nécessite enrichir `MatchRow` (`shots_fired`, `shots_hit`)                     |
|29 | Damage dealt / taken                        | Progression   | **Non**         | —             | `MatchRow` les expose déjà — viz à câbler                                      |
|30 | Rank / Personal score dual-axis             | Progression   | **Non**         | —             | `MatchRow` les expose déjà — viz à câbler                                      |
|31 | Personal score section                      | Progression   | **Non**         | —             | À porter (bonus / objectifs)                                                   |
|32 | Intensity heatmap match × phases            | Avancé        | **Non**         | —             | À ne pas confondre avec heatmap jour × heure actuelle                          |
|33 | Skill Rank LUSR/CSR (tier zones + IC)       | Avancé        | **Non**         | —             | Chart majeur. Source DB `lusr_history` à exposer côté Go                       |
|34 | Cumul K/D + IC 90 %                         | Avancé        | Partiel         | cumul         | Cumul présent, IC absent                                                       |
|35 | Net score / heure (area)                    | Avancé        | **Non**         | —             | Champ JSON stub                                                                |
|36 | EWMA K/D + ligne régression                 | Avancé        | Partiel         | form          | EWMA OK, ligne régression non rendue (juste `kd_slope` en number), pas de marqueurs outcome |
|37 | Confirmation tendance (barre %)             | Avancé        | **Non**         | —             | Représenté par 3 `DeltaCard` au lieu d'une jauge.                              |
|38 | WL heatmap jour × heure (win rate %)        | Avancé        | Partiel         | intensity     | Le Go calcule count + avg_kd, pas le win rate. À ajuster                       |
|39 | Top by week                                 | Avancé        | **Non**         | —             | Calcul à porter                                                                |
|40 | Combat Yield (OC + DR)                      | —             | Oui             | combat        | Inédit Go (S56). Aucune action sauf documenter.                                |

### 3.2 Tableau de couverture par onglet

| Onglet Python  | Visualisations | Présentes Go | Partielles | Absentes | Couverture |
|----------------|---------------:|-------------:|-----------:|---------:|-----------:|
| Résumé         |              7 |            1 |          2 |        4 |       29 % |
| Cartes & Modes |              3 |            0 |          0 |        3 |        0 % |
| Distributions  |             11 |            6 |          1 |        4 |       55 % |
| Progression    |             10 |            0 |          0 |       10 |        0 % |
| Avancé         |              8 |            1 |          3 |        4 |       13 % |
| **Total**      |         **39** |        **8** |      **6** |   **25** |   **~28 %** |

---

## 4. Inventaire des champs à enrichir dans `TimeseriesMatchRow`

`TimeseriesMatchRow` Go expose 14 champs. Pour porter l'onglet Progression complet et les charts associés, il faut ajouter les colonnes ci-dessous (toutes lisibles depuis `match_participants` partagée ou stats joueur, sans nouvelle requête réseau) :

| Colonne                                | Source DB                                       | Utilisé pour                            |
|----------------------------------------|-------------------------------------------------|-----------------------------------------|
| `headshot_kills`                       | `match_participants.headshot_kills`             | Spree/HS/PK                             |
| `max_killing_spree`                    | `match_participants.max_killing_spree`          | Spree/HS/PK                             |
| `perfect_kills`                        | Calculé depuis `medals_earned` (citation existante) | Spree/HS/PK                          |
| `shots_fired`                          | `match_participants.shots_fired`                | Shots/Acc                               |
| `shots_hit`                            | `match_participants.shots_hit`                  | Shots/Acc                               |
| `first_kill_ms`                        | `highlight_events` (event=KILL, min(timestamp)) | First event distribution                |
| `first_death_ms`                       | `highlight_events` (event=DEATH, min(timestamp))| First event distribution                |
| `team_mmr` / `enemy_mmr`               | `match_participants` (déjà chargé pour squad)   | Scatter MMR (conditionnel)              |

Bonus (pas dans `MatchRow` mais nécessaires) :

- `kill_timing_buckets[10]` par match → pour intensity heatmap match × phases. Calculer dans le service via une requête `highlight_events` agrégée par bucket de 10 % de durée.
- `lusr_history` → exposer un nouveau champ top-level `skill_rank_history: []SkillRankPoint{ match_id, rating_type, rating_value, rating_deviation, tier_label, start_time }`.

---

## 5bis. Conformité architecture transversale (applicable à toutes les phases)

Chaque phase doit respecter les principes architecturaux du projet avant d'être marquée complète. Cette section formalise les contrôles qui s'appliquent **en parallèle** des livrables de chaque phase.

### Checklist multi-titre et capability-based design

- [ ] **Pas de branchement sur `slug`** : tout code décisionnel qui varie par titre **doit** utiliser `HasCapability(ctx, titleSlug, CapabilityX)` ou `CapabilityMap.Has(titleSlug, CapabilityX)`, jamais `if titleSlug == "halo_infinite"`.
- [ ] **Déclarations de capability** : si une phase ajoute une nouvelle métrique/champ (e.g., `headshot_kills`, `perfect_kills`, `team_mmr`), déclarer la capability correspondante dans `internal/games/capabilities.go` ou `CapabilityMap` et documenter dans `config/titles/halo_infinite/capabilities.toml`.
- [ ] **Mappings TOML** : si un nouveau champ stats est exposé dans `MatchRow` ou dans les réponses, l'ajouter dans `config/titles/halo_infinite/mappings/fields.toml` avec son `FieldKey` canonique et sa source DB.
- [ ] **Dégradation gracieuse** : tout code qui consomme une capability doit prévoir `ErrCapabilityNotSupported` et soit la retourner proprement (API), soit la gérer silencieusement (frontend `useFieldLabel` retourne `null`). **Pas de panic, pas d'erreur 500.**
- [ ] **PathResolver obligatoire** : aucune ligne de code `filepath.Join(repoRoot, "data", ...)` ou construction de chemin titre-spécifique direct. Toujours utiliser `PathResolver.ResolveStatsDBPath(ctx, titleSlug)` ou équivalent.

### Logging structuré

- [ ] **Erreurs métier** : toute erreur non-triviale (calcul échoue, données manquantes) loggée via `slog.ErrorContext(ctx, "msg", "err", err)` avec clés standardisées : `"titleSlug"`, `"player"`, `"match_id"`, `"duration_ms"`, etc.
- [ ] **Opérations significatives** : chargements DB, transformations pré-calcul, regroupements complexes → `slog.DebugContext` ou `slog.InfoContext` selon importance.
- [ ] **Aucun `fmt.Println`, `log.Printf`, ou `printf` de debug** laissé en code.

### Tests par couche — minima obligatoires

| Couche | Minima | Exemple |
|--------|--------|---------|
| `analysis/*` | Unitaire pur (entrée `[]Match` ou dataframe, sortie calcul) | `TestBuildFormScoreSeries`, `TestBuildPerformanceTimeline` |
| `service/*` | Unitaire avec mock `port.Repository` | `TestGetPageWithoutDataAdapter`, `TestBuildCumulTabAligned` |
| `platform/duckdb/*` | Intégration avec DuckDB `:memory:` | `TestLoadTimeseriesMatchesWithAllFields` |
| `api/handlers/*` | HTTP `httptest` + mock service | `TestHandlerTimeseriesV1OK`, `TestHandlerTimeseriesFilteredContext` |
| Frontend | Hook+component vitest ou E2E Playwright | `it('renders form score', async () => ...)` |

Règle simple : **si un calcul ou une décision de domaine a une logique**, elle a un test.

### Pas de colonnes title-specific côté service

- [ ] **Les services** (`internal/service/timeseries_service.go` et `internal/analysis/*`) ne doivent **jamais** interroger ou manipuler directement des colonnes nommées `halo_infinite_*` ou similaire. Tout passe par `TitleDataAdapter` qui dénormalise les données titre-spécifiques en types canoniques (`canonical.Match`, `canonical.PlayerStats`).
- [ ] **Exception** : `internal/games/halo_infinite/adapter_*.go` peut (et doit) accéder aux colonnes brutes halo-specific ; c'est son rôle.

### Parity & i18n (Phase 7 crítica)

- [ ] **Parité numérique documentée** : chaque écart Go vs Python > 1e-3 noté dans `.ai/thought_log.md` avec justification.
- [ ] **Clés i18n** : ~90 clés mapping vers FR/EN dans `apps/web/src/features/timeseries/i18n.ts`, aucune string hardcodée dans composants.
- [ ] **Labels stats** : via `useFieldLabel()` / `useOutcomeLabel()` (patterns centralisés).

---

## 5. Plan de portage par phases

Le plan cible **une seule branche** (cf. CLAUDE.md § « Stratégie de branches Git ») avec plusieurs commits ordonnés. Branche suggérée : `feat/timeseries-parity-v7` depuis la branche actuelle `feat/multi-title-adapters-and-mappings` (ou depuis `main` une fois cette branche mergée).

### Phase 0 — Cadrage, validation DB et nettoyage (1 jour)

#### 0a. Validation DB préalable (prérequis bloquant, ~1 h)

Avant de lancer le portage, valider que les tables/colonnes nécessaires sont effectivement présentes et peuplées. Probe ad-hoc Go (`cmd/inspect_timeseries_corpus/main.go`, à supprimer après run) qui ATTACH shared/meta sur la DB joueur de référence et :

1. Vérifie que `shared.match_participants` expose bien `headshot_kills`, `max_killing_spree`, `shots_fired`, `shots_hit`, `team_mmr`, `enemy_mmr` (8 colonnes attendues).
2. Vérifie que `shared.highlight_events` est populée (count > 0 sur ≥ 100 matchs récents) avec event types `KILL` et `DEATH`.
3. Vérifie que `shared.medals_earned` permet de compter les Perfect kills (medal_id `1512363953`) — référence : commit `2d32413d`.
4. Vérifie que `stats.match_skill_rank` (player DB) expose `rating_type`, `rating_value`, `rating_deviation`, `tier_label`, `start_time` pour ≥ 50 matchs.
5. Vérifie que `metadata.career_ranks` permet la résolution `tier_label`.
6. Reportez ces vérifs dans `.ai/thought_log.md` avant de continuer.

**Si une vérification échoue** : marquer la phase concernée comme bloquée, retirer les calculs dépendants de la roadmap court-terme, ou prévoir un sprint de remplissage DB.

#### 0b. Aligner les paramètres divergents (sans changer le rendu visible)

1. `minForTrend` 20 → 4 (`timeseries_service.go:223`).
2. Rolling K/D : passer en window adaptatif `max(3, n × 10 / 100)` (`buildCumulTab` ligne 141).
3. Rolling WR distribution : passer de window=14 → window=5 pour aligner Python (`buildRollingWRBuckets`).
4. Documenter ces décisions dans `.ai/thought_log.md`.

#### 0c. Nettoyage du domain

1. **Supprimer les champs `*_chart: PlotlyFigurePayload` morts** (16 occurrences) — décision déjà documentée en haut de `domain/timeseries.go`. Ils sont sources de confusion et le test E2E ne les consomme pas.
2. **Renommer la heatmap actuelle** : la documenter clairement comme `WL heatmap (jour × heure)` côté domain et UI, libérer le terme « intensity » pour la phase 5.
3. Test régression : lancer `slice-3b-timeseries.spec.ts` et `apps/go-api/.../timeseries_service_test.go` après chaque modification.

#### 0d. Capabilities et TOML pour multi-titre

Avant d'ajouter des colonnes/champs à `TimeseriesMatchRow` ou aux réponses API, déclarer les capabilities et les mappings TOML correspondants. Cette phase formalise la fondation multi-titre pour les phases 1–8.

1. **Déclarer les capabilities** dans `apps/go-api/internal/games/capabilities.go` (ou pattern CapabilityMap équivalent) :
   - `CapabilityHeadshotKills` → source `match_participants.headshot_kills`
   - `CapabilityMaxKillingSpree` → source `match_participants.max_killing_spree`
   - `CapabilityPerfectKills` → source dérivée `medals_earned` (medal_id `1512363953`)
   - `CapabilityShotsFired` / `CapabilityShotsHit` → source `match_participants.shots_*`
   - `CapabilityFirstKillMilestone` / `CapabilityFirstDeathMilestone` → source `highlight_events`
   - `CapabilityTeamMMR` / `CapabilityEnemyMMR` → source `match_participants`
   - `CapabilitySkillRatingHistory` → source `match_skill_rank` (si présente)
   - `CapabilityDamageDealtTaken` → source `match_participants`
   - `CapabilityWLHeatmap` (WL win rate par jour × heure) → source agrégée existante
   
   Documenter la source DB, la condition de présence (e.g., « Halo Infinite uniquement »), et l'impact API.

2. **Ajouter mappings TOML** dans `config/titles/halo_infinite/mappings/fields.toml` :
   ```toml
   [headshot_kills]
   field_key = "HEADSHOT_KILLS"
   display_label_fr = "Précision (tirs à la tête)"
   display_label_en = "Headshot Accuracy"
   source = "match_participants.headshot_kills"
   type = "integer"
   capability = "CapabilityHeadshotKills"
   
   # ... repeate pour chaque champ nouveau
   ```

3. **Frontend** : ajouter les `FieldKey` correspondants dans `canonical/fields.ts` (ou équivalent Web) et vérifier que `useFieldLabel(FieldKey.HEADSHOT_KILLS)` retourne la bonne valeur FR/EN.

4. **Tests** : ajouter dans `timeseries_service_test.go` un test `TestCapabilityDeclaredForMatchRowFields` qui vérifie que chaque champ ajouté à `MatchRow` a une capability déclarée et un mappage TOML.

5. **Documenter dans `.ai/thought_log.md`** les capabilities déclarées et leur raison (support pour futures versions Halo).

**Done definition** :
- [ ] Capabilities déclarées dans le code
- [ ] TOML mappings synchronisés
- [ ] Test CapabilityDeclaredForMatchRowFields passe
- [ ] `.ai/thought_log.md` mise à jour

### Phase 1 — Onglet Résumé enrichi (1.5 jour)

**Done definition** (à vérifier avant fermeture) :
- [ ] Tests Go (`go test ./...`) passent, `go vet` clean
- [ ] Tests frontend (vitest + `npm run typecheck`) passent
- [ ] Entrée `.ai/thought_log.md` ajoutée (décision, résultat observé, prochaine étape)
- [ ] Architecture conforme : pas de slug checks, capabilities déclarées, logging structuré

**Livrable** : Onglet summary complet avec KPIs étendus, KDA dual-axis, Form Score, Outcomes, Streaks, Top Weapons.

1. **KPIs** :
   - Étendre `buildTimeseriesSummaryTab` : ajouter durée totale (DHM), K/min, D/min, A/min, lifespan moyen mm:ss.
   - Ajouter une carte « Outcomes » avec barre stacked V/D/T/NF (nouveau type `KpiOutcomeBar` ou retour `[]float64` proportions).
   - Calculer le `trend` ±8 % vs all-time (le service doit avoir accès à un dataset all-time non filtré → ajouter `LoadStatsMatchesAll(ctx)` au repo, ou mémoriser le total avant filtrage et passer les deux sets au builder).
2. **KDA dual-axis** : revoir `<TimeseriesKdaBars>` pour ajouter la **ligne ratio K/D** sur axe Y₂, plus annotations max/min. La donnée source (`match_rows`) suffit.
3. **Form Score** : nouveau calcul backend `BuildFormScoreSeries(matches, fullHistory) → []FormScorePoint{ index, start_time, form_score }` avec rolling 14 vs rolling 90 sur l'historique complet (pas la sélection). Ajouter une carte KPI delta vs baseline. Nouveau composant `<FormScoreChart>` (line) + `<DeltaCard>`.
4. **Outcomes over time** : nouveau calcul `BuildOutcomesOverTime(matches, granularity)` (auto bucketing jour/semaine/mois selon span). Nouveau composant `<OutcomesOverTimeChart>` (stacked bars).
5. **Streaks** : nouveau calcul `BuildStreakSeries(matches)` (cumul consécutif signé). Nouveau composant `<StreakBars>`.
6. **Top Weapons** : câbler `WeaponsService.LoadAggregatedWeaponKills(ctx, matchIDs, xuid)` + grenades + mêlée existant. Nouveau composant `<TopWeaponsBar>` horizontal (réutilisable depuis page Weapons si elle existe).

### Phase 2 — Onglet Cartes & Modes (1 jour)

**Done definition** :
- [ ] Tests Go passent, `go vet` clean
- [ ] Tests frontend passent, TypeScript compile
- [ ] `.ai/thought_log.md` mise à jour
- [ ] Architecture conforme : mode/map parsing via `mode_category.go`, pas de slug magic

**Livrable** : Nouvel onglet `cartes-modes` avec Map breakdown, Mode breakdown, Perf vs History.

1. **Map breakdown** : `BuildMapBreakdown(matches, minMatches=1, maxCategories=12)` → `[]MapOutcomeRow{ map_name, wins, losses, ties, total }`. Composant `<MapBreakdownStackedBars>`.
2. **Mode breakdown** : `BuildModeBreakdown(matches)` avec parsing `pair_name → category`. Composant identique paramétré.
3. **Perf vs history** : `BuildPerfVsHistory(selection, history)` — comparaison side-by-side. Composant `<PerfVsHistoryChart>`.

Nouveau onglet `cartes-modes` à insérer dans le composant `TimeseriesPage.tsx` après `summary`.

### Phase 3 — Onglet Progression complet (3 jours)

**Done definition** :
- [ ] Tous les tests Go passent (y compris `timeseries_service_test.go` pour chaque builder)
- [ ] `go vet` clean, tests E2E passent
- [ ] Frontend TypeScript compile, vitest pass
- [ ] `.ai/thought_log.md` mise à jour avec décisions d'enrichissement MatchRow
- [ ] Architecture conforme : pas de title-specific SQL dans service, capabilities déclarées pour chaque nouveau champ, logging structuré
- [ ] Numeric parity vs Python pour Performance, Assists, Per-min, Lifespan, Damage (fixtures commencées)

**Livrable** : `TimeseriesMatchRow` enrichi + onglet Progression complet (9 timelines + distributions).

C'est la phase la plus dense. Découpage suggéré en 3 commits.

#### 3a. Enrichir `TimeseriesMatchRow` + repo
- Ajouter dans `domain.TimeseriesMatchRow` : `headshot_kills`, `max_killing_spree`, `perfect_kills`, `shots_fired`, `shots_hit`, `first_kill_ms`, `first_death_ms`, `team_mmr`, `enemy_mmr`.
- Adapter `StatsRepository.LoadStatsMatches` (ou un `LoadTimeseriesMatches` dédié) pour LEFT JOIN `match_participants` + sous-requêtes sur `highlight_events` (premier kill/death) et `medals_earned` (perfect kills agrégés). Tester avec un dataset démo.
- Mettre à jour les tests `timeseries_service_test.go` et `multi_title_parity_test.go`.

#### 3b. Backend — calculs Progression
- `BuildPerformanceTimeline(matches, thresholds)` — score 0-100 + classification couleur.
- `BuildAssistsTimeline(matches, smoothWindow=10)`.
- `BuildPerMinuteTimeline(matches)` → KPM/DPM/APM (DPM rendu en valeur négative côté UI pour l'effet symétrique).
- `BuildLifespanTimeline(matches)` (lifespan = `time_played_seconds / max(1, deaths)`).
- `BuildSpreeHsPkTimeline(matches)` → 3 séries.
- `BuildShotsTimeline(matches)` → 2 séries de barres + accuracy ligne.
- `BuildDamageTimeline(matches)` → 2 séries (dealt + taken).
- `BuildRankScoreTimeline(matches)` → 2 séries (rank inversé + score).
- `BuildFirstEventDistribution(matches)` → 2 distributions (kill / death) en buckets de 5 secondes.
- `BuildPersonalScoreSection(matches)` → bonus / objectifs (équivalent `personal_score_awards` côté Python).

Ajouter ces sorties dans un nouvel onglet `domain.TimeseriesProgressionTab` et dans `TimeseriesPageResponse`.

#### 3c. Frontend — onglet Progression
- Ajouter onglet `progression` dans `TimeseriesPage.tsx`.
- Composants à créer (réutiliser `<TimeseriesLineChart>` quand possible, sinon spécialiser) :
  - `<PerformanceColoredBars>` : barres + lissage + range 0-100 + coloration par seuil (utiliser tokens `perf-tier-*`).
  - `<AssistsBars>`.
  - `<PerMinSymmetricBars>` (KPM/DPM/APM, ticks symétriques).
  - `<LifespanBars>`.
  - `<SpreeHsPkGroupedBars>`.
  - `<ShotsAccDualAxis>`.
  - `<DamageDealtTakenBars>`.
  - `<RankScoreDualAxis>`.
  - `<FirstEventHistogram>`.
  - `<PersonalScoreSection>` (table + KPI).

### Phase 4 — Skill Rank LUSR/CSR (~0.8 jour, réduit grâce à `analysis/skill_rating.go`)

**Done definition** :
- [ ] Tests Go passent, `go vet` clean
- [ ] Frontend tests vitest passent, TypeScript compile
- [ ] `.ai/thought_log.md` mise à jour avec tier zone palette
- [ ] Architecture conforme : pas de hex couleur en dur (utiliser `resolveToken`), capability declaration
- [ ] Skill rank numeric parity vs Python vérifiée sur dataset >= 50 matchs

**Livrable** : Composant `<SkillRankChart>` avec IC bandes, zones tier, sélecteur LUSR/CSR.

> **Code existant à réutiliser** : `apps/go-api/internal/analysis/skill_rating.go` (430 L) couvre déjà la résolution LUSR/CSR, le calcul de delta, le regroupement par playlist_group et la résolution tier. `service/career_service.go` charge déjà `match_skill_rank` pour la page Carrière (`buildLUSRSummary`). Cette phase consiste essentiellement à wrapper et à exposer ces calculs sur l'endpoint timeseries.

1. **Repo** : étendre `StatsRepository` (ou réutiliser une méthode existante de `CareerRepository`) pour exposer `LoadSkillRankHistory(ctx, xuid, ratingType?)` retournant la timeline complète (pas juste le summary). Vérifier qu'une telle méthode n'existe pas déjà dans `apps/go-api/internal/platform/duckdb/`.
2. **Service** : `BuildSkillRankSection(matches, ratingType, options)` retournant `[]SkillRankPoint{ match_id, rating_type, rating_value, rating_deviation, tier_label, start_time, map_name? }`. Si len > 50, regrouper par semaine. Ajouter `available_types: []string` (pour UI radio si `len > 1`). Réutiliser les helpers de `skill_rating.go` pour le tier resolution.
3. **Domain** : nouveau champ `skill_rank: { points, available_types, selected_type }` dans la réponse, ou nouvel endpoint dédié.
4. **Frontend** : composant `<SkillRankChart>` (line + markers + bandes IC + zones tier en `Plotly.Layout.shapes`). Sélecteur radio LUSR/CSR conditionnel. Toggle « lissage » (rolling 20). Ajouter dans un nouvel onglet `avance` ou directement dans l'actuel `form`.

Vérifier la palette des tier zones (Bronze/Silver/Gold/Platinum/Diamond/Onyx) — ne pas réintroduire de hex en dur (cf. CLAUDE.md règle 20). Étendre `apps/web/src/lib/accessibility/palettes/` avec une palette `tier-*` ou réutiliser celle existante.

### Phase 5 — Onglet Avancé restauré (1.5 jour)

**Done definition** :
- [ ] Tests Go passent, `go vet` clean
- [ ] Tests E2E passent (heatmap rendue, segmented control fonctionne)
- [ ] TypeScript compile, vitest pass
- [ ] `.ai/thought_log.md` mise à jour avec décisions intensity heatmap / net score / WL heatmap
- [ ] Architecture conforme : pas de colonnes title-specific, logging structuré, tests par couche
- [ ] Numeric parity : cumul+IC, net score/heure, rolling win rate tous vérifiés vs Python

**Livrable** : Onglet Avancé complet (intensity match, cumul+IC, net/heure, regression line, WL heatmap, top by week).

1. **Intensity heatmap match × phases** :
   - Repo : `LoadKillTimingBuckets(ctx, xuid, matchIDs, nBuckets=10)` → pour chaque match, vecteur de 10 entiers (kills par bucket de durée).
   - Service : `BuildMatchIntensityProfile(matches, killTimings, outcomeFilter?)`.
   - Domain : champ `intensity_match_phases: []MatchPhasesRow{ match_id, start_time, outcome, buckets[10] }`.
   - Frontend : composant `<IntensityMatchHeatmap>` (heatmap rows = matchs, cols = 10 phases, tri par date, filtre outcome via `<SegmentedControl>`).
2. **Cumul K/D + IC 90 %** :
   - Service : étendre `buildCumulTab` pour ajouter `cumulative_kd_ci: []CumulativePointWithCI{ index, value, ci_lower, ci_upper }`.
   - Frontend : composant dérivé `<TimeseriesLineWithBand>` (Plotly trace `tonexty` + ligne).
3. **Net score / heure** :
   - Service : `BuildNetScorePerHour(matches, windowSize?)` (fenêtre adaptative).
   - Frontend : `<NetScorePerHourArea>` (zones positives vertes / négatives oranges).
4. **EWMA K/D + ligne régression** :
   - Service : déjà calcule slope/R²/trend ; ajouter dans `form_tab.regression_line_points: []CumulativePoint` les valeurs `slope*x + intercept` pour les n indices.
   - Frontend : superposer une seconde série pointillée sur le chart EWMA + marqueurs `outcome` (cercles V/D).
5. **Confirmation tendance** :
   - Frontend : remplacer ou compléter le bloc 3 `DeltaCard` par une jauge `<TrendConfidenceGauge value={r_squared * 100}>` (vert si trend == "improving", rouge si "declining").
6. **WL heatmap jour × heure (win rate %)** :
   - Service : étendre `IntensityHeatmapPoint` avec `win_rate float64` ou créer un nouveau type `WLHeatmapPoint{ day, hour, matches, wins, losses, win_rate }`.
   - Frontend : ajouter mode `colorBy="win_rate"` dans `<TimeseriesHeatmap>`.
7. **Top by week** :
   - Service : `BuildTopByWeek(matches, perfThreshold=80)` → `[]WeekTopRow{ week_start, total, top }`.
   - Frontend : `<TopByWeekStackedBars>`.

### Phase 6 — Form Score (½ jour)

**Done definition** :
- [ ] Tests Go passent, `go vet` clean
- [ ] Frontend tests pass, TypeScript compile
- [ ] `.ai/thought_log.md` mise à jour avec rationale Form Score distinct EWMA
- [ ] Architecture conforme : calcul pur dans `analysis/`, service orche, logging OK
- [ ] Numeric parity Go vs Python vérifiée sur fixture (tolérance < 1e-3)

**Livrable** : Form Score timeline + KPI card avec delta baseline, intégré à Phase 1 ou Phase 5.

Déjà partiellement traité Phase 1.6 (KPI + chart côté Résumé). Vérifier que le calcul Python est bien reproduit :

```
form_score[t] = mean(kd[max(0, t-13):t+1]) - mean(kd[max(0, t-89):t+1])
baseline = mean of last 14 form_scores BEFORE the selection start
delta = current_avg - baseline
```

À tester sur un dataset Python pour s'assurer de la parité numérique.

### Phase 7 — Polish + Parité numérique vérifiée (1.5 jour)

**Done definition** :
- [ ] Tous les tests Go passent, `go test ./...` + `go vet ./...` clean
- [ ] Tests frontend parity passent (fixture vs Go sur 50 + 500 matches, tolérance verificada)
- [ ] `npm run typecheck` et `npm run lint` pass, E2E Playwright pass
- [ ] `.ai/thought_log.md` mise à jour : EWMA align, fixtures loaded, i18n keys mapped, écarts > 1e-3 documentés
- [ ] Architecture conforme : pas de couleurs hex en dur, capabilities OK, logging OK
- [ ] i18n vérifiée : aucune clé `ts_*` brute dans le DOM

**Livrable** : Page Timeseries **~95 % parité fonctionnelle + ~99 % parité numérique vérifiée + i18n FR/EN complet**.

#### 7a. Polish UX

- **Downsampling** : implémenter LTTB côté Go pour les séries > 200 points. Référentiel : algorithme « Largest Triangle Three Buckets » (Steinarsson 2013). Indispensable pour la fluidité des charts longue plage. Lib candidate : `github.com/dgryski/go-lttb` (à auditer en termes de licence et de stabilité).
- **Hover unifié** : aligner tous les charts sur `hovermode='x unified'` avec `customdata` riche.
- **Annotations extrêmes** : porter `add_extreme_annotations` côté Plotly.js (utility `addExtremeMarkers(figure, x, y, options)`).
- **Couleurs** : auditer pour ne pas réintroduire de hex en dur (`tokenCssVar` / `resolveToken`).

#### 7b. Parité numérique vérifiée (objectif iso strict)

Trois actions pour passer de **« parité fonctionnelle »** (~95 %) à **« parité numérique vérifiée »** (~99 %) :

1. **Aligner EWMA Go sur `adjust=True` Python**.
   - État actuel Go (`timeseries_service.go:198-208`) : `ewma[0] = kd[0]; ewma[i] = α·kd[i] + (1−α)·ewma[i−1]` ≈ `adjust=False` Python.
   - Python v7/cockpit utilise `pl.col('kd').ewm_mean(alpha=0.2, adjust=True)` qui pondère les premières valeurs par `Σ(1−α)^k / Σ(1−α)^k`. Divergence numérique observable sur les 5–10 premiers matchs.
   - Action : porter le calcul `adjust=True` ou décider explicitement `adjust=False` et le documenter dans `.ai/thought_log.md` comme écart accepté.
2. **Fixtures de parité numérique Python ↔ Go**.
   - Exporter depuis `v7/cockpit` un dataset de référence (50 et 500 matchs) avec, pour chaque méthode, les sorties attendues : EWMA K/D, régression (slope, R², trend), cumul K/D + IC 90 %, Form Score (timeline + baseline), performance score, score per minute, rolling win rate, intensity buckets, distributions buckets.
   - Sauvegarder en `apps/go-api/internal/service/testdata/timeseries_parity_50.json` et `..._500.json` (clé par méthode).
   - Procédure d'export : un script Python ad-hoc dans `v7/cockpit` (utilisable une seule fois) qui appelle chaque méthode du `TimeseriesService` Python et sérialise le résultat en JSON. Ne pas committer ce script — seulement les fixtures.
3. **Tests de tolérance Go**.
   - Ajouter dans `timeseries_service_test.go` un sous-ensemble de tests `TestParity_<Method>` qui charge la fixture, rejoue la méthode Go correspondante, et vérifie `math.Abs(go_value - py_value) < 1e-3` pour chaque point de chaque série.
   - Tolérances : `1e-3` pour les valeurs flottantes (cumuls, EWMA, IC, performance), `0` pour les comptages entiers (buckets, count heatmap), `1e-2` pour le R² (sensible aux flottants).
   - Documenter chaque écart accepté > 1e-3 dans `.ai/thought_log.md` avec sa raison.

#### 7c. i18n FR/EN — point critique

L'i18n est **non-négociable** pour la parité v7/cockpit qui supporte FR et EN sur ~90 clés (`src/ui/i18n/pages/timeseries.py`). Action en 4 temps :

1. **Extraire** les ~90 clés Python dans `apps/web/src/features/timeseries/i18n.ts` selon le pattern déjà utilisé dans `apps/web/src/features/notifications/i18n.ts` (FR + EN).
2. **Référencer** par clé sémantique (`ts_form_score_title`, `ts_kda_main`, `ts_intensity_match_title`, etc.) — pas de strings hardcodés dans les composants.
3. **Labels de stats** : utiliser `useFieldLabel()` / `useOutcomeLabel()` pour toutes les métriques affichées (kills, deaths, accuracy, KDA, etc.) afin de réutiliser les mappings centralisés (cf. CLAUDE.md règle 20 et frontend-patterns).
4. **Tooltips Plotly.js** : utiliser également les clés i18n dans les `customdata` et `hovertemplate` — c'est là que les ratés sont les plus fréquents.

Test de validation : `apps/web/e2e/slice-3b-timeseries.spec.ts` étendu pour vérifier qu'aucun chart ne rend une string contenant un identifiant de clé non résolu (`ts_*` brut visible dans le DOM = bug).

#### 7d. Tests E2E enrichis

Étendre `slice-3b-timeseries.spec.ts` pour assert :

- Chaque onglet rend ≥ 1 chart non vide.
- Cardinalité min des séries cumul/EWMA = `total_matches`.
- Présence des KPI cards complets (8 cartes attendues post-Phase 1).
- Form Score : valeur courante numérique (pas `NaN`, pas `null`).
- Skill rank : si `available_types.length > 1`, le radio est rendu et fonctionnel.
- Intensity heatmap match × phases : segmented control (Tous / Victoires / Défaites) modifie le contenu.
- Aucune string `ts_*` brute visible dans le DOM (i18n résolue).

### Phase 8 — Nettoyage final

**Done definition** :
- [ ] `go test ./...` pass, `go vet ./...` clean
- [ ] Linter frontend pass
- [ ] `.ai/thought_log.md` mise à jour : Phase 8 clôturage
- [ ] Documentation mise à jour, plan archives cohérent

**Livrable** : Codebase propre, no dead code, documentation à jour, plan clôturé.

- Supprimer les `*_chart PlotlyFigurePayload` stubs (Phase 0 partielle).
- Mettre à jour `.ai/project_map.md` avec les nouveaux modules.
- Ajouter une section dans `docs/MIGRATION_GAP_PYTHON_TO_GO.md` qui dresse l'état après portage.
- Ajouter `.ai/thought_log.md` pour la conclusion de portage.

---

## 6. Estimation et risques

### 6.1 Charge

Estimation initiale (avant inventaire des modules réutilisables) puis **estimation ajustée** après prise en compte du code existant dans `apps/go-api/internal/analysis/` (cf. § 2.4) :

| Phase | Description                              | Initial (j.) | Ajusté (j.) | Gain                                                    |
|------:|------------------------------------------|-------------|------------|---------------------------------------------------------|
|   0   | Cadrage + validation DB + nettoyage      | 0.5         | 1.0        | +0.5 j (validation DB ajoutée comme prérequis)          |
|   1   | Onglet Résumé enrichi                    | 1.5         | 1.2        | Top weapons accéléré par `weapon_data.go`               |
|   2   | Onglet Cartes & Modes                    | 1.0         | 0.8        | `mode_category.go` + `mode_label.go` réutilisés         |
|   3   | Onglet Progression complet               | 3.0         | 2.5        | `highlight_event_parser.go` couvre first kill/death     |
|   4   | Skill Rank LUSR/CSR                      | 1.5         | 0.8        | `skill_rating.go` couvre l'essentiel des calculs        |
|   5   | Onglet Avancé restauré                   | 1.5         | 1.5        | inchangé                                                |
|   6   | Form Score                               | 0.5         | 0.5        | inchangé (calcul spécifique, pas de helper existant)    |
|   7   | Polish + parité numérique vérifiée + i18n FR/EN | 1.0  | 1.5        | +0.5 j pour fixtures Python ↔ Go + tests tolérance + EWMA `adjust=True` + i18n complet |
|   8   | Nettoyage final                          | 0.5         | 0.5        | inchangé                                                |
| **Total** |                                      | **11.0**    | **10.3**   | Objectif relevé : parité numérique vérifiée            |

### 6.2 Risques

| Risque                                                 | Impact   | Mitigation                                                                  |
|--------------------------------------------------------|----------|-----------------------------------------------------------------------------|
| Données `highlight_events` indisponibles localement     | Élevé    | Tester avec dataset démo, prévoir fallback `null` (cohérent avec v7 sur ce point). |
| Coût de chargement `match_skill_rank` sur petit dataset | Faible   | LEFT JOIN seulement, OK.                                                    |
| Perf side recharts/Plotly sur > 1000 matchs             | Moyen    | LTTB obligatoire Phase 7.                                                   |
| Incohérence Form Score Python vs Go                    | Moyen    | Test de parité numérique avec un fixture commun (`testdata/timeseries_parity.json`). |
| `WithDataAdapter` jamais activé après portage          | Faible   | Ne pas toucher tant que la migration multi-titre n'est pas prioritaire.     |
| Dérive des couleurs vs tokens                          | Moyen    | Lint custom déjà en place dans `apps/web/`. Vigilance manuelle.             |

### 6.3 Ordre d'exécution recommandé

L'utilisateur peut basculer le portage en mode itératif et commit-par-commit :

```
git checkout -b feat/timeseries-parity-v7
# Phase 0 — 1 commit "chore(timeseries): align rolling/minTrend, drop dead Plotly stubs"
# Phase 1 — 4 commits
# Phase 2 — 1 commit "feat(timeseries): add Cartes & Modes tab"
# Phase 3a — 1 commit "feat(timeseries): enrich MatchRow with combat columns"
# Phase 3b — 1 commit "feat(timeseries): backend Progression timelines"
# Phase 3c — 1 commit "feat(timeseries): frontend Progression tab"
# Phase 4 — 2 commits
# Phase 5 — 5 commits
# Phase 6 — 1 commit
# Phase 7 — 2 commits
# Phase 8 — 1 commit
```

---

## 7. Annexes

### 7.1 Mapping des onglets Python → Go (cible)

| Python         | Go cible          | Note                                                          |
|----------------|-------------------|---------------------------------------------------------------|
| Résumé         | `summary`         | Enrichir, conserver le slug actuel.                            |
| Cartes & Modes | `cartes-modes`    | Nouvel onglet à insérer entre `summary` et `distributions`.    |
| Distributions  | `distributions`   | Compléter (lifespan + perf score).                            |
| Progression    | `progression`     | Nouvel onglet à insérer après `cartes-modes`.                 |
| Avancé         | `avance`          | Reprendre les éléments actuels de `cumul`, `form`, `intensity` et y ajouter Skill rank, Cumul+IC, Net score/h, etc. |
| —              | `combat` (S56)    | Conserver tel quel (inédit Go).                                |

Les onglets Go actuels `cumul` / `form` / `intensity` peuvent être absorbés dans `avance` pour réduire la friction utilisateur, OU conservés en sous-sections. Décision UX à valider.

### 7.2 Calculs cibles avec parité Python

| Calcul                       | Code Python ref                                            | Cible Go (méthode service)                             |
|------------------------------|------------------------------------------------------------|--------------------------------------------------------|
| Form score                   | `compute_form_score_history` (`src/analysis/_performance_form.py`) | `BuildFormScoreSeries(history)`                       |
| EWMA K/D                     | `compute_ewma_kd_polars(alpha=0.2)`                        | `buildTimeseriesFormTab` (déjà OK, alpha aligné)       |
| Régression linéaire          | `compute_linear_regression_kd`                             | `computeRegressionStats` (ajouter winrate_slope)       |
| Cumul K/D + IC 90 %          | `compute_cumulative_kd_with_ci(z=1.645)`                   | `BuildCumulativeKDWithCI`                              |
| Net score/heure              | `compute_rolling_net_score_per_hour(window_size=None)`     | `BuildNetScorePerHour` (window adaptatif)              |
| Score per minute             | `compute_score_per_minute`                                 | `buildIntensityTab` (déjà OK)                          |
| Rolling win rate (window=5)  | `compute_rolling_win_rate`                                 | `buildRollingWRBuckets` (window=14, à aligner Phase 0) |
| First event distribution     | `load_first_event_times`                                   | `BuildFirstEventDistribution`                          |
| Perfect kills                | `load_perfect_kills`                                       | Enrichir `MatchRow.perfect_kills`                      |
| Match intensity profile      | `compute_match_intensity_profiles(n_buckets=10)`           | `BuildMatchIntensityProfile`                           |
| Skill rank history           | `_load_lusr_history` (repo Python)                         | `LoadSkillRankHistory` (nouveau repo)                  |

### 7.3 Sources DB nécessaires (rappel)

- `data/players/{gamertag}/stats.duckdb`
  - `match_skill_rank` (rating LUSR/CSR par match)
  - `personal_score_awards`
- `data/warehouse/shared_matches_v2.duckdb`
  - `match_participants` (colonnes manquantes : `headshot_kills`, `max_killing_spree`, `shots_fired`, `shots_hit`, `team_mmr`, `enemy_mmr`)
  - `highlight_events` (premier kill / première mort, kill timing buckets)
  - `medals_earned` (perfect kills via citation Q30 — cf. fix `2d32413d`)
- `data/warehouse/metadata.duckdb`
  - `career_ranks` (tier_label pour skill rank)

### 7.4 Références croisées dans le repo

- Plan jumeau Match View : [PLAN_MATCH_VIEW_GO_PORTAGE.md](PLAN_MATCH_VIEW_GO_PORTAGE.md)
- Plan méta fondations : [PLAN_META_FOUNDATIONS_GO.md](PLAN_META_FOUNDATIONS_GO.md)
- **Spec ECharts graphes** : [SPEC_ECHARTS_TIMESERIES.md](SPEC_ECHARTS_TIMESERIES.md) — `buildOption` pour les 13 graphes, tokens, helpers partagés, champs Go manquants
- Audit teammates : [docs/AUDIT_TEAMMATES_V7_COCKPIT.md](../docs/AUDIT_TEAMMATES_V7_COCKPIT.md)
- Charts existants : [.ai/CHARTS_AND_TABLES.md](CHARTS_AND_TABLES.md)
- Migration gaps : [docs/MIGRATION_GAP_PYTHON_TO_GO.md](../docs/MIGRATION_GAP_PYTHON_TO_GO.md)

### 7.5 Variante allégée — « 60 % de parité en 3 jours »

Si la pleine parité (~10 j-h) n'est pas finançable court terme, une variante minimaliste qui livre rapidement les charts les plus impactants visuellement :

| Phase | Inclus                                                                                         | Charge |
|------:|------------------------------------------------------------------------------------------------|-------:|
|     0 | Validation DB + alignement paramètres + nettoyage (identique au plan complet)                  |    1.0 |
|     1 | KPIs enrichis (8 cartes + outcomes bar) + Streaks + Outcomes over time                         |    0.8 |
|     2 | Onglet Cartes & Modes complet                                                                  |    0.8 |
|   3-bis | Onglet Progression **partiel** : Performance + Assists + Per-min + Damage (4 timelines sur 10) |    1.0 |
| **Total** |                                                                                            | **3.6** |

Couverture résultante : ~60 % (vs 28 % aujourd'hui, vs 95 % en plan complet). Les éléments **non livrés** dans cette variante : Form Score, Skill Rank LUSR/CSR, Lifespan/Spree/Shots/Rank&Score timelines, First event distribution, Intensity heatmap match × phases, Cumul + IC 90 %, Net score/heure, Top by week, Top weapons.

À considérer si on veut prioriser une vraie page « cartes/modes + KPIs riches » avant d'attaquer le morceau le plus dense (Progression + Skill rank).

---

**Fin du plan.** Total estimé : **~10.3 j-h** (ajusté, avec parité numérique vérifiée), variante allégée à ~3.6 j-h. La page passe d'un MVP à ~28 % de couverture à une **parité fonctionnelle ~95 % + parité numérique ~99 % vérifiée par fixtures + i18n FR/EN complet**, plus le bonus Combat Yield (S56) qui reste exclusif au Go.
