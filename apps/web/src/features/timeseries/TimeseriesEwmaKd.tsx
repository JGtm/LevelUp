/**
 * TimeseriesEwmaKd — chart EWMA K/D (timeseries.25).
 *
 * 3 séries :
 *   1. Scatter K/D bruts par match (kd_ratio des matchRows) — opacity 0.5
 *   2. Ligne EWMA (α=0.20, calculée backend) — épaisse, chart-series-1
 *   3. Droite de régression (optionnelle, ssi r²≥0.3) — tiretée, couleur par tendance
 *
 * markLine horizontal à y=1.0 (référence K/D neutre).
 * L'intercept de la régression est recalculé côté client depuis kd_slope +
 * les K/D bruts (l'API Go n'expose pas l'intercept dans regression_stats).
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
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { CumulativePoint, TimeseriesMatchRow, TimeseriesRegressionStats } from '@/lib/api/types'
import { buildMatchCategories } from './matchLabels'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

const R2_THRESHOLD = 0.3

/** Calcule les points de la droite de régression à partir du slope et des K/D bruts. */
function buildRegressionLine(
  kdValues: (number | null)[],
  slope: number,
): (number | null)[] {
  const valid = kdValues
    .map((v, i) => (v != null ? { x: i, y: v } : null))
    .filter((p): p is { x: number; y: number } => p !== null)
  if (valid.length < 2) return new Array(kdValues.length).fill(null)
  const meanX = valid.reduce((s, p) => s + p.x, 0) / valid.length
  const meanY = valid.reduce((s, p) => s + p.y, 0) / valid.length
  const intercept = meanY - slope * meanX
  return kdValues.map((_, i) => Math.round((intercept + slope * i) * 100) / 100)
}

export interface TimeseriesEwmaKdProps {
  ewmaPoints: CumulativePoint[]
  regressionStats: TimeseriesRegressionStats
  matchRows: TimeseriesMatchRow[]
  height?: number
  ewmaLabel: string
  perMatchLabel: string
  refLineLabel: string
  trendLabel: string
}

export function TimeseriesEwmaKd({
  ewmaPoints,
  regressionStats,
  matchRows,
  height = 380,
  ewmaLabel,
  perMatchLabel,
  refLineLabel,
  trendLabel,
}: TimeseriesEwmaKdProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (ewmaPoints.length === 0) return null
    const tc = getEChartsThemeColors()
    const { trend, r_squared, kd_slope, has_enough_for_trend } = regressionStats
    const showRegression =
      has_enough_for_trend && r_squared != null && r_squared >= R2_THRESHOLD && kd_slope != null

    const colEwma = resolveToken('chart-series-1')
    const colScatter = resolveToken('chart-series-3')
    const colRegression =
      trend === 'improving'
        ? resolveToken('outcome-win')
        : trend === 'declining'
        ? resolveToken('outcome-loss')
        : tc.axisLine

    const categories = buildMatchCategories(matchRows)
    // Priorité : kda (FDA stockée en DB depuis l'API Halo), fallback kd_ratio.
    const kdValues: (number | null)[] = matchRows.map((r) => {
      const v = r.kda ?? r.kd_ratio
      return v != null && Number.isFinite(v) ? Math.round(v * 100) / 100 : null
    })
    const ewmaValues = ewmaPoints.map((p) => p.value)
    const regressionLine = showRegression
      ? buildRegressionLine(kdValues, kd_slope!)
      : null

    const series: unknown[] = [
      {
        type: 'scatter',
        name: perMatchLabel,
        data: kdValues,
        symbolSize: 5,
        itemStyle: { color: colScatter, opacity: 0.45 },
        z: 1,
        // Zone de stabilité : bande [0.85, 1.15] autour du K/D neutre 1.0.
        markArea: {
          silent: true,
          itemStyle: { color: tc.axisLine, opacity: 0.08 },
          label: { show: false },
          data: [[{ yAxis: 0.85 }, { yAxis: 1.15 }]],
        },
      },
      {
        type: 'line',
        name: ewmaLabel,
        data: ewmaValues,
        showSymbol: false,
        smooth: false,
        connectNulls: true,
        lineStyle: { color: colEwma, width: 3 },
        itemStyle: { color: colEwma },
        z: 3,
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLine, width: 1, type: 'dashed' },
          data: [
            {
              yAxis: 1,
              label: {
                formatter: refLineLabel,
                position: 'end',
                color: tc.axisLabel,
                fontSize: 10,
              },
            },
          ],
        },
      },
    ]

    if (regressionLine) {
      series.push({
        type: 'line',
        name: trendLabel,
        data: regressionLine,
        showSymbol: false,
        smooth: false,
        connectNulls: true,
        lineStyle: { color: colRegression, width: 1.5, type: 'dashed' },
        itemStyle: { color: colRegression },
        z: 2,
      })
    }

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: { ...getAxisBase(tc), type: 'value', min: 0 },
      series,
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ewmaPoints, regressionStats, matchRows, ewmaLabel, perMatchLabel, refLineLabel, trendLabel, themeVersion])

  if (!option) return null
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
