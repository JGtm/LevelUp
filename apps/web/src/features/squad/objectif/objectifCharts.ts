/**
 * objectifCharts.ts — les deux graphes de la section « Objectif » (lot L3 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), en options ECharts, d'après la maquette C3EW :
 *
 *   - « Rapport de force au fil de la session » : trois courbes cumulées (une par rôle), les
 *     parts par match en petits points pâles (taille = volume du lobby), le point final grossi
 *     et la valeur au bout de chaque courbe, le trait 50 %, puis SOUS L'AXE l'heure, la bande
 *     de résultats (victoire / défaite, encoche de dominance), la carte et le mode ;
 *   - « Rapport de force, soirée après soirée » : dans `eveningsChart.ts` (sorti à la revue L6.1,
 *     seuil de 500 lignes), sur les briques exportées ici.
 *
 * Deux grilles par graphe : la grille du haut porte les courbes, celle du bas la bande de
 * résultats — une même abscisse, donc un alignement exact au pixel. Les marges gauche / droite
 * sont FIXES (pas de `containLabel`) pour que les deux grilles tombent l'une sous l'autre.
 *
 * Toutes les couleurs arrivent résolues (jetons) : aucune valeur en dur ici.
 *
 * Les briques exportées (axe en %, trait 50 %, courbe cumulée, infobulle, API `renderItem`,
 * colonne de ce soir) servent aussi au « Contrôle des ressources au fil de la session » de l'onglet
 * Emprise (`emprise/empriseCharts.ts`, lot L5) : une seule grammaire de courbe cumulée.
 */
import type { EChartsCoreOption } from 'echarts/core'

import { CHART_BG, escapeHtml, getTooltipBase } from '@/components/charts/_utils'
import { withEndPoint } from '@/components/charts/endPoint'
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import { roleToken } from '@/features/_shared/usage/usageMetricKinds'
import { resolveToken } from '@/lib/accessibility'
import { getEChartsThemeColors, type EChartsThemeColors } from '@/lib/echarts/themeColors'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import { OBJECTIVE_ROLES, type ObjectiveRole } from '../formes/model/objectives'
import type { FilMatch } from './objectif.logic'

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
export function tip(text: string): string {
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

/** Le nom d'un match dans les infobulles : « 21:40 · Aquarius ». */
type MatchNamer = (m: FilMatch) => string

/** Les deux séries d'un rôle : la courbe cumulée (point final grossi) et les parts par match. */
function roleFilSeries(
  matches: FilMatch[],
  role: ObjectiveRole,
  first: boolean,
  ctx: { c: ObjectifChartColors; t: FilChartText; matchName: MatchNamer },
) {
  const { c, t, matchName } = ctx
  const tc = c.theme
  const color = c.roles[role]
  const maxLobby = Math.max(0, ...matches.map((m) => m.roles[role].lobby))
  const cum = matches.map((m) => (m.roles[role].cumulative == null ? null : m.roles[role].cumulative! * 100))
  const data = withEndPoint(cum, tc.card, { color })
  const line = lineSeries(t.roles[role], data, color, t.pctFmt, first ? { markLine: parityLine(c.parity) } : {})
  const dots = {
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
  }
  return [line, dots]
}

/** La bande de résultats sous l'axe : une case par match, l'encoche du drapeau de dominance. */
function filBandSeries(matches: FilMatch[], c: ObjectifChartColors, t: FilChartText, matchName: MatchNamer) {
  const tc = c.theme
  return {
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
  }
}

/** Les deux abscisses : l'heure sous les courbes, la carte et le mode sous la bande. */
function filXAxes(matches: FilMatch[], categories: string[], tc: EChartsThemeColors, t: FilChartText) {
  return [
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
  ]
}

export function buildFilOption(matches: FilMatch[], c: ObjectifChartColors, t: FilChartText): EChartsCoreOption {
  const tc = c.theme
  const categories = matches.map((m) => m.matchId)
  const matchName = (m: FilMatch) => `${t.timeOf(m.startTime)} · ${m.map}`
  const series: unknown[] = [
    ...OBJECTIVE_ROLES.flatMap((role, ri) => roleFilSeries(matches, role, ri === 0, { c, t, matchName })),
    filBandSeries(matches, c, t, matchName),
  ]

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
    xAxis: filXAxes(matches, categories, tc, t),
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

/**
 * La colonne de ce soir (la dernière catégorie), grisée sur toute la hauteur de l'axe 0-100,
 * sous les courbes (maquette : fond atténué). Partagée avec « Contrôle des ressources, soirée
 * après soirée » de l'onglet Emprise.
 */
export function tonightColumn(n: number, fill: string) {
  return {
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
      return { type: 'rect', shape: { x: cx - w / 2, y: yTop, width: w, height: yBottom - yTop }, style: { fill } }
    },
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
