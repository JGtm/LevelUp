import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { SynthesisWeaponKillEntry } from '@/lib/api/types'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  weapons?: SynthesisWeaponKillEntry[]
  height?: number
  fillHeight?: boolean
}

interface WeaponPoint {
  label: string
  kills: number
}

function buildWeaponKillsOption(series: ChartSeries<WeaponPoint>[], locale: ManifestLocale): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const numLoc = intlLocale(locale)
  const data = [...(series[0]?.datapoints ?? [])].reverse()
  const color = resolveToken('chart-series-1')
  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 8, left: 8, right: 80, containLabel: true },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: { name: string; value: number }[]) => {
        const p = params[0]
        return `${escapeHtml(p.name ?? '')}<br/><b>${p.value.toLocaleString(numLoc)}</b> frags`
      },
    },
    xAxis: { type: 'value', show: false },
    yAxis: {
      type: 'category',
      data: data.map((d) => d.label),
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
      axisTick: { show: false },
      axisLine: { show: false },
    },
    series: [{
      type: 'bar',
      data: data.map((d) => d.kills),
      itemStyle: { color, borderRadius: [0, 3, 3, 0] },
      barMaxWidth: 20,
      label: {
        show: true,
        position: 'right',
        color: tc.axisLabel,
        fontSize: 11,
        formatter: (p: { value: number }) => p.value.toLocaleString(numLoc),
      },
    }],
  }
}

export function SynthesisWeaponKillsChart({ weapons, height, fillHeight }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const title = formatMessage(synthesisManifest, 'synthesis.charts.weapon_kills_title', locale)
  const emptyMessage = formatMessage(synthesisManifest, 'synthesis.empty.no_data', locale)
  const list = weapons ?? []
  // Série VIDE (et non datapoints vides dans une série) quand aucune arme → ChartCard
  // détecte isEmpty (series.length === 0) et rend le placeholder + emptyMessage, au lieu
  // de `return null` qui n'affichait RIEN (pas même un placeholder, cf. timeseries).
  const series: ChartSeries<WeaponPoint>[] =
    list.length > 0
      ? [{ key: 'weapon-kills', datapoints: list.map((w) => ({ label: w.label, kills: w.kills })) }]
      : []

  const buildOption = useCallback(
    (s: ChartSeries<WeaponPoint>[]) => buildWeaponKillsOption(s, locale),
    [locale]
  )

  const computedHeight = height ?? Math.max(180, list.length * 28 + 16)

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={computedHeight}
      emptyMessage={emptyMessage}
      className={fillHeight ? 'flex-1' : ''}
    />
  )
}
