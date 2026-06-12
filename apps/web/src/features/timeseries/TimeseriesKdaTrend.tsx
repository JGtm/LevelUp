/**
 * TimeseriesKdaTrend — chart timeseries.02 (KPIs : "Évolution Frags / Morts")
 *
 * Frags + Morts en barres groupées par match (étiquettes X `#N\nMap`).
 * La courbe FDA Y2 a été retirée (chart "FDA" dédié sur la même page).
 */
import { useMemo, useState, type ReactNode } from 'react'
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
import { ChartFromOption } from './ChartFromOption'

export interface TimeseriesKdaTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  labels: {
    kills: string
    deaths: string
    yAxis: string
  }
}

export function TimeseriesKdaTrend({ rows, height = 360, title, emptyMessage, labels }: TimeseriesKdaTrendProps) {
  const themeVersion = useThemeVersion()
  const [showBonus, setShowBonus] = useState(false)

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const colKills = resolveToken('chart-series-1')
    const colDeaths = resolveToken('outcome-loss')
    const colBonus = resolveToken('chart-series-5')

    const categories = buildMatchCategories(rows)
    const kills = rows.map((r) => r.kills)
    const deaths = rows.map((r) => r.deaths)
    const bonus = rows.map((r) => r.assists / 3)

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
        data: [labels.kills, labels.deaths, 'Bonus'],
        selected: { Bonus: showBonus },
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
          stack: 'kills',
          data: kills,
          itemStyle: { color: colKills, opacity: 0.85 },
          barGap: 0,
          barMaxWidth: 14,
        },
        {
          name: 'Bonus',
          type: 'bar',
          stack: 'kills',
          data: bonus,
          itemStyle: { color: colBonus, opacity: 0.85 },
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
  }, [rows, labels, themeVersion, showBonus])

  const onEvents = useMemo(
    () => ({
      legendselectchanged: (params: unknown) => {
        const p = params as { selected: Record<string, boolean> }
        setShowBonus(p.selected['Bonus'] ?? false)
      },
    }),
    [],
  )

  return (
    <ChartFromOption
      title={title}
      option={option}
      height={height}
      emptyMessage={emptyMessage}
      onEvents={onEvents}
    />
  )
}
