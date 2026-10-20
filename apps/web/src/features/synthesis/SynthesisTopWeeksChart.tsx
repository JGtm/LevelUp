/**
 * SynthesisTopWeeksChart — synthesis.04.
 * Stacked bar (victoires / autres) + line win_rate% sur axe Y secondaire.
 * Couleurs : victoires=outcome-win, autres=gris neutre, line=outcome-draw (ambre).
 * Noms des séries et de l'axe : manifeste `synthesis.charts.top_weeks_*` (`topWeeksNames`), à la langue de
 * l'application — aucun libellé écrit dans le constructeur d'option.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { barLabel } from '@/components/charts/barLabel'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { axisTooltipFormatter, stackTopFirstOrder } from '@/components/charts/axisTooltip'
import { tipNumber, tipNumberAuto } from '@/components/charts/tipNumber'
import { resolveToken } from '@/lib/accessibility'
import type { TopWeekItem } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  weeks: TopWeekItem[]
  title?: string
  height?: number
}

/** Les noms des trois séries (barres victoires et autres, courbe du taux) et de l'axe de droite. */
export interface TopWeeksNames {
  wins: string
  others: string
  rate: string
  rateAxis: string
}

/** topWeeksNames — les noms à la langue demandée (manifeste `synthesis.charts.top_weeks_*`). */
// eslint-disable-next-line react-refresh/only-export-components
export function topWeeksNames(locale: Locale): TopWeeksNames {
  return {
    wins: formatMessage(synthesisManifest, 'synthesis.charts.top_weeks_wins', locale),
    others: formatMessage(synthesisManifest, 'synthesis.charts.top_weeks_others', locale),
    rate: formatMessage(synthesisManifest, 'synthesis.charts.top_weeks_rate', locale),
    rateAxis: formatMessage(synthesisManifest, 'synthesis.charts.top_weeks_rate_axis', locale),
  }
}

/** Rang de la série « taux » (courbe sur l'axe de droite) dans le tableau de séries. */
const RATE_SERIES_INDEX = 2

// eslint-disable-next-line react-refresh/only-export-components
export function buildTopWeeksOption(weeks: TopWeekItem[], names: TopWeeksNames): EChartsCoreOption {
  if (weeks.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const winColor = resolveToken('outcome-win')

  // Trier chronologiquement par week_start (ISO YYYY-MM-DD).
  // week_label "DD/MM" perd l'année et casse l'ordre lexico aux frontières mois/an.
  const sorted = [...weeks].sort((a, b) => a.week_start.localeCompare(b.week_start))
  const labels = sorted.map((w) => w.week_label)
  const wins = sorted.map((w) => ({ value: w.wins ?? 0 }))
  const others = sorted.map((w) => ({ value: Math.max(0, w.match_count - (w.wins ?? 0)) }))
  const rates = sorted.map((w) => +w.win_rate.toFixed(1))

  const series = [
    {
      name: names.wins,
      type: 'bar',
      stack: 'total',
      data: wins,
      itemStyle: { color: winColor, opacity: 0.85 },
      label: barLabel({ formatter: (p: { value: number }) => tipNumberAuto(p.value) }),
    },
    {
      name: names.others,
      type: 'bar',
      stack: 'total',
      data: others,
      itemStyle: { color: tc.axisLine, opacity: 0.55 },
      label: barLabel({ formatter: (p: { value: number }) => (p.value > 0 ? tipNumberAuto(p.value) : '') }),
    },
    {
      name: names.rate,
      type: 'line',
      yAxisIndex: 1,
      data: rates,
      itemStyle: { color: resolveToken('outcome-draw') },
      lineStyle: { color: resolveToken('outcome-draw'), width: 2 },
      symbol: 'circle',
      symbolSize: 6,
      smooth: false,
    },
  ]

  return {
    backgroundColor: CHART_BG,
    grid: { left: 40, right: 60, top: 30, bottom: 80, containLabel: false },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Barres de la plus haute à la plus basse de la pile, puis la courbe du taux (axe de droite).
      formatter: axisTooltipFormatter({
        value: (r) => (r.seriesIndex === RATE_SERIES_INDEX ? `${tipNumber(r.value, 1)} %` : tipNumberAuto(r.value)),
        order: stackTopFirstOrder(series),
      }),
    },
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
        name: names.rateAxis,
        min: 0,
        max: 100,
        splitLine: { show: false },
        position: 'right',
        axisLabel: { ...axis.axisLabel },
      },
    ],
    series,
  }
}

type Pt = { week_label: string }

export function SynthesisTopWeeksChart({ weeks, title, height }: Props) {
  const locale = useAppShellStore((st) => st.locale)
  const series: ChartSeries<Pt>[] = weeks.length > 0
    ? [{ key: 'topweeks', datapoints: weeks.map((w) => ({ week_label: w.week_label })) }]
    : []

  // Clé de recalcul stable : on ne rebuild l'option que si le contenu change.
  const weeksKey = JSON.stringify(weeks)
  const buildOption = useCallback(
    () => buildTopWeeksOption(weeks, topWeeksNames(locale)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [weeksKey, locale],
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
