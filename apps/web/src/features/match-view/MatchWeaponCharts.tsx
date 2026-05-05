/**
 * MatchWeaponCharts — chart armes de l'onglet Résumé.
 *
 * C10 : MatchWeaponPieChart — match_view.05 : Pie/donut frags par arme
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  getEChartsThemeColors,
  getTooltipBase,
  getLegendBase,
  CHART_BG,
  seriesColor,
} from '@/components/charts/_utils'
import type { MatchWeaponKill } from '@/lib/api/types'
import type { MatchViewText } from './i18n'

const DUMMY_SERIES: ChartSeries<number>[] = [{ key: 'data', datapoints: [1] }]
const EMPTY_SERIES: ChartSeries<number>[] = []

interface WeaponChartsProps {
  weaponKills: MatchWeaponKill[]
  t: MatchViewText
}

// ---------------------------------------------------------------------------
// C10 — MatchWeaponPieChart (match_view.05)
// ---------------------------------------------------------------------------

export function MatchWeaponPieChart({ weaponKills, t }: WeaponChartsProps) {
  const hasData = weaponKills.length > 0

  const buildOption = useCallback(
    (_s: ChartSeries<unknown>[]): EChartsCoreOption => {
      const tc = getEChartsThemeColors()
      const data = [...weaponKills]
        .sort((a, b) => b.kill_count - a.kill_count)
        .map((w, i) => ({
          name: w.weapon_label,
          value: w.kill_count,
          itemStyle: { color: seriesColor(i) },
        }))

      return {
        backgroundColor: CHART_BG,
        tooltip: {
          ...getTooltipBase(tc),
          trigger: 'item',
          formatter: '{b} : <b>{c}</b> ({d}%)',
        },
        legend: {
          ...getLegendBase(tc),
          type: 'scroll' as const,
          orient: 'vertical' as const,
          right: 16,
          top: 'middle',
          data: data.map((d) => d.name),
        },
        series: [
          {
            type: 'pie',
            radius: ['32%', '64%'],
            center: ['28%', '50%'],
            data,
            label: { show: false },
            emphasis: {
              itemStyle: { shadowBlur: 8, shadowOffsetX: 0, shadowColor: 'rgba(0,0,0,0.4)' },
            },
          },
        ],
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [weaponKills],
  )

  return (
    <ChartCard
      title={t.chartWeaponPieTitle}
      series={hasData ? DUMMY_SERIES : EMPTY_SERIES}
      buildOption={buildOption}
      height={280}
    />
  )
}
