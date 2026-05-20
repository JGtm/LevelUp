/**
 * ExplorerActivityHeatmapChart — heatmap 2D heure × jour de l'activité commune
 * avec un joueur cible (Explorer mode Joueur).
 *
 * Variante intensité de SynthesisHeatmapChart : la couleur reflète le `count`
 * (nombre de matchs croisés) via la rampe sémantique heatmap-cold → heatmap-hot,
 * pas le win-rate — l'intention produit est « quand se croise-t-on le plus ? ».
 * Le tooltip rappelle le win-rate à titre informatif.
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useFieldLabel } from '@/lib/i18n/fieldMappings'
import type { HeatmapCell } from '@/lib/api/types'

const DOW_LABELS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']
const HOUR_LABELS = ['00h','01h','02h','03h','04h','05h','06h','07h','08h','09h',
  '10h','11h','12h','13h','14h','15h','16h','17h','18h','19h','20h','21h','22h','23h']

interface Props {
  cells: HeatmapCell[]
  title?: string
  height?: number
}

function buildHeatmapOption(cells: HeatmapCell[], matchesLabel: string): EChartsCoreOption {
  const tc = getEChartsThemeColors()

  const lookup = new Map<string, { count: number; win_rate: number }>()
  let maxCount = 0
  for (const c of cells) {
    if (c.count > 0) {
      lookup.set(`${c.dow}-${c.hour}`, {
        count: c.count,
        win_rate: c.win_rate ?? 0,
      })
      if (c.count > maxCount) maxCount = c.count
    }
  }

  // 168 cellules — null pour count si aucun match (cellule vide non colorée).
  const data: { value: [number, number, number | null]; win_rate: number }[] = []
  for (let h = 0; h < 24; h++) {
    for (let d = 0; d < 7; d++) {
      const cell = lookup.get(`${d}-${h}`)
      data.push({
        value: [h, d, cell ? cell.count : null],
        win_rate: cell ? cell.win_rate : 0,
      })
    }
  }

  const hasData = maxCount > 0

  return {
    backgroundColor: CHART_BG,
    grid: { left: 60, right: 130, top: 30, bottom: 40, containLabel: false },
    tooltip: {
      trigger: 'item',
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text },
      formatter: (params: { data: { value: [number, number, number | null]; win_rate: number } }) => {
        const [h, d, count] = params.data.value
        if (count == null || count === 0) {
          return `${DOW_LABELS[d]} ${HOUR_LABELS[h]}<br>Aucun match commun`
        }
        const wrStr = `${(params.data.win_rate * 100).toFixed(1)}%`
        return `${DOW_LABELS[d]} ${HOUR_LABELS[h]}<br>Matchs communs : ${count}<br>Taux de victoire : ${wrStr}`
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
      max: maxCount,
      calculable: false,
      show: true,
      orient: 'vertical',
      right: 30,
      top: 'center',
      itemWidth: 12,
      itemHeight: 140,
      inRange: {
        color: [
          resolveToken('heatmap-cold'),
          resolveToken('heatmap-hot'),
        ],
      },
      formatter: (val: number) => `${Math.round(val)}`,
      text: [matchesLabel, ''],
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
          formatter: (params: { data: { value: [number, number, number | null] } }) => {
            const count = params.data.value[2]
            return count != null && count > 0 ? String(count) : ''
          },
        },
        emphasis: { itemStyle: { shadowBlur: 8 } },
      },
    ],
  }
}

type Pt = { dow: number; hour: number }

export function ExplorerActivityHeatmapChart({ cells, title, height }: Props) {
  const matchesLabel = useFieldLabel('matches')

  const series: ChartSeries<Pt>[] = cells.length > 0
    ? [{ key: 'heatmap', datapoints: cells.map((c) => ({ dow: c.dow, hour: c.hour })) }]
    : []

  const cellsKey = useMemo(() => JSON.stringify(cells), [cells])
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const buildOption = useCallback(() => buildHeatmapOption(cells, matchesLabel), [cellsKey, matchesLabel])

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      height={height ?? 300}
    />
  )
}
