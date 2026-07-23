/**
 * cumulativeFdaGap — source unique (CLAUDE.md n°6) du CUMUL de l'écart FDA
 * réel vs attendu, partagé par les 3 charts « Écart cumulé au FDA attendu » :
 * Sessions (SessionFdaGapCumulative), Escouade (squadFdaGapChart) et Timeseries
 * (TimeseriesFdaGapTrend).
 *
 * Helper PUR sur des paires DÉJÀ ORDONNÉES (chaque appelant garde son propre tri
 * — start_time côté Sessions, match_order côté Escouade, ordre du service côté
 * Timeseries — et son propre assemblage de labels / forme de série).
 *
 * Différentiel d'un match = `réel − attendu` (FDA réel natif ADR 0006 moins FDA
 * attendu projeté backend), `null` si l'un des termes manque ou est non-fini.
 *
 * D5 : un match sans attendu (`gap === null`) NE fait PAS avancer le cumul — la
 * courbe REPORTE la dernière valeur cumulée (jamais 0, jamais de rupture). Le
 * point figure quand même (porté à la valeur courante). L'identifiant distinctif
 * `carryForward` (accumulateur reporté) est verrouillé hors de ce fichier par
 * `cumulativeFdaGap.guard.test.ts`.
 */

const round2 = (v: number): number => Math.round(v * 100) / 100

function finite(v: number | null | undefined): number | null {
  return v != null && Number.isFinite(v) ? v : null
}

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
}

/**
 * Cumul signé de l'écart au FDA attendu sur des paires DÉJÀ ORDONNÉES.
 * `carryForward` = cumul courant, reporté tel quel quand un match n'a pas
 * d'attendu (D5). Valeurs arrondies à 2 décimales.
 */
export function cumulativeFdaGap(pairs: FdaGapPair[]): FdaGapCumPoint[] {
  let carryForward = 0
  return pairs.map((pair) => {
    const real = finite(pair.real)
    const expected = finite(pair.expected)
    const gap = real != null && expected != null ? round2(real - expected) : null
    if (gap != null) carryForward = round2(carryForward + gap)
    return {
      real: real != null ? round2(real) : null,
      expected: expected != null ? round2(expected) : null,
      gap,
      cumulative: carryForward,
    }
  })
}

/**
 * Écart MOYEN par match, calculé UNIQUEMENT sur les paires avec attendu (D3,
 * pastille KPI « +0,7/match »). `null` si aucune paire exploitable.
 */
export function meanFdaGap(pairs: FdaGapPair[]): number | null {
  let sum = 0
  let count = 0
  for (const pair of pairs) {
    const real = finite(pair.real)
    const expected = finite(pair.expected)
    if (real != null && expected != null) {
      sum += round2(real - expected)
      count += 1
    }
  }
  return count === 0 ? null : sum / count
}
