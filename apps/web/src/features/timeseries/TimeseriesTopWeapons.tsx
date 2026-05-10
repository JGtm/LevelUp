/**
 * TimeseriesTopWeapons — chart timeseries.04
 *
 * Bar chart horizontal des armes triées par kills (top N déjà filtré côté Go).
 * Catégorie sur Y, valeur sur X.
 */
import { Suspense, lazy, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesWeaponKill } from '@/lib/api/types'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export interface TimeseriesTopWeaponsProps {
  weapons: TimeseriesWeaponKill[]
  height?: number
  labels: {
    seriesName: string
    fallbackLabel: (id: number) => string
  }
}

export function TimeseriesTopWeapons({
  weapons,
  height = 360,
  labels,
}: TimeseriesTopWeaponsProps) {
  const themeVersion = useThemeVersion()


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (weapons.length === 0) return null
    const tc = getEChartsThemeColors()
    const accent = resolveToken('chart-series-1')

    // ECharts catégoriel ASC affiche du bas vers le haut → on inverse pour avoir
    // le top arme en haut.
    const reversed = [...weapons].reverse()
    const categories = reversed.map((w) => w.label || labels.fallbackLabel(w.weapon_id))
    const values = reversed.map((w) => w.kills)

    return {
      backgroundColor: CHART_BG,
      grid: { top: 12, right: 32, bottom: 24, left: 132 },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'value',
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, width: 120, overflow: 'truncate' },
      },
      series: [
        {
          name: labels.seriesName,
          type: 'bar',
          data: values,
          itemStyle: { color: accent, opacity: 0.85 },
          label: {
            show: true,
            position: 'right',
            color: tc.axisLabel,
            fontSize: 11,
            formatter: (p: { value: number }) => String(p.value),
          },
          barWidth: '60%',
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [weapons, labels, themeVersion])

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
