// cross-feature-allow: jetons de couleur des joueurs d'escouade (`SQUAD_MAIN_PLAYER_TOKEN`,
// `SQUAD_TEAMMATE_COLOR_TOKENS`) partagés avec la page Escouade — une seule palette d'identité
// joueur dans l'app, celle de la pastille et du sélecteur.
/**
 * tendancesSquadCharts.logic — les définitions PURES des six graphiques de la vue Escouade :
 * de la réponse de l'API, de l'horizon et du pas, à UNE grille de graphiques.
 *
 * ORDRE : taux de victoire avec et sans l'escouade, FDA par membre, part des frags de l'escouade
 * par membre (barres empilées), part des frags de l'équipe, écart de MMR (seulement si le titre
 * mesure le MMR), matchs joués ensemble.
 *
 * LES RÈGLES SONT CELLES DE LA VUE SOLO (`tendancesCharts.logic`) : un point appartient à
 * l'horizon dans `]as_of − horizon, as_of]`, une courbe de moins de 2 points est retirée, un
 * graphique sans courbe n'existe pas. Chaque courbe pleine porte la moyenne de la période
 * d'avant. Au pas « match », le taux de victoire, la part par membre (barres) et le nombre de
 * matchs sont absents : une partie ne donne ni taux ni mélange.
 *
 * COULEURS PAR MEMBRE : le joueur principal prend `SQUAD_MAIN_PLAYER_TOKEN`, les coéquipiers
 * `SQUAD_TEAMMATE_COLOR_TOKENS` dans l'ordre de l'API (principal en premier) — celles de la
 * pastille et du sélecteur, partagées par la courbe de FDA et les barres empilées.
 */
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { SQUAD_MAIN_PLAYER_TOKEN, SQUAD_TEAMMATE_COLOR_TOKENS } from '@/features/squad/colors'
import type { SemanticToken } from '@/lib/accessibility'
import type { TrendsIndicator, TrendsPageResponse } from '@/lib/api/types'

import { getTendancesText } from './i18n'
import { pointsInHorizon } from './tendances.logic'
import {
  assemble,
  compact,
  curveOf,
  indicatorsOf,
  MIN_POINTS,
  type Pending,
  type TendancesChartDef,
  type TendancesChartsParams,
} from './tendancesCharts.logic'
import { intervalCategory } from './tendancesMix.logic'

/** Indicateurs de la vue Escouade (clés de l'API). */
const KEY = {
  winRate: 'win_rate',
  winRateAlone: 'win_rate_alone',
  matchCount: 'match_count',
  teamShare: 'squad_share_of_team_kills',
  mmrGap: 'mmr_gap',
  kda: 'kda',
  memberShare: 'member_share_of_squad_kills',
} as const

/** Groupe des lignes par membre (variante = gamertag). */
const MEMBERS_GROUP = 'members'

/** Repères horizontaux : taux de victoire neutre, FDA et écart de MMR nuls. */
const REFERENCE = { winRate: 0.5, zero: 0 } as const

/** Une part (0..1) s'écrit en pourcentage entier dans les barres. */
const PERCENT = 100

/** Jetons de couleur des deux courbes du premier graphique (stables, même si l'une manque). */
const WITH_SQUAD_TOKEN: SemanticToken = 'chart-series-1'
const ALONE_TOKEN: SemanticToken = 'chart-series-2'

/** Jetons de joueur, dans l'ordre de l'API : principal, puis coéquipiers. */
const MEMBER_TOKENS: readonly SemanticToken[] = [
  SQUAD_MAIN_PLAYER_TOKEN,
  ...SQUAD_TEAMMATE_COLOR_TOKENS,
]

/** Un graphique en barres empilées (une barre par intervalle, un composant par membre). */
export interface TendancesStackedChartDef {
  id: string
  title: string
  info?: string
  stacked: {
    series: ChartSeries<ChartPointStacked>[]
    /** Couleur de chaque composant, par nom affiché (le gamertag). */
    componentColors: Record<string, SemanticToken>
    /** Ordre des composants : celui des membres. */
    componentOrder: string[]
  }
}

export type TendancesAnyChartDef = TendancesChartDef | TendancesStackedChartDef

/** Une grille de graphiques, de l'une ou l'autre vue (`title` vide = pas d'intertitre). */
export interface TendancesAnyGridDef {
  id: string
  title: string
  charts: TendancesAnyChartDef[]
}

/** Vrai pour un graphique en barres empilées (sinon : courbes). */
export function isStackedChart(def: TendancesAnyChartDef): def is TendancesStackedChartDef {
  return 'stacked' in def
}

/** Les lignes d'une clé dans le groupe des membres (une par membre). */
function memberRows(data: TrendsPageResponse, key: string): TrendsIndicator[] {
  return (data.indicators ?? []).filter((i) => i.key === key && i.group === MEMBERS_GROUP)
}

/** Les membres, dans l'ordre de l'API (le joueur principal en premier). */
function memberOrder(data: TrendsPageResponse): string[] {
  const names: string[] = []
  for (const row of [...memberRows(data, KEY.kda), ...memberRows(data, KEY.memberShare)]) {
    const name = row.variant ?? ''
    if (name && !names.includes(name)) names.push(name)
  }
  return names
}

/** Jeton de joueur d'un membre selon sa place dans l'ordre de l'API. */
function memberToken(order: readonly string[], name: string): SemanticToken {
  return MEMBER_TOKENS[Math.max(0, order.indexOf(name)) % MEMBER_TOKENS.length]
}

/** Un graphique de courbes, ou `undefined` quand aucune courbe n'a 2 points sur l'horizon. */
function linesChart(
  p: TendancesChartsParams,
  def: { id: string; title: string; info?: string; reference?: number },
  pendings: Pending[],
): TendancesChartDef | undefined {
  const assembled = assemble(p, pendings)
  if (!assembled) return undefined
  return {
    id: def.id,
    title: def.title,
    info: def.info,
    input: {
      ...assembled.base,
      curves: assembled.curves,
      ...(def.reference !== undefined ? { reference: def.reference } : {}),
    },
  }
}

/** Courbe pleine (avec la moyenne de la période d'avant) d'un indicateur à ligne unique. */
function singleCurve(p: TendancesChartsParams, key: string, name: string, color?: SemanticToken) {
  return curveOf(indicatorsOf(p.data, key)[0], p, { name, color, withPrevMean: true })
}

/**
 * Arrondit des parts (0..1) en pourcentages entiers dont la somme vaut l'arrondi de la somme
 * × 100 : parties entières d'abord, puis les unités manquantes aux plus grands restes (à égalité,
 * le premier dans l'ordre).
 */
export function roundSharesToTotal(values: readonly number[]): number[] {
  const scaled = values.map((v) => v * PERCENT)
  const result = scaled.map((v) => Math.floor(v))
  const target = Math.round(scaled.reduce((sum, v) => sum + v, 0))
  let missing = target - result.reduce((sum, v) => sum + v, 0)
  const byRemainder = scaled
    .map((v, index) => ({ index, remainder: v - Math.floor(v) }))
    .sort((a, b) => b.remainder - a.remainder || a.index - b.index)
  for (const { index } of byRemainder) {
    if (missing <= 0) break
    result[index] += 1
    missing -= 1
  }
  return result
}

/** Les parts d'un bâton (par membre) en pourcentages entiers dont la somme est exacte. */
function roundedComponents(shares: Record<string, number>): Record<string, number> {
  const names = Object.keys(shares)
  const rounded = roundSharesToTotal(names.map((name) => shares[name]))
  return Object.fromEntries(names.map((name, i) => [name, rounded[i]]))
}

/**
 * Barres empilées de la part des frags de l'escouade par membre : un bâton par intervalle du
 * pas courant dans l'horizon (le DÉBUT de l'intervalle y est). Absent au pas « match » et
 * quand moins de deux intervalles sont dans l'horizon.
 */
function memberShareChart(
  p: TendancesChartsParams,
  order: readonly string[],
): TendancesStackedChartDef | undefined {
  const step = p.step
  if (step === 'match') return undefined
  const t = getTendancesText(p.locale)
  const rows = memberRows(p.data, KEY.memberShare)

  // Une entrée par instant (clé : l'instant en millisecondes), toutes lignes confondues.
  const buckets = new Map<number, { t: string; shares: Record<string, number> }>()
  for (const row of rows) {
    const name = row.variant ?? ''
    for (const point of pointsInHorizon(row.series[step], p.data.as_of, p.horizon)) {
      const instant = new Date(point.t).getTime()
      const bucket = buckets.get(instant) ?? { t: point.t, shares: {} }
      bucket.shares[name] = point.value
      buckets.set(instant, bucket)
    }
  }
  if (buckets.size < MIN_POINTS) return undefined

  const datapoints: ChartPointStacked[] = [...buckets.entries()]
    .sort(([a], [b]) => a - b)
    .map(([, bucket]) => ({
      category: intervalCategory(bucket.t, step, p.data.timezone, p.locale),
      components: roundedComponents(bucket.shares),
    }))

  const present = new Set(rows.map((r) => r.variant ?? ''))
  const componentOrder = order.filter((name) => present.has(name))
  const componentColors: Record<string, SemanticToken> = {}
  for (const name of componentOrder) componentColors[name] = memberToken(order, name)

  return {
    id: KEY.memberShare,
    title: t.squadChartMemberShare,
    info: t.squadInfoMemberShare,
    stacked: { series: [{ key: KEY.memberShare, datapoints }], componentColors, componentOrder },
  }
}

/**
 * La grille de la vue Escouade pour l'horizon et le pas choisis. Vide (aucune grille) quand
 * aucun graphique ne se trace : la section affiche alors son état vide.
 */
export function buildTendancesSquadGrids(params: TendancesChartsParams): TendancesAnyGridDef[] {
  const p = params
  const t = getTendancesText(p.locale)
  const order = memberOrder(p.data)

  const kdaCurves = memberRows(p.data, KEY.kda).map((row) => {
    const name = row.variant ?? ''
    return curveOf(row, p, { name, color: memberToken(order, name), withPrevMean: true })
  })

  const charts = compact<TendancesAnyChartDef>([
    linesChart(
      p,
      { id: 'win-rate-with-alone', title: t.squadChartWinRate, info: t.squadInfoWinRate, reference: REFERENCE.winRate },
      compact([
        singleCurve(p, KEY.winRate, t.squadSeriesWith, WITH_SQUAD_TOKEN),
        singleCurve(p, KEY.winRateAlone, t.squadSeriesAlone, ALONE_TOKEN),
      ]),
    ),
    linesChart(p, { id: 'kda-by-member', title: t.squadChartKda, reference: REFERENCE.zero }, compact(kdaCurves)),
    memberShareChart(p, order),
    linesChart(
      p,
      { id: KEY.teamShare, title: t.squadChartTeamShare, info: t.squadInfoTeamShare },
      compact([singleCurve(p, KEY.teamShare, p.labelOf(KEY.teamShare))]),
    ),
    p.data.capabilities.mmr
      ? linesChart(
          p,
          { id: KEY.mmrGap, title: t.squadChartMmrGap, info: t.squadInfoMmrGap, reference: REFERENCE.zero },
          compact([singleCurve(p, KEY.mmrGap, p.labelOf(KEY.mmrGap))]),
        )
      : undefined,
    linesChart(
      p,
      { id: KEY.matchCount, title: t.squadChartMatchCount },
      compact([singleCurve(p, KEY.matchCount, p.labelOf(KEY.matchCount))]),
    ),
  ])
  if (charts.length === 0) return []
  return [{ id: 'squad', title: '', charts }]
}
