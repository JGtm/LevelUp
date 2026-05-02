# Spec ECharts — Page Timeseries

> Blueprint de référence pour l'implémentation des 12 graphes complexes.
> À lire avant d'implémenter `apps/web/src/components/charts/`.
> Tous les wrappers héritent de `<ChartCard buildOption={...}>` (méta-plan §3.2.2).
> Date : 2026-04-27.

---

## 0. Conventions globales

### 0.1 Couleurs — jamais de hex direct

```ts
import { resolveToken } from '@/lib/accessibility'
import { tokenCssVar }  from '@/lib/accessibility/semantic-tokens'
import { perfScale, outcomeScale, kdScale } from '@/lib/accessibility/scales'

// Dans buildOption (appelé à chaque re-render, tokens résolus depuis CSS vars) :
const WIN   = resolveToken('outcome-win')   // → hex résolu dynamiquement
const LOSS  = resolveToken('outcome-loss')
const TIER1 = resolveToken('perf-tier-1')
const DIVPOS= resolveToken('divergent-pos')
const DIVNEG= resolveToken('divergent-neg')

// tokenCssVar() = 'var(--ac-outcome-win)' — NE PAS utiliser dans ECharts car
// ECharts ne parse pas les CSS vars dans les champs color. Utiliser resolveToken().
```

### 0.2 Seuils de performance (perfScale)

```ts
// perfScale(value: number) → SemanticToken
// Seuils : 80 / 65 / 50 / 35 → tiers 1..5
// Utilisation dans buildOption : resolveToken(perfScale(score))
```

### 0.3 Couleur outcome

```ts
// outcomeScale(key) avec key in {'win','loss','draw','dnf'}
// outcome numérique Halo : 2=win, 1=draw, 3=loss, 4=dnf
const OUTCOME_MAP: Record<number, string> = { 2: 'win', 1: 'draw', 3: 'loss', 4: 'dnf' }
const outcomeColor = (o: number | null): string =>
  resolveToken(outcomeScale(OUTCOME_MAP[o ?? -1] ?? 'draw') as any)
```

### 0.4 Axe X commun (timeline par match)

```ts
// Étiquettes : "#N MapName\nDD/MM" — step adaptatif
const matchLabels = (rows: { index: number; playlist_name: string; start_time: string }[]) =>
  rows.map((r, i) => {
    const d = new Date(r.start_time)
    const dd = d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' })
    return `#${i + 1} ${r.playlist_name}\n${dd}`
  })

const tickInterval = (n: number): number => Math.max(1, Math.floor(n / 10))
```

### 0.5 Tooltip formatter standard

```ts
// ECharts tooltip formatter — params peut être array (trigger: 'axis') ou object (trigger: 'item')
type EChartsParams = { name: string; seriesName: string; value: number | number[]; color: string }[]
```

### 0.6 Fond et grille (thème dark)

```ts
const CHART_BG   = 'transparent'  // fond géré par le <Card> parent
const GRID_COLOR = 'rgba(255,255,255,0.06)'
const TEXT_COLOR = 'rgba(255,255,255,0.45)'
const MARK_COLOR = 'rgba(255,255,255,0.15)'
// Ces constantes sont les seules couleurs structurelles autorisées (cf. CLAUDE.md exceptions).
```

---

## 1. Helpers partagés (`apps/web/src/components/charts/_utils.ts`)

```ts
export const CHART_BG   = 'transparent'
export const GRID_COLOR = 'rgba(255,255,255,0.06)'
export const TEXT_COLOR = 'rgba(255,255,255,0.45)'
export const ZERO_LINE  = 'rgba(255,255,255,0.15)'

export const axisBase = {
  axisLine:  { lineStyle: { color: GRID_COLOR } },
  axisTick:  { show: false },
  splitLine: { lineStyle: { color: GRID_COLOR } },
  axisLabel: { color: TEXT_COLOR, fontSize: 10 },
}

export const tooltipBase = {
  backgroundColor: 'rgba(20,24,30,0.92)',
  borderColor:     GRID_COLOR,
  textStyle: { color: 'rgba(255,255,255,0.85)', fontSize: 11 },
  extraCssText: 'border-radius:6px;box-shadow:0 4px 12px rgba(0,0,0,0.4)',
}

export const legendBase = {
  bottom: 0,
  textStyle: { color: TEXT_COLOR, fontSize: 10 },
  itemWidth: 12, itemHeight: 8,
}

// Marqueurs extrêmes (max/min) sur une série
export const extremeMarkPoints = () => ({
  markPoint: {
    data: [
      { type: 'max', symbol: 'triangle', symbolSize: 8, label: { formatter: '{c}', fontSize: 9 } },
      { type: 'min', symbol: 'triangle', symbolSize: 8, symbolRotate: 180, label: { formatter: '{c}', fontSize: 9 } },
    ],
  },
})

// Ligne de référence horizontale
export const refLine = (y: number, label: string, color: string) => ({
  markLine: {
    silent: true,
    symbol: 'none',
    data: [{ yAxis: y, label: { formatter: label, position: 'insideEndTop', fontSize: 9 }, lineStyle: { color, type: 'dashed', width: 1 } }],
  },
})
```

---

## 2. KDA Dual-Axis — barres K/D + ligne ratio

**Wrapper** : `<KdaBarsChart rows={TimeseriesMatchRow[]} />`
**Remplace** : `timeseries-kda-bars.tsx` (Plotly).
**Données** : `match_rows` (déjà dans `TimeseriesPageResponse`).

```ts
// buildOption(rows: TimeseriesMatchRow[]): EChartsCoreOption
export function buildKdaBarsOption(rows: TimeseriesMatchRow[]): EChartsCoreOption {
  const WIN = resolveToken('outcome-win')
  const LOSS = resolveToken('outcome-loss')
  const KD_COLOR = resolveToken('perf-tier-2')
  const xs = matchLabels(rows)
  const n = rows.length

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 48, right: 52 },
    tooltip: {
      ...tooltipBase,
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: EChartsParams[]) => {
        const r = rows[params[0].dataIndex]
        const kd = r.deaths > 0 ? (r.kills / r.deaths).toFixed(2) : r.kills.toFixed(2)
        return `<b>#${r.index + 1}</b> ${new Date(r.start_time).toLocaleDateString('fr-FR')}<br/>
          Kills: <b>${r.kills}</b> · Morts: <b>${r.deaths}</b> · Assists: <b>${r.assists}</b><br/>
          K/D: <b>${kd}</b>${r.accuracy != null ? ` · Précision: ${(r.accuracy * 100).toFixed(1)}%` : ''}`
      },
    },
    legend: { ...legendBase, data: ['Kills', 'Morts', 'K/D'] },
    xAxis: {
      ...axisBase,
      type: 'category',
      data: xs,
      axisLabel: { ...axisBase.axisLabel, interval: tickInterval(n) - 1, rotate: n > 60 ? 30 : 0 },
    },
    yAxis: [
      { ...axisBase, type: 'value', name: 'K / Morts', nameTextStyle: { color: TEXT_COLOR, fontSize: 9 } },
      { ...axisBase, type: 'value', name: 'K/D', position: 'right', splitLine: { show: false },
        nameTextStyle: { color: KD_COLOR, fontSize: 9 }, axisLabel: { ...axisBase.axisLabel, color: KD_COLOR } },
    ],
    series: [
      {
        name: 'Kills',
        type: 'bar',
        barMaxWidth: 12,
        data: rows.map((r) => ({
          value: r.kills,
          itemStyle: { color: `${outcomeColor(r.outcome)}cc`, borderRadius: [2, 2, 0, 0] },
        })),
      },
      {
        name: 'Morts',
        type: 'bar',
        barMaxWidth: 12,
        data: rows.map((r) => ({
          value: -r.deaths, // négatif = symétrique
          itemStyle: { color: `${LOSS}44`, borderRadius: [0, 0, 2, 2] },
        })),
        tooltip: { valueFormatter: (v: number) => String(Math.abs(v)) },
      },
      {
        name: 'K/D',
        type: 'line',
        yAxisIndex: 1,
        data: rows.map((r) => (r.deaths > 0 ? r.kills / r.deaths : r.kills)),
        lineStyle: { color: KD_COLOR, width: 1.5 },
        itemStyle: { color: KD_COLOR },
        symbol: 'circle', symbolSize: 4,
        ...refLine(1, 'K/D = 1', KD_COLOR),
        ...extremeMarkPoints(),
      },
    ],
  }
}
```

**Décisions vs Python** :
- Deaths rendus en valeur négative sur Y1 (même axe que Kills, effet miroir).
- Kills colorés par outcome (avec transparence `cc`), pas une couleur unique.
- Annotations extrêmes via `markPoint.type: 'max'/'min'` (pas d'annotation Python inline).

---

## 3. Performance Colored Bars

**Wrapper** : `<PerformanceBarsChart rows={TimeseriesMatchRow[]} />`
**Données** : `match_rows.perf_score` (à exposer en Phase 3).

```ts
export function buildPerformanceBarsOption(rows: TimeseriesMatchRow[]): EChartsCoreOption {
  const xs = matchLabels(rows)
  const SMOOTH_COLOR = resolveToken('perf-tier-3')

  // Lissage rolling window=10
  const smooth = (arr: number[], w: number) =>
    arr.map((_, i) => {
      const slice = arr.slice(Math.max(0, i - w + 1), i + 1)
      return slice.reduce((a, b) => a + b, 0) / slice.length
    })

  const scores = rows.map((r) => r.perf_score ?? 0)
  const smoothed = smooth(scores, 10)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 48, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'axis' },
    legend: { ...legendBase, data: ['Performance', 'Tendance (×10)'] },
    xAxis: { ...axisBase, type: 'category', data: xs, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(rows.length) - 1 } },
    yAxis: { ...axisBase, type: 'value', min: 0, max: 100, name: 'Score' },
    // visualMap colore chaque barre selon son score (piecewise sur Y)
    visualMap: {
      show: false,
      type: 'piecewise',
      dimension: 1,
      seriesIndex: 0,
      pieces: [
        { min: 80,          color: resolveToken('perf-tier-1') },
        { min: 65, max: 80, color: resolveToken('perf-tier-2') },
        { min: 50, max: 65, color: resolveToken('perf-tier-3') },
        { min: 35, max: 50, color: resolveToken('perf-tier-4') },
        {          max: 35, color: resolveToken('perf-tier-5') },
      ],
    },
    series: [
      { name: 'Performance', type: 'bar', barMaxWidth: 10, data: scores,
        itemStyle: { borderRadius: [2, 2, 0, 0] } },
      { name: 'Tendance (×10)', type: 'line', data: smoothed, smooth: true,
        lineStyle: { color: SMOOTH_COLOR, width: 1.5, type: 'solid' },
        symbol: 'none', itemStyle: { color: SMOOTH_COLOR } },
    ],
  }
}
```

**Décisions** : `visualMap.type: 'piecewise'` sur `seriesIndex: 0` colore chaque barre par sa propre valeur Y. La série de lissage (index 1) n'est pas couverte par le visualMap → garde sa couleur `SMOOTH_COLOR`.

---

## 4. Per-Min Symmetric Bars (KPM / DPM / APM)

**Wrapper** : `<PerMinSymmetricBarsChart rows={TimeseriesMatchRow[]} />`
**Données** : `match_rows` (kills, deaths, assists, time_played_seconds).

```ts
export function buildPerMinBarsOption(rows: TimeseriesMatchRow[]): EChartsCoreOption {
  const KPM_C = resolveToken('perf-tier-1')
  const DPM_C = resolveToken('outcome-loss')
  const APM_C = resolveToken('perf-tier-3')
  const xs = matchLabels(rows)

  const tpm = (r: TimeseriesMatchRow) => (r.time_played_seconds ?? 0) / 60 || 1
  const kpm  = rows.map((r) => r.kills / tpm(r))
  const dpm  = rows.map((r) => -(r.deaths / tpm(r)))  // négatif pour axe symétrique
  const apm  = rows.map((r) => r.assists / tpm(r))

  const smooth = (arr: number[], w = 10) =>
    arr.map((_, i) => { const s = arr.slice(Math.max(0, i-w+1), i+1); return s.reduce((a,b) => a+b, 0) / s.length })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 52, right: 16 },
    tooltip: {
      ...tooltipBase, trigger: 'axis',
      formatter: (params: EChartsParams[]) => {
        const i = params[0].dataIndex; const r = rows[i]
        const t = tpm(r)
        return `<b>#${r.index+1}</b><br/>KPM: ${(r.kills/t).toFixed(2)} · DPM: ${(r.deaths/t).toFixed(2)} · APM: ${(r.assists/t).toFixed(2)}`
      },
    },
    legend: { ...legendBase, data: ['KPM', 'DPM', 'APM'] },
    xAxis: { ...axisBase, type: 'category', data: xs, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(rows.length) - 1 } },
    yAxis: {
      ...axisBase, type: 'value',
      // Axes symétriques autour de 0
      min: (v: { min: number }) => -Math.ceil(Math.abs(v.min) * 1.15 * 10) / 10,
      max: (v: { max: number }) =>  Math.ceil(Math.abs(v.max) * 1.15 * 10) / 10,
    },
    series: [
      { name: 'KPM', type: 'bar', barMaxWidth: 8, stack: 'pos', data: kpm, itemStyle: { color: KPM_C } },
      { name: 'APM', type: 'bar', barMaxWidth: 8, stack: 'pos', data: apm, itemStyle: { color: APM_C } },
      { name: 'DPM', type: 'bar', barMaxWidth: 8, data: dpm, itemStyle: { color: DPM_C },
        tooltip: { valueFormatter: (v: number) => String(Math.abs(v).toFixed(2)) } },
      { name: 'KPM', type: 'line', data: smooth(kpm), lineStyle: { color: KPM_C, type: 'dashed', width: 1 }, symbol: 'none' },
      { name: 'DPM', type: 'line', data: smooth(dpm), lineStyle: { color: DPM_C, type: 'dashed', width: 1 }, symbol: 'none' },
      { name: 'APM', type: 'line', data: smooth(apm), lineStyle: { color: APM_C, type: 'dashed', width: 1 }, symbol: 'none' },
      // Ligne zéro
      { type: 'line', data: rows.map(() => 0), lineStyle: { color: ZERO_LINE, width: 1 }, symbol: 'none', silent: true },
    ],
  }
}
```

**Décisions** :
- KPM et APM `stack: 'pos'` → empilés vers le haut. DPM non-stacké (valeur déjà négative).
- `min`/`max` calculés dynamiquement pour garantir la symétrie autour de 0.
- DPM tooltip `valueFormatter` retire le signe moins pour l'affichage.

---

## 5. EWMA K/D + Ligne de Régression

**Wrapper** : `<EwmaKdChart points={CumulativePoint[]} regression={RegressionLine} trend={string} />`
**Données** : `form_tab.ewma_kd_points` + `form_tab.regression_line_points` (à ajouter Phase 5) + `form_tab.regression_stats.trend`.

```ts
// Type ajouté Go-side : RegressionLine = []CumulativePoint (slope*x + intercept pour chaque index)
export function buildEwmaKdOption(
  points: { index: number; value: number; outcome?: number }[],
  regLine: { index: number; value: number }[],
  trend: 'improving' | 'declining' | 'stable' | null,
): EChartsCoreOption {
  const EWMA_C = resolveToken('perf-tier-2')
  const REG_C  = trend === 'improving' ? resolveToken('divergent-pos') : trend === 'declining' ? resolveToken('divergent-neg') : resolveToken('divergent-neutral')
  const xs = points.map((p) => `#${p.index + 1}`)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 48, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'axis' },
    legend: { ...legendBase, data: ['EWMA K/D', 'Régression'] },
    xAxis: { ...axisBase, type: 'category', data: xs, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(points.length) - 1 } },
    yAxis: { ...axisBase, type: 'value', name: 'K/D lissé' },
    series: [
      {
        name: 'EWMA K/D',
        type: 'line',
        data: points.map((p) => ({
          value: p.value,
          // Marqueur coloré par outcome pour les points (cercle)
          itemStyle: { color: p.outcome != null ? outcomeColor(p.outcome) : EWMA_C },
        })),
        lineStyle: { color: EWMA_C, width: 2 },
        symbol: 'circle', symbolSize: 5,
        markLine: { silent: true, symbol: 'none', data: [{ yAxis: 1, lineStyle: { color: MARK_COLOR, type: 'dashed' }, label: { formatter: 'K/D = 1', fontSize: 9 } }] },
      },
      {
        name: 'Régression',
        type: 'line',
        data: regLine.map((p) => p.value),
        lineStyle: { color: REG_C, width: 1.5, type: 'dashed' },
        symbol: 'none',
        silent: true,
      },
    ],
  }
}
```

**Décisions** :
- Chaque point EWMA est coloré par l'outcome du match correspondant (sur le cercle marker).
- La ligne de régression est calculée **côté Go** (`slope * x + intercept` pour chaque index) → le frontend reçoit un tableau de points prêts, pas slope/intercept bruts.
- Couleur régression : `divergent-pos` si improving, `divergent-neg` si declining, `divergent-neutral` si stable.

---

## 6. Cumul K/D + Bande IC 90 %

**Wrapper** : `<CumulKdWithCIChart points={CumulativePointWithCI[]} />`
**Données** : `cumul_tab.cumulative_kd_ci` (à ajouter Phase 5) — `[]{ index, value, ci_lower, ci_upper }`.

```ts
export function buildCumulKdCIOption(
  points: { index: number; value: number; ci_lower: number; ci_upper: number }[],
): EChartsCoreOption {
  const LINE_C = resolveToken('perf-tier-1')
  const xs = points.map((p) => `#${p.index + 1}`)

  // ECharts IC band : 3 séries stackées
  // Serie 0 : ci_lower (invisible, base du stack)
  // Serie 1 : ci_upper - ci_lower (aire visible)
  // Serie 2 : la ligne K/D cumulée (par-dessus, non stackée)
  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 40, left: 48, right: 16 },
    tooltip: {
      ...tooltipBase, trigger: 'axis',
      formatter: (params: EChartsParams[]) => {
        const i = params[0].dataIndex; const p = points[i]
        return `#${p.index + 1}<br/>K/D cumulé: <b>${p.value.toFixed(3)}</b><br/>IC 90%: [${p.ci_lower.toFixed(3)}, ${p.ci_upper.toFixed(3)}]`
      },
    },
    xAxis: { ...axisBase, type: 'category', data: xs, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(points.length) - 1 } },
    yAxis: { ...axisBase, type: 'value', name: 'K/D' },
    series: [
      // Base invisible pour le stack IC
      { name: '_ci_base', type: 'line', data: points.map((p) => p.ci_lower),
        lineStyle: { opacity: 0 }, symbol: 'none', stack: 'ci',
        areaStyle: { color: 'transparent', opacity: 0 }, silent: true, legendHoverLink: false },
      // Largeur de la bande (ci_upper - ci_lower)
      { name: 'IC 90%', type: 'line', data: points.map((p) => p.ci_upper - p.ci_lower),
        lineStyle: { opacity: 0 }, symbol: 'none', stack: 'ci',
        areaStyle: { color: LINE_C, opacity: 0.15 } },
      // Ligne principale K/D cumulé
      { name: 'K/D cumulé', type: 'line', data: points.map((p) => p.value),
        lineStyle: { color: LINE_C, width: 2 }, itemStyle: { color: LINE_C }, symbol: 'none',
        markLine: { silent: true, symbol: 'none', data: [{ yAxis: 1, lineStyle: { color: MARK_COLOR, type: 'dashed' }, label: { formatter: 'K/D = 1', fontSize: 9 } }] } },
    ],
  }
}
```

**Décisions** :
- La bande IC est rendue via 2 séries stackées (pattern standard ECharts pour les confidence intervals).
- La série base `_ci_base` est masquée de la légende (`legendHoverLink: false`, pas de nom affiché) — la nommer avec `_` est une convention interne.
- Le tooltip affiche les 3 valeurs explicitement.

---

## 7. Intensity Heatmap Match × 10 Phases

**Wrapper** : `<MatchIntensityHeatmap data={MatchPhasesRow[]} outcomeFilter={'all'|'win'|'loss'} />`
**Données** : `intensity_match_phases` (nouveau champ Phase 5) — `[]{ match_id, start_time, outcome, buckets: number[10] }`.

```ts
export function buildMatchIntensityOption(
  rows: { match_id: string; index: number; outcome: number | null; buckets: number[] }[],
): EChartsCoreOption {
  const HOT  = resolveToken('heatmap-hot')
  const COLD = resolveToken('heatmap-cold')

  // ECharts heatmap : [phaseIndex, matchIndex, killCount]
  const data: [number, number, number][] = []
  rows.forEach((r, matchIdx) => {
    r.buckets.forEach((count, phaseIdx) => {
      data.push([phaseIdx, matchIdx, count])
    })
  })

  const maxKills = Math.max(...data.map((d) => d[2]), 1)

  const phaseLabels = ['0-10%', '10-20%', '20-30%', '30-40%', '40-50%', '50-60%', '60-70%', '70-80%', '80-90%', '90-100%']
  const matchLabelsY = rows.map((r, i) => `#${i + 1}`)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 52, right: 80 },
    tooltip: {
      ...tooltipBase, trigger: 'item',
      formatter: ({ data: d }: { data: [number, number, number] }) =>
        `Match #${d[1] + 1} · Phase ${phaseLabels[d[0]]}<br/>Kills: <b>${d[2]}</b>`,
    },
    xAxis: { ...axisBase, type: 'category', data: phaseLabels, splitArea: { show: true, areaStyle: { color: ['transparent', 'rgba(255,255,255,0.02)'] } } },
    yAxis: { ...axisBase, type: 'category', data: matchLabelsY, splitArea: { show: false }, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(rows.length) - 1 } },
    visualMap: {
      min: 0, max: maxKills,
      calculable: true,
      orient: 'vertical',
      right: 8, top: 'center',
      textStyle: { color: TEXT_COLOR, fontSize: 9 },
      inRange: { color: [COLD, HOT] }, // froid (0 kills) → chaud (max kills)
    },
    series: [{
      type: 'heatmap',
      data,
      emphasis: { itemStyle: { shadowBlur: 8, shadowColor: 'rgba(0,0,0,0.5)' } },
      progressive: 200,
    }],
  }
}
```

**Décisions** :
- Les matchs sont en Y (dimension longue, potentiellement > 200), les 10 phases en X.
- `progressive: 200` → ECharts rend la heatmap par blocs pour ne pas bloquer le main thread.
- Le filtre `outcomeFilter` est appliqué **avant** d'appeler `buildOption` (le composant filtre `rows` côté React, puis passe le sous-ensemble à la fonction). Pas de logique de filtre dans `buildOption`.
- `heatmap-cold` / `heatmap-hot` sont des tokens déjà déclarés dans `semantic-tokens.ts`.

---

## 8. Skill Rank LUSR/CSR + Zones Tier + IC

**Wrapper** : `<SkillRankChart points={SkillRankPoint[]} selectedType={'lusr'|'csr'} />`
**Données** : nouveau champ top-level `skill_rank_history` (Phase 4).

```ts
// Bounds tier Halo Infinite (valeurs indicatives, à confirmer via career_ranks DB)
const TIER_BOUNDS = [
  { label: 'Bronze',    min: 0,    max: 999,  token: 'chart-series-6' as const },
  { label: 'Silver',    min: 1000, max: 1999, token: 'chart-series-5' as const },
  { label: 'Gold',      min: 2000, max: 2999, token: 'chart-series-4' as const },
  { label: 'Platinum',  min: 3000, max: 3999, token: 'chart-series-3' as const },
  { label: 'Diamond',   min: 4000, max: 4999, token: 'chart-series-2' as const },
  { label: 'Onyx',      min: 5000, max: 99999, token: 'chart-series-1' as const },
]

export function buildSkillRankOption(
  points: { index: number; rating_value: number; rating_deviation: number; tier_label: string }[],
  showSmooth = false,
): EChartsCoreOption {
  const LINE_C = resolveToken('perf-tier-2')
  const IC_C   = resolveToken('perf-tier-2')
  const xs = points.map((p) => `#${p.index + 1}`)

  // Lissage rolling 20
  const smooth20 = (arr: number[]) =>
    arr.map((_, i) => { const s = arr.slice(Math.max(0, i-19), i+1); return s.reduce((a,b)=>a+b,0)/s.length })

  // Bande IC : ci_lower = value - deviation, ci_upper = value + deviation
  const lower = points.map((p) => p.rating_value - p.rating_deviation)
  const upper = points.map((p) => p.rating_value + p.rating_deviation)
  const band  = points.map((p) => p.rating_deviation * 2)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 40, left: 60, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'axis',
      formatter: (params: EChartsParams[]) => {
        const i = params[0].dataIndex; const p = points[i]
        return `#${p.index+1}<br/>Rating: <b>${p.rating_value.toFixed(0)}</b><br/>IC: ±${p.rating_deviation.toFixed(0)}<br/>Tier: ${p.tier_label}`
      }
    },
    xAxis: { ...axisBase, type: 'category', data: xs, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(points.length) - 1 } },
    yAxis: { ...axisBase, type: 'value', name: 'Rating' },
    series: [
      // Base IC invisible
      { name: '_ic_base', type: 'line', data: lower, lineStyle: { opacity: 0 }, symbol: 'none', stack: 'ic', areaStyle: { opacity: 0 }, silent: true, legendHoverLink: false },
      // Largeur bande IC
      { name: 'IC (±σ)', type: 'line', data: band, lineStyle: { opacity: 0 }, symbol: 'none', stack: 'ic', areaStyle: { color: IC_C, opacity: 0.12 } },
      // Ligne rating principale
      { name: 'Rating', type: 'line', data: points.map((p) => p.rating_value),
        lineStyle: { color: LINE_C, width: 2 }, itemStyle: { color: LINE_C }, symbol: 'circle', symbolSize: 4,
        // Zones tier en markArea sur cette série
        markArea: {
          silent: true,
          data: TIER_BOUNDS.map((t) => [
            { yAxis: t.min, itemStyle: { color: resolveToken(t.token), opacity: 0.08 }, label: { show: true, position: 'insideTopLeft', formatter: t.label, fontSize: 8, color: TEXT_COLOR } },
            { yAxis: t.max },
          ]),
        },
      },
      // Lissage optionnel
      ...(showSmooth ? [{
        name: 'Lissage ×20', type: 'line' as const, data: smooth20(points.map((p) => p.rating_value)),
        lineStyle: { color: resolveToken('perf-tier-4'), type: 'dotted' as const, width: 1.5 },
        symbol: 'none',
      }] : []),
    ],
  }
}
```

**Décisions** :
- `markArea` sur la série rating (index 2) couvre les plages de tiers. Si le rating dépasse la valeur max d'un tier, ECharts clip automatiquement (le markArea ne déborde pas hors du viewport).
- Les couleurs de tier utilisent `chart-series-1..6` (tokens existants, colorés selon palette active) — pas de hex bronze/argent/or en dur.
- `rating_deviation` = équivalent de `rating_deviation` Python (incertitude statistique).
- Le regroupement par semaine si > 50 points est appliqué **côté Go** (Go envoie des points agrégés) → le frontend reçoit toujours le même format.

---

## 9. Net Score par Heure — Zones Pos / Neg

**Wrapper** : `<NetScorePerHourChart points={NetScorePoint[]} />`
**Données** : nouveau champ `intensity_tab.net_score_per_hour` (Phase 5) — `[]{ index, value, outcome? }`.

```ts
export function buildNetScorePerHourOption(
  points: { index: number; value: number; outcome?: number }[],
): EChartsCoreOption {
  const POS = resolveToken('divergent-pos')
  const NEG = resolveToken('divergent-neg')
  const xs  = points.map((p) => `#${p.index + 1}`)
  const vals = points.map((p) => p.value)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 40, left: 52, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'axis',
      formatter: (params: EChartsParams[]) => {
        const i = params[0].dataIndex; const p = points[i]
        const sign = p.value >= 0 ? '+' : ''
        return `#${p.index+1} · Net/h: <b>${sign}${p.value.toFixed(1)}</b>`
      }
    },
    xAxis: { ...axisBase, type: 'category', data: xs, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(points.length) - 1 } },
    yAxis: { ...axisBase, type: 'value', name: 'Net frags/h' },
    // visualMap piecewise sur Y pour colorer pos/neg
    visualMap: {
      show: false,
      type: 'piecewise',
      dimension: 1,
      seriesIndex: 0,
      pieces: [
        { min: 0,   color: POS },
        { max: 0,   color: NEG },
      ],
    },
    series: [
      { name: 'Net/h', type: 'line', data: vals,
        areaStyle: { opacity: 0.5 }, // La couleur de l'aire est gérée par visualMap
        lineStyle: { width: 1.5 },
        symbol: 'none',
        markLine: { silent: true, symbol: 'none', data: [{ yAxis: 0, lineStyle: { color: ZERO_LINE, width: 1 } }] },
      },
    ],
  }
}
```

**Décisions** :
- `visualMap.type: 'piecewise'` sur la série 0 colore ligne ET aire au-dessus/dessous de 0 automatiquement.
- Pas de 2 séries séparées (positive clippée + négative clippée) — ECharts le gère via `visualMap` + `pieces`.

---

## 10. WL Heatmap 7 × 24 (win rate %)

**Wrapper** : `<WLHeatmapChart data={WLHeatmapPoint[]} />`
**Données** : `intensity_tab.heatmap_data` enrichi avec `win_rate` (à ajouter Phase 5) — `[]{ day_of_week, hour, count, avg_kd, win_rate }`.

```ts
const DAY_LABELS_FR = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']
const HOUR_LABELS = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}h`)

export function buildWLHeatmapOption(
  data: { day_of_week: number; hour: number; count: number; avg_kd: number; win_rate: number }[],
  colorBy: 'count' | 'win_rate' = 'win_rate',
): EChartsCoreOption {
  const HOT   = resolveToken('heatmap-hot')
  const COLD  = resolveToken('heatmap-cold')
  const DLOW  = resolveToken('heatmap-divergent-low')
  const DHIGH = resolveToken('heatmap-divergent-high')

  const vals = data.map((d) => colorBy === 'win_rate' ? d.win_rate * 100 : d.count)
  const maxVal = Math.max(...vals, 1)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 40, right: 80 },
    tooltip: {
      ...tooltipBase, trigger: 'item',
      formatter: ({ data: d }: { data: typeof data[0] }) =>
        `${DAY_LABELS_FR[d.day_of_week]} ${HOUR_LABELS[d.hour]}<br/>
        Win rate: <b>${(d.win_rate * 100).toFixed(0)}%</b><br/>
        Matchs: <b>${d.count}</b> · K/D moy.: <b>${d.avg_kd.toFixed(2)}</b>`,
    },
    xAxis: { ...axisBase, type: 'category', data: HOUR_LABELS, splitArea: { show: true, areaStyle: { color: ['transparent', 'rgba(255,255,255,0.02)'] } } },
    yAxis: { ...axisBase, type: 'category', data: DAY_LABELS_FR, splitArea: { show: true, areaStyle: { color: ['transparent', 'rgba(255,255,255,0.02)'] } } },
    visualMap: {
      min: 0, max: colorBy === 'win_rate' ? 100 : maxVal,
      calculable: true,
      orient: 'vertical',
      right: 8, top: 'center',
      textStyle: { color: TEXT_COLOR, fontSize: 9 },
      // win_rate : divergent (rouge <50% → blanc 50% → vert >50%)
      // count : séquentiel (froid → chaud)
      inRange: colorBy === 'win_rate'
        ? { color: [DLOW, 'rgba(255,255,255,0.1)', DHIGH] }
        : { color: [COLD, HOT] },
    },
    series: [{
      type: 'heatmap',
      data: data.map((d) => ({ ...d, value: [d.hour, d.day_of_week, colorBy === 'win_rate' ? d.win_rate * 100 : d.count] })),
      emphasis: { itemStyle: { shadowBlur: 8 } },
    }],
  }
}
```

**Décisions** :
- `colorBy` est un prop React (pas un paramètre du tooltip Go). La toggle est côté client uniquement.
- Win rate divergent : `heatmap-divergent-low` (rouge) → blanc (50%) → `heatmap-divergent-high` (vert). Correspondance directe avec la signification métier.
- `data` passé directement à ECharts avec le spread `{ ...d }` → ECharts ignore les champs extra dans le tooltip formatter si on les réfère via `data`.

---

## 11. Scatter Corrélations + Trendline

**Wrapper** : `<CorrelationScatterChart points={CorrelationDataPair[]} />`
**Données** : `distributions_tab.correlation_points` (déjà présent) + nouveau champ `trendline_points` par paire (Phase 7).
**Remplace** : `timeseries-scatter.tsx` (Plotly).

```ts
// Paires disponibles (label)
const PAIR_LABELS: Record<string, { x: string; y: string; title: string }> = {
  kills_vs_kd:         { x: 'Kills', y: 'K/D',           title: 'Kills → K/D' },
  lifespan_vs_kills:   { x: 'Durée de vie (s)', y: 'Kills', title: 'Durée de vie → Kills' },
  accuracy_vs_kda:     { x: 'Précision (%)', y: 'KDA',    title: 'Précision → KDA' },
  lifespan_vs_deaths:  { x: 'Durée de vie (s)', y: 'Morts', title: 'Durée de vie → Morts' },
  kills_vs_deaths:     { x: 'Kills', y: 'Morts',          title: 'Kills → Morts' },
  mmr_team_vs_enemy:   { x: 'MMR équipe', y: 'MMR adv.', title: 'MMR Équipe vs Adv.' },
}

export function buildCorrelationScatterOption(
  points: { label: string; x: number; y: number; outcome: number | null }[],
  pairLabel: string,
  trendline?: { x: number; y: number }[],
): EChartsCoreOption {
  const cfg = PAIR_LABELS[pairLabel] ?? { x: 'X', y: 'Y', title: pairLabel }
  return {
    backgroundColor: CHART_BG,
    grid: { top: 32, bottom: 40, left: 52, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'item',
      formatter: ({ data: d }: { data: typeof points[0] }) =>
        `${cfg.x}: <b>${d.x.toFixed(2)}</b><br/>${cfg.y}: <b>${d.y.toFixed(2)}</b>`,
    },
    xAxis: { ...axisBase, type: 'value', name: cfg.x, nameTextStyle: { color: TEXT_COLOR, fontSize: 9 } },
    yAxis: { ...axisBase, type: 'value', name: cfg.y, nameTextStyle: { color: TEXT_COLOR, fontSize: 9 } },
    series: [
      {
        name: cfg.title,
        type: 'scatter',
        symbolSize: 7,
        data: points.map((p) => ({
          value: [p.x, p.y],
          itemStyle: { color: outcomeColor(p.outcome) },
        })),
      },
      ...(trendline && trendline.length >= 2 ? [{
        name: 'Trendline',
        type: 'line' as const,
        data: trendline.map((p) => [p.x, p.y]),
        lineStyle: { color: resolveToken('divergent-neutral'), type: 'dashed' as const, width: 1, opacity: 0.6 },
        symbol: 'none', silent: true, legendHoverLink: false,
      }] : []),
    ],
  }
}
```

**Décisions** :
- Trendline calculée **côté Go** (régression linéaire sur la paire, 2 points suffisent) → le frontend reçoit `[{x: min, y: f(min)}, {x: max, y: f(max)}]`.
- Couleur des points par outcome (même mapping que les autres charts).
- Le sélecteur de paire est géré par l'état React du composant parent (pas dans `buildOption`).

---

## 12. Outcomes Over Time (barres empilées par bucket)

**Wrapper** : `<OutcomesOverTimeChart buckets={OutcomeBucket[]} />`
**Données** : nouveau champ `summary_tab.outcomes_over_time` (Phase 1) — `[]{ label, wins, losses, ties, dnf, total }`.

```ts
export function buildOutcomesOverTimeOption(
  buckets: { label: string; wins: number; losses: number; ties: number; dnf: number }[],
): EChartsCoreOption {
  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 48, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { ...legendBase, data: ['Victoires', 'Défaites', 'Égalités', 'Abandons'] },
    xAxis: { ...axisBase, type: 'category', data: buckets.map((b) => b.label) },
    yAxis: { ...axisBase, type: 'value', name: 'Matchs' },
    series: [
      { name: 'Victoires', type: 'bar', stack: 'outcomes', data: buckets.map((b) => b.wins),   itemStyle: { color: resolveToken('outcome-win') } },
      { name: 'Défaites',  type: 'bar', stack: 'outcomes', data: buckets.map((b) => b.losses), itemStyle: { color: resolveToken('outcome-loss') } },
      { name: 'Égalités',  type: 'bar', stack: 'outcomes', data: buckets.map((b) => b.ties),   itemStyle: { color: resolveToken('outcome-draw') } },
      { name: 'Abandons',  type: 'bar', stack: 'outcomes', data: buckets.map((b) => b.dnf),    itemStyle: { color: resolveToken('outcome-dnf') } },
    ],
  }
}
```

---

## 13. Streaks (barres signées V/D)

**Wrapper** : `<StreakBarsChart streaks={StreakPoint[]} />`
**Données** : nouveau champ `summary_tab.streak_series` (Phase 1) — `[]{ index, value, is_win_streak }` (valeur positive = série de victoires, négative = série de défaites).

```ts
export function buildStreakBarsOption(
  streaks: { index: number; value: number; is_win_streak: boolean }[],
): EChartsCoreOption {
  const WIN  = resolveToken('outcome-win')
  const LOSS = resolveToken('outcome-loss')
  const xs   = streaks.map((s) => `#${s.index + 1}`)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 40, left: 44, right: 16 },
    tooltip: { ...tooltipBase, trigger: 'axis',
      formatter: (params: EChartsParams[]) => {
        const s = streaks[params[0].dataIndex]
        return s.is_win_streak
          ? `Série de <b>${s.value}</b> victoires`
          : `Série de <b>${Math.abs(s.value)}</b> défaites`
      },
    },
    xAxis: { ...axisBase, type: 'category', data: xs, axisLabel: { ...axisBase.axisLabel, interval: tickInterval(streaks.length) - 1 } },
    yAxis: { ...axisBase, type: 'value', name: 'Streak' },
    series: [{
      name: 'Séries',
      type: 'bar',
      barMaxWidth: 10,
      data: streaks.map((s) => ({
        value: s.value,
        itemStyle: { color: s.is_win_streak ? WIN : LOSS, borderRadius: s.value > 0 ? [2,2,0,0] : [0,0,2,2] },
      })),
      markLine: { silent: true, symbol: 'none', data: [{ yAxis: 0, lineStyle: { color: ZERO_LINE, width: 1 } }] },
    }],
  }
}
```

---

## 14. Champs Go à ajouter par phase

Récapitulatif des champs manquants côté Go requis par ces specs :

| Phase | Champ Go à ajouter | Type |
|------:|-------------------|----|
| 3 | `match_rows[].perf_score` | `*float64` |
| 3 | `match_rows[].time_played_seconds` | `*int` (déjà déclaré, vérifier peuplement) |
| 4 | `skill_rank_history` top-level | `[]SkillRankPoint` |
| 5 | `cumul_tab.cumulative_kd_ci` | `[]CumulativePointWithCI` |
| 5 | `form_tab.regression_line_points` | `[]CumulativePoint` |
| 5 | `intensity_tab.net_score_per_hour` | `[]CumulativePoint` |
| 5 | `intensity_tab.heatmap_data[].win_rate` | `float64` |
| 5 | `intensity_match_phases` top-level | `[]MatchPhasesRow` |
| 1 | `summary_tab.outcomes_over_time` | `[]OutcomeBucket` |
| 1 | `summary_tab.streak_series` | `[]StreakPoint` |
| 7 | `distributions_tab.correlation_points[].trendline` | par paire, 2 points `{x,y}` |

## 15. Composants ECharts à créer

```
apps/web/src/components/charts/
  _utils.ts                    (§1 de cette spec)
  KdaBarsChart.tsx             (§2)
  PerformanceBarsChart.tsx     (§3)
  PerMinSymmetricBarsChart.tsx (§4)
  EwmaKdChart.tsx              (§5)
  CumulKdWithCIChart.tsx       (§6)
  MatchIntensityHeatmap.tsx    (§7)
  SkillRankChart.tsx           (§8)
  NetScorePerHourChart.tsx     (§9)
  WLHeatmapChart.tsx           (§10)
  CorrelationScatterChart.tsx  (§11)
  OutcomesOverTimeChart.tsx    (§12)
  StreakBarsChart.tsx          (§13)
  ChartCard.tsx                (wrapper base méta-plan §3.2.2 — à créer en Phase 0 méta)
```

---

**Fin de spec.** Tous les graphes sont entièrement définis. Un implémenteur peut écrire chaque composant en lisant uniquement sa section + §0+§1.
