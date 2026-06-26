/**
 * RelationsMomentsHeatmap — heatmap agrégé « Quand tu les croises » (Phase 3a).
 *
 * Variante relation × tranche horaire de ExplorerActivityHeatmapChart : la
 * couleur reflète le `count` (matchs communs) via la rampe sémantique
 * heatmap-cold → heatmap-hot. Axe X = 6 tranches horaires (day-parts), axe Y =
 * top-N relations. Plafonné (top-N × 6 tranches) pour rester lisible.
 *
 * Aligné sur ExplorerActivityHeatmapChart : même visualMap cold→hot, count
 * affiché dans la cellule, tooltip, légende. Strings via palmares.toml (FR/EN).
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { RelationHeatmapCell } from '@/lib/api/types'

interface Props {
  cells: RelationHeatmapCell[]
  daypartLabels: string[] // 6 tranches, index = daypart
  title?: string
  legendLabel: string
  emptyMessage: string
  matchesLabel: (count: number) => string
  height?: number
}

function buildOption(
  cells: RelationHeatmapCell[],
  daypartLabels: string[],
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
    lookup.set(`${c.xuid}-${c.daypart}`, c.count)
    if (c.count > maxCount) maxCount = c.count
  }

  const data: { value: [number, number, number | null] }[] = []
  for (let dp = 0; dp < daypartLabels.length; dp++) {
    for (const xuid of rowOrder) {
      const count = lookup.get(`${xuid}-${dp}`)
      data.push({ value: [dp, rowIndex.get(xuid) ?? 0, count != null && count > 0 ? count : null] })
    }
  }

  const hasData = maxCount > 0

  return {
    backgroundColor: CHART_BG,
    grid: { left: 110, right: 130, top: 30, bottom: 40, containLabel: false },
    tooltip: {
      trigger: 'item',
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text },
      formatter: (params: { data: { value: [number, number, number | null] } }) => {
        const [dp, row, count] = params.data.value
        const who = rowLabels[row] ?? ''
        const when = daypartLabels[dp] ?? ''
        if (count == null || count === 0) return `${who} · ${when}`
        return `${who} · ${when}<br>${matchesLabel(count)}`
      },
    },
    legend: false as unknown as undefined,
    xAxis: {
      type: 'category',
      data: daypartLabels,
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
          show: true,
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

type Pt = { xuid: string; daypart: number }

export function RelationsMomentsHeatmap({
  cells,
  daypartLabels,
  title,
  legendLabel,
  emptyMessage,
  matchesLabel,
  height,
}: Props) {
  const series: ChartSeries<Pt>[] =
    cells.length > 0
      ? [{ key: 'heatmap', datapoints: cells.map((c) => ({ xuid: c.xuid, daypart: c.daypart })) }]
      : []

  const cellsKey = useMemo(() => JSON.stringify(cells), [cells])
  const daypartsKey = useMemo(() => daypartLabels.join('|'), [daypartLabels])
  const build = useCallback(
    () => buildOption(cells, daypartLabels, legendLabel, matchesLabel),
    // cells/daypartLabels capturés via clés stables (cellsKey/daypartsKey).
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [cellsKey, daypartsKey, legendLabel, matchesLabel],
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
