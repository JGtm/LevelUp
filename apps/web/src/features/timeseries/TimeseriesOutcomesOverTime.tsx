/**
 * TimeseriesOutcomesOverTime — chart timeseries.05
 *
 * Stacked bars V/D/N/X agrégés par période (jour/semaine/mois) côté Go.
 * Couleurs par token sémantique outcome-{win,loss,draw,dnf}.
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
import type { OutcomesPeriodPoint } from '@/lib/api/types'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export interface TimeseriesOutcomesOverTimeProps {
  points: OutcomesPeriodPoint[]
  height?: number
  labels: {
    win: string
    loss: string
    tie: string
    dnf: string
  }
}

export function TimeseriesOutcomesOverTime({
  points,
  height = 360,
  labels,
}: TimeseriesOutcomesOverTimeProps) {
  const themeVersion = useThemeVersion()


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (points.length === 0) return null
    const tc = getEChartsThemeColors()
    const colWin = resolveToken('outcome-win')
    const colLoss = resolveToken('outcome-loss')
    const colTie = resolveToken('outcome-draw')
    const colDnf = resolveToken('outcome-dnf')

    const categories = points.map((p) => p.period_label)
    const wins = points.map((p) => p.wins)
    const losses = points.map((p) => p.losses)
    const ties = points.map((p) => p.ties)
    const dnfs = points.map((p) => p.dnf)

    const seriesBase = {
      type: 'bar' as const,
      stack: 'outcome',
      label: {
        show: false,
      },
    }

    return {
      backgroundColor: CHART_BG,
      grid: { top: 32, right: 16, bottom: 56, left: 40 },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      legend: {
        ...getLegendBase(tc),
        bottom: 4,
        data: [labels.win, labels.loss, labels.tie, labels.dnf],
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: {
          ...getAxisBase(tc).axisLabel,
          rotate: categories.length > 12 ? 30 : 0,
        },
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
      },
      series: [
        {
          ...seriesBase,
          name: labels.win,
          data: wins,
          itemStyle: { color: colWin },
        },
        {
          ...seriesBase,
          name: labels.loss,
          data: losses,
          itemStyle: { color: colLoss },
        },
        {
          ...seriesBase,
          name: labels.tie,
          data: ties,
          itemStyle: { color: colTie },
        },
        {
          ...seriesBase,
          name: labels.dnf,
          data: dnfs,
          itemStyle: { color: colDnf },
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [points, labels, themeVersion])

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
