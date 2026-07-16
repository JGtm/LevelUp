/**
 * ExplorerActivityHeatmapChart — heatmap 2D heure × jour de l'activité commune
 * avec un joueur cible (Explorer mode Joueur).
 *
 * Variante intensité de SynthesisHeatmapChart : la couleur reflète le `count`
 * (nombre de matchs croisés) via la rampe NEUTRE de fréquence (mono-teinte,
 * luminance monotone, CVD-safe), pas le win-rate — l'intention produit est
 * « quand se croise-t-on le plus ? ». Le tooltip rappelle le win-rate à titre
 * informatif.
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
          return `${DOW_LABELS[d]} ${HOUR_LABELS[h]}<br>${txt.noCommonMatch}`
        }
        const wrStr = `${(params.data.win_rate * 100).toFixed(1)}%`
        return `${DOW_LABELS[d]} ${HOUR_LABELS[h]}<br>${txt.commonMatches} : ${count}<br>${txt.winRate} : ${wrStr}`
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
      max: maxCount,
      calculable: false,
      show: true,
      orient: 'vertical',
      right: 30,
      top: 'center',
      itemWidth: 12,
      itemHeight: 140,
      // Rampe NEUTRE de fréquence (mono-teinte, luminance monotone, CVD-safe) :
      // intensité de rencontre, pas une perf → rampe centralisée
      // (cf. components/charts/heatmapColors).
      inRange: { color: heatmapRampTokens('frequency').map(resolveToken) },
      formatter: (val: number) => `${Math.round(val)}`,
      text: [txt.matches, ''],
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
