/**
 * MatchWeaponCharts — charts armes de l'onglet Résumé (côte à côte).
 *
 * C10 : MatchWeaponPieChart  — match_view.05 : Pie frags par arme
 * C11 : MatchWeaponTable     — match_view.06 : Tableau frags par arme
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
          right: 4,
          top: 'middle',
          data: data.map((d) => d.name),
        },
        series: [
          {
            type: 'pie',
            radius: ['28%', '62%'],
            center: ['38%', '50%'],
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
      height={240}
    />
  )
}

// ---------------------------------------------------------------------------
// C11 — MatchWeaponTable (match_view.06)
// ---------------------------------------------------------------------------

export function MatchWeaponTable({ weaponKills, t }: WeaponChartsProps) {
  const sorted = [...weaponKills].sort((a, b) => b.kill_count - a.kill_count)
  const total = sorted.reduce((s, w) => s + w.kill_count, 0)

  return (
    <Card>
      <CardHeader className="pb-2 pt-4 px-4">
        <CardTitle className="text-sm font-medium">{t.chartWeaponTableTitle}</CardTitle>
      </CardHeader>
      <CardContent className="px-4 pb-4">
        {sorted.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            {t.noHistData}
          </p>
        ) : (
          <div className="max-h-[196px] overflow-y-auto">
            <table className="w-full text-sm">
              <tbody>
                {sorted.map((w, i) => (
                  <tr key={w.weapon_id} className="border-b border-border/40 last:border-0">
                    <td className="py-1.5 pr-2 text-muted-foreground tabular-nums text-right w-5">
                      {i + 1}
                    </td>
                    <td className="py-1.5 px-2">{w.weapon_label}</td>
                    <td className="py-1.5 pl-2 text-right tabular-nums font-medium">
                      {w.kill_count}
                    </td>
                    <td className="py-1.5 pl-3 text-right text-muted-foreground tabular-nums w-10">
                      {total > 0 ? `${Math.round((w.kill_count / total) * 100)}%` : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
