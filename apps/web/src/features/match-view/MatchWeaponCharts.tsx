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
  escapeHtml,
  seriesColor,
} from '@/components/charts/_utils'
import type { MatchWeaponKill } from '@/lib/api/types'
import type { MatchViewText } from './i18n'

const DUMMY_SERIES: ChartSeries<number>[] = [{ key: 'data', datapoints: [1] }]
const EMPTY_SERIES: ChartSeries<number>[] = []

// Le backend met le weapon_id décimal en label quand il n'a pas de match dans
// weapon_labels (variantes saisonnières, Forge custom, cosmétiques rares).
// 99 % des kills sont reconnus mais ces variantes peuvent introduire 50-100
// entrées en longue queue dans la légende. On les agrège en une slice unique
// "Autres armes (N)" pour garder une légende lisible.
function isUnknownWeapon(label: string): boolean {
  return /^\d+$/.test(label)
}

interface AggregatedSlice {
  name: string
  value: number
}

function aggregateWeaponKills(weaponKills: MatchWeaponKill[], t: MatchViewText): AggregatedSlice[] {
  const known: AggregatedSlice[] = []
  let unknownKills = 0
  let unknownCount = 0
  for (const w of weaponKills) {
    if (isUnknownWeapon(w.weapon_label)) {
      unknownKills += w.kill_count
      unknownCount++
    } else {
      known.push({ name: w.weapon_label, value: w.kill_count })
    }
  }
  known.sort((a, b) => b.value - a.value)
  if (unknownCount > 0) {
    known.push({ name: `${t.weaponOtherGroup} (${unknownCount})`, value: unknownKills })
  }
  return known
}

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
    (): EChartsCoreOption => {
      const tc = getEChartsThemeColors()
      const data = aggregateWeaponKills(weaponKills, t).map((s, i) => ({
        name: s.name,
        value: s.value,
        itemStyle: { color: seriesColor(i) },
      }))

      return {
        backgroundColor: CHART_BG,
        tooltip: {
          ...getTooltipBase(tc),
          trigger: 'item',
          // Formatter fonction (au lieu du template '{b} : <b>{c}</b> ({d}%)') pour
          // échapper le nom d'arme ({b}, donnée asset non constante) — {c}=value et
          // {d}=percent restent des nombres calculés par ECharts (percent = {d}).
          formatter: (p: { name?: string; value?: number; percent?: number }) =>
            `${escapeHtml(p.name ?? '')} : <b>${p.value ?? 0}</b> (${p.percent ?? 0}%)`,
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
