/**
 * SynthesisHeatmapChart — synthesis.03.
 * Heatmap 2D heure × jour (X=heure, Y=jour) colorée par win_rate.
 * Toutes les 168 cellules sont émises — null pour les cases vides.
 * yAxis.inverse: true (Lun en haut, Dim en bas).
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { HeatmapCell } from '@/lib/api/types'

const DOW_LABELS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']
const HOUR_LABELS = ['00h','01h','02h','03h','04h','05h','06h','07h','08h','09h',
  '10h','11h','12h','13h','14h','15h','16h','17h','18h','19h','20h','21h','22h','23h']

interface Props {
  cells: HeatmapCell[]
  title?: string
  height?: number
}

function buildHeatmapOption(cells: HeatmapCell[]): EChartsCoreOption {
  const tc = getEChartsThemeColors()

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
    grid: { left: 60, right: 20, top: 30, bottom: 40, containLabel: false },
    tooltip: {
      trigger: 'item',
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text },
      formatter: (params: { data: { value: [number, number, number | null]; count: number } }) => {
        const [h, d, wr] = params.data.value
        const wrStr = wr == null ? 'n/a' : `${(wr * 100).toFixed(1)}%`
        return `${DOW_LABELS[d]} ${HOUR_LABELS[h]}<br>Win rate : ${wrStr}<br>Matchs : ${params.data.count}`
      },
    },
    legend: false as unknown as undefined,
    xAxis: {
      type: 'category',
      name: 'Heure',
      data: HOUR_LABELS,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { color: tc.axisLabel, fontSize: 10 },
    },
    yAxis: {
      type: 'category',
      name: 'Jour',
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
      right: 10,
      top: 'center',
      inRange: {
        color: [
          resolveToken('outcome-loss'),
          resolveToken('outcome-draw'),
          resolveToken('outcome-win'),
        ],
      },
      formatter: (val: number) => `${(val * 100).toFixed(0)}%`,
      text: ['Win rate', ''],
      textStyle: { color: tc.axisLabel },
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
  const series: ChartSeries<Pt>[] = cells.length > 0
    ? [{ key: 'heatmap', datapoints: cells.map((c) => ({ dow: c.dow, hour: c.hour })) }]
    : []

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const buildOption = useCallback(() => buildHeatmapOption(cells), [JSON.stringify(cells)])

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      height={height ?? 300}
    />
  )
}
