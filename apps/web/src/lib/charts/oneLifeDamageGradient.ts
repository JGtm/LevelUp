/**
 * oneLifeDamageGradient — dégradés des courbes "Rendement & Résistance" exprimées
 * en dégâts BRUTS (solo + escouade), avec le repère **225 dégâts = 1 vie** de
 * Spartan (bouclier + santé).
 *
 * Dégradé vertical linéaire SANS visualMap (cf. SessionNetScoreArea : visualMap
 * rendait historiquement la courbe invisible). Les offsets des couleurs sont
 * calculés depuis l'étendue [min,max] des données de la courbe — donc alignés
 * sur sa boîte englobante de rendu. Couleurs via tokens sémantiques.
 *
 * Sémantique des deux courbes :
 *   - dégâts/frag  : au plus BAS, au plus efficace → monotone (vert sous le
 *     repère 225 = positif : moins d'une vie par frag ; rouge au-dessus =
 *     dégâts gaspillés ; 225 = bascule neutre).
 *   - dégâts/mort  : au plus LOIN au-dessus de 225, meilleure résistance →
 *     ascendant (vert vers le haut, rouge sous le repère).
 */
import { resolveToken } from '@/lib/accessibility'

/**
 * Dégâts d'une vie complète de Spartan — repère commun aux deux courbes.
 * DÉFAUT = Halo Infinite (90 vie + 135 bouclier). Les fonctions ci-dessous
 * acceptent un `oneLife` par titre (115 Halo 5…) ; sans argument elles gardent
 * ce barème Infinite (callers escouade = Infinite-only).
 */
export const ONE_LIFE_DAMAGE = 225

export interface LinearGradient {
  type: 'linear'
  x: number
  y: number
  x2: number
  y2: number
  colorStops: Array<{ offset: number; color: string }>
}

/** Dégâts/frag bruts d'un match : ΣDD / frags. null si non calculable. */
export function damagePerKill(damageDealt?: number | null, kills?: number | null): number | null {
  if (damageDealt == null || kills == null || kills <= 0) return null
  return Math.round(damageDealt / kills)
}

/** Dégâts/mort bruts d'un match : ΣDT / morts. null si non calculable. */
export function damagePerDeath(damageTaken?: number | null, deaths?: number | null): number | null {
  if (damageTaken == null || deaths == null || deaths <= 0) return null
  return Math.round(damageTaken / deaths)
}

/** Fraction verticale (0 = haut/max … 1 = bas/min) où tombe `threshold` dans [min,max]. */
function offsetOf(values: Array<number | null>, threshold: number): number {
  const nums = values.filter((v): v is number => v != null)
  if (nums.length === 0) return 0.5
  const top = Math.max(...nums)
  const bot = Math.min(...nums)
  const span = top - bot
  if (span <= 0) return 0.5
  return Math.min(1, Math.max(0, (top - threshold) / span))
}

function gradient(colorStops: Array<{ offset: number; color: string }>): LinearGradient {
  return { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops }
}

/**
 * Dégradé "dégâts/frag" — monotone : vert quand les dégâts/frag sont FAIBLES
 * (sous 225 = moins d'une vie par frag = efficace = positif), rouge quand ils
 * sont ÉLEVÉS (gaspillage), 225 = bascule neutre. Miroir structurel de
 * `defensiveDamageGradient`. Cas dégénérés (repère hors étendue) → bicolore.
 */
export function offensiveDamageGradient(
  values: Array<number | null>,
  oneLife: number = ONE_LIFE_DAMAGE,
): LinearGradient {
  const pos = resolveToken('divergent-pos')
  const neutral = resolveToken('divergent-neutral')
  const neg = resolveToken('divergent-neg')
  const r = offsetOf(values, oneLife)
  // r = position verticale du repère 225 (0 = haut/max, 1 = bas/min).
  // Toutes sous 225 → entièrement positif (du neutre au repère vers le vert) ;
  // toutes au-dessus → entièrement gaspillage (du rouge vers le neutre au repère).
  if (r <= 0) return gradient([{ offset: 0, color: neutral }, { offset: 1, color: pos }])
  if (r >= 1) return gradient([{ offset: 0, color: neg }, { offset: 1, color: neutral }])
  return gradient([
    { offset: 0, color: neg },
    { offset: r, color: neutral },
    { offset: 1, color: pos },
  ])
}

/**
 * Dégradé "dégâts/mort" — ascendant : vert vers le haut (encaisser plus de
 * 225/mort = bonne résistance), neutre au repère, rouge sous le repère.
 */
export function defensiveDamageGradient(
  values: Array<number | null>,
  oneLife: number = ONE_LIFE_DAMAGE,
): LinearGradient {
  const pos = resolveToken('divergent-pos')
  const neutral = resolveToken('divergent-neutral')
  const neg = resolveToken('divergent-neg')
  const r = offsetOf(values, oneLife)
  if (r <= 0 || r >= 1) return gradient([{ offset: 0, color: pos }, { offset: 1, color: neg }])
  return gradient([
    { offset: 0, color: pos },
    { offset: r, color: neutral },
    { offset: 1, color: neg },
  ])
}

/** Bornes d'axe Y garantissant la visibilité du repère 225 + un peu de marge. */
export function damageAxisBounds(
  values: Array<number | null>,
  oneLife: number = ONE_LIFE_DAMAGE,
): { min: number; max: number } {
  const nums = values.filter((v): v is number => v != null)
  if (nums.length === 0) return { min: 0, max: oneLife * 2 }
  const lo = Math.min(...nums, oneLife)
  const hi = Math.max(...nums, oneLife)
  const pad = Math.max(20, (hi - lo) * 0.1)
  return { min: Math.max(0, Math.floor(lo - pad)), max: Math.ceil(hi + pad) }
}
