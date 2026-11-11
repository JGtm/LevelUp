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
  variant: LeadShareVariant
  /** Colonne divisée (comparaison) : étiquettes de l'axe plus serrées. */
  compact?: boolean
}

/** Les lignes de l'infobulle d'un point : nom, part ou raison du tiret, retournements, détails. */
export function leadShareTooltipLines(point: LeadSharePoint, text: LeadShareText): string[] {
  const lines = [`<strong>${escapeHtml(point.label.replace('\n', ' · '))}</strong>`]
  if (point.share == null) {
    lines.push(escapeHtml(text.notMeasured))
  } else {
    lines.push(`${escapeHtml(text.name)} : <b>${formatLeadShare(point.share)}</b>`)
    const changes = formatLeadChanges(point.changes, text)
    if (changes) lines.push(escapeHtml(changes))
  }
  for (const d of point.details ?? []) lines.push(escapeHtml(d))
  return lines
}

const pct = (share: number) => Math.round(share * 1000) / 10

export function buildLeadShareOption(points: readonly LeadSharePoint[], opts: LeadShareOptionOpts): EChartsCoreOption {
  if (points.length === 0) return { backgroundColor: CHART_BG }
  const { text, variant, compact = false } = opts
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const color = resolveToken('team-ally')
  const neutral = resolveToken('divergent-neutral')
  const hasGap = points.some((p) => p.share == null)
  const interval = points.length > 24 ? Math.floor(points.length / 12) : 0

  const reference = {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'dashed' as const, width: 1, color: neutral },
    label: { show: true, position: 'end' as const, formatter: text.referenceLabel, color: tc.axisLabel, fontSize: 10 },
    data: [{ yAxis: 50 }],
  }

  const shareSeries =
    variant === 'bars'
      ? {
          name: text.name,
          type: 'bar' as const,
          barMaxWidth: compact ? 22 : 36,
          data: points.map((p) => (p.share == null ? null : pct(p.share))),
          itemStyle: { color },
          markLine: reference,
        }
      : {
          name: text.name,
          type: 'line' as const,
          connectNulls: false,
          symbolSize: 5,
          data: points.map((p) => (p.share == null ? null : pct(p.share))),
          itemStyle: { color },
          lineStyle: { color, width: 2 },
          markLine: reference,
        }
  // Colonne en pointillé des matchs non mesurés : une série à part, pleine hauteur, sans fond.
  const gapSeries = {
    name: text.notMeasuredLegend,
    type: 'bar' as const,
    barMaxWidth: compact ? 22 : 36,
    barGap: '-100%',
    data: points.map((p) => (p.share == null ? 100 : null)),
    itemStyle: { color: 'transparent', borderColor: neutral, borderType: 'dashed' as const, borderWidth: 1 },
  }

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
        return point ? leadShareTooltipLines(point, text).join('<br/>') : ''
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: legendEntries([
        { name: text.name, color },
        ...(hasGap && variant === 'bars' ? [{ name: text.notMeasuredLegend, color: neutral, dashed: true }] : []),
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
    series: variant === 'bars' && hasGap ? [shareSeries, gapSeries] : [shareSeries],
  }
}
