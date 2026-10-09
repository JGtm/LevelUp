/**
 * impactHistoryChart.ts — l'option ECharts de « Points d'impact par soirée et par rôle » (maquette
 * validée, forme « Détail des rôles ») :
 *
 *   - trois barres par soirée (une par joueur, dans l'ordre de la page), empilées par rôle en
 *     points : gains au-dessus de zéro, pertes en dessous, le plus fort barème contre l'axe ;
 *   - au pied de chaque barre, la pastille du joueur ;
 *   - une courbe du net par joueur, à sa couleur, détachée par un halo de la couleur de la
 *     carte, chaque point un losange posé sur sa barre ; seuls les deux nets extrêmes écrits ;
 *   - une infobulle par barre : joueur, soirée (matchs, victoires), rôles « ×n · ±points », net.
 *
 * Toutes les couleurs arrivent résolues (jetons, palette des joueurs de l'Escouade).
 */
import type { EChartsCoreOption } from 'echarts/core'

import { CHART_BG, escapeHtml, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'

import type { CustomApi } from '../objectif/objectifCharts'
import { pointKey, showAxisLabel, type ImpactBar, type ImpactHistoryView, type ImpactSegment } from './impactHistory.logic'

/** Les couleurs résolues du graphe. */
export interface ImpactChartColors {
  /** Couleur de chaque joueur, dans l'ordre de `view.players`. */
  players: string[]
  role: (token: SemanticToken) => string
  theme: EChartsThemeColors
}

const IMPACT_TOKENS: SemanticToken[] = [
  'impact-gain-1', 'impact-gain-2', 'impact-gain-3', 'impact-gain-4',
  'impact-loss-1', 'impact-loss-2', 'impact-loss-3',
]

/** Résout les couleurs au moment du rendu (rejoué par ChartCard au changement de thème). */
export function resolveImpactColors(players: string[], colorByPlayer: Record<string, string>): ImpactChartColors {
  const roles = new Map(IMPACT_TOKENS.map((t) => [t, resolveToken(t)]))
  const theme = getEChartsThemeColors()
  return {
    players: players.map((p) => colorByPlayer[p] ?? theme.axisLabel),
    role: (t) => roles.get(t) ?? theme.axisLabel,
    theme,
  }
}

/** Demi-écart entre deux segments voisins (2 px de surface au total). */
const SEG_GAP = 1
/** Distance de la pastille du joueur sous la zone de tracé. */
const PASTILLE_DY = 8
const CORNER = 3
/** Largeur de barre maximale et minimale, part de la bande occupée par le groupe. */
const BAR_MAX = 14
const BAR_MIN = 4
const GROUP_SHARE = 0.72
/** Rang de la série des barres dans `series` (la seule qui ait une infobulle). */
const BARS_SERIES = 1

/** Position de la barre d'un joueur dans le groupe d'une soirée (barres et courbes la partagent). */
function barGeom(api: CustomApi, evening: number, player: number, players: number, narrow: boolean) {
  const band = (api.size([1, 0]) as number[])[0]
  const gap = narrow ? 2 : 3
  const barW = Math.max(BAR_MIN, Math.min(BAR_MAX, (band * GROUP_SHARE - (players - 1) * gap) / players))
  const groupW = players * barW + (players - 1) * gap
  const x = api.coord([evening, 0])[0] - groupW / 2 + player * (barW + gap)
  return { x, barW, gap, mid: x + barW / 2 }
}

/** Les rectangles d'une pile ; le dernier segment porte les coins arrondis du bout. */
function segmentRects(
  api: CustomApi, evening: number, x: number, w: number, segs: ImpactSegment[], c: ImpactChartColors, up: boolean,
) {
  const out: unknown[] = []
  segs.forEach((s, i) => {
    const y0 = api.coord([evening, s.from])[1]
    const y1 = api.coord([evening, s.to])[1]
    const h = Math.abs(y1 - y0) - 2 * SEG_GAP
    if (h <= 0) return
    const last = i === segs.length - 1
    const r = last ? (up ? [CORNER, CORNER, 0, 0] : [0, 0, CORNER, CORNER]) : 0
    out.push({ type: 'rect', shape: { x, y: Math.min(y0, y1) + SEG_GAP, width: w, height: h, r }, style: { fill: c.role(s.token) } })
  })
  return out
}

/** L'infobulle d'une barre (données échappées). */
function barTip(bar: ImpactBar, c: ImpactChartColors, noRole: string, netLabel: string): string {
  const tc = c.theme
  const t = bar.tip
  const head =
    `<div style="color:${tc.axisLabel};margin-bottom:6px"><span style="color:${tc.text};font-weight:500">` +
    `${escapeHtml(t.player)}</span> · ${escapeHtml(t.evening)} · ${escapeHtml(t.matches)}</div>`
  const rows = t.roles.length
    ? t.roles
        .map(
          (r) =>
            `<div style="display:flex;align-items:center;gap:8px;margin-top:4px">` +
            `<span style="width:12px;height:4px;border-radius:1px;background:${c.role(r.token)}"></span>` +
            `<span>${escapeHtml(r.label)} ×${r.count}</span>` +
            `<span style="color:${tc.axisLabel}">· ${escapeHtml(r.points)}</span></div>`,
        )
        .join('')
    : `<div style="color:${tc.axisLabel}">${escapeHtml(noRole)}</div>`
  const net =
    `<div style="border-top:1px solid ${tc.axisLine};margin-top:6px;padding-top:5px">` +
    `${escapeHtml(netLabel)} <b>${escapeHtml(t.net)}</b></div>`
  return head + rows + net
}

/** La série des barres : piles, cible de survol de la colonne, pastille du joueur. */
function barsSeries(view: ImpactHistoryView, c: ImpactChartColors, narrow: boolean) {
  const n = view.players.length
  return {
    type: 'custom' as const,
    data: view.bars.map((b) => [b.evening, b.player, b.gainTotal, b.lossTotal, b.net]),
    encode: { x: 0, y: [2, 3] },
    renderItem: (params: { dataIndex: number; coordSys: { y: number; height: number } }, api: CustomApi) => {
      const bar = view.bars[params.dataIndex]
      if (!bar) return null
      const cs = params.coordSys
      const { x, barW, gap, mid } = barGeom(api, bar.evening, bar.player, n, narrow)
      return {
        type: 'group',
        children: [
          {
            type: 'rect',
            shape: { x: x - gap / 2, y: cs.y, width: barW + gap, height: cs.height + PASTILLE_DY + 6 },
            style: { fill: 'transparent' },
            emphasis: { style: { fill: c.theme.splitAreaA } },
          },
          ...segmentRects(api, bar.evening, x, barW, bar.gains, c, true),
          ...segmentRects(api, bar.evening, x, barW, bar.losses, c, false),
          {
            type: 'circle',
            shape: { cx: mid, cy: cs.y + cs.height + PASTILLE_DY, r: Math.min(3.5, barW / 2 + 0.5) },
            style: { fill: c.players[bar.player] },
          },
        ],
      }
    },
  }
}

/** La série des courbes du net : halo, trait, losanges, et les deux nets extrêmes écrits. */
function netSeries(view: ImpactHistoryView, c: ImpactChartColors, narrow: boolean, signed: (v: number) => string) {
  const n = view.players.length
  const tc = c.theme
  return {
    type: 'custom' as const,
    silent: true,
    z: 5,
    data: view.nets.map((row, p) => [0, p, Math.min(...row), Math.max(...row)]),
    encode: { x: 0, y: [2, 3] },
    renderItem: (params: { coordSys: { x: number } }, api: CustomApi) => {
      const p = api.value(1) as number
      const r = narrow ? 4 : 5
      const color = c.players[p]
      const pts = view.nets[p].map((v, e) => [barGeom(api, e, p, n, narrow).mid, api.coord([e, v])[1]])
      const children: unknown[] = [
        { type: 'polyline', shape: { points: pts }, style: { fill: null, stroke: tc.card, lineWidth: 5, opacity: 0.85, lineJoin: 'round', lineCap: 'round' } },
        { type: 'polyline', shape: { points: pts }, style: { fill: null, stroke: color, lineWidth: 2, lineJoin: 'round', lineCap: 'round' } },
      ]
      pts.forEach(([px, py], e) => {
        children.push({
          type: 'polygon',
          shape: { points: [[px, py - r], [px + r, py], [px, py + r], [px - r, py]] },
          style: { fill: color, stroke: tc.card, lineWidth: 2 },
        })
        if (!view.extremes.has(pointKey(p, e))) return
        const left = p === 0 && px - params.coordSys.x > 34
        children.push({
          type: 'text',
          style: {
            text: signed(view.nets[p][e]),
            x: left ? px - r - 5 : px + r + 5,
            y: py,
            align: left ? 'right' : 'left',
            verticalAlign: 'middle',
            fill: tc.text,
            fontSize: 11,
            fontWeight: 500,
            backgroundColor: tc.card,
            padding: [1, 3],
            borderRadius: 3,
          },
        })
      })
      return { type: 'group', children }
    },
  }
}

export interface ImpactChartText {
  signed: (v: number) => string
  noRole: string
  net: string
}

export function buildImpactHistoryOption(
  view: ImpactHistoryView,
  c: ImpactChartColors,
  t: ImpactChartText,
  narrow: boolean,
): EChartsCoreOption {
  const tc = c.theme
  const count = view.axisLabels.length
  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: { left: narrow ? 36 : 44, right: 6, top: 16, bottom: view.twoLineAxis ? 58 : 44 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      confine: true,
      formatter: (p: { seriesIndex?: number; dataIndex: number }) => {
        const bar = p.seriesIndex === BARS_SERIES ? view.bars[p.dataIndex] : undefined
        return bar ? barTip(bar, c, t.noRole, t.net) : ''
      },
    },
    xAxis: {
      type: 'category',
      data: view.axisLabels,
      axisTick: { show: false },
      axisLine: { show: false },
      axisLabel: {
        color: tc.axisLabel,
        fontSize: narrow ? 10 : 11,
        lineHeight: 13,
        margin: 18,
        interval: (i: number) => showAxisLabel(i, count, narrow),
      },
    },
    yAxis: {
      type: 'value',
      splitNumber: 4,
      axisLabel: { color: tc.axisLabel, fontSize: 11, formatter: (v: number) => t.signed(v) },
      splitLine: { lineStyle: { color: tc.splitLine, width: 1 } },
    },
    series: [
      {
        type: 'line',
        data: [],
        silent: true,
        markLine: {
          silent: true,
          symbol: 'none',
          label: { show: false },
          data: [{ yAxis: 0 }],
          lineStyle: { color: tc.axisLabel, width: 1, type: 'solid', opacity: 0.6 },
          animation: false,
        },
      },
      barsSeries(view, c, narrow),
      netSeries(view, c, narrow, t.signed),
    ],
    legend: { show: false },
    aria: { enabled: true, label: { description: view.aria } },
  }
}
