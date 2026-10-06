// cross-feature-allow: jetons de couleur des chaînes LUSR (`LUSR_GROUP_TOKENS`) partagés avec
// l'onglet Carrière — une seule table de couleurs de types de partie dans l'app.
/**
 * tendancesMix.logic — la série PURE du graphique « Matchs par type de partie » : de
 * `data.mix` au pas courant aux bâtons empilés du wrapper `BarStackedChart`.
 *
 * UN BÂTON PAR INTERVALLE du pas courant, limité à ceux dont le DÉBUT est dans l'horizon (même
 * règle que les courbes). Le pas « match » n'a pas de mélange propre : il se ramène à « day ».
 * CATÉGORIE : la date de l'intervalle (jour et mois court ; mois court seul au pas « mois »),
 * dans le fuseau de la réponse. COMPOSANTS : les types de partie, nommés par `labelOf`. COULEUR :
 * le jeton `LUSR_GROUP_TOKENS` quand la chaîne y figure, sinon `chart-series-N` selon la
 * position du type dans `data.game_types`.
 */
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { LUSR_GROUP_TOKENS } from '@/features/career/lusr-chains'
import type { SemanticToken } from '@/lib/accessibility'
import { intlLocale } from '@/lib/formatters'
import type { TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { pointsInHorizon, type Step } from './tendances.logic'

/** Pas qui portent un mélange dans la réponse. */
export type MixStep = 'day' | 'week' | 'month'

/** Nombre de jetons `chart-series-*` de la palette. */
const SERIES_TOKEN_COUNT = 8

/** Le pas du mélange : « match » se ramène à « day ». */
export function mixStep(step: Step): MixStep {
  return step === 'match' ? 'day' : step
}

export interface MixInput {
  data: Pick<TrendsPageResponse, 'mix' | 'as_of' | 'timezone' | 'game_types'>
  horizon: number
  step: Step
  locale: Locale
  /** Nom d'un type de partie (clé de chaîne) dans la locale. */
  labelOf: (key: string) => string
}

export interface MixChart {
  /** Vide quand aucun intervalle n'est dans l'horizon : le wrapper dit alors « vide ». */
  series: ChartSeries<ChartPointStacked>[]
  /** Couleur de chaque composant, par nom affiché. */
  componentColors: Record<string, SemanticToken>
  /** Ordre des composants (celui de `game_types`). */
  componentOrder: string[]
}

function seriesToken(position: number): SemanticToken {
  return `chart-series-${(position % SERIES_TOKEN_COUNT) + 1}` as SemanticToken
}

/**
 * Catégorie d'un intervalle : sa date (jour et mois court ; mois court seul au pas « mois »),
 * dans le fuseau donné. Partagée avec les barres empilées de la vue Escouade.
 */
export function intervalCategory(t: string, step: MixStep, timeZone: string, locale: Locale): string {
  const options: Intl.DateTimeFormatOptions =
    step === 'month' ? { month: 'short', timeZone } : { day: 'numeric', month: 'short', timeZone }
  return new Intl.DateTimeFormat(intlLocale(locale), options).format(new Date(t))
}

/** Construit le graphique des types de partie. */
export function buildMixChart(input: MixInput): MixChart {
  const step = mixStep(input.step)
  const buckets = pointsInHorizon(input.data.mix[step], input.data.as_of, input.horizon)

  // Ordre de `game_types`, puis les clés vues dans le mélange qu'il ne liste pas.
  const keys = (input.data.game_types ?? []).map((g) => g.key)
  for (const bucket of buckets) {
    for (const key of Object.keys(bucket.counts)) if (!keys.includes(key)) keys.push(key)
  }

  const componentColors: Record<string, SemanticToken> = {}
  const componentOrder: string[] = []
  keys.forEach((key, position) => {
    const name = input.labelOf(key)
    componentColors[name] = LUSR_GROUP_TOKENS[key] ?? seriesToken(position)
    componentOrder.push(name)
  })

  if (buckets.length === 0) return { series: [], componentColors, componentOrder }

  const datapoints: ChartPointStacked[] = buckets.map((bucket) => {
    const components: Record<string, number> = {}
    for (const [key, count] of Object.entries(bucket.counts)) {
      const name = input.labelOf(key)
      components[name] = (components[name] ?? 0) + count
    }
    return { category: intervalCategory(bucket.t, step, input.data.timezone, input.locale), components }
  })
  return { series: [{ key: 'mix', datapoints }], componentColors, componentOrder }
}
