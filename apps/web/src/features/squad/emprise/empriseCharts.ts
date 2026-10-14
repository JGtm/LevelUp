/**
 * empriseCharts.ts — le graphe « Contrôle des ressources au fil de la session » de l'onglet
 * Emprise (lot L5 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), en option ECharts,
 * d'après la maquette de l'onglet (`renderFilSession`) :
 *
 *   - UN graphe (S6 : les séries partagent mesure et échelle), une courbe cumulée par ressource
 *     du bilan, à la couleur de la ressource ;
 *   - la part de chaque match en petits points pâles, leur taille son volume (rayon 1,8 +
 *     1,1 × √prises, maquette) ; un match sans la ressource n'a pas de point, la courbe file ;
 *   - le point final grossi et la valeur courte au bout (« 60 % ») ; le trait 50 % pointillé
 *     `warning` (S5) ;
 *   - sous l'axe, l'heure puis la carte de chaque match, puis la bande de résultats (case
 *     victoire / défaite) avec l'encoche du drapeau de dominance quand il existe (S9).
 *
 * Deux grilles (courbes / bande) sur la même abscisse, marges fixes : alignement au pixel. Les
 * briques de courbe sont celles de la section Objectif (`objectif/objectifCharts.ts`).
 * Toutes les couleurs arrivent résolues (jetons) : aucune valeur en dur ici.
 */
import type { EChartsCoreOption } from 'echarts/core'

import { CHART_BG, escapeHtml, getTooltipBase } from '@/components/charts/_utils'
import {
  footLabelStyle,
  layoutAxisFoot,
  lineHeightFor,
  type AxisFootSpec,
  type FootLabels,
} from '@/components/charts/axisFoot'
import { withEndPoint } from '@/components/charts/endPoint'
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import { resolveToken } from '@/lib/accessibility'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import {
  lineSeries,
  parityLine,
  shortMap,
  yAxisPct,
  type CustomApi,
} from '../objectif/objectifCharts'
import type { ResourceFil, ResourceFilMatch } from './emprise.logic'
import { resolveResourceColor } from './resourceColors'

export interface EmpriseFilColors {
  resource: (resource: string) => string
  win: string
  loss: string
  /** Case d'un match sans résultat connu (égalité, abandon, historique absent). */
  neutral: string
  parity: string
  dominance: (d: DominanceValue) => string
  theme: EChartsThemeColors
}

export interface EmpriseFilText {
  resourceLabel: (resource: string) => string
  /** Part avec une décimale au plus (infobulles). */
  pctFmt: (v: number) => string
  /** Part entière (valeur au bout des courbes, graduation 100 %). */
  pctIntFmt: (v: number) => string
  timeOf: (iso: string) => string
  /** « victoire » / « défaite » (infobulle d'un point), null sans résultat. */
  outcomeOf: (m: ResourceFilMatch) => string | null
  /** « Victoire 3–0 », null sans résultat. */
  resultOf: (m: ResourceFilMatch) => string | null
  dominanceLabel: (d: DominanceValue) => string
  pointTip: (v: { match: string; outcome: string | null; resource: string; us: number; them: number; pct: string; cumUs: number; cumTotal: number; cumPct: string }) => string
  endTip: (resource: string, cumUs: number, cumTotal: number, pct: string) => string
  bandTip: (match: string, result: string | null, dominance: string | null) => string
}

/** Marges fixes (maquette : 36 px à gauche, 44 px à droite pour la valeur de fin). */
const GRID_LEFT = 36
const GRID_RIGHT = 44
const BAND_H = 10
/** Corps des étiquettes sous l'axe (heure, carte, date de mois). */
const LABEL_FONT = 10.5
const LABEL_LINE = lineHeightFor(LABEL_FONT)
/** Encoche de dominance : 4 × 16 px, liseré de la couleur de la carte (maquette). */
const NOTCH_W = 4
const NOTCH_H = 16

/** Rayon du point d'un match selon son volume de prises (maquette : 1,8 + 1,1 × √n). */
export function pickupRadius(pickups: number): number {
  return 1.8 + Math.sqrt(Math.max(0, pickups)) * 1.1
}

/**
 * Sur l'axe PÉRIODE seulement, au-delà de ce nombre de matchs, des points plus petits (maquette
 * des Séries temporelles) ; l'axe par match de l'Escouade garde le rayon de sa maquette.
 */
const MANY_MATCHES = 120

/** Rayon d'un point quand le graphe porte beaucoup de matchs : 0,8 + 0,45 × √n. */
function smallPickupRadius(pickups: number): number {
  return 0.8 + Math.sqrt(Math.max(0, pickups)) * 0.45
}

/**
 * L'axe du graphe : un match par colonne avec son heure et sa carte (`match`, une soirée de
 * l'Escouade), ou une PÉRIODE (`period`, la fenêtre des Séries temporelles) — la date du premier
 * match de chaque mois sous l'axe, aucune mention de couverture, pas d'encoche de
 * dominance (plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, D12).
 */
export type FilAxe =
  | { kind: 'match'; compact?: boolean }
  | { kind: 'period'; dateOf: (iso: string) => string }

/**
 * Le mode match COMPACT (tiroir de comparaison de Sessions, maquette Sessions `renderFil` avec `cp`) :
 * pied réduit à la seule bande (rien sous l'axe), graduations 0 / 50 / 100, points 1,6 + √n.
 */
const compactPickupRadius = (pickups: number) => 1.6 + Math.sqrt(Math.max(0, pickups))
const isCompact = (axe: FilAxe) => axe.kind === 'match' && axe.compact === true

/**
 * Le pied du graphe selon l'axe — calculé (`layoutAxisFoot`) depuis les étiquettes qu'il porte,
 * jamais écrit à la main : match = l'heure puis la carte (2 lignes) au-dessus de la bande ;
 * période = la date du mois (1 ligne) ; compact = la bande seule, rien sous l'axe.
 */
const MATCH_LABELS: FootLabels = { lines: 2, lineHeight: LABEL_LINE, margin: 4 }
const PERIOD_LABELS: FootLabels = { lines: 1, lineHeight: LABEL_LINE, margin: 4 }
const FIL_BAND = { height: BAND_H, gap: 4 }
const MATCH_FOOT: AxisFootSpec = { above: MATCH_LABELS, band: FIL_BAND, below: null, pad: 4 }
const PERIOD_FOOT: AxisFootSpec = { above: PERIOD_LABELS, band: FIL_BAND, below: null, pad: 4 }
const COMPACT_FOOT: AxisFootSpec = { above: null, band: { height: BAND_H, gap: 8 }, below: null, pad: 12 }

function footSpecFor(axe: FilAxe): AxisFootSpec {
  if (isCompact(axe)) return COMPACT_FOOT
  return axe.kind === 'period' ? PERIOD_FOOT : MATCH_FOOT
}

/** Le rayon des points selon l'axe : compact, période chargée, ou celui de la maquette. */
function radiusFor(axe: FilAxe, matches: number): (pickups: number) => number {
  if (isCompact(axe)) return compactPickupRadius
  return axe.kind === 'period' && matches > MANY_MATCHES ? smallPickupRadius : pickupRadius
}

/** Le mois d'un match (heure locale), clé des étiquettes du mode période ; vide sans date. */
function monthKey(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : `${d.getFullYear()}-${d.getMonth()}`
}

/** Les étiquettes du mode période : la date du premier match de chaque mois, vide ailleurs. */
function periodLabels(matches: ResourceFilMatch[], dateOf: (iso: string) => string): string[] {
  let previous = ''
  return matches.map((m) => {
    const key = monthKey(m.startTime)
    if (key === '' || key === previous) return ''
    previous = key
    return dateOf(m.startTime)
  })
}

/** Une infobulle multiligne, échappée, la première ligne en gras (maquette). */
export function tipHtml(text: string): string {
  const [head, ...rest] = text.split('\n')
  return [`<b>${escapeHtml(head)}</b>`, ...rest.map(escapeHtml)].join('<br>')
}

const matchName = (m: ResourceFilMatch, t: EmpriseFilText) => [t.timeOf(m.startTime), m.map].filter(Boolean).join(' · ')

/**
 * Les deux séries d'une ressource : la courbe cumulée (point final grossi) et les points par match
 * au rayon `o.radius`.
 */
function resourceSeries(
  matches: ResourceFilMatch[],
  resource: string,
  c: EmpriseFilColors,
  t: EmpriseFilText,
  o: { first: boolean; radius: (pickups: number) => number },
): unknown[] {
  const color = c.resource(resource)
  const label = t.resourceLabel(resource)
  const { first, radius } = o
  const cum = matches.map((m) => (m.points[resource] ? m.points[resource]!.cumulative * 100 : null))
  const data = withEndPoint(cum, c.theme.card, {
    color,
    extra: (i) => {
      const p = matches[i].points[resource]!
      return { tip: t.endTip(label, p.cumUs, p.cumTotal, t.pctFmt(p.cumulative * 100)) }
    },
  })
  const dots = {
    name: label,
    type: 'scatter' as const,
    xAxisIndex: 0,
    yAxisIndex: 0,
    z: 2,
    data: matches.map((m, i) => {
      const p = m.points[resource]
      if (!p) return null
      return {
        value: [i, p.share * 100],
        symbolSize: 2 * radius(p.us + p.them),
        tip: t.pointTip({
          match: matchName(m, t),
          outcome: t.outcomeOf(m),
          resource: label,
          us: p.us,
          them: p.them,
          pct: t.pctFmt(p.share * 100),
          cumUs: p.cumUs,
          cumTotal: p.cumTotal,
          cumPct: t.pctFmt(p.cumulative * 100),
        }),
      }
    }),
    itemStyle: { color, opacity: 0.45, borderColor: c.theme.card, borderWidth: 1 },
  }
  return [lineSeries(label, data, color, t.pctIntFmt, first ? { markLine: parityLine(c.parity) } : {}), dots]
}

/** La bande de résultats sous l'axe : une case par match, l'encoche du drapeau de dominance. */
function bandSeries(matches: ResourceFilMatch[], c: EmpriseFilColors, t: EmpriseFilText, notch: boolean) {
  return {
    type: 'custom' as const,
    xAxisIndex: 1,
    yAxisIndex: 1,
    data: matches.map((m, i) => ({
      value: [i, 0],
      tip: t.bandTip(matchName(m, t), t.resultOf(m), m.dominance ? t.dominanceLabel(m.dominance) : null),
    })),
    renderItem: (_params: unknown, api: CustomApi) => {
      const i = api.value(0) as number
      const m = matches[i]
      const [cx, cy] = api.coord([i, 0])
      const w = (api.size([1, 0]) as number[])[0] - 3
      const fill = m.outcome === 'win' ? c.win : m.outcome === 'loss' ? c.loss : c.neutral
      const children: unknown[] = [
        { type: 'rect', shape: { x: cx - w / 2, y: cy - BAND_H / 2, width: w, height: BAND_H, r: 2 }, style: { fill } },
      ]
      if (notch && m.dominance) {
        children.push({
          type: 'rect',
          shape: { x: cx - NOTCH_W / 2, y: cy - NOTCH_H / 2, width: NOTCH_W, height: NOTCH_H, r: 1 },
          style: { fill: c.dominance(m.dominance), stroke: c.theme.card, lineWidth: 1.5 },
        })
      }
      return { type: 'group', children }
    },
  }
}

export function buildResourceFilOption(
  fil: ResourceFil,
  c: EmpriseFilColors,
  t: EmpriseFilText,
  axe: FilAxe = { kind: 'match' },
): EChartsCoreOption {
  const tc = c.theme
  const { matches } = fil
  const categories = matches.map((m) => m.matchId)
  // Sur une période, le nom d'un match dans les infobulles porte sa date devant l'heure.
  const tipText: EmpriseFilText =
    axe.kind === 'period' ? { ...t, timeOf: (iso) => [axe.dateOf(iso), t.timeOf(iso)].filter(Boolean).join(' ') } : t
  const radius = radiusFor(axe, matches.length)
  const compact = isCompact(axe)
  const spec = footSpecFor(axe)
  const foot = layoutAxisFoot(spec)
  const series = [
    ...fil.resources.flatMap((resource, ri) => resourceSeries(matches, resource, c, tipText, { first: ri === 0, radius })),
    bandSeries(matches, c, tipText, axe.kind === 'match'),
  ]
  const months = axe.kind === 'period' ? periodLabels(matches, axe.dateOf) : []
  const label = compact
    ? () => ''
    : axe.kind === 'period'
      ? (_v: string, i: number) => (months[i] ? `{d|${months[i]}}` : '')
      : (_v: string, i: number) => `{t|${t.timeOf(matches[i]?.startTime ?? '')}}\n{m|${shortMap(matches[i]?.map ?? '')}}`
  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: [
      { left: GRID_LEFT, right: GRID_RIGHT, top: 10, bottom: foot.gridBottom },
      { left: GRID_LEFT, right: GRID_RIGHT, bottom: foot.bandBottom, height: foot.bandHeight },
    ],
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: { data?: { tip?: string } }) => (p.data?.tip ? tipHtml(p.data.tip) : ''),
    },
    xAxis: [
      {
        gridIndex: 0,
        type: 'category',
        data: categories,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          ...footLabelStyle(spec.above ?? MATCH_LABELS),
          formatter: label,
          rich: {
            t: { color: tc.axisLabel, fontSize: LABEL_FONT },
            m: { color: tc.text, fontSize: LABEL_FONT },
            d: { color: tc.axisLabel, fontSize: LABEL_FONT },
          },
        },
      },
      { gridIndex: 1, type: 'category', data: categories, show: false },
    ],
    yAxis: [
      { ...yAxisPct(t.pctIntFmt, tc), ...(compact ? { interval: 50 } : {}) },
      { gridIndex: 1, type: 'value', min: -1, max: 1, show: false },
    ],
    series,
    // La légende est rendue HORS canvas (pied de carte, S2).
    legend: { show: false },
    aria: { enabled: true },
  }
}

/**
 * resolveEmpriseFilColors — les couleurs du graphe, résolues depuis les jetons AU MOMENT du
 * rendu (appelé dans `buildOption`, que `ChartCard` rejoue à chaque changement de thème ou de
 * palette).
 */
export function resolveEmpriseFilColors(): EmpriseFilColors {
  const theme = getEChartsThemeColors()
  return {
    resource: resolveResourceColor,
    win: resolveToken('outcome-win'),
    loss: resolveToken('outcome-loss'),
    neutral: theme.splitLine,
    parity: resolveToken('warning'),
    dominance: (d) => resolveToken(DOMINANCE_COLOR_TOKENS[d]),
    theme,
  }
}
