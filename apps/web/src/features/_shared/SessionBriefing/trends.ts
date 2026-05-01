/**
 * trends.ts — calcul des trends ▲/▼ pour le SessionBriefing.
 *
 * Comparaison intra-session : KPI individuel vs moyenne d'équipe sur le scope
 * filtré (PAS vs all-time du joueur). Le seuil par défaut (5%) évite le bruit
 * autour de la moyenne — en deçà on affiche "near" (━).
 *
 * Pour les morts (lower_is_better), on inverse le résultat : `value > ref`
 * devient "below" (rouge ▼) et `value < ref` devient "above" (vert ▲).
 */

export type TrendState = 'above' | 'near' | 'below' | 'none'

export interface TrendOptions {
  /** Si true, valeur basse = mieux (deaths). Default false. */
  lowerIsBetter?: boolean
  /** Seuil ratio relatif (default 0.05 = ±5%). En deçà → "near". */
  threshold?: number
}

export function computeTrend(
  value: number | null | undefined,
  reference: number | null | undefined,
  options: TrendOptions = {},
): TrendState {
  const { lowerIsBetter = false, threshold = 0.05 } = options
  if (value == null || reference == null || reference === 0) return 'none'
  const ratio = value / reference
  if (ratio >= 1 + threshold) return lowerIsBetter ? 'below' : 'above'
  if (ratio <= 1 - threshold) return lowerIsBetter ? 'above' : 'below'
  return 'near'
}

export function trendSymbol(state: TrendState): string {
  if (state === 'above') return '▲'
  if (state === 'below') return '▼'
  if (state === 'near') return '━'
  return ''
}
