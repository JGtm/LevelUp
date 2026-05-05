/**
 * squadMapHeatmapChart — teammates.03 : heatmap perf joueur × carte.
 *
 * Spec : .ai/charts_specs/teammates/03_squad_heatmap.yaml
 *
 * visualMap discret (5 paliers) sur les seuils SCORE_THRESHOLDS (75/60/45/30).
 * yAxis = joueurs (moi en haut). xAxis = toutes les cartes jouées en escouade,
 * triées par fréquence décroissante.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadMapHeatmap, SquadMapHeatmapCell } from '@/lib/api/types'

export interface SquadMapHeatmapOpts {
  mapLabelOf: (mapUI: string) => string
  pieceLabels: { tier1: string; tier2: string; tier3: string; tier4: string; tier5: string }
  noScoreLabel: string
}

export function buildSquadMapHeatmapOption(
  series: ChartSeries<SquadMapHeatmap>[],
  opts: SquadMapHeatmapOpts,
): EChartsCoreOption {
  const heatmap = series[0]?.datapoints[0]
  if (!heatmap || heatmap.players.length === 0 || heatmap.maps_topn.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const xLabels = heatmap.maps_topn.map(opts.mapLabelOf)
  const yLabels = heatmap.players

  // Map (player, map) → cell pour lookup O(1).
  const cellByKey = new Map<string, SquadMapHeatmapCell>()
  for (const c of heatmap.cells) {
    cellByKey.set(`${c.player}|${c.map_ui}`, c)
  }

  const data: Array<[number, number, number | null]> = []
  for (let yi = 0; yi < heatmap.players.length; yi += 1) {
    for (let xi = 0; xi < heatmap.maps_topn.length; xi += 1) {
      const c = cellByKey.get(`${heatmap.players[yi]}|${heatmap.maps_topn[xi]}`)
      const v = c?.perf_avg !== undefined ? Number(c.perf_avg.toFixed(1)) : null
      data.push([xi, yi, v])
    }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 110, left: 8, right: 8, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: unknown) => {
        const point = p as { data?: [number, number, number | null] }
        const d = point?.data
        if (!d) return ''
        const [xi, yi, v] = d
        const cell = cellByKey.get(`${heatmap.players[yi]}|${heatmap.maps_topn[xi]}`)
        const perf = v === null ? opts.noScoreLabel : v.toFixed(1)
        const n = cell?.match_count ?? 0
        return `${heatmap.players[yi]} — ${xLabels[xi]}<br/>Perf: ${perf}<br/>N: ${n}`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: { ...axis.axisLabel, rotate: -35, interval: 0 },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: yLabels,
      inverse: true,
    },
    visualMap: {
      type: 'piecewise',
      pieces: [
        { lt: 30, color: resolveToken('perf-tier-5'), label: opts.pieceLabels.tier5 },
        { gte: 30, lt: 45, color: resolveToken('perf-tier-4'), label: opts.pieceLabels.tier4 },
        { gte: 45, lt: 60, color: resolveToken('perf-tier-3'), label: opts.pieceLabels.tier3 },
        { gte: 60, lt: 75, color: resolveToken('perf-tier-2'), label: opts.pieceLabels.tier2 },
        { gte: 75, color: resolveToken('perf-tier-1'), label: opts.pieceLabels.tier1 },
      ],
      orient: 'horizontal',
      left: 'center',
      bottom: 4,
      textStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        type: 'heatmap',
        data,
        label: { show: false },
        emphasis: { itemStyle: { borderColor: tc.text, borderWidth: 1 } },
      },
    ],
  }
}
