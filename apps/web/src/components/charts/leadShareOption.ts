/**
 * leadShareOption — L'OPTION ECHARTS DE LA MAINMISE, commune aux quatre pages qui l'affichent
 * (Sessions : un match par colonne ; Séries temporelles : un mois par colonne ; Escouade : une
 * courbe par match).
 *
 * NORMALISÉE : l'axe va TOUJOURS de 0 à 100 %, jamais à l'échelle des données. Deux sessions
 * comparées côte à côte, ou deux mois, se lisent donc sur la même règle, et la ligne pointillée
 * à 50 % (ni devant ni derrière) tombe au même endroit partout.
 *
 * UN MATCH NON MESURÉ N'EST PAS UN ZÉRO. En colonnes, il se dessine en pointillé pleine hauteur
 * (le bloc existe, la mesure non) ; en courbe, la ligne se coupe. L'infobulle en dit la raison.
 *
 * Un seul axe, une légende en bas, des jetons de couleur : l'équipe du joueur pour la part,
 * le neutre pour le repère et le pointillé.
 */
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import { formatLeadChanges, formatLeadShare, type LeadSharePoint } from '@/lib/charts/leadShare/model'
import type { LeadShareText } from '@/lib/charts/leadShare/text'
import type { Locale } from '@/lib/i18n/locale'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from './_utils'

export type LeadShareVariant = 'bars' | 'line'

export interface LeadShareOptionOpts {
  text: LeadShareText
  /** La langue de l'affichage : le pluriel des retournements suit sa règle. */
  locale: Locale
  variant: LeadShareVariant
  /** Colonne divisée (comparaison) : étiquettes de l'axe plus serrées. */
  compact?: boolean
}

/** Les lignes de l'infobulle d'un point : nom, part ou raison du tiret, retournements, détails. */
export function leadShareTooltipLines(point: LeadSharePoint, text: LeadShareText, locale: Locale): string[] {
  const lines = [`<strong>${escapeHtml(point.label.replace('\n', ' · '))}</strong>`]
  if (point.share == null) {
    lines.push(escapeHtml(text.notMeasured))
  } else {
    lines.push(`${escapeHtml(text.name)} : <b>${formatLeadShare(point.share)}</b>`)
    const changes = formatLeadChanges(point.changes, text, locale)
    if (changes) lines.push(escapeHtml(changes))
  }
  for (const d of point.details ?? []) lines.push(escapeHtml(d))
  return lines
}

const pct = (share: number) => Math.round(share * 1000) / 10

/** Les encres du graphe : la part à la couleur de l'équipe du joueur, le repère et le pointillé en neutre. */
interface Inks {
  color: string
  neutral: string
}

/** La ligne pointillée à 50 % (ni devant ni derrière), commune aux deux variantes. */
function referenceLine(text: LeadShareText, inks: Inks, axisLabel: string) {
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'dashed' as const, width: 1, color: inks.neutral },
    label: { show: true, position: 'end' as const, formatter: text.referenceLabel, color: axisLabel, fontSize: 10 },
    data: [{ yAxis: 50 }],
  }
}

/** La série de la part : colonnes (un point par colonne) ou courbe coupée aux matchs non mesurés. */
function shareSeriesOf(
  points: readonly LeadSharePoint[],
  opts: LeadShareOptionOpts,
  inks: Inks,
  reference: ReturnType<typeof referenceLine>,
) {
  const { text, variant, compact = false } = opts
  const data = points.map((p) => (p.share == null ? null : pct(p.share)))
  if (variant === 'bars') {
    return {
      name: text.name,
      type: 'bar' as const,
      barMaxWidth: compact ? 22 : 36,
      data,
      itemStyle: { color: inks.color },
      markLine: reference,
    }
  }
  return {
    name: text.name,
    type: 'line' as const,
    connectNulls: false,
    symbolSize: 5,
    data,
    itemStyle: { color: inks.color },
    lineStyle: { color: inks.color, width: 2 },
    markLine: reference,
  }
}

/** Colonne en pointillé des matchs non mesurés : une série à part, pleine hauteur, sans fond. */
function gapSeriesOf(points: readonly LeadSharePoint[], opts: LeadShareOptionOpts, inks: Inks) {
  return {
    name: opts.text.notMeasuredLegend,
    type: 'bar' as const,
    barMaxWidth: opts.compact ? 22 : 36,
    barGap: '-100%',
    data: points.map((p) => (p.share == null ? 100 : null)),
    itemStyle: { color: 'transparent', borderColor: inks.neutral, borderType: 'dashed' as const, borderWidth: 1 },
  }
}

export function buildLeadShareOption(points: readonly LeadSharePoint[], opts: LeadShareOptionOpts): EChartsCoreOption {
  if (points.length === 0) return { backgroundColor: CHART_BG }
  const { text, locale, variant, compact = false } = opts
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const inks: Inks = { color: resolveToken('team-ally'), neutral: resolveToken('divergent-neutral') }
  const hasGap = points.some((p) => p.share == null)
  const interval = points.length > 24 ? Math.floor(points.length / 12) : 0
  const shareSeries = shareSeriesOf(points, opts, inks, referenceLine(text, inks, tc.axisLabel))
  const withGap = variant === 'bars' && hasGap

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 56, left: 44, right: 40 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: variant === 'bars' ? 'shadow' : 'line' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        const idx = (arr[0] as { dataIndex?: number } | undefined)?.dataIndex
        const point = idx == null ? undefined : points[idx]
        return point ? leadShareTooltipLines(point, text, locale).join('<br/>') : ''
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: legendEntries([
        { name: text.name, color: inks.color },
        ...(withGap ? [{ name: text.notMeasuredLegend, color: inks.neutral, dashed: true }] : []),
      ]),
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval, ...(compact ? { fontSize: 9 } : {}) },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      interval: 25,
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), formatter: (v: number) => `${v} %` },
    },
    series: withGap ? [shareSeries, gapSeriesOf(points, opts, inks)] : [shareSeries],
  }
}
