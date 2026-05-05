/**
 * BarStackedChart — wrapper ECharts pour `ChartSeries<ChartPointStacked>`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ category, components }`.
 *   - Les `components` sont les sous-clés empilées (ex. win/loss/tie/dnf).
 *
 * Cas d'usage Squad V2 :
 *   - S5 Impact ranking par rôle (vertical, components = nb par joueur)
 *   - S6 Cadence (vertical, components = kills par phase)
 *   - S7 HS+PK (vertical, components = headshots + power_weapons)
 *
 * Variants :
 *   - `orientation` : 'vertical' (default) ou 'horizontal' (lollipop-like)
 *   - `componentColors` : map sous-clé → SemanticToken pour palette stable
 *
 * Le ChartCard parent gère les états loading/error/empty.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  seriesColor,
} from './_utils'

export interface ChartPointStacked {
  category: string
  components: Record<string, number>
}

export interface BarStackedChartProps {
  title?: string
  series: ChartSeries<ChartPointStacked>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Vertical (default) ou horizontal (categories sur Y). */
  orientation?: 'vertical' | 'horizontal'
  /**
   * Map sous-clé component → SemanticToken pour coloration cohérente.
   * Ex: { win: 'outcome-win', loss: 'outcome-loss' }.
   */
  componentColors?: Record<string, SemanticToken>
  /** Tableau des sous-clés à inclure dans l'ordre voulu (default: collecte auto). */
  componentOrder?: string[]
}

export function BarStackedChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  orientation = 'vertical',
  componentColors,
  componentOrder,
}: BarStackedChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointStacked>[]) =>
      buildBarStackedOption(s, { orientation, componentColors, componentOrder }),
    [orientation, componentColors, componentOrder],
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
  orientation?: 'vertical' | 'horizontal'
  componentColors?: Record<string, SemanticToken>
  componentOrder?: string[]
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildBarStackedOption(
  series: ChartSeries<ChartPointStacked>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { orientation = 'vertical', componentColors, componentOrder } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  // 1 série attendue (le wrapper agit sur la première).
  const main = series[0]
  const dps = main.datapoints

  const categories = dps.map((d) => d.category)

  // Collecter l'ordre des composants (preserve l'ordre componentOrder si fourni).
  const componentSet = new Set<string>()
  for (const dp of dps) {
    for (const k of Object.keys(dp.components)) componentSet.add(k)
  }
  const components = componentOrder
    ? componentOrder.filter((c) => componentSet.has(c))
    : Array.from(componentSet)

  // 1 ECharts series par component (toutes empilées sur le même stack).
  const echartsSeries = components.map((comp, idx) => {
    const color = componentColors?.[comp]
      ? resolveToken(componentColors[comp])
      : seriesColor(idx)
    return {
      name: comp,
      type: 'bar' as const,
      stack: 'total',
      barMaxWidth: 24,
      itemStyle: { color, borderRadius: 2 },
      data: dps.map((d) => d.components[comp] ?? 0),
    }
  })

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const valueAxis = { ...axis, type: 'value' as const }
  const categoryAxis = {
    ...axis,
    type: 'category' as const,
    data: categories,
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 20, bottom: 40, left: 60, right: 16 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
    },
    legend: { ...getLegendBase(tc), data: components },
    xAxis: orientation === 'horizontal' ? valueAxis : categoryAxis,
    yAxis: orientation === 'horizontal' ? categoryAxis : valueAxis,
    series: echartsSeries,
  }
}
