import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { SynthesisWeaponAccuracyEntry } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  weapons?: SynthesisWeaponAccuracyEntry[]
  height?: number
  fillHeight?: boolean
}

interface AccuracyPoint {
  label: string
  /** Précision en pourcentage (0..100) — l'API fournit 0..1, converti ici. */
  accuracyPct: number
  shotsFired: number
  shotsLanded: number
}

function buildWeaponAccuracyOption(series: ChartSeries<AccuracyPoint>[]): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  // L'axe catégoriel ECharts empile de bas en haut → reverse pour afficher la
  // meilleure précision EN HAUT (les datapoints arrivent triés desc côté Go).
  const data = [...(series[0]?.datapoints ?? [])].reverse()
  const color = resolveToken('chart-series-1')
  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 8, left: 8, right: 64, containLabel: true },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: { name: string; value: number; dataIndex: number }[]) => {
        const p = params[0]
        const d = data[p.dataIndex]
        return `${p.name}<br/><b>${p.value.toFixed(1)} %</b><br/>${d.shotsLanded.toLocaleString('fr-FR')} / ${d.shotsFired.toLocaleString('fr-FR')} tirs`
      },
    },
    xAxis: { type: 'value', show: false, max: 100 },
    yAxis: {
      type: 'category',
      data: data.map((d) => d.label),
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
      axisTick: { show: false },
      axisLine: { show: false },
    },
    series: [{
      type: 'bar',
      data: data.map((d) => d.accuracyPct),
      itemStyle: { color, borderRadius: [0, 3, 3, 0] },
      barMaxWidth: 20,
      label: {
        show: true,
        position: 'right',
        color: tc.axisLabel,
        fontSize: 11,
        formatter: (p: { value: number }) => `${p.value.toFixed(1)} %`,
      },
    }],
  }
}

export function SynthesisWeaponAccuracyChart({ weapons, height, fillHeight }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const title = formatMessage(synthesisManifest, 'synthesis.charts.weapon_accuracy_title', locale)
  const emptyMessage = formatMessage(synthesisManifest, 'synthesis.empty.no_data', locale)
  const list = weapons ?? []
  // Série VIDE (et non datapoints vides) quand aucune arme → ChartCard détecte
  // isEmpty (series.length === 0) et rend le placeholder + emptyMessage.
  const series: ChartSeries<AccuracyPoint>[] =
    list.length > 0
      ? [{
          key: 'weapon-accuracy',
          datapoints: list.map((w) => ({
            label: w.label,
            accuracyPct: w.accuracy * 100,
            shotsFired: w.shots_fired,
            shotsLanded: w.shots_landed,
          })),
        }]
      : []

  const buildOption = useCallback(
    (s: ChartSeries<AccuracyPoint>[]) => buildWeaponAccuracyOption(s),
    []
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
