/**
 * makeDivergentScale.ts — Constructeur d'échelle divergente.
 *
 * Mappe une valeur numérique signée vers pos / neutral / neg.
 * La bande neutre est inclusive aux deux bornes.
 *
 * Exemple : makeDivergentScale({ neutralBand: [-10, 10], ... })
 *   value > 10  → positive
 *   -10 <= value <= 10 → neutral
 *   value < -10 → negative
 *
 * Cas strict : neutralBand: [0, 0]
 *   value > 0 → positive
 *   value === 0 → neutral
 *   value < 0 → negative
 */
import { log } from '../_logger'
import type { SemanticToken } from '../semantic-tokens'

export interface DivergentScaleConfig {
  positive: SemanticToken
  neutral: SemanticToken
  negative: SemanticToken
  /** Bornes inclusives [min, max]. Pour strict-zéro : [0, 0]. */
  neutralBand: [number, number]
}

export function makeDivergentScale(config: DivergentScaleConfig): (value: number) => SemanticToken {
  const { positive, neutral, negative, neutralBand } = config
  const [bandMin, bandMax] = neutralBand

  if (bandMin > bandMax) {
    throw new Error(
      `makeDivergentScale : neutralBand inversée [${bandMin}, ${bandMax}] — bandMin doit être <= bandMax`,
    )
  }

  const scaleKey = `${positive}|${neutral}|${negative}`

  return function divergentScale(value: number): SemanticToken {
    // NaN = valeur invalide → neutral. ±Infinity suit la logique normale
    // (Infinity > bandMax = true → positive ; -Infinity < bandMin = true → negative).
    if (isNaN(value)) {
      log.warn(`divergent:invalid:${scaleKey}`, `divergentScale reçoit NaN — retourne neutral`)
      return neutral
    }

    if (value > bandMax) return positive
    if (value < bandMin) return negative
    return neutral
  }
}
