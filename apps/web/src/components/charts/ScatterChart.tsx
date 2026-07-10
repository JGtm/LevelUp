/**
 * ScatterChart — wrapper ECharts pour un nuage de points multi-séries.
 *
 * Consomme `ChartSeries<ChartPointScatter>[]` où chaque série porte un nom
 * (label) et une couleur dérivée d'un token. Les datapoints sont des paires
 * {x, y} numériques.
 *
 * Cas d'usage Phase 3 :
 *   - Corrélations Timeseries : 5+ paires (kills↔K/D, lifespan↔kills,
 *     accuracy↔KDA, kills↔deaths, mmr_team↔mmr_enemy…). Couleur par
 *     outcome (win / loss / unknown).
 *
 * Le ChartCard parent gère les états loading/error/empty.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getLegendBase, getTooltipBase, seriesColor } from './_utils'

export interface ChartPointScatter {
  x: number
  y: number
}

export interface ScatterChartProps {
  title?: string
  series: ChartSeries<ChartPointScatter>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Libellé i18n résolu pour l'axe X. */
  xAxisLabel?: string
  /** Libellé i18n résolu pour l'axe Y. */
  yAxisLabel?: string
  /**
   * Map série.key → SemanticToken pour coloration par catégorie (ex. outcome).
   * Si absent, fallback sur seriesColor(idx).
   */
  seriesColorTokens?: Record<string, SemanticToken>
  /** Nom à afficher pour une série dans la légende (default : meta.gamertag). */
  seriesNameResolver?: (s: ChartSeries<ChartPointScatter>) => string
  /** Taille des marqueurs (default 5). */
  symbolSize?: number
}

export function ScatterChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  xAxisLabel,
  yAxisLabel,
  seriesColorTokens,
  seriesNameResolver,
  symbolSize,
}: ScatterChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointScatter>[]) =>
      buildScatterOption(s, {
        xAxisLabel,
        yAxisLabel,
        seriesColorTokens,
        seriesNameResolver,
        symbolSize,
      }),
    [xAxisLabel, yAxisLabel, seriesColorTokens, seriesNameResolver, symbolSize],
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
  xAxisLabel?: string
  yAxisLabel?: string
  seriesColorTokens?: Record<string, SemanticToken>
  seriesNameResolver?: (s: ChartSeries<ChartPointScatter>) => string
  symbolSize?: number
}

function defaultSeriesName(s: ChartSeries<ChartPointScatter>): string {
  const meta = s.meta as { gamertag?: unknown } | undefined
  return typeof meta?.gamertag === 'string' ? meta.gamertag : s.key
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildScatterOption(
  series: ChartSeries<ChartPointScatter>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const {
    xAxisLabel,
    yAxisLabel,
    seriesColorTokens,
    seriesNameResolver = defaultSeriesName,
    symbolSize = 5,
  } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const echartsSeries = series.map((s, idx) => {
    const color = seriesColorTokens?.[s.key]
      ? resolveToken(seriesColorTokens[s.key])
      : seriesColor(idx)
    return {
      type: 'scatter' as const,
      name: seriesNameResolver(s),
      data: s.datapoints.map((p) => [p.x, p.y]),
      symbolSize,
      itemStyle: { color, opacity: 0.75 },
    }
  })

  const legendNames = series.map((s) => seriesNameResolver(s))
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 56, left: 56, right: 16 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (params: unknown) => {
        const p = params as { seriesName?: string; value?: [number, number] }
        const xLabel = xAxisLabel ?? 'X'
        const yLabel = yAxisLabel ?? 'Y'
        const x = p.value?.[0] ?? '—'
        const y = p.value?.[1] ?? '—'
        return `${escapeHtml(p.seriesName ?? '')}<br>${xLabel} : <b>${x}</b><br>${yLabel} : <b>${y}</b>`
      },
    },
    legend: { ...getLegendBase(tc), data: legendNames },
    xAxis: {
      ...axis,
      type: 'value',
      name: xAxisLabel,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      name: yAxisLabel,
      nameLocation: 'middle',
      nameGap: 40,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: echartsSeries,
  }
}
