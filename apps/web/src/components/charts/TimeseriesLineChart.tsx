/**
 * TimeseriesLineChart — wrapper ECharts pour `ChartSeries<ChartPoint2D>[]`
 * en mode "1 trace par série, X = chronologie".
 *
 * Cas d'usage Squad V2 :
 *   - S4 Timeline performance multi-joueurs (1 trace/joueur, marker outcome)
 *   - S4 Form Score (1 trace lissée LOWESS)
 *   - S7 5 charts trio (assists, KDA, accuracy, avg life, performance)
 *   - S7 Killing spree max lissé
 *
 * Le composant accepte plusieurs `ChartSeries<ChartPoint2D>` ; chaque série
 * devient une trace ECharts. Les couleurs sont résolues via tokens
 * (chart-series-1..8 cyclées).
 *
 * Le `Label` d'un ChartPoint2D peut porter un outcome ("win"/"loss"/"tie"/
 * "dnf") — dans ce cas, le marker est coloré selon l'outcome.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from './ChartCard'
import {
  CHART_BG,
  axisBase,
  legendBase,
  outcomeColor,
  seriesColor,
  tooltipBase,
} from './_utils'

export interface ChartPoint2D {
  x: string | number | Date
  y: number
  label?: string
}

export interface TimeseriesLineChartProps {
  title?: string
  series: ChartSeries<ChartPoint2D>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Si true, X est traité comme un axe temporel (datetime). Default true. */
  timeAxis?: boolean
  /** Si true, marker outcome utilise outcomeColor. Default true. */
  outcomeMarkers?: boolean
  /** Optionnel : nom d'une série côté front (override LabelKey backend). */
  seriesNameResolver?: (s: ChartSeries<ChartPoint2D>) => string
}

export function TimeseriesLineChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  timeAxis = true,
  outcomeMarkers = true,
  seriesNameResolver,
}: TimeseriesLineChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPoint2D>[]) =>
      buildTimeseriesLineOption(s, {
        timeAxis,
        outcomeMarkers,
        seriesNameResolver,
      }),
    [timeAxis, outcomeMarkers, seriesNameResolver],
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
  timeAxis?: boolean
  outcomeMarkers?: boolean
  seriesNameResolver?: (s: ChartSeries<ChartPoint2D>) => string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildTimeseriesLineOption(
  series: ChartSeries<ChartPoint2D>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { timeAxis = true, outcomeMarkers = true, seriesNameResolver } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const echartsSeries = series.map((s, idx) => {
    const color = seriesColor(idx)
    const name =
      seriesNameResolver?.(s) ??
      (s.meta && typeof s.meta.gamertag === 'string' ? s.meta.gamertag : s.key)

    const data = s.datapoints.map((dp) => {
      const x = timeAxis && !(dp.x instanceof Date) ? new Date(dp.x as string) : dp.x
      const itemStyle =
        outcomeMarkers && dp.label
          ? { color: outcomeColor(dp.label) }
          : undefined
      return {
        value: [x, dp.y] as [string | number | Date, number],
        itemStyle,
      }
    })

    return {
      name,
      type: 'line' as const,
      smooth: false,
      showSymbol: true,
      symbolSize: 6,
      lineStyle: { color, width: 1.5 },
      itemStyle: { color },
      data,
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 56, left: 56, right: 16 },
    tooltip: {
      ...tooltipBase,
      trigger: 'axis',
      axisPointer: { type: 'cross' },
    },
    legend: { ...legendBase, data: echartsSeries.map((s) => s.name) },
    xAxis: {
      ...axisBase,
      type: timeAxis ? 'time' : 'category',
    },
    yAxis: { ...axisBase, type: 'value' },
    series: echartsSeries,
  }
}
