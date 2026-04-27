/**
 * DonutChart — wrapper ECharts pour un donut/pie depuis
 * `ChartSeries<ChartPointDonut>[]` (1 série, plusieurs slices).
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ name, value }`.
 *   - Couleurs par slice via `sliceColors` (map name → SemanticToken) ou
 *     fallback sur `seriesColor(idx)`.
 *
 * Cas d'usage Phase 3 :
 *   - SessionComparePage : répartition des outcomes (wins / losses / other)
 *     d'une paire de sessions.
 *
 * Le ChartCard parent gère les états loading/error/empty.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, legendBase, seriesColor, tooltipBase } from './_utils'

export interface ChartPointDonut {
  name: string
  value: number
}

export interface DonutChartProps {
  title?: string
  series: ChartSeries<ChartPointDonut>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Map slice.name → SemanticToken pour coloration sémantique. */
  sliceColors?: Record<string, SemanticToken>
  /** Largeur intérieure du donut (default '50%'). 0% = pie plein. */
  innerRadius?: string
  /** Rayon externe (default '75%'). */
  outerRadius?: string
  /** Si true, affiche les pourcentages dans les labels (default true). */
  showPercent?: boolean
}

export function DonutChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  sliceColors,
  innerRadius,
  outerRadius,
  showPercent,
}: DonutChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointDonut>[]) =>
      buildDonutOption(s, { sliceColors, innerRadius, outerRadius, showPercent }),
    [sliceColors, innerRadius, outerRadius, showPercent],
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
  sliceColors?: Record<string, SemanticToken>
  innerRadius?: string
  outerRadius?: string
  showPercent?: boolean
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildDonutOption(
  series: ChartSeries<ChartPointDonut>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const {
    sliceColors,
    innerRadius = '50%',
    outerRadius = '75%',
    showPercent = true,
  } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints
  if (dps.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const data = dps.map((p, idx) => {
    const color = sliceColors?.[p.name]
      ? resolveToken(sliceColors[p.name])
      : seriesColor(idx)
    return {
      name: p.name,
      value: p.value,
      itemStyle: { color },
    }
  })

  const legendNames = dps.map((p) => p.name)

  return {
    backgroundColor: CHART_BG,
    tooltip: {
      ...tooltipBase,
      trigger: 'item',
      formatter: '{b} : <b>{c}</b> ({d}%)',
    },
    legend: { ...legendBase, data: legendNames },
    series: [
      {
        type: 'pie',
        radius: [innerRadius, outerRadius],
        avoidLabelOverlap: true,
        label: {
          show: showPercent,
          color: 'rgba(255,255,255,0.85)',
          fontSize: 11,
          formatter: showPercent ? '{b}\n{d}%' : '{b}',
        },
        labelLine: { length: 8, length2: 6 },
        data,
      },
    ],
  }
}
