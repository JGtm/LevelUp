/**
 * oneLifeDamageGradient — repère commun « une vie » des surfaces Rendement &
 * Résistance (escouade + solo), et son cadre de lecture PARTAGÉ.
 *
 * Le pivot est le même partout : **une vie de Spartan**. Il s'exprime à 100 %
 * dans l'unité normalisée des cartes Escouade (`rendement_offensif` /
 * `resistance_defensive` × 100) et à `oneLife` dégâts dans l'unité brute de la
 * courbe solo (dégâts/frag, dégâts/mort).
 *
 * Deux invariants tiennent l'alignement des deux surfaces :
 *   1. **Fenêtre FIXE** autour du pivot — {@link oneLifeWindowBounds} : demi-pivot
 *      … double pivot. Elle ne dépend JAMAIS des données affichées, sinon la même
 *      valeur change de position (et de couleur) d'une session à l'autre.
 *   2. **Zones de lecture** — {@link oneLifeZonesMarkArea} : `divergent-pos`
 *      au-dessus du pivot, `divergent-neg` en dessous, à des opacités distinctes
 *      (le rouge pèse plus que le vert à opacité égale). Elles sont posées en
 *      coordonnées d'axe, donc exactes, là où un dégradé est ancré sur la boîte
 *      englobante du tracé.
 *
 * Les dégradés de trait restent un rappel décoratif : ECharts ancre un gradient
 * linéaire sur la boîte englobante de la série, pas sur l'axe — d'où des arrêts
 * exprimés en fraction de la fenêtre fixe (constante), et non des données.
 */
import { resolveToken } from '@/lib/accessibility'

/**
 * Dégâts d'une vie complète de Spartan — repère commun aux deux courbes.
 * DÉFAUT = Halo Infinite (90 vie + 135 bouclier). Les fonctions ci-dessous
 * acceptent un `oneLife` par titre (115 Halo 5…) ; sans argument elles gardent
 * ce barème Infinite (callers escouade = Infinite-only).
 */
export const ONE_LIFE_DAMAGE = 225

/**
 * Demi-amplitude multiplicative de la fenêtre de lecture : de `pivot / 2` à
 * `pivot × 2`. En unité normalisée (pivot 100 %) cela donne la fenêtre 50…200 %
 * des cartes Rendement / Résistance ; en dégâts bruts (pivot 225) la fenêtre
 * 112,5…450. Une seule constante pour les deux surfaces.
 */
export const ONE_LIFE_WINDOW_FACTOR = 2

/** Opacités des deux zones de lecture. Le rouge est volontairement PLUS discret
 *  que le vert : à opacité égale il pèse davantage à l'œil. */
export const ONE_LIFE_ZONE_OPACITY = { pos: 0.08, neg: 0.06 } as const

export interface LinearGradient {
  type: 'linear'
  x: number
  y: number
  x2: number
  y2: number
  colorStops: Array<{ offset: number; color: string }>
}

/** Bornes de la fenêtre FIXE encadrant le pivot (jamais dérivées des données). */
export function oneLifeWindowBounds(pivot: number = ONE_LIFE_DAMAGE): {
  min: number
  max: number
} {
  return { min: pivot / ONE_LIFE_WINDOW_FACTOR, max: pivot * ONE_LIFE_WINDOW_FACTOR }
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

/**
 * Fraction verticale (0 = haut/max … 1 = bas/min) du pivot dans la fenêtre FIXE.
 * Constante par construction (2/3 pour un facteur 2) : c'est précisément ce qui
 * rend le dégradé comparable d'une session à l'autre.
 */
export function oneLifeNeutralOffset(): number {
  const { min, max } = oneLifeWindowBounds(1)
  return (max - 1) / (max - min)
}

function gradient(colorStops: Array<{ offset: number; color: string }>): LinearGradient {
  return { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops }
}

/**
 * Dégradé « dégâts/frag » — monotone : vert quand les dégâts par frag sont
 * FAIBLES (moins d'une vie dépensée par frag = efficace), rouge quand ils sont
 * ÉLEVÉS (gaspillage), neutre au pivot. Miroir de {@link defensiveDamageGradient}.
 */
export function offensiveDamageGradient(): LinearGradient {
  return gradient([
    { offset: 0, color: resolveToken('divergent-neg') },
    { offset: oneLifeNeutralOffset(), color: resolveToken('divergent-neutral') },
    { offset: 1, color: resolveToken('divergent-pos') },
  ])
}

/**
 * Dégradé « dégâts/mort » — ascendant : vert vers le haut (encaisser plus d'une
 * vie par mort = bonne résistance), neutre au pivot, rouge en dessous.
 */
export function defensiveDamageGradient(): LinearGradient {
  return gradient([
    { offset: 0, color: resolveToken('divergent-pos') },
    { offset: oneLifeNeutralOffset(), color: resolveToken('divergent-neutral') },
    { offset: 1, color: resolveToken('divergent-neg') },
  ])
}

/** Une zone `markArea` ECharts : paire [borne basse, borne haute] sur l'axe Y. */
type MarkAreaZone = [
  { yAxis: number; itemStyle: { color: string; opacity: number } },
  { yAxis: number },
]

/**
 * Sens de lecture de la grandeur tracée, par rapport au repère « une vie ».
 * OBLIGATOIRE à l'appel : c'est la seule chose qui change d'une surface à
 * l'autre, et l'uniformiser par défaut produit un contresens visuel.
 *   - `above-is-good` : l'indicateur MONTE quand le joueur va mieux (taux
 *     normalisés `rendement_offensif` / `resistance_defensive` des cartes
 *     Escouade ; dégâts encaissés par mort, où encaisser plus est une force) ;
 *   - `below-is-good` : l'indicateur DESCEND quand le joueur va mieux (dégâts
 *     BRUTS dépensés par frag : moins d'une vie par frag = efficace).
 */
export type OneLifeZoneDirection = 'above-is-good' | 'below-is-good'

/**
 * markArea des DEUX zones de lecture d'une carte « une vie », en coordonnées
 * d'axe : la zone « favorable » (`divergent-pos`) et la zone « défavorable »
 * (`divergent-neg`) de part et d'autre du pivot, placées selon `direction`.
 * Source unique des deux surfaces (cartes Escouade en % et courbe solo en dégâts
 * bruts) — ne jamais ré-écrire les deux zones à la main.
 *
 * L'opacité suit le TOKEN, pas la position : le rouge reste le plus discret quel
 * que soit le côté où il tombe (il pèse davantage à l'œil à opacité égale).
 *
 * Ordre de sortie STABLE : zone haute (pivot → `bounds.max`) puis zone basse
 * (`bounds.min` → pivot).
 */
export function oneLifeZonesMarkArea(
  pivot: number,
  direction: OneLifeZoneDirection,
  bounds: { min: number; max: number } = oneLifeWindowBounds(pivot),
): { silent: true; data: MarkAreaZone[] } {
  const good = {
    color: resolveToken('divergent-pos'),
    opacity: ONE_LIFE_ZONE_OPACITY.pos,
  }
  const bad = {
    color: resolveToken('divergent-neg'),
    opacity: ONE_LIFE_ZONE_OPACITY.neg,
  }
  const aboveIsGood = direction === 'above-is-good'
  return {
    silent: true,
    data: [
      [{ yAxis: pivot, itemStyle: aboveIsGood ? good : bad }, { yAxis: bounds.max }],
      [{ yAxis: bounds.min, itemStyle: aboveIsGood ? bad : good }, { yAxis: pivot }],
    ],
  }
}
