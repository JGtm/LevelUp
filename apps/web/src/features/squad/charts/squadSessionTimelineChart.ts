/**
 * squadSessionTimelineChart — teammates.04 : performance d'escouade par session.
 *
 * Spec : .ai/charts_specs/teammates/04_squad_timeline.yaml
 *
 * Composition :
 *   - Bars perf squad colorées par tier (perf-tier-1..5 selon seuils 75/60/45/30)
 *   - Line winrate (% sur même axe Y) si winRate non nul sur ≥1 point
 *   - Line MMR moyen équipe sur axe Y2 (pointillé) si team_mmr_avg dispo
 *   - tooltip trigger='axis' (hover unifié par session)
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken } from '@/lib/accessibility'
import { perfSessionScale } from '@/lib/accessibility/scales'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  seriesColor,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadSessionPoint } from '@/lib/api/types'

export interface SquadSessionTimelineOpts {
  perfLabel: string
  winRateLabel: string
  mmrLabel: string
  perfAxisLabel: string
  mmrAxisLabel: string
}

function round1(v: number): number {
  return parseFloat(v.toFixed(1))
}

export function buildSquadSessionTimelineOption(
  series: ChartSeries<SquadSessionPoint>[],
  opts: SquadSessionTimelineOpts,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const labels = points.map((p) => p.session_label)
  const perfData = points.map((p) => ({
    value: round1(p.squad_perf),
    itemStyle: { color: resolveToken(perfSessionScale(p.squad_perf)), opacity: 0.85 },
  }))
  const hasWinRate = points.some((p) => p.win_rate !== undefined)
  const hasMmr = points.some((p) => p.team_mmr_avg !== undefined)
  const winRateData = points.map((p) =>
    p.win_rate !== undefined ? round1(p.win_rate * 100) : null,
  )
  const mmrData = points.map((p) => (p.team_mmr_avg !== undefined ? round1(p.team_mmr_avg) : null))

  const legendData = [opts.perfLabel]
  if (hasWinRate) legendData.push(opts.winRateLabel)
  if (hasMmr) legendData.push(opts.mmrLabel)

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const yAxis: object[] = [
    {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      name: opts.perfAxisLabel,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value}' },
    },
  ]
  if (hasMmr) {
    yAxis.push({
      ...axis,
      type: 'value',
      name: opts.mmrAxisLabel,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      splitLine: { show: false },
      axisLabel: { ...axis.axisLabel, formatter: '{value}' },
    })
  }

  const echSeries: object[] = [
    {
      name: opts.perfLabel,
      type: 'bar',
      data: perfData,
      barMaxWidth: 18,
      yAxisIndex: 0,
    },
  ]
  if (hasWinRate) {
    echSeries.push({
      name: opts.winRateLabel,
      type: 'line',
      data: winRateData,
      smooth: false,
      yAxisIndex: 0,
      lineStyle: { width: 2, color: seriesColor(2) },
      itemStyle: { color: seriesColor(2) },
      symbol: 'circle',
      symbolSize: 6,
    })
  }
  if (hasMmr) {
    echSeries.push({
      name: opts.mmrLabel,
      type: 'line',
      data: mmrData,
      smooth: false,
      yAxisIndex: 1,
      lineStyle: { width: 2, type: 'dotted', color: seriesColor(4) },
      itemStyle: { color: seriesColor(4) },
      symbol: 'diamond',
      symbolSize: 7,
    })
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: 40, left: 8, right: hasMmr ? 60 : 24, containLabel: true },
    tooltip: { ...getTooltipBase(tc), trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { ...getLegendBase(tc), data: legendData },
    xAxis: {
      ...axis,
      type: 'category',
      data: labels,
      axisLabel: { ...axis.axisLabel, rotate: labels.length > 8 ? 30 : 0 },
    },
    yAxis,
    series: echSeries,
  }
}
