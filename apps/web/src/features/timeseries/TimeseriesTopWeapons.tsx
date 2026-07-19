/**
 * TimeseriesTopWeapons — chart timeseries.04
 *
 * Bar chart horizontal des armes triées par kills (top N déjà filtré côté Go).
 * Catégorie sur Y, valeur sur X.
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { fragClassColor } from '@/lib/accessibility/scales'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesWeaponKill } from '@/lib/api/types'
import { ChartFromOption } from './ChartFromOption'

export interface TimeseriesTopWeaponsProps {
  weapons: TimeseriesWeaponKill[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  labels: {
    seriesName: string
    fallbackLabel: (id: number) => string
  }
}

export function TimeseriesTopWeapons({
  weapons,
  height = 360,
  title,
  emptyMessage,
  labels,
}: TimeseriesTopWeaponsProps) {
  const themeVersion = useThemeVersion()


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (weapons.length === 0) return null
    const tc = getEChartsThemeColors()

    // ECharts catégoriel ASC affiche du bas vers le haut → on inverse pour avoir
    // le top arme en haut. Chaque barre est recolorée par la CLASSE de l'arme
    // (fragClassColor) pour la cohérence visuelle avec le sunburst v2 ; les armes
    // sans classe résolue retombent sur la teinte neutre.
    const reversed = [...weapons].reverse()
    const categories = reversed.map((w) => w.label || labels.fallbackLabel(w.weapon_id))
    const values = reversed.map((w) => ({
      value: w.kills,
      itemStyle: { color: fragClassColor(w.class) },
    }))

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

  return (
    <ChartFromOption title={title} option={option} height={height} emptyMessage={emptyMessage} />
  )
}
