/**
 * HistogramChart — wrapper ECharts pour un histogramme depuis
 * `ChartSeries<ChartPointHistogram>[]`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ binStart, binEnd, count }`.
 *   - Les buckets sont rendus comme des barres adjacentes (catégories).
 *
 * Cas d'usage Phase 3 :
 *   - Distributions Timeseries : K/D, kills/match, précision, score/min,
 *     win rate glissant.
 *
 * Le ChartCard parent gère les états loading/error/empty.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase, seriesColor } from './_utils'

export interface ChartPointHistogram {
  binStart: number
  binEnd: number
  count: number
}

export interface HistogramChartProps {
  title?: string
  series: ChartSeries<ChartPointHistogram>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Token couleur pour la barre (default chart-series-1). */
  colorToken?: SemanticToken
  /** Libellé de l'axe X (ex. "K/D", "Kills / match"). */
  xAxisLabel?: string
  /** Libellé de l'axe Y (default = nb de matchs en FR). */
  yAxisLabel?: string
  /**
   * Format des bornes de bucket. Default : "binStart–binEnd" arrondi à 2
   * décimales si non-entier.
   */
  formatBin?: (point: ChartPointHistogram) => string
}

export function HistogramChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  colorToken,
  xAxisLabel,
  yAxisLabel,
  formatBin,
}: HistogramChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHistogram>[]) =>
      buildHistogramOption(s, { colorToken, xAxisLabel, yAxisLabel, formatBin }),
    [colorToken, xAxisLabel, yAxisLabel, formatBin],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={buildOption}
    />
  )
}

interface BuildOpts {
  colorToken?: SemanticToken
  xAxisLabel?: string
  yAxisLabel?: string
  formatBin?: (point: ChartPointHistogram) => string
}

function defaultFormatBin(point: ChartPointHistogram): string {
  const fmt = (n: number): string => (Number.isInteger(n) ? String(n) : n.toFixed(2))
  return `${fmt(point.binStart)}–${fmt(point.binEnd)}`
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildHistogramOption(
  series: ChartSeries<ChartPointHistogram>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { colorToken, xAxisLabel, yAxisLabel: yLabelOpt, formatBin = defaultFormatBin } = opts
  const yAxisLabel = yLabelOpt ?? 'Matchs'
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints
  if (dps.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const categories = dps.map((d) => formatBin(d))
  const counts = dps.map((d) => d.count)
  const color = colorToken ? resolveToken(colorToken) : seriesColor(0)

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 48, right: 12 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: categories,
      name: xAxisLabel,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, rotate: -30 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      name: yAxisLabel,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        type: 'bar',
        data: counts,
        barCategoryGap: '10%',
        itemStyle: { color, borderRadius: 2 },
      },
    ],
  }
}
