/**
 * combatChartOptions — pure builders ECharts pour le "Profil de combat" Explorer.
 *
 * Deux builders exportés et testables (modelés sur buildKdaBarsOption de
 * features/timeseries) : G1 FDA + Frags/Morts/Assists (double axe Y) et
 * G3 Score + Placement (axe placement inversé). Le pattern est recopié plutôt
 * qu'importé depuis features/timeseries pour respecter lint-cross-feature-imports.
 *
 * Couleurs uniquement via tokens sémantiques (resolveToken). Cf.
 * PLAN_explorer_combat_profile_charts.md §7 + §Spécification par graphe.
 */
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'

/** Format court JJ/MM d'un start_time ISO (axe catégoriel temporel). */
function fmtDayMonth(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}`
}

// ─── G1 : FDA + Frags / Morts / Assistances ───────────────────────────────

export interface CombatFdaPoint {
  x: string
  kills: number
  deaths: number
  assists: number
  kda: number
  outcome: number | null
}

export interface CombatFdaLabels {
  kills: string
  deaths: string
  assists: string
  fda: string
  yAxisLeft: string
  yAxisRight: string
}

/**
 * Builder G1 — 3 barres groupées (kills/deaths/assists, PAS de stack) + 1 ligne
 * FDA sur l'axe Y secondaire (yAxisIndex:1) → double axe Y (yAxis.length===2).
 * Exporté pour test unitaire.
 */
export function buildCombatFdaOption(
  series: ChartSeries<CombatFdaPoint>[],
  labels: CombatFdaLabels,
): EChartsCoreOption {
  if (series.length === 0 || series[0].datapoints.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const dps = series[0].datapoints
  const xs = dps.map((p) => fmtDayMonth(p.x))

  const killsColor = resolveToken('chart-series-1')
  const deathsColor = resolveToken('outcome-loss')
  const assistsColor = resolveToken('chart-series-3')
  const fdaColor = resolveToken('perf-tier-2')
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 56, right: 56 },
    tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
    legend: {
      ...getLegendBase(tc),
      data: [labels.kills, labels.deaths, labels.assists, labels.fda],
    },
    xAxis: { ...axis, type: 'category', data: xs },
    yAxis: [
      {
        ...axis,
        type: 'value',
        name: labels.yAxisLeft,
        nameLocation: 'middle',
        nameGap: 40,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      },
      {
        ...axis,
        type: 'value',
        position: 'right',
        name: labels.yAxisRight,
        nameLocation: 'middle',
        nameGap: 32,
        nameTextStyle: { color: fdaColor, fontSize: 10 },
        axisLabel: { ...axis.axisLabel, color: fdaColor },
        min: 0,
      },
    ],
    series: [
      {
        type: 'bar',
        name: labels.kills,
        data: dps.map((p) => p.kills),
        itemStyle: { color: killsColor },
        barMaxWidth: 10,
      },
      {
        type: 'bar',
        name: labels.deaths,
        data: dps.map((p) => p.deaths),
        itemStyle: { color: deathsColor },
        barMaxWidth: 10,
      },
      {
        type: 'bar',
        name: labels.assists,
        data: dps.map((p) => p.assists),
        itemStyle: { color: assistsColor },
        barMaxWidth: 10,
      },
      {
        type: 'line',
        name: labels.fda,
        yAxisIndex: 1,
        data: dps.map((p) => p.kda),
        symbol: 'circle',
        symbolSize: 4,
        lineStyle: { color: fdaColor, width: 1.5 },
        itemStyle: { color: fdaColor },
      },
    ],
  }
}

// ─── G3 : Score + Placement (axe placement inversé) ───────────────────────

export interface CombatScorePoint {
  x: string
  score: number
  /** null si DNF/non classé → point null, jamais 0 (fausserait l'axe inversé). */
  rank: number | null
}

export interface CombatScoreLabels {
  score: string
  placement: string
  yAxisLeft: string
  yAxisRight: string
}

/**
 * Builder G3 — barres score + courbe placement sur l'axe Y secondaire inversé
 * (yAxis[1] = { position:'right', inverse:true, min:1 } → rang 1 en haut).
 * `rank===null` → point `null` + `connectNulls:false`. Exporté pour test.
 */
export function buildCombatScoreOption(
  series: ChartSeries<CombatScorePoint>[],
  labels: CombatScoreLabels,
): EChartsCoreOption {
  if (series.length === 0 || series[0].datapoints.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const dps = series[0].datapoints
  const xs = dps.map((p) => fmtDayMonth(p.x))

  const scoreColor = resolveToken('chart-series-1')
  const rankColor = resolveToken('perf-tier-2')
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 64, right: 56 },
    tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
    legend: { ...getLegendBase(tc), data: [labels.score, labels.placement] },
    xAxis: { ...axis, type: 'category', data: xs },
    yAxis: [
      {
        ...axis,
        type: 'value',
        name: labels.yAxisLeft,
        nameLocation: 'middle',
        nameGap: 44,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      },
      {
        ...axis,
        type: 'value',
        position: 'right',
        inverse: true,
        min: 1,
        name: labels.yAxisRight,
        nameLocation: 'middle',
        nameGap: 28,
        nameTextStyle: { color: rankColor, fontSize: 10 },
        axisLabel: { ...axis.axisLabel, color: rankColor },
      },
    ],
    series: [
      {
        type: 'bar',
        name: labels.score,
        data: dps.map((p) => p.score),
        itemStyle: { color: scoreColor },
        barMaxWidth: 14,
      },
      {
        type: 'line',
        name: labels.placement,
        yAxisIndex: 1,
        data: dps.map((p) => (p.rank == null ? null : p.rank)),
        connectNulls: false,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { color: rankColor, width: 1.5 },
        itemStyle: { color: rankColor },
      },
    ],
  }
}
