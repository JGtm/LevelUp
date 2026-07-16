/**
 * winRateVsHistoryBulletChart — bullet chart « Winrate session vs historique » (teammates.02).
 *
 * Spec : .ai/charts_specs/teammates/02_map_winrate_bullet.yaml
 *
 * Deux barres horizontales superposées (`barGap: '-100%'`) :
 *   - Historique → token chart-series-1 (rose Okabe-Ito), opacity 0.85
 *   - Session   → couleur conditionnelle divergent-pos / -neg / -neutral selon Δ ±5 pp
 *
 * + markLine pointillé à 50 % (parité) sur la série historique
 * + markPoint rectangle vertical pour les cartes 0 % (toutes défaites — barre invisible sinon)
 *
 * Tri canonique : match_count desc + yAxis inverse (carte la plus jouée en haut).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken, tokenCssVar } from '@/lib/accessibility'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'

const DIVERGENCE_THRESHOLD = 0.05

export interface WinRateVsHistoryBulletOpts {
  mapLabelOf: (mapUI: string) => string
  sessionLabel: string
  historyLabel: string
  parityLabel?: string
  zeroWinrateLabel?: string
  /** Ligne « Session : X parties · Historique : Y parties » du tooltip. */
  countsLabel?: (session: number, history?: number) => string
}

function sessionItemColor(row: MapBreakdownRow): string {
  const hist = row.historical_win_rate
  if (hist === undefined) return resolveToken('divergent-neutral')
  if (row.win_rate - hist > DIVERGENCE_THRESHOLD) return resolveToken('divergent-pos')
  if (hist - row.win_rate > DIVERGENCE_THRESHOLD) return resolveToken('divergent-neg')
  return resolveToken('divergent-neutral')
}

function toPercent(v: number): number {
  return parseFloat((v * 100).toFixed(1))
}

type BulletTooltipParam = { dataIndex?: number; marker?: string; seriesName?: string; value?: number | null }

/**
 * bulletTooltipFormatter — tooltip custom : titre carte + valeurs par série +
 * ligne « nombre de parties » (session ET historique). escapeHtml sur les
 * données non constantes (nom de carte UGC) — garde-rail XSS tooltip.
 */
function bulletTooltipFormatter(
  sorted: MapBreakdownRow[],
  mapLabelOf: (mapUI: string) => string,
  countsLabel?: (session: number, history?: number) => string,
) {
  return (params: unknown): string => {
    const arr = (Array.isArray(params) ? params : [params]) as BulletTooltipParam[]
    const row = sorted[arr[0]?.dataIndex ?? 0]
    if (!row) return ''
    const header = escapeHtml(mapLabelOf(row.map_ui))
    const lines = arr
      .map((p) => {
        const val = typeof p.value === 'number' ? `${p.value.toFixed(1)}%` : '-'
        return `${p.marker ?? ''}${escapeHtml(p.seriesName ?? '')}: ${val}`
      })
      .join('<br/>')
    const counts = countsLabel
      ? `<br/>${escapeHtml(countsLabel(row.match_count, row.historical_match_count))}`
      : ''
    return `${header}<br/>${lines}${counts}`
  }
}

export function buildWinRateVsHistoryBulletOption(
  series: ChartSeries<MapBreakdownRow>[],
  opts: WinRateVsHistoryBulletOpts,
): EChartsCoreOption {
  const rows = series[0]?.datapoints ?? []
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  const { mapLabelOf, sessionLabel, historyLabel, parityLabel, zeroWinrateLabel, countsLabel } = opts
  const sorted = [...rows].sort((a, b) => b.match_count - a.match_count)
  // Suffixe « (n) » = nombre de parties de la session sur la carte (indicateur
  // discret toujours visible ; le tooltip détaille session + historique).
  const mapLabels = sorted.map((r) => `${mapLabelOf(r.map_ui)} (${r.match_count})`)
  const negColor = resolveToken('divergent-neg')
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const histColor = tokenCssVar('chart-series-1')
  const histData = sorted.map((r) =>
    r.historical_win_rate !== undefined
      ? { value: toPercent(r.historical_win_rate), itemStyle: { color: histColor, opacity: 0.85 } }
      : { value: null, itemStyle: { color: histColor, opacity: 0.85 } },
  )
  const sessionData = sorted.map((r) => ({
    value: toPercent(r.win_rate),
    itemStyle: { color: sessionItemColor(r) },
  }))

  const zeroIndices = sorted.flatMap((r, i) =>
    r.match_count > 0 && r.win_rate === 0 ? [i] : [],
  )

  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 32, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: bulletTooltipFormatter(sorted, mapLabelOf, countsLabel),
    },
    legend: { ...getLegendBase(tc), data: [historyLabel, sessionLabel] },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { ...axis.axisLabel, formatter: '{value}%' },
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
        barGap: '-100%',
        data: histData,
        markLine: {
          silent: true,
          symbol: 'none',
          label: { show: false },
          lineStyle: { color: tc.splitLine, type: 'dotted', width: 1.5 },
          data: [{ xAxis: 50, name: parityLabel ?? '50%' }],
        },
      },
      {
        name: sessionLabel,
        type: 'bar',
        barMaxWidth: 12,
        barGap: '-100%',
        data: sessionData,
        ...(zeroIndices.length > 0
          ? {
              markPoint: {
                symbol: 'rect',
                symbolSize: [3, 14],
                itemStyle: { color: negColor },
                label: { show: false },
                data: zeroIndices.map((i) => ({
                  xAxis: 0,
                  yAxis: i,
                  name: zeroWinrateLabel ?? '0%',
                })),
              },
            }
          : {}),
      },
    ],
  }
}
