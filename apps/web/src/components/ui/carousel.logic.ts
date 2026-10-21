/**
 * Logique pure du carrousel : « proche de la fin » (chargement de la page suivante d'un carrousel
 * qui se remplit au fil du défilement, Accueil : matchs récents).
 */

/** Nombre de tuiles restantes à droite à partir duquel la page suivante est demandée. */
export const CAROUSEL_NEAR_END_TILES = 5

/** Mesures de la piste de défilement (valeurs du DOM, en pixels). */
export interface CarouselMetrics {
  scrollLeft: number
  clientWidth: number
  scrollWidth: number
  /** Pas d'une tuile à la suivante (largeur + espace). 0 tant que rien n'est mesuré. */
  itemSpan: number
}

/**
 * Le défilement est à `tiles` tuiles ou moins de la fin. Jamais vrai tant que la piste n'est pas
 * mesurée (`scrollWidth` nul : élément non monté ou sans dimensions).
 */
export function isNearEnd(m: CarouselMetrics, tiles: number = CAROUSEL_NEAR_END_TILES): boolean {
  if (m.scrollWidth <= 0) return false
  const remaining = m.scrollWidth - (m.scrollLeft + m.clientWidth)
  return remaining <= tiles * m.itemSpan
}

/** Pas d'une tuile : écart entre les deux premières tuiles, sinon largeur de la première. */
export function itemSpanOf(track: Pick<HTMLElement, 'children'>): number {
  const first = track.children[0] as HTMLElement | undefined
  if (!first) return 0
  const second = track.children[1] as HTMLElement | undefined
  return second ? second.offsetLeft - first.offsetLeft : first.offsetWidth
}
