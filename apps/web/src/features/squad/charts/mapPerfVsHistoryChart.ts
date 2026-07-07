/**
 * mapPerfVsHistoryChart — Performance par carte « Session vs Historique » (teammates.13).
 *
 * Spec : .ai/charts_specs/teammates/13_map_perf_vs_history.yaml
 *
 * Barres horizontales groupées (côte à côte, pas overlay) :
 *   - Historique (gris neutre rgba(120,120,120,0.45)) — référence visuelle
 *   - Session (couleur par palier perf-tier-1..5 selon performance_avg)
 *
 * Filtrage : jointure interne sur les cartes ayant un performance_avg ET un
 * historical_performance_avg non nuls. Tri par performance_avg ASC + head 20.
 * markLine pointillé à xAxis: 0 (référence visuelle, cohérent avec spec).
 *
 * Échelle perf 0..100 (cf. SCORE_THRESHOLDS Python : 75/60/45/30).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken } from '@/lib/accessibility'
import { perfSessionScale } from '@/lib/accessibility/scales'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'

const PERF_HISTORY_COLOR = 'rgba(120, 120, 120, 0.45)' // color-allow: gris neutre comparable historique (sur les 2 thèmes)
const MAX_MAPS = 20

export interface MapPerfVsHistoryOpts {
  mapLabelOf: (mapUI: string) => string
  sessionLabel: string
  historyLabel: string
}

function round1(v: number): number {
  return parseFloat(v.toFixed(1))
}

interface JoinedRow {
  mapUI: string
  perfSession: number
  perfHistory: number
  matchCount: number
}

function joinAndSort(rows: MapBreakdownRow[]): JoinedRow[] {
  const joined: JoinedRow[] = []
  for (const r of rows) {
    if (r.performance_avg === undefined || r.historical_performance_avg === undefined) continue
    joined.push({
      mapUI: r.map_ui,
      perfSession: r.performance_avg,
      perfHistory: r.historical_performance_avg,
      matchCount: r.match_count,
    })
  }
  // Tri par nombre de matchs DESC (les cartes les plus jouées en haut).
  joined.sort((a, b) => b.matchCount - a.matchCount)
  return joined.slice(0, MAX_MAPS)
}

export function buildMapPerfVsHistoryOption(
  series: ChartSeries<MapBreakdownRow>[],
  opts: MapPerfVsHistoryOpts,
): EChartsCoreOption {
  const rows = series[0]?.datapoints ?? []
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  const { mapLabelOf, sessionLabel, historyLabel } = opts
  const joined = joinAndSort(rows)
  if (joined.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const mapLabels = joined.map((r) => mapLabelOf(r.mapUI))
  const sessionData = joined.map((r) => ({
    value: round1(r.perfSession),
    itemStyle: { color: resolveToken(perfSessionScale(r.perfSession)), opacity: 0.85 },
  }))
  const historyData = joined.map((r) => ({
    value: round1(r.perfHistory),
    itemStyle: { color: PERF_HISTORY_COLOR },
  }))

  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 32, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      valueFormatter: (v: unknown) => (typeof v === 'number' ? v.toFixed(1) : '-'),
    },
    legend: { ...getLegendBase(tc), data: [historyLabel, sessionLabel] },
    xAxis: {
      ...axis,
      type: 'value',
      axisLabel: { ...axis.axisLabel },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: mapLabels,
      inverse: true,
      axisLabel: { ...axis.axisLabel, width: 100, overflow: 'truncate' },
    },
    series: [
      {
        name: historyLabel,
        type: 'bar',
        barMaxWidth: 12,
        data: historyData,
        markLine: {
          silent: true,
          symbol: 'none',
          label: { show: false },
          lineStyle: { color: tc.splitLine, type: 'dotted', width: 1 },
          data: [{ xAxis: 0 }],
        },
      },
      {
        name: sessionLabel,
        type: 'bar',
        barMaxWidth: 12,
        data: sessionData,
      },
    ],
  }
}
