# Plan de portage Squad — Python `v7/cockpit` → Go + React (ECharts)

> Plan enfant Phase 1 du méta-plan `PLAN_META_FOUNDATIONS_GO.md`.
> Pilote (avec MatchView) qui valide les fondations Phase 0 sur le cas le plus
> complexe : 22 sections distinctes, heatmap 8 rôles d'impact, radar 6 axes,
> kill timing intensity, etc.
>
> Branche cible : `feat/foundations-axes-1-3-4` (continuation Phase 0).
> Date : 2026-04-27.
> Source d'audit : `docs/AUDIT_TEAMMATES_V7_COCKPIT.md` (~630 L, ~22 sections).
> Plan jumeau : `PLAN_MATCH_VIEW_GO_PORTAGE.md` (à réécrire en parallèle).
> Plan complémentaire UX : `PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md` (sessions
> squad-only, multi-sélect, filtre amis) — intègre les sections P1 et P5.

---

## 0. Synthèse exécutive

La page Squad Python `v7/cockpit` est la plus dense du produit : **22 sections
distinctes** réparties en 2 onglets (Synergies / Contributions) avec heatmap
8 rôles d'impact (`silent_hero`, `false_brother`, `top_killer`, etc.), radar
6 axes (Combat/Survie/Soutien/Score/Objectif/Impact), tableaux d'armes,
galerie de médailles, cadence et intensité par phase de match.

La version Go actuelle représente **~20-25 %** de la richesse fonctionnelle :
3 charts opérationnels (heatmap simpliste 1D, timeline 2 traces, HS/PK
partiel), 1 radar approximatif, 0 tableau d'armes, 0 galerie médailles, 0
heatmap impact 8 rôles, 0 cadence, 0 intensité. Voir
`docs/AUDIT_TEAMMATES_V7_COCKPIT.md` § 7 pour le détail.

**Bonne nouvelle** : **toutes les briques sont prêtes côté fondations Phase 0**.
Le plan se concentre sur l'**adoption** des fondations + la création des
wrappers ECharts spécialisés + la composition React.

### Effort estimé

| Phase | Effort | Bloque |
|---|---:|---|
| P1 — Service Squad sur `LoadPlayerMatches` × N + sessions squad-only | 1.5j | P2-P9 |
| P2 — En-têtes (KPI personnels + score d'équipe + cartes joueurs ▲▼) | 1j | — |
| P3 — Synergies par carte (lollipop, bullet, perf vs historique, heatmap 2D) | 1.5j | — |
| P4 — Timeline + Form Score (LOWESS) | 1j | — |
| P5 — Impact 8 rôles (heatmap + ranking MVP/Boulet) | 1.5j | — |
| P6 — Cadence + Intensité (LoadHighlightEvents + nouveaux algos) | 1.5j | — |
| P7 — Contributions (per-min, 6 charts trio dédiés, killing spree, HS/PK) | 2j | — |
| P8 — Radar 6 axes (`ComputeParticipationProfile`) | 1j | — |
| P9 — Tableau historique + armes + médailles + légende flottante | 2j | — |
| **Total Phase 1 Squad** | **~13j** | |

Aligné avec l'estimation méta-plan § 6.1 (12-13j pour les 2 pilotes Squad +
MatchView).

### Critères de succès

- [ ] Page Squad fonctionne end-to-end avec 1 + 2 coéquipiers + 3 coéquipiers.
- [ ] 22 sections de l'audit présentes (ou explicitement skippées avec `<CapabilityGap>`).
- [ ] Aucun import `plotly.js` ni `recharts` dans `apps/web/src/features/squad/`.
- [ ] Aucune string hardcodée dans `features/squad/` (ESLint `error` localement).
- [ ] Manifest `apps/web/src/lib/i18n/manifests/squad.toml` complet (~80 clés).
- [ ] Couverture Go ≥ 80 % sur `service/squad_service.go`.
- [ ] Couverture frontend ≥ 80 % sur `features/squad/`.
- [ ] Test E2E `e2e/squad.spec.ts` passe sur Chromium / Firefox / WebKit.
- [ ] Test golden parity sur `cmd/foundations_golden_parity --page squad`.
- [ ] `dominance_flag` peuplé sur la DB de test via `engine.RunBackfillComebackBadges(ctx, true)`.
- [ ] Capability gating testé (titre synthétique sans `match.history` → dégradation gracieuse).

---

## 1. Périmètre — mapping audit → sections cibles

Récap des 22 sections de l'audit (`docs/AUDIT_TEAMMATES_V7_COCKPIT.md` § 2-4)
mappées vers les fondations Phase 0 disponibles.

### 1.1 En-têtes (§ 2 audit)

| # | Section | Fondations utilisées | Phase |
|---|---|---|---|
| 1 | Bandeau KPI personnels (8 cartes + tendance vs all-time) | `LoadPlayerMatches` (scope: courant + all-time) | P2 |
| 2 | Score d'équipe + grade lettre + bonus | `analysis.ComputeSquadPerformanceScore` (existe déjà) | P2 |
| 3 | Cartes scores individuels avec badge ▲▼ | `analysis.ComputeSessionPerformanceScore` (existe déjà) | P2 |
| 4 | Sentinelle légende `#llp-squad-start` | composant transverse | P9 |

### 1.2 Onglet Synergies (§ 3 audit)

| # | Section | Fondations utilisées | Phase |
|---|---|---|---|
| 5 | Lollipop W/L par carte | `analysis/breakdown.ByMap` + `<Lollipop>` | P3 |
| 6 | Bullet winrate session vs historique | `analysis/breakdown.ByMap` + `CompareToHistorical` + `<Bullet>` | P3 |
| 7 | Perf vs historique par carte | `analysis/breakdown.CompareToHistorical` + `<BarStacked>` | P3 |
| 8 | Heatmap escouade joueur × carte | `LoadPlayerMatches` × N + `<Heatmap2D>` | P3 |
| 9 | Timeline performance multi-joueurs + outcome marker | `LoadPlayerMatches` × N + `<TimeseriesLine>` | P4 |
| 10 | Form Score lissé (LOWESS) | nouveau `analysis/temporal.LowessSmooth` (chunk dédié) | P4 |
| 11 | Impact 8 rôles (heatmap + ranking MVP/Boulet) | `LoadHighlightEvents` + `narrative.IdentifyImpactRoles` ✓ | P5 |
| 12 | Tableau historique escouade | `LoadPlayerMatches` × N intersection | P9 |
| 13 | Cadence trio (kills/phase 60s) | `LoadHighlightEvents` + nouvel algo bucket 60s | P6 |

### 1.3 Onglet Contributions (§ 4 audit)

| # | Section | Fondations utilisées | Phase |
|---|---|---|---|
| 14 | Stats par minute (frags/morts/assists par min) | `LoadPlayerMatches` + projection per-min | P7 |
| 15 | Radar Participation 6 axes | `narrative.ComputeParticipationProfile` ✓ + `<Radar>` | P8 |
| 16 | Frags ↑ / Morts ↓ combinés | `LoadPlayerMatches` × N + `<BarGrouped>` | P7 |
| 17 | Assists / KDA / Précision / Vie moyenne / Performance (5 charts) | `LoadPlayerMatches` × N + `<TimeseriesLine>` × 5 | P7 |
| 18 | Killing spree (max + smoothing) | `LoadPlayerMatches` + `temporal.RollingMeanAdaptive` ✓ | P7 |
| 19 | HS+PK stacked + records hachurés | `<BarStacked>` + nouvelle prop `recordOverlay` | P7 |
| 20 | Heatmap intensité match × 10 phases | `LoadHighlightEvents` + nouvel algo `ComputeIntensityProfile` | P6 |
| 21 | Tableau armes + slider min kills + grenade/mêlée | nouveau repo `LoadWeaponKillsAggregated` | P9 |
| 22 | Galerie médailles (top 20 matchs partagés) | nouveau repo `LoadMedalsForMatches` | P9 |

### 1.4 Hors périmètre Phase 1 (différé)

- Sessions trio détectées (`_detect_trio_session`) : reporter Phase 2 ou plus tard.
- Records hachurés `SquadRecordSet` : implémentés en Phase 1 mais simplifié (records joueur uniquement, pas dominant_pair).
- Légende joueurs flottante via `IntersectionObserver` : Phase 1 (transverse).

---

## 2. État actuel Go (référence rapide)

```
apps/web/src/features/squad/
  SquadLayout.tsx              navigation 2 onglets
  SquadContext.ts              sélection coéquipiers
  SquadSynergiesPage.tsx       227 L, 1 barplot générique + 3 charts simplistes
  SquadContributionsPage.tsx   143 L, 1 radar approximatif
  charts/heatmapChart.ts       72 L, heatmap 1D (carte × win rate)
  charts/timelineChart.ts      79 L, line chart 2 traces
  charts/hsPkChart.ts          79 L, HS/PK stacked (sans records)
  queries.ts, metrics.ts
```

**~931 L vs ~6000 L Python**. Tout va être réécrit en Phase 1.

```
apps/go-api/internal/service/squad_service.go
apps/go-api/internal/service/teammates_service.go
apps/go-api/internal/platform/duckdb/squad_repo.go        Q29/Q30/Q31
apps/go-api/internal/analysis/squad_score.go              ✓ ComputeSquadPerformanceScore
apps/go-api/internal/analysis/squad_impact.go             4 rôles bilatéral, à étendre via narrative
```

---

## 3. Architecture cible (réutilisation fondations Phase 0)

### 3.1 Service Squad refactoré

```go
// apps/go-api/internal/service/squad_service.go (~400L cible)
type SquadService struct {
    matchesRepo port.PlayerMatchesRepository  // Phase 0 chunk 6
    eventsRepo  port.HighlightEventsRepository // Phase 0 chunk 7
    pool        *duckdb.PlayerPool             // existant
}

func (s *SquadService) GetSquadPage(
    ctx context.Context,
    mainGT string,
    teammateGTs []string, // 1..3 coéquipiers
    period temporal.Period,
) (*domain.SquadPage, error) {
    // 1. Charger les matchs du joueur principal + chaque coéquipier
    // 2. Intersection in-memory sur match_id (matchs où tous étaient présents)
    // 3. Selon les sections demandées :
    //    - breakdown.ByMap pour lollipop/bullet/perf
    //    - LoadHighlightEvents pour impact/cadence/intensity
    //    - narrative.IdentifyImpactRoles pour heatmap 8 rôles
    //    - narrative.ComputeParticipationProfile pour radar 6 axes
    //    - temporal.RollingMeanAdaptive pour Form Score / smoothing
    // 4. Capability gating : si match.history absent → games.ErrCapabilityNotSupported
    //    avec section flagguée capability_gap dans le DTO partiel.
}
```

### 3.2 DTOs Squad alignés `ChartSeries[T]`

Suppression de tout payload Plotly server-side. Les données passent en
`ChartSeries[ChartPoint2D]` ou `ChartSeries[ChartPointHeatmap]`.

```go
type SquadPageResponse struct {
    Header struct {
        SoloKPIs        canonical.KPIStats    // 8 cartes + tendance
        SquadScore      *SquadScoreCard       // base+bonus+grade
        PlayerCards     []PlayerScoreCard     // ▲▼ vs avg
    }
    Synergies struct {
        MapBreakdown        *ChartSeries[ChartPointStacked] // lollipop W/L
        BulletWinrate       *ChartSeries[ChartPointStacked] // bullet vs hist
        PerfVsHistorical    *ChartSeries[ChartPoint2D]      // delta perf
        HeatmapPlayerMap    *ChartSeries[ChartPointHeatmap] // 2D
        Timeline            []*ChartSeries[ChartPoint2D]    // 1 par joueur
        FormScore           *ChartSeries[ChartPoint2D]      // LOWESS
        ImpactRoles         *ImpactRolesMatrix              // 8 rôles × N joueurs
        ImpactRanking       []ImpactRanking                 // table 8 colonnes
        Cadence             *ChartSeries[ChartPointStacked] // kills/phase 60s
        History             []HistoryRow                    // tableau matchs
    }
    Contributions struct {
        PerMinute           *ChartSeries[ChartPointStacked]
        Radar               *ChartSeries[ParticipationScore] // 6 axes
        FragsDeaths         *ChartSeries[ChartPointStacked]  // groupé
        Assists, KDA, Accuracy, AvgLife, Performance []*ChartSeries[ChartPoint2D]
        KillingSpree        *ChartSeries[ChartPoint2D] // smoothed
        HsPk                *ChartSeries[ChartPointStacked]
        Intensity           *ChartSeries[ChartPointHeatmap] // match × phase
        WeaponsTable        *WeaponsTable
        WeaponsBarplot      *ChartSeries[ChartPointStacked]
        MedalsGallery       []MedalGalleryRow
    }
    Capabilities []domain.CapabilityGap // sections désactivées
}
```

### 3.3 Wrappers ECharts spécialisés

À créer dans `apps/web/src/components/charts/` (extension du `<ChartCard>`
livré chunk 11) :

| Wrapper | Sections consommatrices |
|---|---|
| `<Lollipop>` | P3 lollipop W/L + Career map history (réutilisé) |
| `<Bullet>` | P3 bullet winrate + Career bullet (réutilisé) |
| `<BarStacked>` (déjà à créer) | Multiple |
| `<BarGrouped>` | P7 frags/deaths combinés |
| `<Heatmap2D>` | P3 player×map + P6 intensity + Synthesis WL |
| `<Radar>` | P8 + MatchView radar |
| `<Cadence>` | P6 (Squad + MatchView + Timeseries) |
| `<TimeseriesLine>` | P4 + P7 (réutilisé Career, Timeseries) |
| `<Donut>` | P9 weapons top |

**Tous étendent `<ChartCard buildOption={...}>`** (chunk 11). Leur API prop
prend `ChartSeries<T>[]` directement.

### 3.4 Composants UI partagés

| Composant | Section | À créer |
|---|---|---|
| `<NarrativeBadge>` (chunk 14 ✓) | Impact roles + dominance | déjà livré |
| `<CapabilityGap>` (chunk 14 ✓) | Sections désactivées | déjà livré |
| `<KPIStrip>` | Header bandeau personnel | nouveau, transverse Phase 1 |
| `<PlayerScoreCard>` | Header cartes joueurs ▲▼ | spécifique Squad/MatchView |
| `<FloatingLegend>` | Légende observer fixée | nouveau, IntersectionObserver |
| `<ImpactRolesMatrix>` | Heatmap 8 rôles avec emojis | nouveau, P5 |
| `<MedalsGallery>` | Galerie médailles | nouveau, P9 |
| `<WeaponsTable>` | Tableau armes filtré | nouveau, P9 |

---

## 4. Phases d'implémentation

### Phase P1 — Service Squad sur fondations + sessions squad-only (~1.5j)

> Intègre `PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md` partiel : multi-sélect sessions
> + filtre coéquipiers amis uniquement.

**Tâches Go** :
- [ ] Refactor `SquadService.GetSquadPage(ctx, mainGT, teammateGTs[], period)` sur `LoadPlayerMatches` × (1 + N).
- [ ] Helper `intersectMatches(slices [][]canonical.PlayerMatchRow) []canonical.PlayerMatchRow` (intersection in-memory sur match_id).
- [ ] Endpoint `GET /api/v1/players/{slug}/pages/squad?teammates=gt1,gt2,gt3&period=1y`.
- [ ] Capability gating : `games.ErrCapabilityNotSupported` → DTO partiel + `[]CapabilityGap`.
- [ ] Pré-requis `dominance_flag` : appeler `engine.RunBackfillComebackBadges(ctx, false)` une fois en seed CI.

**Tâches frontend** :
- [ ] `SquadContext` étendu : `pickedSquadSessions: string[]` (multi-sélect).
- [ ] `useSquadPage(slug, teammates, period)` hook avec invalidation propre.

**Tests** :
- [ ] `service/squad_service_test.go` : intersection 1/2/3 coéquipiers, dégradation capability.
- [ ] `api/handlers/squad_handler_test.go` : 200/400/404/503.
- [ ] `e2e/squad.spec.ts` (skeleton, étendu phase suivante).

### Phase P2 — En-têtes (KPI + score équipe + cartes joueurs) (~1j)

**Tâches Go** :
- [ ] `service.computeSoloKPIs(rows []canonical.PlayerMatchRow, scope) canonical.KPIStats` (réutilise composants `analysis/`).
- [ ] `service.buildPlayerScoreCards(rows, teammates) []PlayerScoreCard`.
- [ ] DTO `SquadHeader` rempli.

**Tâches frontend** :
- [ ] `<KPIStrip>` partagé (8 cartes + tendance vs all-time) — composant transverse.
- [ ] `<PlayerScoreCard>` (compact, badge ▲▼ vs avg).
- [ ] Manifest `squad.toml` § header (~15 clés).

**Tests** :
- [ ] Vitest `<KPIStrip>` (8 états + tendance).
- [ ] Service test computeSoloKPIs avec fixture all-time.

### Phase P3 — Synergies par carte (~1.5j)

**Wrappers à livrer** :
- [ ] `<Lollipop>` ECharts (chunk dédié).
- [ ] `<Bullet>` ECharts.
- [ ] `<BarStacked>` ECharts (perf vs hist + outcome stacked).
- [ ] `<Heatmap2D>` ECharts (joueur × carte).

**Tâches Go** :
- [ ] `service.buildMapBreakdownLollipop(rows) ChartSeries[ChartPointStacked]`.
- [ ] `service.buildBulletWinrate(rows, historicalRows) ChartSeries`.
- [ ] `service.buildHeatmap2DPlayerMap(rowsPerPlayer map[string][]Row) ChartSeries[ChartPointHeatmap]`.

**Tests** :
- [ ] Vitest snapshot des `option` ECharts pour chaque wrapper.
- [ ] Service tests fixtures canoniques (3 cartes, 3 outcomes).

### Phase P4 — Timeline + Form Score (~1j)

**Pré-requis algorithmique** : `analysis/temporal.LowessSmooth(points, alpha)`
n'existe pas en Phase 0. À ajouter ici (ou implémentation simple via moyenne
mobile pondérée si LOWESS gonum trop lourd).

**Tâches Go** :
- [ ] Décision LOWESS : (a) lib gonum/stat ; (b) implémentation locale 50L.
- [ ] `service.buildTimelineMultiPlayer(rowsPerPlayer) []ChartSeries`.
- [ ] `service.buildFormScore(rowsMain, alpha=0.3) ChartSeries`.

**Tâches frontend** :
- [ ] `<TimeseriesLine>` ECharts multi-trace (déjà à créer en P1 réutilisé).
- [ ] Outcome markers (W/L/T) en symbol custom.

### Phase P5 — Impact 8 rôles (~1.5j)

> Plus gros morceau visuel. Utilise `narrative.IdentifyImpactRoles` ✓ livré
> chunk 5.

**Tâches Go** :
- [ ] `service.buildImpactRolesMatrix(events, teamOutcomes, squad) []ImpactRolesMatchRow` :
  groupé par match × xuid, chaque cellule = liste de `RoleAssignment`.
- [ ] `service.buildImpactRanking(matrix, period) []ImpactRanking` : 8 colonnes
  (1 par rôle), tri par count desc, gradient couleur via narrative.

**Tâches frontend** :
- [ ] `<ImpactRolesMatrix>` : grille HTML (cellules avec emojis ⚡🎯💀🐌🪦🛡️🗡️💥), fond outcome (vert win, rouge loss, gris tie).
- [ ] `<ImpactRankingTable>` : tableau 8 colonnes avec gradient Okabe-Ito (inversion pour rôles "négatifs").
- [ ] Toggle viz heatmap / scatter (segmented control).
- [ ] Popover légende (déjà localisée via manifest).

**Tests** :
- [ ] Service test : 5 scénarios canoniques de match (Domination, Humiliation, Comeback, etc.).
- [ ] Vitest `<ImpactRolesMatrix>` : 8 rôles isolés + combinaisons.

### Phase P6 — Cadence + Intensité (~1.5j)

> Utilise `LoadHighlightEvents` ✓ livré chunk 7.

**Pré-requis algorithmique** : 2 nouveaux helpers à ajouter dans
`analysis/narrative` (ou nouveau sous-package `analysis/cadence`) :
- [ ] `ComputeCadenceProfiles(events, squad, phaseSeconds=60) []CadenceProfile`
- [ ] `ComputeMatchIntensityProfiles(events, nBuckets=10) []IntensityProfile`

**Tâches Go** :
- [ ] Ajouter les 2 helpers ci-dessus avec tests purs (5+ scénarios chacun).
- [ ] `service.buildCadenceChart(events, squad) ChartSeries[ChartPointStacked]`.
- [ ] `service.buildIntensityHeatmap(events, matchIDs) ChartSeries[ChartPointHeatmap]`.

**Tâches frontend** :
- [ ] `<Cadence>` ECharts (kills/phase 60s, barres empilées par joueur).
- [ ] `<Heatmap2D>` réutilisé pour intensité (match × 10 buckets).
- [ ] Segmented control "Tous / joueur1 / joueur2 / joueur3" pour intensité.

### Phase P7 — Contributions (per-min + 6 charts trio + spree + HS/PK) (~2j)

**Tâches Go** :
- [ ] `service.buildPerMinuteStats(rowsPerPlayer) ChartSeries[ChartPointStacked]` (3 barres groupées).
- [ ] `service.buildFragsDeathsCombined(rowsPerPlayer) ChartSeries[ChartPointStacked]`.
- [ ] `service.buildAssistsChart`, `KDAChart`, `AccuracyChart`, `AvgLifeChart`, `PerformanceChart` (5 charts).
- [ ] `service.buildKillingSpreeMax(rowsPerPlayer) ChartSeries[ChartPoint2D]` avec `temporal.RollingMeanAdaptive`.
- [ ] `service.buildHsPkStacked(rowsPerPlayer) ChartSeries[ChartPointStacked]` avec records overlay.

**Tâches frontend** :
- [ ] `<BarGrouped>` (per-min, 3 barres × N joueurs).
- [ ] `<TimeseriesLine>` × 5 charts trio.
- [ ] Records overlay : nouvelle prop `recordsOverlay` sur `<BarStacked>` (motif hachuré ECharts).

### Phase P8 — Radar Participation 6 axes (~1j)

> Utilise `narrative.ComputeParticipationProfile` ✓ livré chunk 5.

**Tâches Go** :
- [ ] `service.buildRadar(awardsPerPlayer, modeFamily) ChartSeries[ParticipationScore]`.
- [ ] Endpoint dédié `GET /api/v1/players/{slug}/pages/squad/radar?teammates=...&period=...`.

**Tâches frontend** :
- [ ] `<Radar>` ECharts (6 axes, traces multi-joueurs colorées).
- [ ] Note post-graphe `tm_note_radar` localisée.

**Tests** :
- [ ] Service test : 3 familles de mode (slayer/ctf/strongholds), normalisation correcte.

### Phase P9 — Tableau historique + armes + médailles + légende (~2j)

**Tâches Go** :
- [ ] Nouveau repo `LoadWeaponKillsAggregated(ctx, slug, gt, matchIDs, opts) []WeaponRow`.
- [ ] Nouveau repo `LoadGrenadeMeleeKills(ctx, slug, gt, matchIDs)` (réinjection des kills filmés exclus).
- [ ] Nouveau repo `LoadMedalsForMatchesByXUID(ctx, slug, xuids, matchIDs) []MedalRow`.
- [ ] `service.buildHistoryTable(rowsIntersect) []HistoryRow`.
- [ ] `service.buildWeaponsTable(weaponRows, grenadeRows, params) []WeaponsTableRow`.

**Tâches frontend** :
- [ ] `<WeaponsTable>` (tri, slider min kills, colonnes dynamiques par joueur).
- [ ] `<MedalsGallery>` (grille de cartes match avec icônes).
- [ ] `<FloatingLegend>` joueurs avec `IntersectionObserver` (sentinelles `#llp-squad-start` / `#llp-medals-start`).
- [ ] Tableau historique avec date relative, mode normalisé, lien Waypoint.

---

## 5. Done definition globale Phase 1 Squad

- [ ] 22 sections de l'audit présentes (livrées ou skippées via `<CapabilityGap>`).
- [ ] Tous les wrappers ECharts spécialisés livrés (`<Lollipop>`, `<Bullet>`, `<BarStacked>`, `<BarGrouped>`, `<Heatmap2D>`, `<Radar>`, `<Cadence>`, `<TimeseriesLine>`, `<Donut>`).
- [ ] Composants spécifiques livrés (`<KPIStrip>`, `<PlayerScoreCard>`, `<FloatingLegend>`, `<ImpactRolesMatrix>`, `<ImpactRankingTable>`, `<MedalsGallery>`, `<WeaponsTable>`).
- [ ] Manifest `squad.toml` complet (~80 clés FR + EN, vérifié par tests cross-référence).
- [ ] ESLint `@levelup/no-hardcoded-strings` actif en `error` localement sur `apps/web/src/features/squad/`.
- [ ] Couverture Go ≥ 80 % `service/squad_service.go`.
- [ ] Couverture frontend ≥ 80 % `features/squad/`.
- [ ] Tests E2E Playwright sur 3 navigateurs.
- [ ] Test golden parity `cmd/foundations_golden_parity --page squad` passe.
- [ ] `dominance_flag` peuplé via `RunBackfillComebackBadges` sur DB de test (vérifié dans setup E2E).
- [ ] Capability gating : test sur titre synthétique sans `match.history` → page partielle, pas 5xx.
- [ ] Aucun import `plotly.js` ni `recharts` dans `features/squad/`.
- [ ] Entrée `thought_log.md` `[YYYY-MM-DD] Phase 1 Squad — pilote livré` complétée.

---

## 6. Risques et dépendances

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| LOWESS Go non-trivial à porter (Form Score P4) | Moyenne | Moyen | Implémentation locale ~50L (moyenne mobile pondérée gaussienne) au lieu de gonum. |
| Records overlays ECharts (motif hachuré) complexe | Faible | Moyen | Utiliser `itemStyle.decal` (pattern_shape équivalent ECharts). |
| `IntersectionObserver` + sentinelles légende fixe → friction mobile | Moyenne | Faible | Hors scope mobile (méta-plan a explicitement écarté). |
| Volume de wrappers ECharts à livrer (9) → Phase 1 dépasse 13j | Haute | Haut | Prioriser : `<Heatmap2D>`, `<Radar>`, `<Cadence>`, `<BarGrouped>` en P1. Les autres au fil des phases. |
| Sections excédentaires Go actuelles (heatmap 1D etc.) → confusion lors du refacto | Faible | Faible | Suppression nette des fichiers `charts/heatmapChart.ts` etc. en début de phase. |
| Squad sessions multi-select pas dans Phase 0 fondations | Moyenne | Moyen | Intégré P1 directement (cf. PLAN_SQUAD_STATS_SESSIONS_OVERHAUL). |
| Manque de tests fixtures `dominance_flag` côté integration | Moyenne | Haut | Setup E2E appelle `RunBackfillComebackBadges` une fois sur DB seed. |

### Dépendances externes

- ✅ Phase 0 méta-plan complète (livrée 2026-04-27).
- ✅ `dominance_flag` peuplable via `RunBackfillComebackBadges` (chunk 8).
- ✅ `LoadPlayerMatches` + `LoadHighlightEvents` opérationnels (chunks 6+7).
- ✅ `narrative.IdentifyImpactRoles` (8 rôles) + `ComputeParticipationProfile` (6 axes) livrés (chunk 5).
- ⏳ `analysis/temporal.LowessSmooth` à ajouter en P4 (~50L, ~30 min).
- ⏳ `analysis/narrative.ComputeCadenceProfiles` + `ComputeMatchIntensityProfiles` à ajouter en P6 (~120L total, ~1h).
- ⏳ Repos `LoadWeaponKillsAggregated`, `LoadGrenadeMeleeKills`, `LoadMedalsForMatchesByXUID` à ajouter en P9 (~200L total + tests, ~2h).

---

## 7. Annexes

### 7.1 Fichiers nouveaux ou réécrits

```
apps/go-api/internal/service/squad_service.go         (~400L réécrit)
apps/go-api/internal/service/squad_service_test.go     (~400L)
apps/go-api/internal/api/handlers/squad.go             (~80L)
apps/go-api/internal/api/handlers/squad_test.go        (~150L)
apps/go-api/internal/platform/duckdb/weapons_repo.go   (~150L) [P9]
apps/go-api/internal/platform/duckdb/medals_repo.go    (~100L) [P9]
apps/go-api/internal/analysis/temporal/lowess.go       (~50L) [P4]
apps/go-api/internal/analysis/narrative/cadence.go     (~80L) [P6]
apps/go-api/internal/analysis/narrative/intensity.go   (~80L) [P6]

apps/web/src/features/squad/                           (réécriture totale)
  SquadHeader.tsx
  SquadSynergiesPage.tsx                               (refonte 227L → ~250L)
  SquadContributionsPage.tsx                           (refonte 143L → ~180L)
  components/PlayerScoreCard.tsx
  components/ImpactRolesMatrix.tsx
  components/ImpactRankingTable.tsx
  components/WeaponsTable.tsx
  components/MedalsGallery.tsx
  components/FloatingLegend.tsx

apps/web/src/components/charts/                        (extension chunk 11)
  Lollipop.tsx
  Bullet.tsx
  BarStacked.tsx
  BarGrouped.tsx
  Heatmap2D.tsx
  Radar.tsx
  Cadence.tsx
  TimeseriesLine.tsx
  Donut.tsx
  __tests__/*.test.tsx

apps/web/src/components/layout/KPIStrip.tsx            (transverse)

apps/web/src/lib/i18n/manifests/squad.toml             (~80 clés)
apps/web/src/lib/i18n/manifests/narrative.toml         (~30 clés badges + roles)
apps/web/e2e/squad.spec.ts
```

### 7.2 Liste des 22 sections (récap)

```
Header :
  1. Bandeau KPI personnels (8 cartes + tendance)
  2. Score d'équipe + grade lettre + bonus
  3. Cartes scores individuels avec badge ▲▼
  4. Sentinelle légende #llp-squad-start

Synergies :
  5. Lollipop W/L par carte
  6. Bullet winrate session vs historique
  7. Perf vs historique par carte
  8. Heatmap escouade joueur × carte
  9. Timeline performance multi-joueurs + outcome marker
  10. Form Score lissé (LOWESS)
  11. Impact 8 rôles (heatmap + ranking MVP/Boulet)
  12. Tableau historique escouade
  13. Cadence trio (kills/phase 60s)

Contributions :
  14. Stats par minute (frags/morts/assists par min)
  15. Radar Participation 6 axes
  16. Frags ↑ / Morts ↓ combinés
  17-21. Assists / KDA / Précision / Vie moyenne / Performance (5 charts)
  18. Killing spree (max + smoothing)
  19. HS+PK stacked + records hachurés
  20. Heatmap intensité match × 10 phases
  21. Tableau armes + slider min kills + grenade/mêlée
  22. Galerie médailles (top 20 matchs partagés)
```

### 7.3 Ordre de chunkification recommandé

Pour livrer en **chunks committables de 1-3h** comme la Phase 0 :

1. **Chunk S1** : Service Squad refactor + handler + tests (P1) — ~3h
2. **Chunk S2** : `<KPIStrip>` + `<PlayerScoreCard>` + Header DTO (P2) — ~2h
3. **Chunk S3** : Wrappers ECharts `<Lollipop>` + `<Bullet>` + `<Heatmap2D>` (P3) — ~3h
4. **Chunk S4** : Synergies SquadSynergiesPage refactor + manifest squad.toml header/synergies (P3) — ~2h
5. **Chunk S5** : LOWESS + Form Score + `<TimeseriesLine>` (P4) — ~2h
6. **Chunk S6** : `<ImpactRolesMatrix>` + ranking + algos cadence + intensity (P5+P6 algos) — ~3h
7. **Chunk S7** : Heatmap intensité + Cadence wrapper (P6 frontend) — ~2h
8. **Chunk S8** : Contributions per-min + 6 charts trio (P7) — ~3h
9. **Chunk S9** : Radar 6 axes + manifest contributions (P8) — ~2h
10. **Chunk S10** : Tableau armes + médailles + repo (P9) — ~3h
11. **Chunk S11** : Légende flottante + cleanup + E2E + golden parity (P9 + clôture) — ~2h

**Total chunks Squad : 11, ~27h** (estimation Phase 1 méta-plan : 12-13j théorique = ~50-60h pour Squad+MatchView en parallèle, donc Squad seul ~25-30h cohérent).

### 7.4 Plans complémentaires à intégrer

- `PLAN_SQUAD_STATS_SESSIONS_OVERHAUL.md` : amélioration UX sessions multi-sélect + filtre coéquipiers amis. **À intégrer P1**.
- Le plan `PLAN_MATCH_VIEW_GO_PORTAGE.md` (jumeau) suivra le même pattern et partagera plusieurs wrappers ECharts (`<Heatmap2D>`, `<Radar>`, `<Cadence>`, `<BarStacked>`, `<TimeseriesLine>`).
