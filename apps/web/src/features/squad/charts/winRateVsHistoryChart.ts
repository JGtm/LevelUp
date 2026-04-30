/**
 * winRateVsHistoryChart — Taux de victoire session vs historique par carte.
 *
 * Barres horizontales groupées (yAxis=cartes, xAxis=winRate%).
 * La série session est colorée par token divergent selon l'écart à l'historique :
 *   > +5pp → divergent-pos  /  < -5pp → divergent-neg  /  ±5pp → divergent-neutral
 * La série historique utilise chart-series-1 (couleur de référence stable).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { tokenCssVar } from '@/lib/accessibility'
import { CHART_BG, axisBase, tooltipBase, legendBase } from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'

const DIVERGENCE_THRESHOLD = 0.05

export interface WinRateVsHistoryOpts {
  mapLabelOf: (mapUI: string) => string
  sessionLabel: string
  historyLabel: string
}

function sessionItemColor(row: MapBreakdownRow): string {
  const hist = row.historical_win_rate
  if (hist === undefined) return tokenCssVar('divergent-neutral')
  if (row.win_rate - hist > DIVERGENCE_THRESHOLD) return tokenCssVar('divergent-pos')
  if (hist - row.win_rate > DIVERGENCE_THRESHOLD) return tokenCssVar('divergent-neg')
  return tokenCssVar('divergent-neutral')
}

function toPercent(v: number): number {
  return parseFloat((v * 100).toFixed(1))
}

export function buildWinRateVsHistoryOption(
  series: ChartSeries<MapBreakdownRow>[],
  opts: WinRateVsHistoryOpts,
): EChartsCoreOption {
  const rows = series[0]?.datapoints ?? []
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  const { mapLabelOf, sessionLabel, historyLabel } = opts
  const sorted = [...rows].sort((a, b) => a.win_rate - b.win_rate)
  const mapLabels = sorted.map((r) => mapLabelOf(r.map_ui))
  const histColor = tokenCssVar('chart-series-1')

  const histData = sorted.map((r) =>
    r.historical_win_rate !== undefined ? toPercent(r.historical_win_rate) : null,
  )
  const sessionData = sorted.map((r) => ({
    value: toPercent(r.win_rate),
    itemStyle: { color: sessionItemColor(r) },
  }))

  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 28, left: 8, right: 40, containLabel: true },
    tooltip: {
      ...tooltipBase,
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
    },
    legend: { ...legendBase, data: [historyLabel, sessionLabel] },
    xAxis: {
      ...axisBase,
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { ...axisBase.axisLabel, formatter: '{value}%' },
    },
    yAxis: {
      ...axisBase,
      type: 'category',
      data: mapLabels,
      axisLabel: { ...axisBase.axisLabel, width: 100, overflow: 'truncate' },
    },
    series: [
      {
        name: historyLabel,
        type: 'bar',
        barMaxWidth: 10,
        itemStyle: { color: histColor },
        data: histData,
      },
      {
        name: sessionLabel,
        type: 'bar',
        barMaxWidth: 10,
        data: sessionData,
      },
    ],
  }
}
