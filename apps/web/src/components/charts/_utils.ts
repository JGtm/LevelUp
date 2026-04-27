/**
 * _utils.ts — Helpers partagés des wrappers ECharts Squad V2.
 *
 * Conformément au PLAN_META_FOUNDATIONS_GO § 5.2 : tous les wrappers
 * (BarStacked, BarGrouped, TimeseriesLine, Heatmap2D, Radar) consomment
 * `ChartSeries<T>[]` (mirror du domain.ChartSeries[T] Go) et reposent
 * sur les helpers ci-dessous pour cohérence visuelle.
 *
 * Couleurs : jamais de hex direct. Tout passe par resolveToken() pour
 * permettre le thème dark/light + accessibilité (cf.
 * apps/web/src/lib/accessibility/).
 */
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

export const CHART_BG = 'transparent'
export const GRID_COLOR = 'rgba(255,255,255,0.06)'
export const TEXT_COLOR = 'rgba(255,255,255,0.45)'
export const ZERO_LINE = 'rgba(255,255,255,0.15)'

/** Base axis style (axes X/Y). À spread avant les overrides spécifiques. */
export const axisBase = {
  axisLine: { lineStyle: { color: GRID_COLOR } },
  axisTick: { show: false },
  splitLine: { lineStyle: { color: GRID_COLOR } },
  axisLabel: { color: TEXT_COLOR, fontSize: 10 },
} as const

/** Base tooltip style. À spread avant `formatter` ou `trigger`. */
export const tooltipBase = {
  backgroundColor: 'rgba(20,24,30,0.92)',
  borderColor: GRID_COLOR,
  textStyle: { color: 'rgba(255,255,255,0.85)', fontSize: 11 },
  extraCssText: 'border-radius:6px;box-shadow:0 4px 12px rgba(0,0,0,0.4)',
} as const

/** Base legend style (bas du chart). */
export const legendBase = {
  bottom: 0,
  textStyle: { color: TEXT_COLOR, fontSize: 10 },
  itemWidth: 12,
  itemHeight: 8,
} as const

/**
 * Couleurs des outcomes Halo (win/loss/tie/dnf).
 * Résolu côté composant via resolveToken (pas de hex direct).
 */
export function outcomeColor(outcome: string | undefined): string {
  switch (outcome) {
    case 'win':
      return resolveToken('outcome-win')
    case 'loss':
      return resolveToken('outcome-loss')
    case 'tie':
      return resolveToken('outcome-draw')
    case 'dnf':
      return resolveToken('outcome-dnf')
    default:
      return resolveToken('chart-series-1')
  }
}

/**
 * Couleurs séries (chart-series-1..8) cyclées modulo 8.
 * Pour wrappers multi-séries (ex. TimeseriesLine, Radar).
 */
export function seriesColor(index: number): string {
  const tokens: SemanticToken[] = [
    'chart-series-1',
    'chart-series-2',
    'chart-series-3',
    'chart-series-4',
    'chart-series-5',
    'chart-series-6',
    'chart-series-7',
    'chart-series-8',
  ]
  return resolveToken(tokens[index % tokens.length])
}

/**
 * Composant générique d'axe X catégoriel — paramètre les ticks selon le
 * nombre de catégories (rotation labels au-delà de 60 entrées).
 */
export function categoricalXAxis(categories: string[]): EChartsCoreOption['xAxis'] {
  const n = categories.length
  return {
    ...axisBase,
    type: 'category',
    data: categories,
    axisLabel: {
      ...axisBase.axisLabel,
      interval: tickInterval(n) - 1,
      rotate: n > 60 ? 30 : n > 20 ? 15 : 0,
    },
  }
}

/**
 * Calcule l'intervalle de tick pour un nombre N de points (cap visuel).
 * Utilisé pour éviter les axes surchargés.
 */
export function tickInterval(n: number): number {
  if (n <= 10) return 1
  if (n <= 30) return 2
  if (n <= 60) return 5
  if (n <= 120) return 10
  return Math.ceil(n / 12)
}

/**
 * Format date FR court (DD/MM) pour les axes timeseries.
 */
export function formatDateShort(d: Date | string | number): string {
  const date = d instanceof Date ? d : new Date(d)
  return date.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' })
}

/**
 * Format chiffre arrondi (1 décimale par défaut) pour tooltips.
 */
export function formatNumber(v: number, decimals = 1): string {
  if (!Number.isFinite(v)) return '-'
  return v.toFixed(decimals)
}
