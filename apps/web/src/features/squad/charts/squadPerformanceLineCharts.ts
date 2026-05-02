/**
 * squadPerformanceLineCharts — teammates.16 : 8 sous-charts performance escouade.
 *
 * Spec : .ai/charts_specs/teammates/16_trio_performance_charts.yaml (étendue
 *        à 8 charts par demande utilisateur — ajout MaxKillingSpree + HS+Perfect).
 *
 * Builders disponibles :
 *   - buildPerformanceLineOption        : 1 line par joueur sur une métrique simple.
 *   - buildKillsDeathsButterflyOption   : sous-chart spécial #1 (kills positifs / deaths négatifs).
 *   - buildHsPerfectOption              : sous-chart #8 — HS line normale + perfect_kills emphasés.
 *
 * Tous consomment `Record<gamertag, SquadPerformanceSeriesPoint[]>` aligné côté
 * serveur sur les matchs partagés (intersection). xAxis = match_order 0..N-1.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { CHART_BG, axisBase, legendBase, tooltipBase } from '@/components/charts/_utils'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

const ZERO_LINE_COLOR = 'rgba(255, 255, 255, 0.55)'

export type PerformanceMetricKey =
  | 'kda'
  | 'assists'
  | 'accuracy'
  | 'avg_life_seconds'
  | 'performance_score'
  | 'max_killing_spree'

interface CommonOpts {
  /** Mapping gamertag → couleur hex. */
  colorByPlayer: Record<string, string>
  /** Ordre des séries (sinon ordre alphabétique). */
  playerOrder?: string[]
}

export interface PerformanceLineOpts extends CommonOpts {
  metric: PerformanceMetricKey
  /** Format `value.toFixed(decimals)` pour le tooltip. */
  decimals?: number
  /** Suffixe pour le label de l'axe Y et le tooltip (ex: " %", " s"). */
  unitSuffix?: string
  /** Multiplicateur appliqué avant affichage (ex: 100 pour accuracy 0..1 → %). */
  scale?: number
}

function extractValue(p: SquadPerformanceSeriesPoint, metric: PerformanceMetricKey): number | null {
  switch (metric) {
    case 'kda':
      return p.kda ?? null
    case 'assists':
      return p.assists
    case 'accuracy':
      return p.accuracy ?? null
    case 'avg_life_seconds':
      return p.avg_life_seconds ?? null
    case 'performance_score':
      return p.performance_score ?? null
    case 'max_killing_spree':
      return p.max_killing_spree ?? null
  }
}

function fmtVal(v: number | null, decimals = 1, suffix = '', scale = 1): string {
  if (v === null) return '-'
  return `${(v * scale).toFixed(decimals)}${suffix}`
}

function orderedPlayers(rows: Record<string, SquadPerformanceSeriesPoint[]>, playerOrder?: string[]): string[] {
  if (playerOrder && playerOrder.length > 0) {
    return playerOrder.filter((p) => rows[p] !== undefined)
  }
  return Object.keys(rows).sort()
}

function maxLength(rows: Record<string, SquadPerformanceSeriesPoint[]>, players: string[]): number {
  let max = 0
  for (const p of players) {
    if (rows[p].length > max) max = rows[p].length
  }
  return max
}

function xAxisLabels(n: number): string[] {
  return Array.from({ length: n }, (_, i) => `#${i + 1}`)
}

export function buildPerformanceLineOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: PerformanceLineOpts,
): EChartsCoreOption {
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }

  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }

  const xLabels = xAxisLabels(n)
  const scale = opts.scale ?? 1
  const decimals = opts.decimals ?? 1
  const suffix = opts.unitSuffix ?? ''

  const series = players.map((player) => {
    const data = new Array<number | null>(n).fill(null)
    for (const p of rows[player]) {
      const idx = p.match_order
      const v = extractValue(p, opts.metric)
      if (v === null) {
        data[idx] = null
      } else {
        data[idx] = Number((v * scale).toFixed(decimals))
      }
    }
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    return {
      name: player,
      type: 'line' as const,
      data,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      symbol: 'circle' as const,
      symbolSize: 5,
      connectNulls: true,
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 28, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...tooltipBase,
      trigger: 'axis',
      axisPointer: { type: 'line' },
      valueFormatter: (v: unknown) => fmtVal(typeof v === 'number' ? v : null, decimals, suffix),
    },
    legend: { ...legendBase, data: players },
    xAxis: {
      ...axisBase,
      type: 'category',
      data: xLabels,
      axisLabel: {
        ...axisBase.axisLabel,
        interval: n > 30 ? Math.floor(n / 12) : 0,
      },
    },
    yAxis: {
      ...axisBase,
      type: 'value',
      axisLabel: {
        ...axisBase.axisLabel,
        formatter: (v: number) => `${v.toFixed(decimals)}${suffix}`,
      },
    },
    series,
  }
}

// ---------------------------------------------------------------------------
// Sous-chart #1 — Frags / Morts (butterfly)
// ---------------------------------------------------------------------------

export interface KillsDeathsButterflyOpts extends CommonOpts {
  killsLabel: string
  deathsLabel: string
}

const DEATHS_OPACITY = 0.45

export function buildKillsDeathsButterflyOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: KillsDeathsButterflyOpts,
): EChartsCoreOption {
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }
  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }
  const xLabels = xAxisLabels(n)

  const seriesPerPlayer: Array<Record<string, unknown>> = []
  for (const player of players) {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const killsData = new Array<number | null>(n).fill(null)
    const deathsData = new Array<number | null>(n).fill(null)
    for (const p of rows[player]) {
      killsData[p.match_order] = p.kills
      deathsData[p.match_order] = -p.deaths
    }
    seriesPerPlayer.push({
      name: `${player} — ${opts.killsLabel}`,
      type: 'bar',
      stack: `kills-${player}`,
      barMaxWidth: 14,
      itemStyle: { color },
      data: killsData,
    })
    seriesPerPlayer.push({
      name: `${player} — ${opts.deathsLabel}`,
      type: 'bar',
      stack: `deaths-${player}`,
      barMaxWidth: 14,
      itemStyle: { color, opacity: DEATHS_OPACITY },
      data: deathsData,
    })
  }

  const legendData = players.flatMap((p) => [`${p} — ${opts.killsLabel}`, `${p} — ${opts.deathsLabel}`])

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...tooltipBase,
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      valueFormatter: (v: unknown) => (typeof v === 'number' ? `${Math.abs(v)}` : '-'),
    },
    legend: { ...legendBase, data: legendData, type: 'scroll' },
    xAxis: { ...axisBase, type: 'category', data: xLabels },
    yAxis: {
      ...axisBase,
      type: 'value',
      axisLine: { lineStyle: { color: ZERO_LINE_COLOR, width: 1.5 } },
      axisLabel: { ...axisBase.axisLabel, formatter: (v: number) => `${Math.abs(v)}` },
    },
    series: seriesPerPlayer,
  }
}

// ---------------------------------------------------------------------------
// Sous-chart #8 — Tirs à la tête + frags parfaits (perfect mis en valeur)
// ---------------------------------------------------------------------------

export interface HsPerfectOpts extends CommonOpts {
  hsLabel: string
  perfectLabel: string
}

export function buildHsPerfectOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  opts: HsPerfectOpts,
): EChartsCoreOption {
  const players = orderedPlayers(rows, opts.playerOrder)
  if (players.length === 0) return { backgroundColor: CHART_BG }
  const n = maxLength(rows, players)
  if (n === 0) return { backgroundColor: CHART_BG }
  const xLabels = xAxisLabels(n)

  // Pour chaque joueur : 1 line HS (couleur normale, dotted, fine) + 1 line
  // perfect_kills (couleur emphasée — opacity 1, width 3, marker carré, area
  // pour faire ressortir les pics de perfect kills).
  const series: Array<Record<string, unknown>> = []
  for (const player of players) {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const hsData = new Array<number | null>(n).fill(null)
    const perfectData = new Array<number | null>(n).fill(null)
    for (const p of rows[player]) {
      hsData[p.match_order] = p.headshot_kills ?? null
      perfectData[p.match_order] = p.perfect_kills ?? null
    }
    // HS : ligne fine, dashed, opacity réduite.
    series.push({
      name: `${player} — ${opts.hsLabel}`,
      type: 'line',
      data: hsData,
      lineStyle: { color, width: 1.5, type: 'dashed', opacity: 0.7 },
      itemStyle: { color, opacity: 0.7 },
      symbol: 'circle',
      symbolSize: 4,
      connectNulls: true,
    })
    // Perfect kills : ligne épaisse + areaStyle pour mise en valeur.
    series.push({
      name: `${player} — ${opts.perfectLabel}`,
      type: 'line',
      data: perfectData,
      lineStyle: { color, width: 3 },
      itemStyle: { color },
      areaStyle: { color, opacity: 0.18 },
      symbol: 'diamond',
      symbolSize: 8,
      emphasis: { focus: 'series', scale: 1.5 },
      connectNulls: true,
    })
  }

  const legendData = players.flatMap((p) => [`${p} — ${opts.hsLabel}`, `${p} — ${opts.perfectLabel}`])

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: { ...tooltipBase, trigger: 'axis', axisPointer: { type: 'line' } },
    legend: { ...legendBase, data: legendData, type: 'scroll' },
    xAxis: { ...axisBase, type: 'category', data: xLabels },
    yAxis: {
      ...axisBase,
      type: 'value',
      axisLabel: { ...axisBase.axisLabel },
    },
    series,
  }
}
