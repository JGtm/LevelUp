/**
 * RelationsMomentsHeatmap — heatmap agrégé « Quand tu les croises » (Phase 3a).
 *
 * Générique : relation × bucket (tranche horaire OU jour de semaine selon le
 * toggle du parent, #8). La couleur reflète le `count` (matchs communs) via la
 * rampe sémantique heatmap-cold → heatmap-hot. Axe X = buckets (labels fournis),
 * axe Y = top-N relations. Aligné sur ExplorerActivityHeatmapChart : même
 * visualMap cold→hot, count dans la cellule, tooltip, légende. Strings via
 * palmares.toml (FR/EN).
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'

/** Cellule générique : bucket = index de tranche horaire OU de jour de semaine. */
export interface HeatmapBucketCell {
  xuid: string
  gamertag: string
  bucket: number
  count: number
}

interface Props {
  cells: HeatmapBucketCell[]
  bucketLabels: string[] // index = bucket (tranche horaire ou jour)
  title?: string
  legendLabel: string
  emptyMessage: string
  matchesLabel: (count: number) => string
  height?: number
}

function buildOption(
  cells: HeatmapBucketCell[],
  bucketLabels: string[],
  legendLabel: string,
  matchesLabel: (count: number) => string,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()

  // Ordre Y : relations triées par total décroissant (les plus actives en haut).
  const totals = new Map<string, number>()
  for (const c of cells) {
    totals.set(c.xuid, (totals.get(c.xuid) ?? 0) + c.count)
  }
  const rowOrder = [...totals.entries()].sort((a, b) => b[1] - a[1]).map(([xuid]) => xuid)
  const xuidToGamertag = new Map(cells.map((c) => [c.xuid, c.gamertag]))
  const rowLabels = rowOrder.map((x) => xuidToGamertag.get(x) ?? '')
  const rowIndex = new Map(rowOrder.map((x, i) => [x, i]))

  const lookup = new Map<string, number>()
  let maxCount = 0
  for (const c of cells) {
    lookup.set(`${c.xuid}-${c.bucket}`, c.count)
    if (c.count > maxCount) maxCount = c.count
  }

  const data: { value: [number, number, number | null] }[] = []
  for (let b = 0; b < bucketLabels.length; b++) {
    for (const xuid of rowOrder) {
      const count = lookup.get(`${xuid}-${b}`)
      data.push({ value: [b, rowIndex.get(xuid) ?? 0, count != null && count > 0 ? count : null] })
    }
  }

  const hasData = maxCount > 0

  return {
    backgroundColor: CHART_BG,
    grid: { left: 110, right: 20, top: 30, bottom: 40, containLabel: false },
    tooltip: {
      trigger: 'item',
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text },
      formatter: (params: { data: { value: [number, number, number | null] } }) => {
        const [b, row, count] = params.data.value
        const who = rowLabels[row] ?? ''
        const when = bucketLabels[b] ?? ''
        if (count == null || count === 0) return `${who} · ${when}`
        return `${who} · ${when}<br>${matchesLabel(count)}`
      },
    },
    legend: false as unknown as undefined,
    xAxis: {
      type: 'category',
      data: bucketLabels,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: rowLabels,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
    },
    visualMap: hasData
      ? {
          min: 0,
          max: maxCount,
          calculable: false,
          show: false,
          orient: 'vertical',
          right: 30,
          top: 'center',
          itemWidth: 12,
          itemHeight: 140,
          inRange: { color: [resolveToken('heatmap-cold'), resolveToken('heatmap-hot')] },
          formatter: (val: number) => `${Math.round(val)}`,
          text: [legendLabel, ''],
          textStyle: { color: tc.axisLabel, fontSize: 10 },
        }
      : undefined,
    series: [
      {
        type: 'heatmap',
        data,
        label: {
          show: true,
          fontSize: 11,
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

type Pt = { xuid: string; bucket: number }

export function RelationsMomentsHeatmap({
  cells,
  bucketLabels,
  title,
  legendLabel,
  emptyMessage,
  matchesLabel,
  height,
}: Props) {
  const series: ChartSeries<Pt>[] =
    cells.length > 0
      ? [{ key: 'heatmap', datapoints: cells.map((c) => ({ xuid: c.xuid, bucket: c.bucket })) }]
      : []

  const cellsKey = useMemo(() => JSON.stringify(cells), [cells])
  const bucketsKey = useMemo(() => bucketLabels.join('|'), [bucketLabels])
  const build = useCallback(
    () => buildOption(cells, bucketLabels, legendLabel, matchesLabel),
    // cells/bucketLabels capturés via clés stables (cellsKey/bucketsKey).
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [cellsKey, bucketsKey, legendLabel, matchesLabel],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={build as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      emptyMessage={emptyMessage}
      height={height ?? 320}
    />
  )
}
