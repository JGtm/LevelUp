/**
 * TimeseriesKdaTrend — chart timeseries.02 (KPIs : "Évolution Frags / Morts")
 *
 * Frags + Morts en barres groupées par match (étiquettes X `#N\nMap`).
 * La courbe FDA Y2 a été retirée (chart "FDA" dédié sur la même page).
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
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { buildMatchCategories } from './matchLabels'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export interface TimeseriesKdaTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  labels: {
    kills: string
    deaths: string
    yAxis: string
  }
}

export function TimeseriesKdaTrend({ rows, height = 360, labels }: TimeseriesKdaTrendProps) {
  const themeVersion = useThemeVersion()


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const colKills = resolveToken('chart-series-1')
    const colDeaths = resolveToken('outcome-loss')

    const categories = buildMatchCategories(rows)
    const kills = rows.map((r) => r.kills)
    const deaths = rows.map((r) => r.deaths)

    return {
      backgroundColor: CHART_BG,
      grid: { top: 32, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      legend: {
        ...getLegendBase(tc),
        bottom: 0,
        data: [labels.kills, labels.deaths],
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: {
          ...getAxisBase(tc).axisLabel,
          interval: 0,
          fontSize: 9,
        },
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
        name: labels.yAxis,
        nameTextStyle: { color: tc.axisLabel, fontSize: 11 },
        minInterval: 1,
      },
      series: [
        {
          name: labels.kills,
          type: 'bar',
          data: kills,
          itemStyle: { color: colKills, opacity: 0.85 },
          barGap: 0,
          barMaxWidth: 14,
        },
        {
          name: labels.deaths,
          type: 'bar',
          data: deaths,
          itemStyle: { color: colDeaths, opacity: 0.85 },
          barMaxWidth: 14,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, labels, themeVersion])

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
