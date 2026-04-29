/**
 * TimeseriesKdaBars — barres K (haut, coloré par outcome) + barres D
 * (bas, fixe) + ligne K/D ratio sur axe Y secondaire, rendu via ECharts.
 *
 * Phase 3 P3.G : remplace l'ancien wrapper Plotly `TimeseriesKdaBars`.
 * Construit côté client depuis TimeseriesMatchRow[].
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { resolveToken } from '@/lib/accessibility'
import { outcomeScale } from '@/lib/accessibility/scales'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import {
  CHART_BG,
  axisBase,
  legendBase,
  tooltipBase,
} from '@/components/charts/_utils'
import type { TimeseriesMatchRow } from '@/lib/api/types'

const OUTCOME_INT_KEY: Record<number, string> = {
  2: 'win',
  1: 'draw',
  3: 'loss',
  4: 'dnf',
}

function outcomeColor(outcome: number | null): string {
  const key = outcome != null ? OUTCOME_INT_KEY[outcome] : null
  const token = key ? outcomeScale(key) : null
  return token ? resolveToken(token) : resolveToken('outcome-draw')
}

interface KdaBarPoint {
  x: string
  kills: number
  deaths: number
  kdRatio: number
  outcome: number | null
}

export interface TimeseriesKdaBarsProps {
  rows: TimeseriesMatchRow[]
  height?: number
  labels: {
    kills: string
    deaths: string
    kdRatio: string
    yAxisLeft: string
    yAxisRight: string
    emptyTitle: string
    emptyDescription: string
  }
}

export function TimeseriesKdaBars({
  rows,
  height = 320,
  labels,
}: TimeseriesKdaBarsProps) {
  const { data: fieldMappings } = useFieldMappings()
  const killsLabel = fieldMappings?.fields['kills']?.label ?? labels.kills
  const deathsLabel = fieldMappings?.fields['deaths']?.label ?? labels.deaths

  const series = useMemo<ChartSeries<KdaBarPoint>[]>(() => {
    if (rows.length === 0) return []
    return [
      {
        key: 'timeseries.kda_bars',
        meta: { gamertag: 'kda_bars' },
        datapoints: rows.map((r) => ({
          x: r.start_time,
          kills: r.kills,
          deaths: r.deaths,
          // P4.4 (revue 2026-04-29 B3) : utilise r.kd_ratio expose par
          // l'API (P2.5) au lieu de recalculer kills/deaths cote front.
          // Fallback sur le recompute si kd_ratio manquant (vieux DTO).
          kdRatio: r.kd_ratio ?? (r.deaths > 0 ? r.kills / r.deaths : r.kills),
          outcome: r.outcome,
        })),
      },
    ]
  }, [rows])

  const buildOption = useCallback(
    (s: ChartSeries<KdaBarPoint>[]) =>
      buildKdaBarsOption(s, {
        killsLabel,
        deathsLabel,
        kdRatioLabel: labels.kdRatio,
        yAxisLeft: labels.yAxisLeft,
        yAxisRight: labels.yAxisRight,
      }),
    [killsLabel, deathsLabel, labels.kdRatio, labels.yAxisLeft, labels.yAxisRight],
  )

  if (rows.length === 0) {
    return (
      <EmptyStateNotice title={labels.emptyTitle} description={labels.emptyDescription} />
    )
  }

  return (
    <ChartCard series={series} buildOption={buildOption} height={height} />
  )
}

interface BuildLabels {
  killsLabel: string
  deathsLabel: string
  kdRatioLabel: string
  yAxisLeft: string
  yAxisRight: string
}

/**
 * Pure builder — exporté pour test unitaire.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildKdaBarsOption(
  series: ChartSeries<KdaBarPoint>[],
  labels: BuildLabels,
): EChartsCoreOption {
  if (series.length === 0 || series[0].datapoints.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints

  const xs = dps.map((p) => p.x)
  const killsData = dps.map((p) => ({
    value: p.kills,
    itemStyle: { color: outcomeColor(p.outcome), opacity: 0.85 },
  }))
  const deathsData = dps.map((p) => -p.deaths)
  const kdData = dps.map((p) => p.kdRatio)

  const lossColor = resolveToken('outcome-loss')
  const kdColor = resolveToken('perf-tier-2')

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 56, left: 56, right: 56 },
    tooltip: { ...tooltipBase, trigger: 'axis' },
    legend: {
      ...legendBase,
      data: [labels.killsLabel, labels.deathsLabel, labels.kdRatioLabel],
    },
    xAxis: {
      ...axisBase,
      type: 'category',
      data: xs,
      axisLabel: {
        ...axisBase.axisLabel,
        formatter: (value: string) => {
          const d = new Date(value)
          if (Number.isNaN(d.getTime())) return value
          return `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}`
        },
      },
    },
    yAxis: [
      {
        ...axisBase,
        type: 'value',
        name: labels.yAxisLeft,
        nameLocation: 'middle',
        nameGap: 40,
        nameTextStyle: { color: 'rgba(255,255,255,0.65)', fontSize: 10 },
      },
      {
        ...axisBase,
        type: 'value',
        position: 'right',
        name: labels.yAxisRight,
        nameLocation: 'middle',
        nameGap: 32,
        nameTextStyle: { color: kdColor, fontSize: 10 },
        axisLabel: { ...axisBase.axisLabel, color: kdColor },
        min: 0,
      },
    ],
    series: [
      {
        type: 'bar',
        name: labels.killsLabel,
        stack: 'kda',
        data: killsData,
        barMaxWidth: 12,
      },
      {
        type: 'bar',
        name: labels.deathsLabel,
        stack: 'kda',
        data: deathsData,
        itemStyle: { color: `${lossColor}66` },
        barMaxWidth: 12,
      },
      {
        type: 'line',
        name: labels.kdRatioLabel,
        yAxisIndex: 1,
        data: kdData,
        symbol: 'none',
        lineStyle: { color: kdColor, width: 1.5 },
        itemStyle: { color: kdColor },
      },
    ],
  }
}
