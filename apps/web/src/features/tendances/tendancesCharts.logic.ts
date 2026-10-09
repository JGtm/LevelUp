// cross-feature-allow: jetons de couleur des chaînes LUSR (`LUSR_GROUP_TOKENS`) partagés avec
// l'onglet Carrière — une seule table de chaînes dans l'app.
/**
 * tendancesCharts.logic — les définitions PURES des graphiques de la section « Évolution » :
 * de la réponse de l'API, de l'horizon et du pas, aux deux grilles de graphiques à tracer.
 *
 * RÈGLES. Un point appartient à l'horizon quand son instant est dans `]as_of − horizon, as_of]`.
 * Une courbe de moins de 2 points est retirée ; un graphique sans courbe de gauche n'existe
 * pas ; une grille sans graphique n'existe pas. Le taux de victoire n'a pas de sens au pas
 * « match » (une partie vaut 0 ou 1) : il y est retiré.
 *
 * LE MMR ADVERSE de la première grille passe sur l'axe de droite de chaque graphique quand le
 * titre sait le mesurer (`capabilities.mmr`) ET que la série a au moins 2 points sur l'horizon ;
 * sinon chaque graphique n'a qu'un axe, et les titres perdent « face au MMR adverse ».
 *
 * La MOYENNE DE LA PÉRIODE D'AVANT (`prev_value` de l'horizon courant) n'accompagne que les
 * courbes pleines de la seconde grille.
 */
import { LUSR_GROUP_TOKENS } from '@/features/career/lusr-chains'
import type { SemanticToken } from '@/lib/accessibility'
import type { TrendsIndicator, TrendsPageResponse, TrendsPoint } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { variantLabel } from './labels'
import { AXIS_INK, type CurveColor, type LinesCurve, type TendancesLinesInput } from './tendancesLines.logic'
import { pointsInHorizon, type Step } from './tendances.logic'

/** Indicateurs dont les courbes sont définies ici (clés de l'API). */
const KEY = {
  kda: 'kda',
  kills: 'kills_per_match',
  deaths: 'deaths_per_match',
  winRate: 'win_rate',
  performance: 'performance_score',
  accuracy: 'accuracy',
  life: 'avg_life_seconds',
  yield: 'offensive_conversion',
  resistance: 'defensive_resistance',
  enemyMmr: 'enemy_mmr',
  teamMmr: 'team_mmr',
  csr: 'csr_value',
  lusr: 'lusr_value',
  damageDealt: 'avg_damage_dealt',
  damageTaken: 'avg_damage_taken',
  take: 'objective_take_share',
  defend: 'objective_defend_share',
  hold: 'objective_hold_share',
  parity: 'objective_parity',
} as const

/** Repères horizontaux de la première grille : valeur neutre de l'indicateur. */
const REFERENCE = { kda: 0, winRate: 0.5, performance: 50, yieldResistance: 1 } as const

/**
 * Indicateurs sans sens au pas « match » : un taux de victoire vaut 0 ou 1 sur une partie, et un
 * nombre de matchs joués ensemble vaut 1 (la vue Escouade y retire ses trois courbes concernées).
 */
const ABSENT_AT_MATCH_STEP: ReadonlySet<string> = new Set([
  KEY.winRate,
  'win_rate_alone',
  'match_count',
])

/** Nombre minimal de points sur l'horizon pour tracer une courbe. */
export const MIN_POINTS = 2

const DAY_MS = 24 * 60 * 60 * 1000

/** Jeton de la courbe du MMR adverse dans la seconde grille (celui de l'axe de droite, `seriesColor(4)`). */
const ENEMY_MMR_TOKEN: SemanticToken = 'chart-series-5'

export interface TendancesChartDef {
  /** Identifiant stable (clé React, test). */
  id: string
  title: string
  /** Bulle d'information du titre, absente quand le graphique n'en porte pas. */
  info?: string
  input: TendancesLinesInput
}

export interface TendancesGridDef {
  id: 'versus-mmr' | 'level'
  title: string
  charts: TendancesChartDef[]
}

export interface TendancesChartsParams {
  data: TrendsPageResponse
  horizon: number
  step: Step
  locale: Locale
  /** Libellé d'un indicateur (clé, variante) : `useIndicatorLabeler`. */
  labelOf: (key: string, variant?: string) => string
}

/** Jeton d'ordre de la n-ième courbe d'un graphique : `chart-series-1`, `-2`... */
function defaultToken(index: number): SemanticToken {
  return `chart-series-${(index % 8) + 1}` as SemanticToken
}

export function indicatorsOf(data: TrendsPageResponse, key: string): TrendsIndicator[] {
  return (data.indicators ?? []).filter((i) => i.key === key)
}

/** Points d'un indicateur sur l'horizon, au pas courant. */
function horizonPointsOf(ind: TrendsIndicator, p: TendancesChartsParams): TrendsPoint[] {
  return pointsInHorizon(ind.series[p.step], p.data.as_of, p.horizon)
}

/** Une courbe en attente de sa couleur d'ordre, avec l'indicateur qui règle l'axe de gauche. */
export interface Pending {
  ind: TrendsIndicator
  name: string
  color?: CurveColor
  points: TrendsPoint[]
  dashed?: boolean
  prevMean?: number
}

export interface CurveOptions {
  name: string
  color?: CurveColor
  dashed?: boolean
  withPrevMean?: boolean
}

/** Courbe d'un indicateur, ou `undefined` quand elle a moins de 2 points sur l'horizon. */
export function curveOf(
  ind: TrendsIndicator | undefined,
  p: TendancesChartsParams,
  options: CurveOptions,
): Pending | undefined {
  if (!ind) return undefined
  if (p.step === 'match' && ABSENT_AT_MATCH_STEP.has(ind.key)) return undefined
  const points = horizonPointsOf(ind, p)
  if (points.length < MIN_POINTS) return undefined
  const prev = options.withPrevMean
    ? ind.horizons?.find((h) => h.days === p.horizon)?.prev_value
    : undefined
  return {
    ind,
    name: options.name,
    color: options.color,
    points,
    dashed: options.dashed,
    prevMean: prev ?? undefined,
  }
}

export function compact<T>(items: (T | undefined)[]): T[] {
  return items.filter((x): x is T => x !== undefined)
}

/**
 * Assemble les courbes d'un graphique : couleur d'ordre pour celles qui n'en imposent pas,
 * unité et décimales de l'axe de gauche prises sur la première. `undefined` sans courbe.
 */
export function assemble(
  p: TendancesChartsParams,
  pendings: Pending[],
): { curves: LinesCurve[]; base: ReturnType<typeof baseInput> } | undefined {
  if (pendings.length === 0) return undefined
  const curves = pendings.map((c, i): LinesCurve => ({
    name: c.name,
    color: c.color ?? defaultToken(i),
    points: c.points,
    ...(c.dashed ? { dashed: true } : {}),
    ...(c.prevMean != null ? { prevMean: c.prevMean } : {}),
  }))
  return { curves, base: baseInput(p, pendings[0].ind) }
}

function baseInput(
  p: TendancesChartsParams,
  first: TrendsIndicator,
): Omit<TendancesLinesInput, 'curves' | 'right' | 'reference'> {
  const to = new Date(p.data.as_of).getTime()
  const text = getTendancesText(p.locale)
  return {
    from: to - p.horizon * DAY_MS,
    to,
    unit: first.unit,
    decimals: first.decimals,
    step: p.step,
    locale: p.locale,
    timeZone: p.data.timezone || undefined,
    labels: { tooltipLine: text.evolutionTooltipLine, weekOf: text.evolutionWeekOf },
  }
}

interface ChartSpec {
  id: string
  /** Libellé du graphique, avant l'éventuel « face au MMR adverse ». */
  label: string
  keys: { key: string; color?: CurveColor }[]
  reference?: number
}

function versusMmrSpecs(p: TendancesChartsParams, t: ReturnType<typeof getTendancesText>): ChartSpec[] {
  const L = (key: string) => p.labelOf(key)
  return [
    { id: KEY.kda, label: L(KEY.kda), keys: [{ key: KEY.kda }], reference: REFERENCE.kda },
    {
      id: KEY.kills,
      label: L(KEY.kills),
      keys: [{ key: KEY.kills, color: 'stat-kills' }],
    },
    {
      id: KEY.deaths,
      label: L(KEY.deaths),
      keys: [{ key: KEY.deaths, color: 'stat-deaths' }],
    },
    {
      id: KEY.winRate,
      label: L(KEY.winRate),
      keys: [{ key: KEY.winRate }],
      reference: REFERENCE.winRate,
    },
    {
      id: KEY.performance,
      label: L(KEY.performance),
      keys: [{ key: KEY.performance }],
      reference: REFERENCE.performance,
    },
    { id: KEY.accuracy, label: L(KEY.accuracy), keys: [{ key: KEY.accuracy }] },
    { id: KEY.life, label: L(KEY.life), keys: [{ key: KEY.life }] },
    {
      id: 'yield-resistance',
      label: t.chartYieldResistance,
      keys: [{ key: KEY.yield }, { key: KEY.resistance }],
      reference: REFERENCE.yieldResistance,
    },
  ]
}

/** Grille « Face au MMR adverse » (« Résultats et combat » sans capacité MMR). */
function versusMmrGrid(p: TendancesChartsParams): TendancesGridDef | undefined {
  const t = getTendancesText(p.locale)
  const enemy = indicatorsOf(p.data, KEY.enemyMmr)[0]
  const enemyPoints = enemy && p.data.capabilities.mmr ? horizonPointsOf(enemy, p) : []
  const withMmr = enemyPoints.length >= MIN_POINTS

  const charts = compact(
    versusMmrSpecs(p, t).map((spec): TendancesChartDef | undefined => {
      const assembled = assemble(
        p,
        compact(
          spec.keys.map((k) =>
            curveOf(indicatorsOf(p.data, k.key)[0], p, { name: p.labelOf(k.key), color: k.color }),
          ),
        ),
      )
      if (!assembled) return undefined
      return {
        id: spec.id,
        title: withMmr ? t.chartVersusMmr(spec.label) : spec.label,
        input: {
          ...assembled.base,
          curves: assembled.curves,
          reference: spec.reference,
          ...(withMmr ? { right: { name: p.labelOf(KEY.enemyMmr), points: enemyPoints } } : {}),
        },
      }
    }),
  )
  if (charts.length === 0) return undefined
  return {
    id: 'versus-mmr',
    title: withMmr ? t.gridVersusMmr : t.gridResults,
    charts,
  }
}

/** Un graphique de la grille « Niveau, combat et style » (jamais de repère : la page n'en pose pas). */
function levelChart(
  p: TendancesChartsParams,
  id: string,
  title: string,
  pendings: Pending[],
  info?: string,
): TendancesChartDef | undefined {
  const assembled = assemble(p, pendings)
  if (!assembled) return undefined
  return { id, title, info, input: { ...assembled.base, curves: assembled.curves } }
}

/** Courbes de toutes les variantes d'une clé (CSR par file, LUSR par type de partie). */
function variantCurves(
  p: TendancesChartsParams,
  key: string,
  nameOf: (variant: string) => string,
  colorOf: (variant: string) => CurveColor | undefined,
): Pending[] {
  return compact(
    indicatorsOf(p.data, key).map((ind) =>
      curveOf(ind, p, {
        name: nameOf(ind.variant ?? ''),
        color: colorOf(ind.variant ?? ''),
        withPrevMean: true,
      }),
    ),
  )
}

function singleCurve(
  p: TendancesChartsParams,
  key: string,
  options: { color?: CurveColor; dashed?: boolean; withPrevMean?: boolean } = {},
): Pending | undefined {
  return curveOf(indicatorsOf(p.data, key)[0], p, { name: p.labelOf(key), ...options })
}

/** Grille « Niveau, combat et style ». */
function levelGrid(p: TendancesChartsParams): TendancesGridDef | undefined {
  const t = getTendancesText(p.locale)
  const caps = p.data.capabilities
  const solid = { withPrevMean: true }
  const charts = compact<TendancesChartDef>([
    caps.csr
      ? levelChart(
          p,
          'csr',
          t.chartCsr,
          variantCurves(p, KEY.csr, (v) => variantLabel(KEY.csr, v, p.locale), () => undefined),
          t.infoCsr,
        )
      : undefined,
    caps.lusr
      ? levelChart(
          p,
          'lusr',
          t.chartLusr,
          variantCurves(
            p,
            KEY.lusr,
            (v) => variantLabel(KEY.lusr, v, p.locale),
            (v) => LUSR_GROUP_TOKENS[v],
          ),
        )
      : undefined,
    caps.mmr
      ? levelChart(
          p,
          'mmr-pair',
          t.chartMmrPair,
          compact([
            singleCurve(p, KEY.enemyMmr, { color: ENEMY_MMR_TOKEN, ...solid }),
            singleCurve(p, KEY.teamMmr, solid),
          ]),
          t.infoMmrPair,
        )
      : undefined,
    levelChart(
      p,
      'damage',
      t.chartDamage,
      compact([singleCurve(p, KEY.damageDealt, solid), singleCurve(p, KEY.damageTaken, solid)]),
    ),
    caps.objectives
      ? levelChart(
          p,
          'objectives',
          t.chartObjectives,
          compact([
            singleCurve(p, KEY.take, { color: 'objective-role-take', ...solid }),
            singleCurve(p, KEY.defend, { color: 'objective-role-defend', ...solid }),
            singleCurve(p, KEY.hold, { color: 'objective-role-hold', ...solid }),
            singleCurve(p, KEY.parity, { color: AXIS_INK, dashed: true }),
          ]),
          t.infoObjectives,
        )
      : undefined,
  ])
  if (charts.length === 0) return undefined
  return { id: 'level', title: t.gridLevel, charts }
}

/**
 * Les grilles à afficher pour l'horizon et le pas choisis, dans l'ordre de la page. Vide
 * quand rien ne se trace : la section affiche alors son état vide.
 */
export function buildTendancesGrids(params: TendancesChartsParams): TendancesGridDef[] {
  return compact([versusMmrGrid(params), levelGrid(params)])
}
