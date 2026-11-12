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
import { formatLeadShare } from '@/lib/charts/leadShare/model'
import type { LeadShareText } from '@/lib/charts/leadShare/text'
import { fillTemplate } from '@/lib/i18n/template'

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

/** Les encres : la sélection à la couleur de l'équipe du joueur, l'historique en neutre. */
interface Inks {
  color: string
  neutral: string
}

/** Une ligne d'infobulle : le périmètre, sa part et le nombre de matchs MESURÉS qui la fondent. */
function tooltipLineOf(text: LeadShareText, label: string, share: number | undefined, n: number | undefined) {
  return share == null
    ? null
    : `${escapeHtml(label)} : <b>${formatLeadShare(share)}</b> (${escapeHtml(
        fillTemplate(n === 1 ? text.measuredOne : text.measuredOther, { n: n ?? 0 }),
      )})`
}

/** L'infobulle d'une carte : son nom, puis la sélection et l'historique. */
function tooltipOf(kept: readonly MapBreakdownRow[], opts: LeadShareMapOpts) {
  const { text, mapLabelOf, scopeLabel, historyLabel } = opts
  return (params: unknown): string => {
    const arr = Array.isArray(params) ? params : []
    const idx = (arr[0] as { dataIndex?: number } | undefined)?.dataIndex
    const row = idx == null ? undefined : kept[idx]
    if (!row) return ''
    return [
      `<strong>${escapeHtml(mapLabelOf(row.map_ui))}</strong>`,
      tooltipLineOf(text, scopeLabel, row.lead_share, row.lead_share_matches),
      tooltipLineOf(text, historyLabel, row.historical_lead_share, row.historical_lead_share_matches),
    ]
      .filter((l): l is string => l != null)
      .join('<br/>')
  }
}

/** Les deux séries : l'historique (porte le repère à 50 %) puis la sélection. */
function seriesOf(kept: readonly MapBreakdownRow[], opts: LeadShareMapOpts, inks: Inks, axisLabel: string) {
  const { text, scopeLabel, historyLabel } = opts
  return [
    {
      name: historyLabel,
      type: 'bar' as const,
      barMaxWidth: 12,
      data: kept.map((r) => pct(r.historical_lead_share)),
      itemStyle: { color: inks.neutral, opacity: 0.45 },
      markLine: {
        silent: true,
        symbol: 'none',
        lineStyle: { type: 'dashed' as const, width: 1, color: inks.neutral },
        label: { show: true, position: 'end' as const, formatter: text.referenceLabel, color: axisLabel, fontSize: 10 },
        data: [{ xAxis: 50 }],
      },
    },
    {
      name: scopeLabel,
      type: 'bar' as const,
      barMaxWidth: 12,
      data: kept.map((r) => pct(r.lead_share)),
      itemStyle: { color: inks.color },
    },
  ]
}

export function buildLeadShareMapOption(rows: readonly MapBreakdownRow[], opts: LeadShareMapOpts): EChartsCoreOption {
  const kept = leadShareMapRows(rows)
  if (kept.length === 0) return { backgroundColor: CHART_BG }
  const { mapLabelOf, scopeLabel, historyLabel } = opts
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const inks: Inks = { color: resolveToken('team-ally'), neutral: resolveToken('divergent-neutral') }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 40, left: 8, right: 40, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: tooltipOf(kept, opts),
    },
    legend: {
      ...getLegendBase(tc),
      data: legendEntries([
        { name: historyLabel, color: inks.neutral },
        { name: scopeLabel, color: inks.color },
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
    series: seriesOf(kept, opts, inks, tc.axisLabel),
  }
}
