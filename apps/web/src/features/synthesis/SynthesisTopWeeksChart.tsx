/**
 * SynthesisTopWeeksChart — synthesis.04.
 * Stacked bar (victoires / autres) + line win_rate% sur axe Y secondaire.
 * Couleurs : victoires=outcome-win, autres=gris neutre, line=outcome-draw (ambre).
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { TopWeekItem } from '@/lib/api/types'

interface Props {
  weeks: TopWeekItem[]
  title?: string
  height?: number
}

function buildTopWeeksOption(weeks: TopWeekItem[]): EChartsCoreOption {
  if (weeks.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  // Trier chronologiquement par week_start (ISO YYYY-MM-DD).
  // week_label "DD/MM" perd l'année et casse l'ordre lexico aux frontières mois/an.
  const sorted = [...weeks].sort((a, b) => a.week_start.localeCompare(b.week_start))
  const labels = sorted.map((w) => w.week_label)
  const wins = sorted.map((w) => ({ value: w.wins ?? 0 }))
  const others = sorted.map((w) => ({ value: Math.max(0, w.match_count - (w.wins ?? 0)) }))
  const rates = sorted.map((w) => +w.win_rate.toFixed(1))

  return {
    backgroundColor: CHART_BG,
    grid: { left: 40, right: 60, top: 30, bottom: 80, containLabel: false },
    tooltip: { ...getTooltipBase(tc), trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { ...getLegendBase(tc), orient: 'horizontal', bottom: 5, left: 'center', top: undefined },
    xAxis: {
      ...axis,
      type: 'category',
      data: labels,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { ...axis.axisLabel },
    },
    yAxis: [
      {
        ...axis,
        type: 'value',
        splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      },
      {
        ...axis,
        name: 'Win %',
        min: 0,
        max: 100,
        splitLine: { show: false },
        position: 'right',
        axisLabel: { ...axis.axisLabel },
      },
    ],
    series: [
      {
        name: 'Top 1',
        type: 'bar',
        stack: 'total',
        data: wins,
        itemStyle: { color: resolveToken('outcome-win'), opacity: 0.85 },
        label: {
          show: true,
          position: 'inside',
          formatter: (p: { value: number }) => String(p.value),
        },
      },
      {
        name: 'Autres',
        type: 'bar',
        stack: 'total',
        data: others,
        itemStyle: { color: tc.axisLine, opacity: 0.55 },
        label: {
          show: true,
          position: 'inside',
          formatter: (p: { value: number }) => p.value > 0 ? String(p.value) : '',
        },
      },
      {
        name: 'Taux Top (%)',
        type: 'line',
        yAxisIndex: 1,
        data: rates,
        itemStyle: { color: resolveToken('outcome-draw') },
        lineStyle: { color: resolveToken('outcome-draw'), width: 2 },
        symbol: 'circle',
        symbolSize: 6,
        smooth: false,
      },
    ],
  }
}

type Pt = { week_label: string }

export function SynthesisTopWeeksChart({ weeks, title, height }: Props) {
  const series: ChartSeries<Pt>[] = weeks.length > 0
    ? [{ key: 'topweeks', datapoints: weeks.map((w) => ({ week_label: w.week_label })) }]
    : []

  // Clé de recalcul stable : on ne rebuild l'option que si le contenu change.
  const weeksKey = JSON.stringify(weeks)
  const buildOption = useCallback(
    () => buildTopWeeksOption(weeks),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [weeksKey],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      height={height ?? 360}
    />
  )
}
