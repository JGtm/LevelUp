/**
 * squadRankStripsChart — l'option ECharts du sous-graphe « Rang » de la page Contributions
 * (Escouade › Performances) : un BANDEAU par joueur, grilles empilées, abscisse partagée (les
 * matchs de la session).
 *
 * Dans un bandeau : la courbe du rang (CSR ou LUSR) dans la SEULE couleur du joueur, un marqueur
 * par match — triangle plein vers le haut si le rang n'a pas reculé, triangle creux (contour à la
 * couleur du joueur) vers le bas s'il a reculé. Le sens se lit à la forme, jamais à une couleur
 * opposée. À gauche : le nom, le rang en fin de session et l'écart sur la session
 * (« 1 512 (+18) »). Les bornes du bandeau (min et max) sont en graduation discrète. Tous les
 * bandeaux ont la même amplitude, centrés chacun sur son joueur (`squadRankStrips.logic.ts`). Le
 * MMR d'équipe, tireté et à la couleur neutre du thème, a son propre bandeau et sa propre échelle.
 *
 * Aucune légende de série : l'identité est dans l'étiquette de gauche ; la légende ne nomme que
 * les deux sens de marqueur.
 */
import type { ChartOption } from '@/components/charts/chartOption'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  legendEntries,
} from '@/components/charts/_utils'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { displayRatingLabel } from '@/lib/formatters/rating'

import {
  halfAmplitude,
  mmrBounds,
  rankStrips,
  stripBounds,
  teamMmrByMatch,
  type RankKind,
  type RankPoint,
  type RankStrip,
  type StripBounds,
} from './squadRankStrips.logic'

export interface RankStripsOpts {
  /** gamertag → couleur hex (palette de la page). */
  colorByPlayer: Record<string, string>
  playerOrder?: string[]
  /** Étiquettes des matchs (« #1\nCarte »), indexées par `match_order`. */
  xLabels?: string[]
  mmrLabel: string
  /** Légende des deux marqueurs. */
  upLabel: string
  downLabel: string
  /** « variation +18 » (le delta arrive signé et formaté). */
  changeLabel: (delta: string) => string
  /** Entier avec les séparateurs de la langue (« 1 512 »). */
  fmtInt: (v: number) => string
  /** Variation signée avec les séparateurs de la langue (« +18 », « −7 »). */
  fmtDelta: (v: number) => string
}

/** Géométrie (px) : un bandeau, l'écart entre deux, la marge haute, l'axe des matchs, la légende. */
export const STRIP_HEIGHT = 56
export const STRIP_GAP = 14
const TOP = 8
const LEFT = 150
const RIGHT = 16
const AXIS_BLOCK = 46
const LEGEND_BLOCK = 30
const MARKER_SIZE = 10
const MMR_MARKER_SIZE = 5
/** Pied de l'étiquette de gauche : hauteur du nom et du rang, qu'on centre sur le bandeau. */
const LABEL_BLOCK = 34
/** Icône de légende du triangle pointe en bas (le type `triangle` d'ECharts pointe en haut). */
const TRIANGLE_DOWN_ICON = 'path://M0,0 L10,0 L5,9 Z'

/** Hauteur de la carte pour `strips` bandeaux (joueurs, plus le MMR d'équipe quand il existe). */
export function rankStripsHeight(strips: number): number {
  const n = Math.max(strips, 1)
  return TOP + n * STRIP_HEIGHT + (n - 1) * STRIP_GAP + AXIS_BLOCK + LEGEND_BLOCK
}

const stripTop = (i: number) => TOP + i * (STRIP_HEIGHT + STRIP_GAP)

/** Libellé de la méthode de note : la normalisation canonique (CSR tel quel, toute la famille LUSR en « LUSR »). */
const kindLabel = (k: RankKind) => displayRatingLabel(k) ?? k
const oneLine = (label: string) => label.replace(/\n/g, ' · ')

interface PointDatum {
  value: number
  symbol: string
  symbolRotate: number
  symbolSize: number
  itemStyle: { color: string; borderColor: string; borderWidth: number }
  tip: string
}

/** Le marqueur d'un match : plein vers le haut (rang tenu ou gagné), creux vers le bas (rang perdu). */
function pointDatum(strip: RankStrip, p: RankPoint, color: string, o: RankStripsOpts, xLabel: string): PointDatum {
  const lost = p.delta != null && p.delta < 0
  const lines = [
    `<b>${escapeHtml(strip.gamertag)}</b>`,
    escapeHtml(xLabel),
    `${kindLabel(p.kind)} ${o.fmtInt(p.rating)}`,
  ]
  if (p.delta != null) lines.push(escapeHtml(o.changeLabel(o.fmtDelta(p.delta))))
  return {
    value: p.rating,
    // Sans variation servie : un rond plein, ni gain ni perte affirmés.
    symbol: p.delta == null ? 'circle' : 'triangle',
    symbolRotate: lost ? 180 : 0,
    symbolSize: MARKER_SIZE,
    itemStyle: lost
      ? { color: 'transparent', borderColor: color, borderWidth: 2 }
      : { color, borderColor: color, borderWidth: 2 },
    tip: lines.join('<br/>'),
  }
}

/** Les séries d'un bandeau de joueur : une courbe par nature de rang (le plus souvent une seule). */
function playerSeries(
  strip: RankStrip,
  index: number,
  color: string,
  n: number,
  o: RankStripsOpts,
  xLabels: string[],
): Record<string, unknown>[] {
  const kinds = [...new Set(strip.points.map((p) => p.kind))]
  return kinds.map((kind) => {
    const data = new Array<PointDatum | null>(n).fill(null)
    for (const p of strip.points.filter((x) => x.kind === kind)) {
      data[p.order] = pointDatum(strip, p, color, o, oneLine(xLabels[p.order] ?? `#${p.order + 1}`))
    }
    return {
      type: 'line',
      name: `${strip.gamertag} — ${kindLabel(kind)}`,
      xAxisIndex: index,
      yAxisIndex: index,
      data,
      connectNulls: true,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      z: 3,
    }
  })
}

/** La série du MMR d'équipe : tiretée, couleur neutre du thème, petits ronds. */
function mmrSeries(values: (number | null)[], index: number, color: string, o: RankStripsOpts, xLabels: string[]) {
  return {
    type: 'line',
    name: o.mmrLabel,
    xAxisIndex: index,
    yAxisIndex: index,
    data: values.map((v, i) =>
      v === null
        ? null
        : {
            value: v,
            tip: [`<b>${escapeHtml(o.mmrLabel)}</b>`, escapeHtml(oneLine(xLabels[i] ?? `#${i + 1}`)), o.fmtInt(v)].join('<br/>'),
          },
    ),
    connectNulls: false,
    symbol: 'circle',
    symbolSize: MMR_MARKER_SIZE,
    lineStyle: { color, width: 2, type: 'dashed' },
    itemStyle: { color },
    z: 3,
  }
}

type Tc = ReturnType<typeof getEChartsThemeColors>

/** L'étiquette de gauche d'un bandeau : nom (à sa couleur) et « rang (écart) » dessous. */
function stripTitle(text: string, subtext: string, index: number, color: string, tc: Tc) {
  return {
    text,
    subtext,
    left: 8,
    top: stripTop(index) + (STRIP_HEIGHT - LABEL_BLOCK) / 2,
    itemGap: 4,
    textStyle: { color, fontSize: 12, fontWeight: 600 },
    subtextStyle: { color: tc.axisLabel, fontSize: 11 },
  }
}

/** Les axes d'un bandeau : abscisse partagée (étiquettes sur le dernier seulement), ordonnée à deux bornes. */
function stripAxes(index: number, last: boolean, xLabels: string[], b: StripBounds, o: RankStripsOpts, tc: Tc) {
  const axis = getAxisBase(tc)
  return {
    x: {
      ...axis,
      gridIndex: index,
      type: 'category',
      data: xLabels,
      axisLine: { show: last, lineStyle: { color: tc.axisLine } },
      axisTick: { show: false },
      // Guides verticaux : un match se suit d'un bandeau à l'autre.
      splitLine: { show: true, lineStyle: { color: tc.splitLine, type: 'dashed' } },
      axisLabel: { ...axis.axisLabel, show: last, fontSize: 9, hideOverlap: true },
    },
    y: {
      ...axis,
      gridIndex: index,
      type: 'value',
      min: b.min,
      max: b.max,
      // Deux graduations, les bornes : l'échelle se lit sans encombrer le bandeau.
      interval: b.max - b.min,
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { ...axis.axisLabel, fontSize: 9, formatter: (v: number) => o.fmtInt(v), showMinLabel: true, showMaxLabel: true },
    },
  }
}

const tipOf = (p: { data?: { tip?: string } }) => p.data?.tip ?? ''

/** Nombre de bandeaux de `buildRankStripsOption` : les joueurs avec un rang, plus le MMR d'équipe s'il existe. */
export function rankStripsCount(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  players: string[],
  n: number,
): number {
  return rankStrips(rows, players).length + (mmrBounds(teamMmrByMatch(rows, players, n)) ? 1 : 0)
}

export function buildRankStripsOption(
  rows: Record<string, SquadPerformanceSeriesPoint[]>,
  players: string[],
  n: number,
  opts: RankStripsOpts,
): ChartOption {
  const strips = rankStrips(rows, players)
  const xLabels = opts.xLabels ?? Array.from({ length: n }, (_, i) => `#${i + 1}`)
  const mmr = teamMmrByMatch(rows, players, n)
  const mmrRange = mmrBounds(mmr)
  const total = strips.length + (mmrRange ? 1 : 0)
  if (total === 0 || n === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const half = halfAmplitude(strips)
  const grid: unknown[] = []
  const xAxis: unknown[] = []
  const yAxis: unknown[] = []
  const title: unknown[] = []
  const series: unknown[] = []

  const addStrip = (i: number, b: StripBounds) => {
    const axes = stripAxes(i, i === total - 1, xLabels, b, opts, tc)
    grid.push({ left: LEFT, right: RIGHT, top: stripTop(i), height: STRIP_HEIGHT })
    xAxis.push(axes.x)
    yAxis.push(axes.y)
  }

  strips.forEach((strip, i) => {
    const color = opts.colorByPlayer[strip.gamertag] ?? tc.text
    addStrip(i, stripBounds(strip, half))
    title.push(stripTitle(strip.gamertag, `${opts.fmtInt(strip.endRating)} (${opts.fmtDelta(strip.sessionDelta)})`, i, color, tc))
    series.push(...playerSeries(strip, i, color, n, opts, xLabels))
  })
  if (mmrRange) {
    const i = strips.length
    const last = [...mmr].reverse().find((v) => v !== null) ?? 0
    addStrip(i, mmrRange)
    title.push(stripTitle(opts.mmrLabel, opts.fmtInt(last), i, tc.text, tc))
    series.push(mmrSeries(mmr, i, tc.text, opts, xLabels))
  }

  // Légende : les deux sens de marqueur, en neutre (la couleur est celle du joueur dans le graphe).
  const marker = legendEntries([
    { name: opts.upLabel, color: tc.text },
    { name: opts.downLabel, color: tc.text },
  ])
  const legendData = [
    { ...marker[0], icon: 'triangle' },
    { ...marker[1], icon: TRIANGLE_DOWN_ICON, itemStyle: { color: 'transparent', borderColor: tc.text, borderWidth: 1.5 } },
  ]
  // Séries sans donnée : elles portent seulement les deux entrées de légende.
  series.push(
    { type: 'line', name: opts.upLabel, data: [], silent: true, itemStyle: { color: tc.text }, lineStyle: { color: tc.text } },
    { type: 'line', name: opts.downLabel, data: [], silent: true, itemStyle: { color: tc.text }, lineStyle: { color: tc.text } },
  )

  return {
    backgroundColor: CHART_BG,
    animation: false,
    title,
    grid,
    xAxis,
    yAxis,
    tooltip: { ...getTooltipBase(tc), trigger: 'item', formatter: tipOf },
    legend: { ...getLegendBase(tc), bottom: 4, left: 'center', selectedMode: false, data: legendData },
    series,
  }
}
