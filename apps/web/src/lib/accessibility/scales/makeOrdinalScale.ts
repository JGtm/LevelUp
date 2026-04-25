/**
 * makeOrdinalScale.ts — Constructeur d'échelle ordinale.
 *
 * Mappe une valeur numérique vers un SemanticToken selon des seuils décroissants.
 * tier[0] = meilleur (valeur >= thresholds[0])
 * tier[N-1] = pire   (valeur < thresholds[N-2])
 *
 * Exemple : makeOrdinalScale({ tiers: ['A','B','C'], thresholds: [80, 50] })
 *   value >= 80 → 'A'
 *   50 <= value < 80 → 'B'
 *   value < 50 → 'C'
 */
import { log } from '../_logger'
import type { SemanticToken } from '../semantic-tokens'

export interface OrdinalScaleConfig {
  tiers: [SemanticToken, ...SemanticToken[]]
  /** Longueur = tiers.length - 1, valeurs décroissantes. */
  thresholds: number[]
}

export function makeOrdinalScale(config: OrdinalScaleConfig): (value: number) => SemanticToken {
  const { tiers, thresholds } = config

  if (tiers.length !== thresholds.length + 1) {
    throw new Error(
      `makeOrdinalScale : tiers.length (${tiers.length}) doit être égal à thresholds.length + 1 (${thresholds.length + 1})`,
    )
  }

  for (let i = 0; i < thresholds.length - 1; i++) {
    if (thresholds[i] <= thresholds[i + 1]) {
      throw new Error(
        `makeOrdinalScale : thresholds doit être strictement décroissant. Violation : thresholds[${i}]=${thresholds[i]} <= thresholds[${i + 1}]=${thresholds[i + 1]}`,
      )
    }
  }

  const scaleKey = tiers.join('|')

  return function ordinalScale(value: number): SemanticToken {
    // NaN = valeur invalide → tier le plus bas. ±Infinity suit la logique normale
    // (Infinity >= n = true → tier-1 ; -Infinity >= n = false → tier final).
    if (isNaN(value)) {
      log.warn(`ordinal:invalid:${scaleKey}`, `ordinalScale reçoit NaN — retourne le tier le plus bas`)
      return tiers[tiers.length - 1]
    }

    for (let i = 0; i < thresholds.length; i++) {
      if (value >= thresholds[i]) return tiers[i]
    }
    return tiers[tiers.length - 1]
  }
}
