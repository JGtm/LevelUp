/**
 * BarGroupedChart — wrapper ECharts pour bars groupées (côte à côte).
 *
 * Consomme `ChartSeries<ChartPointStacked>` mais NE PAS empile : chaque
 * sous-clé est rendue comme une barre adjacente. Cas d'usage Squad V2 :
 *   - S7 Per-minute stats (kills/deaths/assists par joueur côte à côte)
 *   - S7 Frags/Deaths combined
 *
 * Diffère de BarStackedChart par l'absence de `stack`.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import type { ChartPointStacked } from './BarStackedChart'
import {
  CHART_BG,
  axisBase,
  legendBase,
  seriesColor,
  tooltipBase,
} from './_utils'

export interface BarGroupedChartProps {
  title?: string
  series: ChartSeries<ChartPointStacked>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  componentColors?: Record<string, SemanticToken>
  componentOrder?: string[]
}

export function BarGroupedChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  componentColors,
  componentOrder,
}: BarGroupedChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointStacked>[]) =>
      buildBarGroupedOption(s, { componentColors, componentOrder }),
    [componentColors, componentOrder],
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
  componentColors?: Record<string, SemanticToken>
  componentOrder?: string[]
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildBarGroupedOption(
  series: ChartSeries<ChartPointStacked>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { componentColors, componentOrder } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints
  const categories = dps.map((d) => d.category)

  const componentSet = new Set<string>()
  for (const dp of dps) {
    for (const k of Object.keys(dp.components)) componentSet.add(k)
  }
  const components = componentOrder
    ? componentOrder.filter((c) => componentSet.has(c))
    : Array.from(componentSet)

  const echartsSeries = components.map((comp, idx) => {
    const color = componentColors?.[comp]
      ? resolveToken(componentColors[comp])
      : seriesColor(idx)
    return {
      name: comp,
      type: 'bar' as const,
      barMaxWidth: 14,
      itemStyle: { color, borderRadius: 2 },
      data: dps.map((d) => d.components[comp] ?? 0),
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 20, bottom: 40, left: 56, right: 16 },
    tooltip: {
      ...tooltipBase,
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
    },
    legend: { ...legendBase, data: components },
    xAxis: { ...axisBase, type: 'category', data: categories },
    yAxis: { ...axisBase, type: 'value' },
    series: echartsSeries,
  }
}
