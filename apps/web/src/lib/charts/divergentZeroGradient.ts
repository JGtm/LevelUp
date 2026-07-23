/**
 * divergentZeroGradient — dégradé vertical DIVERGENT vert (au-dessus de 0) / rouge
 * (en dessous), à bascule EXACTE sur la ligne 0, pour une aire/courbe ancrée à 0
 * (`areaStyle.origin: 0`).
 *
 * SANS visualMap (cf. SessionNetScoreArea : visualMap rendait historiquement la
 * courbe invisible). Le dégradé est calculé depuis la boîte englobante de l'aire
 * ancrée à 0 : en Y elle vaut `[min(valeurs, 0), max(valeurs, 0)]`, donc 0 tombe
 * à la fraction `zeroRatio` depuis le HAUT. Le MÊME dégradé colore ligne ET aire.
 *
 * Source unique (CLAUDE.md n°6) : consommé par les 3 charts « aire signée » —
 * SessionNetScoreArea (net score cumulé), TimeseriesFdaGapTrend (écart au FDA
 * attendu par match) et SessionFdaGapCumulative (écart cumulé au FDA attendu).
 * L'identifiant `zeroRatio` est verrouillé hors de ce fichier par
 * `divergentZeroGradient.guard.test.ts`.
 */
import { resolveToken } from '@/lib/accessibility'

export interface LinearGradient {
  type: 'linear'
  x: number
  y: number
  x2: number
  y2: number
  colorStops: Array<{ offset: number; color: string }>
}

/**
 * Construit le dégradé divergent à bascule sur 0 pour `values` (les valeurs de la
 * courbe/aire ancrée à 0). Cas dégénéré (une seule valeur, ou étendue nulle) →
 * `zeroRatio = 1` (entièrement du côté positif), robuste au rendu.
 */
export function divergentZeroGradient(values: number[]): LinearGradient {
  const posColor = resolveToken('divergent-pos')
  const negColor = resolveToken('divergent-neg')
  const top = Math.max(...values, 0)
  const bot = Math.min(...values, 0)
  const span = top - bot
  const zeroRatio = span > 0 ? Math.min(1, Math.max(0, top / span)) : 1
  return {
    type: 'linear',
    x: 0,
    y: 0,
    x2: 0,
    y2: 1,
    colorStops: [
      { offset: 0, color: posColor },
      { offset: zeroRatio, color: posColor },
      { offset: zeroRatio, color: negColor },
      { offset: 1, color: negColor },
    ],
  }
}
