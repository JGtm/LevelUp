/**
 * makeCategoricalScale.ts — Constructeur d'échelle catégorielle.
 *
 * Mappe une clé string typée vers un SemanticToken.
 * Lève une erreur explicite sur clé inconnue — pas de fallback silencieux.
 */
import { log } from '../_logger'
import type { SemanticToken } from '../semantic-tokens'

export function makeCategoricalScale<K extends string>(
  map: Record<K, SemanticToken>,
): (key: K | string | null | undefined) => SemanticToken | null {
  const knownKeys = Object.keys(map) as K[]
  const scaleKey = knownKeys.join('|')

  return function categoricalScale(key: K | string | null | undefined): SemanticToken | null {
    if (key == null) return null

    if (key in map) return map[key as K]

    log.error(
      `categorical:unknown:${scaleKey}:${key}`,
      `categoricalScale : clé inconnue "${key}". Clés valides : ${knownKeys.join(', ')}`,
    )
    return null
  }
}
