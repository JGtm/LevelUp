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
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import { resolveToken } from '@/lib/accessibility'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import {
  endPoint,
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
/** Réservé sous le graphe : heure (+14), carte (+27), bande (+33 à +43) — maquette, B = 50. */
const FOOT = 50
const BAND_H = 10
const BAND_BOTTOM = FOOT - 33 - BAND_H
/** Encoche de dominance : 4 × 16 px, liseré de la couleur de la carte (maquette). */
const NOTCH_W = 4
const NOTCH_H = 16

/** Rayon du point d'un match selon son volume de prises (maquette : 1,8 + 1,1 × √n). */
export function pickupRadius(pickups: number): number {
  return 1.8 + Math.sqrt(Math.max(0, pickups)) * 1.1
}

/** Une infobulle multiligne, échappée, la première ligne en gras (maquette). */
function tipHtml(text: string): string {
  const [head, ...rest] = text.split('\n')
  return [`<b>${escapeHtml(head)}</b>`, ...rest.map(escapeHtml)].join('<br>')
}

const matchName = (m: ResourceFilMatch, t: EmpriseFilText) => [t.timeOf(m.startTime), m.map].filter(Boolean).join(' · ')

/** Les deux séries d'une ressource : la courbe cumulée (point final grossi) et les points par match. */
function resourceSeries(matches: ResourceFilMatch[], resource: string, first: boolean, c: EmpriseFilColors, t: EmpriseFilText): unknown[] {
  const color = c.resource(resource)
  const label = t.resourceLabel(resource)
  let last = -1
  matches.forEach((m, i) => {
    if (m.points[resource]) last = i
  })
  const data = matches.map((m, i) => {
    const p = m.points[resource]
    if (!p) return null
    const v = p.cumulative * 100
    if (i !== last) return v
    return { ...endPoint(v, color, c.theme.card), tip: t.endTip(label, p.cumUs, p.cumTotal, t.pctFmt(v)) }
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
        symbolSize: 2 * pickupRadius(p.us + p.them),
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
function bandSeries(matches: ResourceFilMatch[], c: EmpriseFilColors, t: EmpriseFilText) {
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
      if (m.dominance) {
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

export function buildResourceFilOption(fil: ResourceFil, c: EmpriseFilColors, t: EmpriseFilText): EChartsCoreOption {
  const tc = c.theme
  const { matches } = fil
  const categories = matches.map((m) => m.matchId)
  const series = [
    ...fil.resources.flatMap((resource, ri) => resourceSeries(matches, resource, ri === 0, c, t)),
    bandSeries(matches, c, t),
  ]
  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: [
      { left: GRID_LEFT, right: GRID_RIGHT, top: 10, bottom: FOOT },
      { left: GRID_LEFT, right: GRID_RIGHT, bottom: BAND_BOTTOM, height: BAND_H },
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
          interval: 0,
          margin: 4,
          lineHeight: 13,
          formatter: (_v: string, i: number) =>
            `{t|${t.timeOf(matches[i]?.startTime ?? '')}}\n{m|${shortMap(matches[i]?.map ?? '')}}`,
          rich: {
            t: { color: tc.axisLabel, fontSize: 10.5 },
            m: { color: tc.text, fontSize: 10.5 },
          },
        },
      },
      { gridIndex: 1, type: 'category', data: categories, show: false },
    ],
    yAxis: [yAxisPct(t.pctIntFmt, tc), { gridIndex: 1, type: 'value', min: -1, max: 1, show: false }],
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
