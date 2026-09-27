/**
 * objectifCharts.ts — les deux graphes de la section « Objectif » (lot L3 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), en options ECharts, d'après la maquette C3EW :
 *
 *   - « Rapport de force au fil de la session » : trois courbes cumulées (une par rôle), les
 *     parts par match en petits points pâles (taille = volume du lobby), le point final grossi
 *     et la valeur au bout de chaque courbe, le trait 50 %, puis SOUS L'AXE l'heure, la bande
 *     de résultats (victoire / défaite, encoche de dominance), la carte et le mode ;
 *   - « Rapport de force, soirée après soirée » : trois courbes (une soirée par point), ce soir à
 *     droite dans une colonne grisée, la médiane des soirées précédentes en pointillé fin de la
 *     couleur de chaque courbe, le trait 50 %, et sous chaque soirée : la date, la barre
 *     victoires / défaites, « x sur y » et le mélange de modes.
 *
 * Deux grilles par graphe : la grille du haut porte les courbes, celle du bas la bande de
 * résultats — une même abscisse, donc un alignement exact au pixel. Les marges gauche / droite
 * sont FIXES (pas de `containLabel`) pour que les deux grilles tombent l'une sous l'autre.
 *
 * Toutes les couleurs arrivent résolues (jetons) : aucune valeur en dur ici.
 *
 * Les briques exportées (axe en %, trait 50 %, point final grossi, courbe cumulée, API
 * `renderItem`) servent aussi au « Contrôle des ressources au fil de la session » de l'onglet
 * Emprise (`emprise/empriseCharts.ts`, lot L5) : une seule grammaire de courbe cumulée.
 */
import type { EChartsCoreOption } from 'echarts/core'

import { CHART_BG, escapeHtml, getTooltipBase } from '@/components/charts/_utils'
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import { roleToken } from '@/features/_shared/usage/usageMetricKinds'
import { resolveToken } from '@/lib/accessibility'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import { OBJECTIVE_ROLES, type ObjectiveRole } from '../formes/model/objectives'
import type { EveningPoint, FilMatch } from './objectif.logic'

/** Les couleurs résolues des deux graphes. */
export interface ObjectifChartColors {
  roles: Record<ObjectiveRole, string>
  win: string
  loss: string
  /** Le trait 50 % (jeton `warning`). */
  parity: string
  dominance: (d: DominanceValue) => string
  theme: EChartsThemeColors
}

/** Marges fixes des grilles (maquette : 36 px à gauche, place de la valeur de fin à droite). */
const GRID_LEFT = 36
const GRID_RIGHT = 46
/** Hauteur réservée sous le graphe (heure, bande, carte, mode). */
const FOOT = 62
const BAND_H = 8

/** Une infobulle multiligne, échappée. */
function tip(text: string): string {
  return text.split('\n').map(escapeHtml).join('<br>')
}

export function yAxisPct(pctFmt: (v: number) => string, tc: EChartsThemeColors, gridIndex = 0) {
  return {
    gridIndex,
    type: 'value' as const,
    min: 0,
    max: 100,
    interval: 25,
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { lineStyle: { color: tc.splitLine } },
    axisLabel: {
      color: tc.axisLabel,
      fontSize: 10.5,
      formatter: (v: number) => (v === 100 ? pctFmt(100) : String(v)),
    },
  }
}

/** Le trait 50 % (repère, jamais une donnée), posé sur la première courbe. */
export function parityLine(color: string) {
  return {
    silent: true,
    symbol: 'none',
    label: { show: false },
    lineStyle: { color, type: [4, 3] as number[], width: 1.5 },
    data: [{ yAxis: 50 }],
  }
}

/** Le point final grossi, liseré à la couleur de la carte (maquette : r 4,5). */
export function endPoint(value: number, color: string, card: string) {
  return { value, symbol: 'circle', symbolSize: 9, itemStyle: { color, borderColor: card, borderWidth: 2 } }
}

export function lineSeries(
  name: string,
  data: unknown[],
  color: string,
  pctFmt: (v: number) => string,
  extra: Record<string, unknown> = {},
) {
  return {
    name,
    type: 'line' as const,
    xAxisIndex: 0,
    yAxisIndex: 0,
    data,
    symbol: 'none',
    connectNulls: true,
    lineStyle: { color, width: 2.5 },
    itemStyle: { color },
    endLabel: {
      show: true,
      color,
      fontSize: 12,
      fontWeight: 700,
      distance: 6,
      formatter: (p: { value?: unknown }) => (typeof p.value === 'number' ? pctFmt(p.value) : ''),
    },
    labelLayout: { moveOverlap: 'shiftY' as const },
    z: 3,
    ...extra,
  }
}

// ---------------------------------------------------------------------------
// Au fil de la session
// ---------------------------------------------------------------------------

export interface FilChartText {
  roles: Record<ObjectiveRole, string>
  pctFmt: (v: number) => string
  countFmt: (v: number, duration: boolean) => string
  timeOf: (iso: string) => string
  familyLabel: (family: string) => string
  /** « Drapeau, défaite 1–3 » (mode seul sans résultat connu). */
  contextOf: (m: FilMatch) => string
  dominanceLabel: (d: DominanceValue) => string
  pointTip: (v: {
    match: string
    context: string
    role: string
    value: string
    lobby: string
    pct: string
    cumulative: string
  }) => string
  bandTip: (match: string, context: string, dominance: string | null) => string
}

/** Le nom court d'une carte sous l'axe (maquette : au-delà de 11 signes, 10 + point). */
export function shortMap(map: string): string {
  return map.length > 11 ? `${map.slice(0, 10)}.` : map
}

/** Rayon du point d'un match : 2 px + jusqu'à 5 px selon son volume (maquette). */
export function volumeRadius(lobby: number, maxLobby: number): number {
  return maxLobby > 0 ? 2 + Math.sqrt(lobby / maxLobby) * 5 : 2
}

export function buildFilOption(matches: FilMatch[], c: ObjectifChartColors, t: FilChartText): EChartsCoreOption {
  const tc = c.theme
  const categories = matches.map((m) => m.matchId)
  const matchName = (m: FilMatch) => `${t.timeOf(m.startTime)} · ${m.map}`
  const series: unknown[] = []

  OBJECTIVE_ROLES.forEach((role, ri) => {
    const color = c.roles[role]
    const maxLobby = Math.max(0, ...matches.map((m) => m.roles[role].lobby))
    const cum = matches.map((m) => (m.roles[role].cumulative == null ? null : m.roles[role].cumulative! * 100))
    let last = -1
    cum.forEach((v, i) => {
      if (v != null) last = i
    })
    const data = cum.map((v, i) => (i === last && v != null ? endPoint(v, color, tc.card) : v))
    series.push(
      lineSeries(t.roles[role], data, color, t.pctFmt, ri === 0 ? { markLine: parityLine(c.parity) } : {}),
    )
    series.push({
      name: t.roles[role],
      type: 'scatter' as const,
      xAxisIndex: 0,
      yAxisIndex: 0,
      z: 2,
      data: matches.map((m, i) => {
        const r = m.roles[role]
        if (r.share == null) return null
        return {
          value: [i, r.share * 100],
          symbolSize: 2 * volumeRadius(r.lobby, maxLobby),
          tip: t.pointTip({
            match: matchName(m),
            context: t.contextOf(m),
            role: t.roles[role],
            value: t.countFmt(r.us, role === 'hold'),
            lobby: t.countFmt(r.lobby, role === 'hold'),
            pct: t.pctFmt(r.share * 100),
            cumulative: r.cumulative == null ? '—' : t.pctFmt(r.cumulative * 100),
          }),
        }
      }),
      itemStyle: { color, opacity: 0.45, borderColor: tc.card, borderWidth: 1 },
    })
  })

  series.push({
    type: 'custom' as const,
    xAxisIndex: 1,
    yAxisIndex: 1,
    data: matches.map((m, i) => ({
      value: [i, 0],
      tip: t.bandTip(matchName(m), t.contextOf(m), m.dominance ? t.dominanceLabel(m.dominance) : null),
    })),
    renderItem: (_params: unknown, api: CustomApi) => {
      const i = api.value(0) as number
      const m = matches[i]
      const [cx, cy] = api.coord([i, 0])
      const w = (api.size([1, 0]) as number[])[0] - 3
      const fill = m.outcome === 'win' ? c.win : m.outcome === 'loss' ? c.loss : tc.splitLine
      const children: unknown[] = [
        { type: 'rect', shape: { x: cx - w / 2, y: cy - BAND_H / 2, width: w, height: BAND_H, r: 2 }, style: { fill } },
      ]
      if (m.dominance) {
        const h = BAND_H + 6
        children.push(
          { type: 'rect', shape: { x: cx - 2.5, y: cy - h / 2, width: 5, height: h }, style: { fill: tc.card } },
          { type: 'rect', shape: { x: cx - 1.5, y: cy - h / 2, width: 3, height: h }, style: { fill: c.dominance(m.dominance) } },
        )
      }
      return { type: 'group', children }
    },
  })

  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: [
      { left: GRID_LEFT, right: GRID_RIGHT, top: 10, bottom: FOOT },
      { left: GRID_LEFT, right: GRID_RIGHT, bottom: FOOT - 8 - BAND_H, height: BAND_H },
    ],
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: { data?: { tip?: string }; seriesType?: string; seriesName?: string; value?: unknown }) => {
        if (p.data?.tip) return tip(p.data.tip)
        if (p.seriesType === 'line' && typeof p.value === 'number') return tip(`${p.seriesName} : ${t.pctFmt(p.value)}`)
        return ''
      },
    },
    xAxis: [
      {
        gridIndex: 0,
        type: 'category',
        data: categories,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          color: tc.axisLabel,
          fontSize: 9.5,
          margin: 4,
          interval: 0,
          formatter: (_v: string, i: number) => t.timeOf(matches[i]?.startTime ?? ''),
        },
      },
      {
        gridIndex: 1,
        type: 'category',
        data: categories,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          interval: 0,
          margin: 6,
          lineHeight: 12,
          formatter: (_v: string, i: number) =>
            `{map|${shortMap(matches[i]?.map ?? '')}}\n{mode|${t.familyLabel(matches[i]?.family ?? '')}}`,
          rich: {
            map: { color: tc.text, fontSize: 9.5 },
            mode: { color: tc.axisLabel, fontSize: 9.5 },
          },
        },
      },
    ],
    yAxis: [yAxisPct(t.pctFmt, tc), { gridIndex: 1, type: 'value', min: -1, max: 1, show: false }],
    series,
    // Jamais plus de trois courbes : la légende est rendue HORS canvas (pied de carte).
    legend: { show: false },
    aria: { enabled: true },
  }
}

/** Le sous-ensemble de l'API `renderItem` d'ECharts utilisé ici. */
export interface CustomApi {
  value: (dim: number) => unknown
  coord: (v: number[]) => number[]
  size: (v: number[]) => unknown
}

// ---------------------------------------------------------------------------
// Soirée après soirée
// ---------------------------------------------------------------------------

export interface EveningsChartText {
  roles: Record<ObjectiveRole, string>
  pctFmt: (v: number, digits?: number) => string
  tonight: string
  dateOf: (iso: string) => string
  outOfFmt: (wins: number, matches: number) => string
  mixOf: (p: EveningPoint) => string
  pointTip: (role: string, evening: string, value: string, median: string | null) => string
  bandTip: (evening: string, wins: number, matches: number, mix: string) => string
  eveningOf: (date: string) => string
  medianTip: (role: string, value: string) => string
}

const EVENING_FOOT = 64

export function buildEveningsOption(
  points: EveningPoint[],
  medians: Record<ObjectiveRole, number | null>,
  c: ObjectifChartColors,
  t: EveningsChartText,
): EChartsCoreOption {
  const tc = c.theme
  const n = points.length
  const categories = points.map((_, i) => String(i))
  const eveningName = (p: EveningPoint) => (p.current ? t.tonight : t.eveningOf(t.dateOf(p.startTime)))
  const series: unknown[] = []

  OBJECTIVE_ROLES.forEach((role, ri) => {
    const color = c.roles[role]
    const med = medians[role]
    const data = points.map((p, i) => {
      const v = p.shares[role]
      if (v == null) return null
      const item = {
        value: v,
        symbol: 'circle',
        symbolSize: i === n - 1 ? 10 : 6,
        itemStyle: { color, borderColor: tc.card, borderWidth: 1.5 },
        tip: t.pointTip(t.roles[role], eveningName(p), t.pctFmt(v, 1), med == null ? null : t.pctFmt(med, 1)),
      }
      return item
    })
    const extra: Record<string, unknown> = {
      symbol: 'circle',
      markLine: {
        symbol: 'none',
        label: { show: false },
        lineStyle: { color, type: [2, 3] as number[], width: 1 },
        data: [
          ...(med == null ? [] : [{ yAxis: med, tip: t.medianTip(t.roles[role], t.pctFmt(med, 1)) }]),
          ...(ri === 0 ? [{ yAxis: 50, lineStyle: { color: c.parity, type: [4, 3], width: 1.5 }, silent: true }] : []),
        ],
      },
    }
    series.push(lineSeries(t.roles[role], data, color, (v) => t.pctFmt(v), extra))
  })

  // La colonne de ce soir, grisée (maquette : muted à 75 %), sous les courbes.
  series.push({
    type: 'custom' as const,
    xAxisIndex: 0,
    yAxisIndex: 0,
    silent: true,
    z: 0,
    data: [[n - 1, 0]],
    renderItem: (_params: unknown, api: CustomApi) => {
      const [cx, yBottom] = api.coord([n - 1, 0])
      const yTop = api.coord([n - 1, 100])[1]
      const w = (api.size([1, 0]) as number[])[0]
      return { type: 'rect', shape: { x: cx - w / 2, y: yTop, width: w, height: yBottom - yTop }, style: { fill: tc.splitAreaB } }
    },
  })

  series.push({
    type: 'custom' as const,
    xAxisIndex: 1,
    yAxisIndex: 1,
    data: points.map((p, i) => ({ value: [i, 0], tip: t.bandTip(eveningName(p), p.wins, p.matches, t.mixOf(p)) })),
    renderItem: (_params: unknown, api: CustomApi) => {
      const i = api.value(0) as number
      const p = points[i]
      const [cx, cy] = api.coord([i, 0])
      const bw = (api.size([1, 0]) as number[])[0] - 6
      const wv = p.matches > 0 ? (p.wins / p.matches) * bw : 0
      const x0 = cx - bw / 2
      const h = 10
      const children: unknown[] = []
      if (p.wins > 0) children.push({ type: 'rect', shape: { x: x0, y: cy - h / 2, width: wv, height: h, r: 2 }, style: { fill: c.win } })
      if (p.wins < p.matches) {
        children.push({ type: 'rect', shape: { x: x0 + wv, y: cy - h / 2, width: bw - wv, height: h, r: 2 }, style: { fill: c.loss } })
      }
      return { type: 'group', children }
    },
  })

  return {
    backgroundColor: CHART_BG,
    animation: false,
    grid: [
      { left: 40, right: 52, top: 12, bottom: EVENING_FOOT },
      { left: 40, right: 52, bottom: EVENING_FOOT - 8 - 10, height: 10 },
    ],
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: { data?: { tip?: string } }) => (p.data?.tip ? tip(p.data.tip) : ''),
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
          formatter: (_v: string, i: number) =>
            points[i]?.current ? `{cur|${t.tonight}}` : `{d|${t.dateOf(points[i]?.startTime ?? '')}}`,
          rich: {
            d: { color: tc.axisLabel, fontSize: 10.5 },
            cur: { color: tc.text, fontSize: 10.5, fontWeight: 600 },
          },
        },
      },
      {
        gridIndex: 1,
        type: 'category',
        data: categories,
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: {
          interval: 0,
          margin: 6,
          lineHeight: 13,
          formatter: (_v: string, i: number) =>
            `{n|${t.outOfFmt(points[i]?.wins ?? 0, points[i]?.matches ?? 0)}}\n{m|${points[i] ? t.mixOf(points[i]) : ''}}`,
          rich: {
            n: { color: tc.text, fontSize: 9.5 },
            m: { color: tc.axisLabel, fontSize: 9.5 },
          },
        },
      },
    ],
    yAxis: [yAxisPct((v) => t.pctFmt(v), tc), { gridIndex: 1, type: 'value', min: -1, max: 1, show: false }],
    series,
    legend: { show: false },
    aria: { enabled: true },
  }
}

/**
 * resolveObjectifColors — les couleurs des deux graphes, résolues depuis les jetons AU MOMENT
 * du rendu (appelé dans `buildOption`, que `ChartCard` rejoue à chaque changement de thème ou
 * de palette).
 */
export function resolveObjectifColors(): ObjectifChartColors {
  return {
    roles: {
      take: resolveToken(roleToken('take')),
      defend: resolveToken(roleToken('defend')),
      hold: resolveToken(roleToken('hold')),
    },
    win: resolveToken('outcome-win'),
    loss: resolveToken('outcome-loss'),
    parity: resolveToken('warning'),
    dominance: (d) => resolveToken(DOMINANCE_COLOR_TOKENS[d]),
    theme: getEChartsThemeColors(),
  }
}
