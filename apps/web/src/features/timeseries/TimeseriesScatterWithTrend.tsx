/**
 * TimeseriesScatterWithTrend — scatter coloré par outcome + ligne de tendance OLS.
 *
 * Wrapper local (page Cumul) : reprend les séries win/loss/unknown produites
 * par correlationPointsToSeries et ajoute une ligne de régression linéaire
 * calculée sur tous les points (cf. mock timeseries.10).
 */
import { Suspense, lazy, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { CorrelationDataPair } from '@/lib/api/types'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export interface TimeseriesScatterWithTrendProps {
  /** Tous les correlation points (le composant filtre lui-même par metricKey). */
  points: CorrelationDataPair[]
  metricXKey: string
  metricYKey: string
  height?: number
  xAxisLabel: string
  yAxisLabel: string
  outcomeLabels: { win: string; loss: string; tie?: string; dnf?: string; unknown: string }
  trendLabel: string
  emptyTitle: string
  emptyDescription: string
}

/** OLS sur des points (x, y). Null si <2 points ou variance nulle. */
function fitOLS(xs: number[], ys: number[]): { slope: number; intercept: number } | null {
  const n = xs.length
  if (n < 2) return null
  let sumX = 0
  let sumY = 0
  let sumXY = 0
  let sumX2 = 0
  for (let i = 0; i < n; i++) {
    sumX += xs[i]
    sumY += ys[i]
    sumXY += xs[i] * ys[i]
    sumX2 += xs[i] * xs[i]
  }
  const denom = n * sumX2 - sumX * sumX
  if (denom === 0) return null
  const slope = (n * sumXY - sumX * sumY) / denom
  const intercept = (sumY - slope * sumX) / n
  return { slope, intercept }
}

const OUTCOME_TIE = 1
const OUTCOME_WIN = 2
const OUTCOME_LOSS = 3
const OUTCOME_DNF = 4

export function TimeseriesScatterWithTrend({
  points,
  metricXKey,
  metricYKey,
  height = 260,
  xAxisLabel,
  yAxisLabel,
  outcomeLabels,
  trendLabel,
  emptyTitle,
  emptyDescription,
}: TimeseriesScatterWithTrendProps) {
  const themeVersion = useThemeVersion()

  const filtered = useMemo(
    () =>
      points.filter(
        (p) => p.metric_x_key === metricXKey && p.metric_y_key === metricYKey,
      ),
    [points, metricXKey, metricYKey],
  )


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (filtered.length === 0) return null
    const tc = getEChartsThemeColors()
    const colWin = resolveToken('outcome-win')
    const colLoss = resolveToken('outcome-loss')
    const colTie = resolveToken('outcome-draw')
    const colDnf = resolveToken('outcome-dnf')
    const colUnknown = resolveToken('divergent-neutral')

    const wins = filtered.filter((p) => p.outcome === OUTCOME_WIN)
    const losses = filtered.filter((p) => p.outcome === OUTCOME_LOSS)
    const ties = filtered.filter((p) => p.outcome === OUTCOME_TIE)
    const dnfs = filtered.filter((p) => p.outcome === OUTCOME_DNF)
    const unknowns = filtered.filter(
      (p) =>
        p.outcome !== OUTCOME_WIN &&
        p.outcome !== OUTCOME_LOSS &&
        p.outcome !== OUTCOME_TIE &&
        p.outcome !== OUTCOME_DNF,
    )

    const xs = filtered.map((p) => p.x_value)
    const ys = filtered.map((p) => p.y_value)
    const fit = fitOLS(xs, ys)
    let trendData: [number, number][] = []
    if (fit) {
      const xMin = Math.min(...xs)
      const xMax = Math.max(...xs)
      trendData = [
        [xMin, fit.slope * xMin + fit.intercept],
        [xMax, fit.slope * xMax + fit.intercept],
      ]
    }

    const series: unknown[] = []
    if (wins.length > 0) {
      series.push({
        type: 'scatter',
        name: outcomeLabels.win,
        data: wins.map((p) => [p.x_value, p.y_value]),
        symbolSize: 5,
        itemStyle: { color: colWin, opacity: 0.7 },
      })
    }
    if (losses.length > 0) {
      series.push({
        type: 'scatter',
        name: outcomeLabels.loss,
        data: losses.map((p) => [p.x_value, p.y_value]),
        symbolSize: 5,
        itemStyle: { color: colLoss, opacity: 0.7 },
      })
    }
    if (ties.length > 0 && outcomeLabels.tie) {
      series.push({
        type: 'scatter',
        name: outcomeLabels.tie,
        data: ties.map((p) => [p.x_value, p.y_value]),
        symbolSize: 5,
        itemStyle: { color: colTie, opacity: 0.7 },
      })
    }
    if (dnfs.length > 0 && outcomeLabels.dnf) {
      series.push({
        type: 'scatter',
        name: outcomeLabels.dnf,
        data: dnfs.map((p) => [p.x_value, p.y_value]),
        symbolSize: 5,
        itemStyle: { color: colDnf, opacity: 0.7 },
      })
    }
    if (unknowns.length > 0) {
      series.push({
        type: 'scatter',
        name: outcomeLabels.unknown,
        data: unknowns.map((p) => [p.x_value, p.y_value]),
        symbolSize: 5,
        itemStyle: { color: colUnknown, opacity: 0.6 },
      })
    }
    if (trendData.length === 2) {
      series.push({
        type: 'line',
        name: trendLabel,
        data: trendData,
        showSymbol: false,
        smooth: false,
        lineStyle: { color: tc.axisLabel, width: 1.5, type: 'dashed', opacity: 0.7 },
        z: 5,
      })
    }

    return {
      backgroundColor: CHART_BG,
      // Bottom élargi pour ne pas chevaucher le label X et la légende.
      grid: { top: 32, right: 16, bottom: 80, left: 56 },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'item',
      },
      legend: {
        ...getLegendBase(tc),
        bottom: 0,
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'value',
        name: xAxisLabel,
        nameLocation: 'middle',
        nameGap: 28,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
        name: yAxisLabel,
        nameLocation: 'middle',
        nameGap: 40,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      },
      series,
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filtered, xAxisLabel, yAxisLabel, outcomeLabels, trendLabel, themeVersion])

  if (!option) {
    return <EmptyStateNotice title={emptyTitle} description={emptyDescription} />
  }

  return (
    <Suspense fallback={null}>
      <ReactECharts
        option={option}
        style={{ height, width: '100%' }}
        notMerge
        lazyUpdate
        theme={undefined}
      />
    </Suspense>
  )
}
