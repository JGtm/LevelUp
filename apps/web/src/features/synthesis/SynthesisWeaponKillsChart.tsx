import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { SynthesisWeaponKillEntry } from '@/lib/api/types'

interface Props {
  weapons: SynthesisWeaponKillEntry[]
  height?: number
  fillHeight?: boolean
}

interface WeaponPoint {
  label: string
  kills: number
}

function buildWeaponKillsOption(series: ChartSeries<WeaponPoint>[]): EChartsCoreOption {
  const tc = getEChartsThemeColors()
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
        return `${p.name}<br/><b>${p.value.toLocaleString('fr-FR')}</b> frags`
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
        formatter: (p: { value: number }) => p.value.toLocaleString('fr-FR'),
      },
    }],
  }
}

export function SynthesisWeaponKillsChart({ weapons, height, fillHeight }: Props) {
  const series: ChartSeries<WeaponPoint>[] = [{
    key: 'weapon-kills',
    datapoints: weapons.map((w) => ({ label: w.label, kills: w.kills })),
  }]

  const buildOption = useCallback(
    (s: ChartSeries<WeaponPoint>[]) => buildWeaponKillsOption(s),
    []
  )

  const computedHeight = height ?? Math.max(180, weapons.length * 28 + 16)

  if (weapons.length === 0) return null

  return (
    <ChartCard
      title="Frags par arme"
      series={series}
      buildOption={buildOption}
      height={computedHeight}
      className={fillHeight ? 'flex-1' : ''}
    />
  )
}
