import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SynthesisDetailedStats } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  stats: SynthesisDetailedStats
  height?: number
}

interface KillTypePoint {
  name: string
  value: number
  color: string
}

function buildDonutOption(series: ChartSeries<KillTypePoint>[]): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const data = series[0]?.datapoints ?? []
  return {
    backgroundColor: CHART_BG,
    tooltip: {
      trigger: 'item',
      formatter: (p: { name: string; value: number; percent: number }) =>
        `${p.name}<br/><b>${p.value.toLocaleString('fr-FR')}</b> frags (${p.percent}%)`,
    },
    series: [{
      type: 'pie',
      radius: ['42%', '68%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: true,
      label: {
        show: true,
        color: tc.axisLabel,
        fontSize: 11,
        formatter: (p: { name: string; value: number; percent: number }) =>
          `${p.name}\n${p.value.toLocaleString('fr-FR')} (${p.percent}%)`,
      },
      labelLine: { show: true, length: 10, length2: 14 },
      emphasis: { label: { fontSize: 12, fontWeight: 'bold' } },
      data: data.map((d) => ({ name: d.name, value: d.value, itemStyle: { color: d.color } })),
    }],
  }
}

export function SynthesisKillTypesDonut({ stats, height = 280 }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const title = formatMessage(synthesisManifest, 'synthesis.charts.kill_types_title', locale)
  const total =
    stats.total_headshot_kills +
    stats.total_grenade_kills +
    stats.total_melee_kills +
    stats.total_power_weapon_kills

  const series: ChartSeries<KillTypePoint>[] = [{
    key: 'kill-types',
    datapoints: [
      { name: fieldMappings?.fields['headshot_kills']?.label ?? 'Tirs à la tête', value: stats.total_headshot_kills,     color: resolveToken('chart-series-1') },
      { name: fieldMappings?.fields['grenade_kills']?.label ?? 'Grenade',         value: stats.total_grenade_kills,      color: resolveToken('chart-series-2') },
      { name: fieldMappings?.fields['melee_kills']?.label ?? 'Corps à corps',     value: stats.total_melee_kills,        color: resolveToken('chart-series-3') },
      { name: fieldMappings?.fields['power_weapon_kills']?.label ?? 'Arme lourde', value: stats.total_power_weapon_kills, color: resolveToken('chart-series-4') },
    ].filter((d) => d.value > 0),
  }]

  const buildOption = useCallback(
    (s: ChartSeries<KillTypePoint>[]) => buildDonutOption(s),
    []
  )

  if (total === 0) return null

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={height}
    />
  )
}
