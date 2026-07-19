/**
 * FragSunburst — carte hiérarchique « Répartition des frags » v2 (ECharts sunburst
 * 2 anneaux). Anneau INTERNE = classe (axe manipulation : Épaule/Poing/Lourde/
 * Mêlée/Grenade/Capacités spartanes), anneau EXTERNE = rôle (fonction de combat).
 * « Non attribué » = résidu hachuré. Centre = total. Composant PARTAGÉ (Synthesis/
 * Match view/Timeseries/Sessions) — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P1.4.
 *
 * Couleurs : 1 token/classe via fragClassColor (source unique, CVD-safe Okabe-Ito) ;
 * rôles = teintes de luminosité de la classe (fragRoleColor). Double encodage
 * couleur + label + position. Rend null si total 0 (aucune donnée à montrer).
 *
 * Provenance/autorité : chaque classe porte `authoritative` (exact = totaux API ;
 * estimé = registre) → badge dans le tooltip.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { fragClassColor, fragRoleColor, FRAG_CLASS_UNATTRIBUTED } from '@/lib/accessibility/scales'
import type { FragDistribution, FragClassEntry } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from './_utils'

/** Libellés/formatters injectés (builder pur, testable sans i18n). */
export interface FragSunburstLabels {
  classLabel: (className: string) => string
  roleLabel: (role: string) => string
  centerLabel: string
  authorityExact: string
  authorityEstimated: string
  formatValue: (n: number) => string
  shareTotal: (pct: string) => string
  shareClass: (pct: string, className: string) => string
}

interface SunburstNode {
  name: string
  value: number
  itemStyle: Record<string, unknown>
  children?: SunburstNode[]
  authoritative: boolean
  className: string
  classKills: number
}

const UNATTRIBUTED_DECAL = {
  color: 'rgba(0,0,0,0.38)',
  dashArrayX: [1, 0] as [number, number],
  dashArrayY: [4, 3] as [number, number],
  rotation: -Math.PI / 4,
}

function buildClassNode(entry: FragClassEntry, labels: FragSunburstLabels): SunburstNode {
  const isUnattributed = entry.class === FRAG_CLASS_UNATTRIBUTED
  const itemStyle: Record<string, unknown> = { color: fragClassColor(entry.class) }
  if (isUnattributed) itemStyle.decal = UNATTRIBUTED_DECAL
  const roles = entry.roles ?? []
  const children =
    roles.length > 0
      ? roles.map((r, i) => ({
          name: labels.roleLabel(r.role),
          value: r.kills,
          itemStyle: { color: fragRoleColor(entry.class, i, roles.length) },
          authoritative: entry.authoritative,
          className: labels.classLabel(entry.class),
          classKills: entry.kills,
        }))
      : undefined
  return {
    name: labels.classLabel(entry.class),
    value: entry.kills,
    itemStyle,
    children,
    authoritative: entry.authoritative,
    className: labels.classLabel(entry.class),
    classKills: entry.kills,
  }
}

function tooltipFormatter(labels: FragSunburstLabels, total: number) {
  return (p: { name?: string; value?: number; data?: Partial<SunburstNode>; treePathInfo?: unknown[] }) => {
    const value = typeof p.value === 'number' ? p.value : 0
    const data = p.data ?? {}
    const depth = Array.isArray(p.treePathInfo) ? p.treePathInfo.length - 1 : 1
    const badge = data.authoritative ? labels.authorityExact : labels.authorityEstimated
    const pctTotal = total > 0 ? ((value / total) * 100).toFixed(1) : '0'
    const lines = [`<b>${escapeHtml(p.name ?? '')}</b> — ${labels.formatValue(value)}`]
    lines.push(labels.shareTotal(pctTotal))
    // Niveau 2 (rôle) : ajouter la part de la classe parente.
    if (depth >= 2 && data.classKills && data.classKills > 0) {
      const pctClass = ((value / data.classKills) * 100).toFixed(1)
      lines.push(labels.shareClass(pctClass, escapeHtml(data.className ?? '')))
    }
    lines.push(`<span style="opacity:0.7">(${escapeHtml(badge)})</span>`)
    return lines.join('<br/>')
  }
}

/** Builder PUR — exporté pour tester l'option ECharts sans monter le React tree. */
// eslint-disable-next-line react-refresh/only-export-components
export function buildFragSunburstOption(
  classes: FragClassEntry[],
  total: number,
  labels: FragSunburstLabels,
): EChartsCoreOption {
  if (total <= 0 || classes.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const data = classes.map((c) => buildClassNode(c, labels))
  return {
    backgroundColor: CHART_BG,
    graphic: {
      type: 'group',
      left: 'center',
      top: 'center',
      children: [
        { type: 'text', top: -12, style: { text: labels.formatValue(total), fontSize: 22, fontWeight: 'bold', fill: tc.text, textAlign: 'center' } },
        { type: 'text', top: 14, style: { text: labels.centerLabel, fontSize: 11, fill: tc.axisLabel, textAlign: 'center' } },
      ],
    },
    tooltip: {
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text, fontSize: 11 },
      extraCssText: 'border-radius:6px;box-shadow:0 4px 12px rgba(0,0,0,0.4)',
      trigger: 'item',
      formatter: tooltipFormatter(labels, total) as unknown as string,
    },
    series: [
      {
        type: 'sunburst',
        radius: ['24%', '92%'],
        data,
        sort: undefined, // conserver l'ordre canonique fourni (jamais par valeur)
        emphasis: { focus: 'ancestor' },
        label: { color: tc.text, fontSize: 10, minAngle: 8 },
        itemStyle: { borderColor: CHART_BG, borderWidth: 2 },
        levels: [
          {},
          { r0: '24%', r: '58%', label: { rotate: 'tangential' } },
          { r0: '58%', r: '92%', label: { rotate: 'tangential' } },
        ],
      },
    ],
  }
}

function useSunburstLabels(): FragSunburstLabels {
  const appLocale = useAppShellStore((s) => s.locale)
  const numLoc = intlLocale(appLocale)
  const classLabel = (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale)
  const roleLabel = (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, appLocale)
  return {
    classLabel,
    roleLabel,
    centerLabel: formatMessage(fragsManifest, 'frags.charts.center_total_label', appLocale),
    authorityExact: formatMessage(fragsManifest, 'frags.authority.exact', appLocale),
    authorityEstimated: formatMessage(fragsManifest, 'frags.authority.estimated', appLocale),
    formatValue: (n: number) => n.toLocaleString(numLoc),
    shareTotal: (pct: string) => formatMessage(fragsManifest, 'frags.tooltip.share_total', appLocale, { pct }),
    shareClass: (pct: string, className: string) =>
      formatMessage(fragsManifest, 'frags.tooltip.share_class', appLocale, { pct, className }),
  }
}

export interface FragSunburstProps {
  distribution?: FragDistribution | null
  title?: string
  height?: number
}

export function FragSunburst({ distribution, title, height }: FragSunburstProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const labels = useSunburstLabels()
  const total = distribution?.total_kills ?? 0
  const classes = distribution?.classes ?? []

  const buildOption = useCallback(
    (series: ChartSeries<FragClassEntry>[]) => buildFragSunburstOption(series[0]?.datapoints ?? [], total, labels),
    // labels dépend de appLocale ; on l'inclut plutôt que l'objet (référence neuve à chaque rendu)
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [total, appLocale],
  )

  // Rend null si total 0 (rien à montrer) — contrat P1.4.
  if (total <= 0 || classes.length === 0) return null

  const cardTitle = title ?? formatMessage(fragsManifest, 'frags.charts.sunburst_title', appLocale)
  const series: ChartSeries<FragClassEntry>[] = [{ key: 'frag-classes', datapoints: classes }]

  return <ChartCard title={cardTitle} series={series} buildOption={buildOption} height={height ?? 320} />
}
