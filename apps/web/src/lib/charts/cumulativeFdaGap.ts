/**
 * cumulativeFdaGap — CUMUL de l'écart FDA réel vs attendu, partagé par les 3
 * charts « Écart cumulé au FDA attendu » : Sessions (SessionFdaGapCumulative),
 * Escouade (squadFdaGapChart) et Timeseries (TimeseriesFdaGapTrend).
 *
 * Helper PUR sur des paires DÉJÀ ORDONNÉES (chaque appelant garde son propre tri
 * — start_time côté Sessions, match_order côté Escouade, ordre du service côté
 * Timeseries — et son propre assemblage de labels / forme de série).
 *
 * Différentiel d'un match = `réel − attendu` (FDA réel natif ADR 0006 moins FDA
 * attendu projeté backend), `null` si l'un des termes manque ou est non-fini.
 *
 * DÉLÈGUE le cumul signé + report D5 au helper générique `cumulativeSeries`
 * (source unique de l'accumulateur `carryForward`, CLAUDE.md n°6) : un match sans
 * attendu (`gap === null`) ne fait pas avancer le cumul, la courbe reporte la
 * dernière valeur cumulée.
 */

import { cumulativeSigned, finiteOrNull, meanOfValid } from './cumulativeSeries'

const round2 = (v: number): number => Math.round(v * 100) / 100

/** Une paire FDA réel / attendu d'un match (ordre déjà fixé par l'appelant). */
export interface FdaGapPair {
  real: number | null
  expected: number | null
}

/** Un point du cumul : valeurs natives + écart du match + somme cumulée signée. */
export interface FdaGapCumPoint {
  /** FDA réel du match, arrondi 2 décimales (null si absent/non-fini). */
  real: number | null
  /** FDA attendu du match, arrondi 2 décimales (null si absent/non-fini). */
  expected: number | null
  /** Écart du match `réel − attendu`, arrondi 2 décimales (null — D5). */
  gap: number | null
  /** Écart cumulé (reporte la dernière valeur si le match n'a pas d'attendu). */
  cumulative: number
  /**
   * FDA réel CUMULÉ, avec le MÊME skip conjoint que `gap` (D5) : un match sans
   * paire réel/attendu valide ne fait avancer NI ce cumul NI `cumulativeExpected`
   * (report des deux ensemble), afin de préserver l'identité
   * `cumulativeReal − cumulativeExpected === cumulative`.
   */
  cumulativeReal: number
  /** FDA attendu CUMULÉ — même skip conjoint que `cumulativeReal` (cf. ci-dessus). */
  cumulativeExpected: number
}

/** Écart signé d'une paire (`réel − attendu`), `null` si un terme manque (D5). */
function gapOf(pair: FdaGapPair): number | null {
  const real = finiteOrNull(pair.real)
  const expected = finiteOrNull(pair.expected)
  return real != null && expected != null ? round2(real - expected) : null
}

/**
 * Cumul signé de l'écart au FDA attendu sur des paires DÉJÀ ORDONNÉES, délégué au
 * helper générique `cumulativeSigned`. Un match sans attendu reporte le cumul (D5).
 */
export function cumulativeFdaGap(pairs: FdaGapPair[]): FdaGapCumPoint[] {
  const gaps = pairs.map(gapOf)
  const cum = cumulativeSigned(gaps)
  // Masque conjoint (D5) : réel/attendu n'alimentent leurs cumuls respectifs que
  // si la paire du match est valide (même condition que `gap`) — sinon les 3
  // cumuls reportent ENSEMBLE, ce qui préserve cumulativeReal − cumulativeExpected
  // === cumulative en tout point.
  const cumReal = cumulativeSigned(pairs.map((pair, i) => (gaps[i] != null ? finiteOrNull(pair.real) : null)))
  const cumExpected = cumulativeSigned(
    pairs.map((pair, i) => (gaps[i] != null ? finiteOrNull(pair.expected) : null)),
  )
  return pairs.map((pair, i) => {
    const real = finiteOrNull(pair.real)
    const expected = finiteOrNull(pair.expected)
    return {
      real: real != null ? round2(real) : null,
      expected: expected != null ? round2(expected) : null,
      gap: cum[i].value,
      cumulative: cum[i].cumulative,
      cumulativeReal: cumReal[i].cumulative,
      cumulativeExpected: cumExpected[i].cumulative,
    }
  })
}

/**
 * Écart MOYEN par match, calculé UNIQUEMENT sur les paires avec attendu (D3,
 * pastille KPI « +0,7/match »), délégué au helper générique `meanOfValid`.
 * `null` si aucune paire exploitable.
 */
export function meanFdaGap(pairs: FdaGapPair[]): number | null {
  return meanOfValid(pairs.map(gapOf))
}
