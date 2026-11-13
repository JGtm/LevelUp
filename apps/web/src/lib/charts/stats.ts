/**
 * stats — moyenne, médiane et moyenne mobile des séries des graphes : UNE implémentation, que tous les
 * graphes appellent au lieu de réécrire `reduce(...) / length` (9 copies relevées le 2026-11-13, au
 * traitement des valeurs absentes divergent).
 *
 * Règle commune (D5) : une valeur `null`, `undefined` ou non finie est ABSENTE ; elle ne compte ni au
 * numérateur ni au dénominateur. Sans aucune valeur valide, le résultat est `null`. La somme se fait dans
 * l'ordre des valeurs, de zéro : le résultat d'une série entièrement valide est celui de
 * `reduce((s, v) => s + v, 0) / n`, au bit près.
 */
import { finiteOrNull } from './cumulativeSeries'

export type MaybeNumber = number | null | undefined

/** Moyenne des valeurs valides (non-null, finies) — `null` si aucune. */
export function meanOfValid(values: readonly MaybeNumber[]): number | null {
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

/**
 * Médiane des valeurs valides — `null` si aucune. Nombre pair : moyenne des deux valeurs centrales
 * (`(a + b) / 2`) ; nombre impair : la valeur centrale elle-même.
 */
export function medianOfValid(values: readonly MaybeNumber[]): number | null {
  const sorted: number[] = []
  for (const raw of values) {
    const v = finiteOrNull(raw)
    if (v != null) sorted.push(v)
  }
  if (sorted.length === 0) return null
  sorted.sort((a, b) => a - b)
  const mid = Math.floor(sorted.length / 2)
  return sorted.length % 2 === 0 ? (sorted[mid - 1] + sorted[mid]) / 2 : sorted[mid]
}

export interface MovingMeanOptions {
  /** `'trailing'` (défaut) : la fenêtre se termine au point ; `'centered'` : elle l'entoure. */
  align?: 'trailing' | 'centered'
  /** Fenêtre qui se termine au point : `null` tant que les `window − 1` premiers points n'ont pas une fenêtre pleine. */
  fullWindowOnly?: boolean
  /** Nombre minimal de valeurs valides dans la fenêtre pour rendre une moyenne (défaut 1). */
  minValid?: number
}

/**
 * Moyenne mobile : pour chaque point, la moyenne des valeurs valides de sa fenêtre (`window` points). La
 * fenêtre est tronquée aux bords de la série, sauf `fullWindowOnly`. Rend `null` quand la fenêtre compte
 * moins de `minValid` valeurs valides. Centrée, une fenêtre de 1 point ou moins rend la série telle quelle
 * (absentes en `null`).
 */
export function movingMean(
  values: readonly MaybeNumber[],
  window: number,
  { align = 'trailing', fullWindowOnly = false, minValid = 1 }: MovingMeanOptions = {},
): (number | null)[] {
  if (align === 'centered' && window <= 1) return values.map((v) => v ?? null)
  const half = Math.floor(window / 2)
  return values.map((_, i) => {
    if (align === 'trailing' && fullWindowOnly && i < window - 1) return null
    const lo = align === 'centered' ? Math.max(0, i - half) : Math.max(0, i - (window - 1))
    const hi = align === 'centered' ? Math.min(values.length - 1, i + half) : i
    let sum = 0
    let count = 0
    for (let k = lo; k <= hi; k++) {
      const v = finiteOrNull(values[k])
      if (v != null) {
        sum += v
        count += 1
      }
    }
    return count < Math.max(1, minValid) ? null : sum / count
  })
}
