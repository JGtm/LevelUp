# Axe 1 — Agnosticisme données ↔ charts

Date : 2026-04-29
Branche : feat/multi-title-static-fs-rescope
Périmètre : apps/go-api/internal/{analysis,service,api,domain,games/canonical}/ + apps/web/src/{components/charts,features,lib}/

## Synthèse (3-5 lignes max)

Verdict global : **moyen, avec un découplage à deux vitesses**. La couche basse charts (`components/charts/*` + `domain.ChartSeries[T]`) est saine — un contrat générique `ChartSeries<T>` mirroré Go↔TS, builders `buildXxxOption` testables, tokens couleur sémantiques. **Mais le contrat n'est pas appliqué uniformément** : Squad V2 + Timeseries respectent bien le pattern, alors que Home, Synthesis, Career, Match-View et Session-Compare exposent des DTOs pré-formatés (couleurs hex, labels, scores arrondis), avec 4–5 implémentations divergentes du même indicateur côté Go (unités 0..1 vs 0..100 pour `win_rate` ET `accuracy`) et 2 recomputes K/D côté front (1 par oubli de DTO, 1 mathématiquement faux). Le canonical `PlayerMatchRow` existe mais n'est consommé que par Squad V2 — les autres services lisent des `domain.HomeMatchRow`/`StatsMatchRow`/`SquadMatchRow` calqués sur les colonnes DuckDB.

> **Amendement 2026-04-29** : KDA n'est jamais recomputé côté front (vérifié). Les 2 recomputes front initialement pris pour des KDA sont en fait des K/D (KDR), à supprimer également (décision produit). Le BLOQUANT « Win Rate » est étendu à `accuracy` (même divergence d'unité) + précision décimale à harmoniser.

## Constats

### [BLOQUANT] Win Rate ET Accuracy — unités divergentes 0..1 vs 0..100 + précision décimale non harmonisée

> Amendement 2026-04-29 : étendu à `accuracy` (même divergence d'unité) et à la précision décimale (toFixed 0/1/2/3 selon le module).

#### Win Rate — 7+ implémentations Go avec unités divergentes

- **Fichier:ligne** :
  - `apps/go-api/internal/analysis/home.go:216` → `WinRate: float64(wins) / float64(total)` (0..1)
  - `apps/go-api/internal/analysis/squad_breakdown.go:44` → `wr = math.Round(float64(wins)/float64(totalWL)*1000) / 10` (0..100, 1 décimale)
  - `apps/go-api/internal/analysis/squad_breakdown.go:358` → `kpis.WinRate = math.Round(float64(wins)/float64(totalWL)*1000) / 1000` (0..1)
  - `apps/go-api/internal/service/session_compare_service.go:329` → `return float64(wins) / float64(len(matches)) * 100` (0..100)
  - `apps/go-api/internal/service/squad_service.go:192` → `winRates = append(winRates, float64(wins)/float64(total)*100)` (0..100)
  - `apps/go-api/internal/service/squad_service_v2.go:500` → `card.WinRate = float64(wins) / float64(wins+losses)` (0..1)
  - `apps/go-api/internal/service/teammates_service.go:375,414` → `WinRate: round2(float64(wins) / float64(n) * 100)` (0..100)
  - `apps/go-api/internal/service/timeseries_service.go:102,624` → `winRate := float64(wins) / float64(n) * 100` (0..100)
  - `apps/go-api/internal/service/stats_service.go:172` → `winRate = float64(wins) / float64(len(matches)) * 100.0` (0..100)
  - `apps/go-api/internal/prestige/evaluator.go:166` → `return float64(wins) / float64(len(matches)) * 100.0`

#### Accuracy — divergence 0..1 vs 0..100 selon le DTO

- **Fichier:ligne** :
  - **0..1** : `apps/go-api/internal/domain/match_view.go:107,278` (`Accuracy *float64`) consommé avec `*100` côté front dans `apps/web/src/features/match-view/PlayerDetailPanel.tsx:51` et `apps/web/src/features/compare/CompareSurface.tsx:57`
  - **0..100** : `apps/go-api/internal/domain/squad_v2.go:125` (commentaire explicite `// 0..100`), consommé tel quel dans `apps/web/src/components/shell/KPIBar.tsx:59` (`avg_accuracy.toFixed(1) + ' %'`) et `apps/web/src/components/ui/match-card.tsx:401` (`m.accuracy.toFixed(0) %`)

#### Précision décimale non standardisée

- `toFixed(0)` dans `match-card.tsx:401`, `toFixed(1)` dans `KPIBar.tsx:59`, `toFixed(2)` dans `match-card.tsx:363`, arrondi `Math.round(x*1000)/10` côté Go (`squad_breakdown.go:44`) — aucune convention.

- **Extrait** :
  ```go
  // home.go:216 — ratio 0..1
  WinRate: float64(wins) / float64(total),
  // session_compare_service.go:329 — pourcent 0..100
  return float64(wins) / float64(len(matches)) * 100
  // squad_v2.go:125 — accuracy 0..100 (commentaire explicite)
  AvgAccuracy float64 `json:"avg_accuracy"` // 0..100
  ```
- **Problème** : `win_rate` transporte 0..1 dans certains endpoints (Home, Synthesis, SquadV2) et 0..100 dans d'autres (Stats, Teammates, Compare, Timeseries). Idem pour `accuracy` : ratio dans MatchView/Compare, pourcent dans Home/Squad/MatchCard. 14+ endroits front font `* 100` parfois en double, parfois jamais (cf. `apps/web/src/features/squad/metrics.ts:46,54,73`, `HomePage.tsx:775`, `SynthesisPage.tsx:151,181,197,230`, `MatchHistoryTable.tsx:231`, `KPIBar.tsx:57`, `CompareSurface.tsx:54`, `SquadLayout.tsx:185`, `squad/charts/timelineChart.ts:35`). Précision décimale variable selon le module.
- **Action** :
  1. Centraliser dans `internal/analysis/indicators.go` : `WinRate(wins, total int) float64` et `Accuracy(hits, fired int) float64` retournant **toujours 0..1** (convention canonique).
  2. Supprimer tous les `*100` côté Go ; ne plus exposer ni stocker du 0..100 dans les DTOs.
  3. Faire le formatage `*100` + arrondi décimal **uniquement à l'affichage** côté front via un helper `formatPercent(ratio, decimals = 1)`.
  4. Standardiser la précision : 1 décimale par défaut, 2 décimales pour les ratios sub-unitaires (KDA, KDR), 0 décimale pour les compteurs.

### [BLOQUANT] KDA et K/D ratio (KDR) — recomputes en cascade, aucun n'est justifié

> Amendement 2026-04-29 : reclassé après vérif des usages. Le constat initial mélangeait KDA et K/D — le front ne recompute jamais un KDA, mais recompute du K/D (KDR) à 2 endroits dont 1 mathématiquement faux. **Décision produit : aucun recompute KDA/KDR/K/D n'est justifié, tout doit s'appuyer sur la valeur API.**

#### Backend — formule KDA dupliquée 3x inline au lieu d'un helper canonique

- `apps/go-api/internal/analysis/performance_score.go:175` → `kda := float64(row.Kills+row.Assists) / math.Max(1.0, float64(row.Deaths))`
- `apps/go-api/internal/analysis/performance_score.go:374-376,382` → recompute conditionnel si KDA nil (même formule)
- `apps/go-api/internal/sync/performance.go:75-78` → `kda := row.KDA; if kda == 0 { kda = (kills+assists)/math.Max(1, deaths) }`

Note : `apps/go-api/internal/analysis/squad_breakdown.go:48,151,241,300` (4 sites) sont des **agrégations** `avg(KDA)` à partir d'une KDA déjà calculée, pas des duplications de la formule — à centraliser pour cohérence d'arrondi mais moins critique.

#### Frontend — recomputes K/D (KDR), pas KDA, **tous à supprimer**

- `apps/web/src/features/timeseries/TimeseriesKdaBars.tsx:78` → `kdRatio: r.deaths > 0 ? r.kills / r.deaths : r.kills`
  - Cause : `domain.TimeseriesMatchRow` (`internal/domain/timeseries.go:117`) n'expose ni `kda` ni `kd_ratio`, seulement `accuracy`. Le front compense → **oubli backend**, pas une justification.
- `apps/web/src/features/synthesis/SynthesisPage.tsx:139-141` → `kd = overview.total_deaths > 0 ? (overview.total_kills / overview.total_deaths).toFixed(2) : '—'`
  - **Mathématiquement faux** : `sum(K)/sum(D) ≠ avg(K/D)`. `kpis.global_ratio` existe déjà côté API et donne le bon ratio global ; sinon exposer un `total_kdr` proprement calculé.

- **Extrait** :
  ```go
  // performance_score.go:175 (formule canonique répétée 3x inline en Go)
  kda := float64(row.Kills+row.Assists) / math.Max(1.0, float64(row.Deaths))
  ```
  ```ts
  // TimeseriesKdaBars.tsx:78 — recompute K/D parce que le DTO ne l'expose pas
  kdRatio: r.deaths > 0 ? r.kills / r.deaths : r.kills,
  // SynthesisPage.tsx:139-141 — recompute K/D faux sur totaux agrégés
  const kd = overview.total_deaths > 0
    ? (overview.total_kills / overview.total_deaths).toFixed(2)
    : '—'
  ```
- **Problème** : la formule canonique KDA = (K+A)/max(1,D) est documentée dans `apps/web/src/features/help/i18n.ts:326` mais répétée 3x inline en Go. Côté front, 2 recomputes K/D existent : l'un par oubli de DTO (`TimeseriesMatchRow` n'expose pas la donnée), l'autre par calcul incorrect (sum/sum sur totaux). KDA et KDR sont des **métriques distinctes et légitimes** (Halo expose les deux), mais l'API doit faire foi pour les deux — pas de recompute consommateur.
- **Action** :
  1. Créer `internal/analysis/indicators.go` avec `KDA(k, a, d int) float64` et `KDR(k, d int) float64` + tests, migrer les 3 sites Go (`performance_score.go:175,374-376`, `sync/performance.go:75-78`).
  2. Ajouter `kd_ratio` (et `kda`) sur `domain.TimeseriesMatchRow` (`internal/domain/timeseries.go:117`) → supprimer le recompute `TimeseriesKdaBars.tsx:78`.
  3. Exposer un `total_kdr` correct sur `domain.SynthesisOverview` (ou consommer `kpis.global_ratio` déjà disponible) → supprimer le recompute `SynthesisPage.tsx:139-141`.
  4. Audit final : aucun recompute KDA, KDR ou K/D ne doit subsister côté front. Toute donnée vient de l'API.

### [BLOQUANT] Couleurs hex dur servies par l'API (anti-pattern documenté CLAUDE.md règle 20)

- **Fichier:ligne** :
  - `apps/go-api/internal/service/match_view_service.go:34-39` (table `outcomeColors`)
  - `apps/go-api/internal/service/match_view_service.go:491,607,1168,1174-1180` (perfColor + outcomeColor sites)
  - `apps/go-api/internal/service/squad_service.go:47-48` → `analysis.ComputeParticipationProfile(myMatches, gamertag, "#4FC3F7")` et `"#FF8A65"`
  - `apps/go-api/internal/domain/chart/base.go:57-94` (HaloColors + OkabeIto + OutcomeColor + PerfColor)
- **Extrait** :
  ```go
  // match_view_service.go:34-39
  var outcomeColors = map[int]string{
      1: "#8b5cf6", 2: "#22c55e", 3: "#ef4444", 4: "#8b5cf6",
  }
  // squad_service.go:47-48
  myProfile := analysis.ComputeParticipationProfile(myMatches, gamertag, "#4FC3F7")
  tmProfile := analysis.ComputeTeammateProfile(tmMatches, tmName, "#FF8A65")
  ```
- **Problème** : l'API émet des hex `#22c55e`, `#ef4444`, `#4FC3F7` directement dans la payload (cf. `MatchViewHeader.OutcomeColor` toujours présent), ce qui couple le DTO au thème Tailwind du front et empêche dark/light/accessibilité Okabe-Ito. La règle 20 du `CLAUDE.md` (côté Python) interdit explicitement les hex côté backend ; la version Go expose `OutcomeColorToken` mais conserve `OutcomeColor` "deprecated" toujours rempli.
- **Action** : supprimer `OutcomeColor`/`PerfColor` (champs deprecated) du DTO `MatchViewHeader`/`MatchPersonalResult` après vérif qu'aucun front ne les lit, retirer les `outcomeColors`/`perfColor`/`HaloColors`/`OkabeIto` Go et le param `color` de `ComputeParticipationProfile` — les tokens (`outcome-win`, `perf-tier-1`, `chart-series-1`) suffisent.

### [BLOQUANT] Canonical `PlayerMatchRow` non utilisé par les services produit principaux

- **Fichier:ligne** : `apps/go-api/internal/games/canonical/match.go:137` (PlayerMatchRow défini), consommé uniquement dans `apps/go-api/internal/service/squad_service_v2*.go` + `explorer_service.go` + `match_history_service.go`
- **Extrait** :
  ```go
  // canonical/match.go:137 — contrat censé être consommé partout
  type PlayerMatchRow struct {
      Summary    MatchSummary
      Self       MatchParticipant
      Enrichment PlayerMatchEnrichment
  }
  ```
- **Problème** : `home_service.go`, `career_service.go`, `synthesis_service.go`, `timeseries_service.go`, `match_view_service.go`, `session_compare_service.go`, `citations_service.go` lisent leurs propres `domain.HomeMatchRow` / `domain.StatsMatchRow` / `domain.SquadMatchRow` / `domain.HomeMatchRow` / `domain.SynthesisMatchRow` calqués 1-1 sur les colonnes DuckDB Halo Infinite. Conséquence : pour ajouter un titre B, il faut soit forker N row-types soit migrer N services. Multi-titres ne fonctionne aujourd'hui que pour Squad V2 et le `MultiTitlePreviewHandler`.
- **Action** : faire converger les services restants vers `canonical.PlayerMatchRow` en plusieurs sprints — commencer par `home_service.go` et `synthesis_service.go` (faible volume de stats, ROI rapide). Documenter dans une ADR le plan de bascule progressive.

### [BLOQUANT] DTOs Timeseries / Synthesis pré-shape pour ECharts (perte d'agnosticisme)

- **Fichier:ligne** :
  - `apps/go-api/internal/domain/timeseries.go:73-78` `IntensityHeatmapPoint{DayOfWeek, Hour, Count, AvgKD}`
  - `apps/go-api/internal/domain/timeseries.go:91-95` `DistributionBucket{BinStart, BinEnd, Count}`
  - `apps/go-api/internal/domain/timeseries.go:100-105` `CorrelationDataPair{Label, X, Y, Outcome}`
  - `apps/go-api/internal/domain/squad.go:185-190` `HeatmapCell{RowKey, ColKey, Value, Count}`
  - `apps/go-api/internal/domain/squad.go:215-221` `ComparisonMetricItem{Label, SoloValue, SquadValue, SoloText, SquadText}` (texte pré-formaté côté backend)
  - `apps/go-api/internal/domain/timeseries.go:31-37` `TimeseriesKpiCard{Label, Value, Delta, Color}` (Color géré côté backend)
- **Problème** : ces DTOs portent les noms d'axes ECharts (`X`/`Y`, `BinStart`/`BinEnd`, `RowKey`/`ColKey`, `Label`/`Value`/`Color`) au lieu de noms métier (`hour_of_day`, `kda_bin_lower`, `map_name`, `mode_name`). Les histogrammes Timeseries sont pré-binnés côté Go avant transit JSON — changer le nombre de bins requiert un appel API au lieu d'un re-render front. `ComparisonMetricItem.SoloText`/`SquadText` contient déjà du texte localisé formaté.
- **Action** : renommer les champs vers leur sémantique métier dans les DTOs (`Hour`/`DayOfWeek` au lieu de `X`/`Y` héritage chart). Supprimer `SoloText`/`SquadText` — laisser le front formater. Pour les buckets, exposer la donnée brute (matchs avec leurs valeurs) si la cardinalité < 1000, ou alors documenter explicitement que ce bucket est figé pour ECharts.

### [DETTE] Fonctions service `winRate`, `avgKD`, `killsPerGame`, `deathsPerGame` privées et redéfinies par feature

- **Fichier:ligne** : `apps/go-api/internal/service/session_compare_service.go:319-365` (4 helpers privés) ; equivalents inline dans `analysis/home.go:1663`, `analysis/squad_breakdown.go:19-61`, `service/squad_service.go:170-200`, `service/teammates_service.go:370-415`, `service/stats_service.go:165-180`
- **Extrait** :
  ```go
  // session_compare_service.go:347-355
  func killsPerGame(matches []domain.StatsMatchRow) float64 {
      if len(matches) == 0 { return 0 }
      var total int
      for _, m := range matches { total += m.Kills }
      return float64(total) / float64(len(matches))
  }
  ```
- **Problème** : 5+ helpers privés identiques par module, chacun typé sur son propre row-type (`StatsMatchRow` / `SquadMatchRow` / `HomeMatchRow`). Pas de risque de divergence aujourd'hui mais dette technique évidente.
- **Action** : déplacer ces helpers vers `internal/analysis/indicators.go` (générique sur une interface `IndicatorSource{ Kills() int; Deaths() int; Outcome() Outcome }`). Aujourd'hui `analysis.ComputeKPIStats` pour `canonical.PlayerMatchRow` joue déjà ce rôle ; étendre.

### [DETTE] DTOs HTTP pré-formatés (labels FR + scores formatés)

- **Fichier:ligne** : `apps/go-api/internal/domain/match_history.go:53-72` (`MatchHistoryRow` : `OutcomeLabel`, `ScoreLabel`, `StartTimeLabel`, `AverageLifeMMSS`, `MatchURL`)
  ; `apps/go-api/internal/domain/home.go:228-234` (`RecentMatchItem` : `Title`, `Detail`, `OutcomeLabel`, `OutcomeTone`)
  ; `apps/go-api/internal/domain/squad.go:217-219` `ComparisonMetricItem.SoloText`/`SquadText`
- **Problème** : la frontière API expose des strings localisées (`OutcomeLabel = "Victoire"`, `MatchURL = "/match/abc"`) qui forcent une langue à l'API ou demandent une couche `Accept-Language`. Le pattern moderne adopté pour les charts (`LabelKey` + résolution front via `useFieldLabel`) n'est pas appliqué aux DTOs de tableau / hero card.
- **Action** : pour les nouveaux DTOs, passer `outcome_code` (canonical.Outcome) + `match_id` brut + `started_at_utc` (ISO) ; laisser le front formater via `useFieldLabel('outcome.win')`. Migrer `MatchHistoryRow` quand la table sera retouchée (pas urgent — la structure est fonctionnelle).

### [DETTE] `domain.charts.go` (ChartSeries[T]) cohabite avec `domain/chart/*` (legacy MatchSeries / SingleSeriesChartData)

- **Fichier:ligne** :
  - `apps/go-api/internal/domain/charts.go:21-60` (`ChartSeries[T]`, `ChartPoint2D`, `ChartPointStacked`, `ChartPointHeatmap`)
  - `apps/go-api/internal/domain/chart/base.go:16-50` (`DataPoint`, `LabeledValue`, `MatchSeries`, `SingleSeriesChartData`, `MultiSeriesChartData`, `NamedSeries`)
  - `apps/go-api/internal/domain/chart/antagonists.go` (`AntagonistEntry`, `DuelChartData`, `ImpactTimelineData`, `DominanceChartData`)
- **Problème** : deux contrats coexistent. Le moderne `ChartSeries[T]` (générique, mirroré TS) est utilisé par Squad V2. Le legacy `domain/chart/*` (port direct du Python `src/visualization/`) inclut hex `HaloColors`/`OkabeIto`. Aucun "deprecated" explicite ; un nouveau dev ne sait pas lequel utiliser.
- **Action** : ajouter un commentaire `// Deprecated: utiliser domain.ChartSeries[T] (charts.go)` en tête de `domain/chart/base.go` + `antagonists.go`, retirer `HaloColors`/`OkabeIto`/`OutcomeColor`/`PerfColor` (couleurs côté Go = anti-pattern), planifier la conversion progressive des consommateurs.

### [DETTE] ~~Recompute K/D côté front~~ — fusionné dans le BLOQUANT KDA/KDR ci-dessus

> Amendement 2026-04-29 : ce constat est désormais couvert par le BLOQUANT « KDA et K/D ratio — recomputes en cascade ». Décision produit : aucun recompute K/D front n'est justifié.

### [AMÉLIORATION] `SquadEngagementView` construit son propre `EChartsCoreOption` hors du pattern wrapper

- **Fichier:ligne** : `apps/web/src/features/squad/v2/SquadEngagementView.tsx:120-180` (`buildSquadEngagementOption`) ; `apps/web/src/features/synthesis/SynthesisPage.tsx:50-94` (`buildBipolaireOption`) ; `apps/web/src/features/timeseries/TimeseriesKdaBars.tsx` ; `apps/web/src/features/timeseries/TimeseriesCombatYield.tsx`
- **Problème** : 4 features importent directement `EChartsCoreOption` et construisent leur option ECharts à la main, hors `components/charts/`. Le test mental "remplacer ECharts par Recharts" coûte donc 4 fichiers + 11 wrappers = 15 fichiers. Acceptable mais à surveiller.
- **Action** : extraire `BipolarBarChart` et `EngagementCurve` (déjà dans `components/charts/EngagementCurve.tsx` !) en wrappers réutilisables — l'engagement curve est déjà extraite mais SquadEngagementView ne l'utilise pas. Documenter qu'au-delà de 2 occurrences d'un pattern custom, on extrait.

### [AMÉLIORATION] Fallback inline accuracy/ratio borné dans `metrics.ts` au lieu d'un type SquadMetric.ceiling

- **Fichier:ligne** : `apps/web/src/features/squad/metrics.ts:69-81` — `norm(v, max)` et plafonds magiques (3 pour KDR, 20 pour KPG, 10 pour APG, 1 pour accuracy)
- **Extrait** :
  ```ts
  const norm = (v: number | null, max: number): number =>
    v != null ? Math.min(100, (v / max) * 100) : 0
  ```
- **Problème** : ces plafonds sont valides pour Halo Infinite (cf. commentaire L65-67 qui le reconnaît : "à terme, exposer un `ceiling` dans le SquadMetric pour le rendre title-aware"). Pour synthetic_title_b, ces plafonds n'ont pas de sens.
- **Action** : ajouter `ceiling?: number` sur `SquadMetric` et le faire descendre du backend via le `fields.toml` du titre (champ `radar_ceiling`).

## Cartographie : flux d'un indicateur (KDA)

Backend Halo Infinite :
- Source brute : `shared.match_participants.kda` (DuckDB, déjà calculé au sync)
- Sync recompute (fallback si null) : `apps/go-api/internal/sync/performance.go:75-78`
- Lu en `domain.HomeMatchRow.KDA`, `domain.SquadMatchRow.KDA`, `domain.StatsMatchRow.KDA` (3 row-types distincts)
- Aggregation : `analysis/squad_breakdown.go:48,151,241,300` (4 sites avg KDA), `analysis/home.go:228-230` (kpis.AvgKDA), `analysis/kpi_stats.go` n'agrège PAS KDA explicitement
- Recompute si nil : `analysis/performance_score.go:175` + `:374-376` + `:382` (3 sites)
- DTO output : `HeroKPIs.AvgKDA` (Home), `SynthesisOverview.AvgKDA` + `SynthesisMatchHighlight.KDA` + `TopWeekEntry.AvgKDA`, `RecentMatchItem.KDA`, `MatchHistoryRawRow.KDA` (interne uniquement)

Frontend :
- KDA : aucun recompute — `match-card.tsx:363` consomme `m.kda.toFixed(2)`, `SynthesisHighlightsSection.tsx:18` consomme `item.kda.toFixed(2)`, `kpis.kd_ratio?.toFixed(2)` à `SynthesisPage.tsx:182`. **OK**.
- K/D (KDR) : 2 recomputes **non justifiés** à supprimer (cf. BLOQUANT) :
  - `SynthesisPage.tsx:139-141` recalcule `total_kills/total_deaths` (faux, sum/sum ≠ avg)
  - `TimeseriesKdaBars.tsx:78` recalcule `kills/deaths` faute d'avoir `kd_ratio` dans le DTO `TimeseriesMatchRow` (oubli backend, pas une justification)
- Wrapper consommateur : `<TimeseriesLineChart>` (KDA timeseries) / `<TimeseriesKdaBars>` (custom) / `<RadarChart>` (Squad V2 contributions)

**Étapes de transformation à risque** :
1. Sync calcule KDA → DB
2. Sync recalcule KDA si nil (fallback) — formule inline dupliquée 3x au lieu d'un helper
3. Service Go aggrège `avg(KDA)` différemment selon module (4 sites, arrondis variables)
4. DTO expose `kda` / `avg_kda` / `kd_ratio` / `global_ratio` selon endpoint (mais pas sur `TimeseriesMatchRow`)
5. Front recompute K/D dans 2 endroits — à supprimer (1 oubli DTO, 1 bug sum/sum)

## Suivi recommandé

- **Follow-up 1 (axe 1 sous-sujet)** : audit ciblé des unités au passage de la frontière HTTP. Lister tous les `*float64 / float32` exposés en JSON et les classer par convention (0..1, 0..100, brut, ratio, secondes, ms). Sortir un tableau "field → unité canonique" et corriger la divergence en une passe.
- **Follow-up 2 (axe 1 sous-sujet)** : ADR « formule canonique des indicateurs » qui tranche : KDA = (K+A)/max(1,D), KDR = K/max(1,D), WinRate = wins/total (0..1), Accuracy = ratio (0..1), durations en secondes, MMR brut. Centraliser dans `analysis/indicators.go`.
- **Follow-up 3 (hors axe)** : conversion progressive des services produit vers `canonical.PlayerMatchRow` (relève de l'axe multi-titres).

Constats hors-axe à reverser ailleurs : multi-titres (canonical non propagé aux services produit), Go layering (helpers privés dupliqués entre service/ et analysis/), i18n DTO (labels pré-localisés au lieu de keys + résolution front).
