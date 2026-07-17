/**
 * SynthesisHeatmapChart — synthesis.03.
 * Heatmap 2D heure × jour (X=heure, Y=jour) colorée par win_rate via la rampe
 * DIVERGENTE centralisée (perdant → neutre 50 % → gagnant) : le win_rate est un
 * indicateur signé autour de 0,5, et la rampe divergente est CVD-safe par
 * construction — neutre gris en palette daltonienne (cf. heatmapColors).
 * Toutes les 168 cellules sont émises — null pour les cases vides.
 * yAxis.inverse: true (Lun en haut, Dim en bas).
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { dowLabels, HOUR_LABELS, calendarChartText } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { HeatmapCell } from '@/lib/api/types'

interface Props {
  cells: HeatmapCell[]
  title?: string
  height?: number
}

function buildHeatmapOption(cells: HeatmapCell[], locale: ManifestLocale): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const DOW_LABELS = dowLabels(locale)
  const txt = calendarChartText(locale)

  // Indexer les données reçues par (dow, hour)
  const lookup = new Map<string, { win_rate: number; count: number }>()
  for (const c of cells) {
    if (c.count > 0) {
      lookup.set(`${c.dow}-${c.hour}`, {
        win_rate: c.win_rate ?? 0,
        count: c.count,
      })
    }
  }

  // Générer les 168 cellules — null pour win_rate si aucun match
  const data: { value: [number, number, number | null]; count: number }[] = []
  for (let h = 0; h < 24; h++) {
    for (let d = 0; d < 7; d++) {
      const cell = lookup.get(`${d}-${h}`)
      data.push({
        value: [h, d, cell ? cell.win_rate : null],
        count: cell ? cell.count : 0,
      })
    }
  }

  const hasData = cells.length > 0

  return {
    backgroundColor: CHART_BG,
    grid: { left: 60, right: 130, top: 30, bottom: 40, containLabel: false },
    tooltip: {
      trigger: 'item',
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text },
      formatter: (params: { data: { value: [number, number, number | null]; count: number } }) => {
        const [h, d, wr] = params.data.value
        const wrStr = wr == null ? 'n/a' : `${(wr * 100).toFixed(1)}%`
        return `${DOW_LABELS[d]} ${HOUR_LABELS[h]}<br>${txt.winRate} : ${wrStr}<br>${txt.matches} : ${params.data.count}`
      },
    },
    legend: false as unknown as undefined,
    xAxis: {
      type: 'category',
      name: txt.hourAxis,
      data: HOUR_LABELS,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { color: tc.axisLabel, fontSize: 10 },
    },
    yAxis: {
      type: 'category',
      name: txt.dayAxis,
      inverse: true,
      data: DOW_LABELS,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { color: tc.axisLabel },
    },
    visualMap: hasData ? {
      min: 0,
      max: 1,
      calculable: false,
      show: true,
      orient: 'vertical',
      right: 30,
      top: 'center',
      itemWidth: 12,
      itemHeight: 140,
      // Rampe DIVERGENTE centralisée (bas → neutre → haut) : le win_rate est un
      // indicateur signé autour de 0,5 → neutre gris en CVD (cf. heatmapColors).
      inRange: { color: heatmapRampTokens('divergent').map(resolveToken) },
      formatter: (val: number) => `${(val * 100).toFixed(0)}%`,
      text: [txt.wins, ''],
      textStyle: { color: tc.axisLabel, fontSize: 10 },
    } : undefined,
    series: [
      {
        type: 'heatmap',
        data,
        label: {
          show: true,
          fontSize: 10,
          color: tc.text,
          formatter: (params: { data: { count: number } }) =>
            params.data.count > 0 ? String(params.data.count) : '',
        },
        emphasis: { itemStyle: { shadowBlur: 8 } },
      },
    ],
  }
}

type Pt = { dow: number; hour: number }

export function SynthesisHeatmapChart({ cells, title, height }: Props) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const series: ChartSeries<Pt>[] = cells.length > 0
    ? [{ key: 'heatmap', datapoints: cells.map((c) => ({ dow: c.dow, hour: c.hour })) }]
    : []

  const cellsKey = useMemo(() => JSON.stringify(cells), [cells])
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const buildOption = useCallback(() => buildHeatmapOption(cells, locale), [cellsKey, locale])

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      height={height ?? 300}
    />
  )
}
