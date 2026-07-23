/**
 * cumulativeSeries — cumul signé GÉNÉRIQUE avec report D5, source unique
 * (CLAUDE.md n°6) de l'accumulateur `carryForward`.
 *
 * Une contribution `null` (donnée manquante / non-finie) NE fait PAS avancer le
 * cumul : la courbe REPORTE la dernière valeur cumulée (jamais 0, jamais de
 * rupture) tout en figurant au point courant. Partagé par :
 *   - `cumulativeFdaGap` (écart FDA réel vs attendu) ;
 *   - `netLives` (« Balance des dégâts »).
 * Toute réimplémentation de l'identifiant distinctif `carryForward` hors de ce
 * fichier (et de son délégant `cumulativeFdaGap.ts`) est interdite par
 * `cumulativeFdaGap.guard.test.ts`.
 */

const round2 = (v: number): number => Math.round(v * 100) / 100

/** Renvoie la valeur si finie, sinon `null` (garde D5 partagée). */
export function finiteOrNull(v: number | null | undefined): number | null {
  return v != null && Number.isFinite(v) ? v : null
}

export interface CumulativePoint {
  /** Contribution du point (arrondie 2 déc.), `null` si absente/non-finie (D5). */
  value: number | null
  /** Cumul signé courant (reporte la dernière valeur si `value === null`). */
  cumulative: number
}

/**
 * Cumul signé sur des contributions DÉJÀ ORDONNÉES. `carryForward` = cumul
 * courant, reporté tel quel quand une contribution est absente (D5). Valeurs
 * arrondies à 2 décimales.
 */
export function cumulativeSigned(values: Array<number | null | undefined>): CumulativePoint[] {
  let carryForward = 0
  return values.map((raw) => {
    const value = finiteOrNull(raw)
    if (value != null) carryForward = round2(carryForward + value)
    return { value: value != null ? round2(value) : null, cumulative: carryForward }
  })
}

/**
 * Moyenne des contributions valides (non-null, finies) — `null` si aucune.
 * Sert les pastilles KPI « moyenne par match » des charts cumulés.
 */
export function meanOfValid(values: Array<number | null | undefined>): number | null {
  let sum = 0
  let count = 0
  for (const raw of values) {
    const v = finiteOrNull(raw)
    if (v != null) {
      sum += v
      count += 1
    }
  }
  return count === 0 ? null : sum / count
}
