/**
 * leadShareMapOption — « Mainmise par carte », la sélection contre l'historique : une paire de
 * barres horizontales par carte (historique en neutre, sélection à la couleur de l'équipe du
 * joueur), sur un axe NORMALISÉ de 0 à 100 % avec le repère pointillé à 50 %.
 *
 * Seules les cartes qui portent au moins une mesure sont tracées : une carte dont aucun match
 * n'est mesuré (mode à peu de points, film non lu) n'a ni barre ni zéro. L'infobulle donne, pour
 * chaque barre, la part et le nombre de matchs MESURÉS qui la fondent.
 */
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import type { MapBreakdownRow } from '@/lib/api/types'
import { formatLeadShare, formatTemplate } from '@/lib/charts/leadShare/model'
import type { LeadShareText } from '@/lib/charts/leadShare/text'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from './_utils'

const MAX_MAPS = 20

export interface LeadShareMapOpts {
  text: LeadShareText
  mapLabelOf: (mapUI: string) => string
  /** « Session » / « Sélection » : le périmètre filtré de la page. */
  scopeLabel: string
  historyLabel: string
}

/** Les cartes retenues : au moins une mesure, les plus jouées d'abord retenues, l'ordre reçu gardé. */
export function leadShareMapRows(rows: readonly MapBreakdownRow[]): MapBreakdownRow[] {
  const measured = rows.filter((r) => r.lead_share != null || r.historical_lead_share != null)
  const keep = new Set(
    [...measured].sort((a, b) => b.match_count - a.match_count).slice(0, MAX_MAPS).map((r) => r.map_ui),
  )
  return measured.filter((r) => keep.has(r.map_ui))
}

const pct = (v: number | undefined) => (v == null ? null : Math.round(v * 1000) / 10)

export function buildLeadShareMapOption(rows: readonly MapBreakdownRow[], opts: LeadShareMapOpts): EChartsCoreOption {
  const kept = leadShareMapRows(rows)
  if (kept.length === 0) return { backgroundColor: CHART_BG }
  const { text, mapLabelOf, scopeLabel, historyLabel } = opts
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const color = resolveToken('team-ally')
  const neutral = resolveToken('divergent-neutral')

  const tooltipLine = (label: string, share: number | undefined, n: number | undefined) =>
    share == null
      ? null
      : `${escapeHtml(label)} : <b>${formatLeadShare(share)}</b> (${escapeHtml(
          formatTemplate(n === 1 ? text.measuredOne : text.measuredOther, { n: n ?? 0 }),
        )})`

  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 40, left: 8, right: 40, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        const idx = (arr[0] as { dataIndex?: number } | undefined)?.dataIndex
        const row = idx == null ? undefined : kept[idx]
        if (!row) return ''
        return [
          `<strong>${escapeHtml(mapLabelOf(row.map_ui))}</strong>`,
          tooltipLine(scopeLabel, row.lead_share, row.lead_share_matches),
          tooltipLine(historyLabel, row.historical_lead_share, row.historical_lead_share_matches),
        ]
          .filter((l): l is string => l != null)
          .join('<br/>')
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: legendEntries([
        { name: historyLabel, color: neutral },
        { name: scopeLabel, color },
      ]),
    },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      interval: 25,
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), formatter: (v: number) => `${v} %` },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: kept.map((r) => mapLabelOf(r.map_ui)),
      inverse: true,
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), width: 100, overflow: 'truncate' },
    },
    series: [
      {
        name: historyLabel,
        type: 'bar',
        barMaxWidth: 12,
        data: kept.map((r) => pct(r.historical_lead_share)),
        itemStyle: { color: neutral, opacity: 0.45 },
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { type: 'dashed' as const, width: 1, color: neutral },
          label: { show: true, position: 'end' as const, formatter: text.referenceLabel, color: tc.axisLabel, fontSize: 10 },
          data: [{ xAxis: 50 }],
        },
      },
      {
        name: scopeLabel,
        type: 'bar',
        barMaxWidth: 12,
        data: kept.map((r) => pct(r.lead_share)),
        itemStyle: { color },
      },
    ],
  }
}
