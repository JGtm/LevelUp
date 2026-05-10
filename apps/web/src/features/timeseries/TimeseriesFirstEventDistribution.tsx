/**
 * TimeseriesFirstEventDistribution — chart timeseries.11
 *
 * Distribution superposée du premier kill (vert) vs première mort (rouge)
 * par bin de N secondes depuis le début du match. markLine verticale par
 * série pour la moyenne (Moy. 38s).
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
import type { FirstEventDistribution } from '@/lib/api/types'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export interface TimeseriesFirstEventDistributionProps {
  data: FirstEventDistribution
  height?: number
  killsLabel: string
  deathsLabel: string
  meanLabel: string
  xAxisLabel: string
}

export function TimeseriesFirstEventDistribution({
  data,
  height = 360,
  killsLabel,
  deathsLabel,
  meanLabel,
  xAxisLabel,
}: TimeseriesFirstEventDistributionProps) {
  const themeVersion = useThemeVersion()


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (data.buckets.length === 0) return null
    const tc = getEChartsThemeColors()
    const colKills = resolveToken('outcome-win')
    const colDeaths = resolveToken('outcome-loss')

    // Format `XmYYs` (toujours 2 chiffres pour les secondes) — ex. 0m45s, 1m25s.
    const fmtMinSec = (sec: number): string => {
      const m = Math.floor(sec / 60)
      const s = Math.round(sec) % 60
      return `${m}m${String(s).padStart(2, '0')}s`
    }
    const categories = data.buckets.map((b) => fmtMinSec(b.lower_seconds))
    const kills = data.buckets.map((b) => b.first_kills)
    const deaths = data.buckets.map((b) => b.first_deaths)

    // Position markLine sur l'axe catégorie : index du bucket contenant la
    // moyenne (avec interpolation interne).
    const meanIdx = (mean: number | null | undefined): number | null => {
      if (mean == null || !Number.isFinite(mean)) return null
      for (let i = 0; i < data.buckets.length; i++) {
        const b = data.buckets[i]
        if (mean >= b.lower_seconds && mean < b.upper_seconds) {
          const frac =
            (mean - b.lower_seconds) / (b.upper_seconds - b.lower_seconds)
          return i + frac
        }
      }
      return data.buckets.length - 1
    }
    const meanKill = data.mean_first_kill_seconds ?? null
    const meanDeath = data.mean_first_death_seconds ?? null
    const meanKillIdx = meanIdx(meanKill)
    const meanDeathIdx = meanIdx(meanDeath)

    const buildMarkLine = (xAxisIdx: number, mean: number, color: string) => ({
      silent: true,
      symbol: 'none' as const,
      lineStyle: { color, width: 1.5, type: 'dashed' as const },
      label: {
        formatter: `${meanLabel} ${fmtMinSec(mean)}`,
        color,
        fontSize: 11,
        fontWeight: 700,
        backgroundColor: tc.tooltipBg,
        padding: [2, 4],
        borderRadius: 2,
        position: 'insideEndTop' as const,
      },
      data: [{ xAxis: xAxisIdx }],
    })

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 56, left: 48 },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      legend: {
        ...getLegendBase(tc),
        bottom: 0,
        data: [killsLabel, deathsLabel],
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        name: xAxisLabel,
        nameLocation: 'middle',
        nameGap: 28,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        axisLabel: {
          ...getAxisBase(tc).axisLabel,
          interval: Math.max(0, Math.floor(categories.length / 10) - 1),
        },
      },
      yAxis: { ...getAxisBase(tc), type: 'value' },
      series: [
        {
          type: 'bar',
          name: killsLabel,
          data: kills,
          itemStyle: { color: colKills, opacity: 0.85 },
          barGap: '10%',
          markLine:
            meanKillIdx != null && meanKill != null
              ? buildMarkLine(meanKillIdx, meanKill, colKills)
              : undefined,
        },
        {
          type: 'bar',
          name: deathsLabel,
          data: deaths,
          itemStyle: { color: colDeaths, opacity: 0.85 },
          markLine:
            meanDeathIdx != null && meanDeath != null
              ? buildMarkLine(meanDeathIdx, meanDeath, colDeaths)
              : undefined,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data, killsLabel, deathsLabel, meanLabel, xAxisLabel, themeVersion])

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
