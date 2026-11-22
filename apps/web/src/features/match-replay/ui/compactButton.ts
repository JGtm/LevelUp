/**
 * compactButton — LE FORMAT D'UNE COMMANDE ORDINAIRE DE LA BARRE D'UNE LIGNE (Cinéma, Plein cadre) : un carré de
 * `boutonPx`, au rayon `boutonRayonPx`, sans bordure ni fond au repos. Le fond ne vient qu'au survol ; l'état marqué
 * (caméra qui suit, cadre courant) garde le sien. Le bouton de lecture reste rond et plein, et la barre complète du
 * Standard garde ses formes : chaque composant partagé reçoit une variante `compact`, ce module n'en change aucun.
 *
 * Les cotes s'appliquent en `style` (une classe Tailwind arbitraire ne naît que si elle est écrite en clair).
 */
import type { CSSProperties } from 'react'

import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'

/** Le carré : largeur et hauteur de `boutonPx`. */
export const COMPACT_BUTTON_SQUARE: CSSProperties = Object.freeze({
  width: C.boutonPx,
  height: C.boutonPx,
  borderRadius: C.boutonRayonPx,
})

/** Le carré qui peut s'élargir (une valeur écrite, un pourcentage) : largeur minimale de `boutonPx`, marge intérieure de 4 px. */
export const COMPACT_BUTTON_WIDENABLE: CSSProperties = Object.freeze({
  minWidth: C.boutonPx,
  height: C.boutonPx,
  borderRadius: C.boutonRayonPx,
  padding: '0 4px',
})

/** Les classes communes : centré, fond au survol seulement, anneau au focus du clavier. */
export const COMPACT_BUTTON_CLASS =
  'inline-flex shrink-0 items-center justify-center transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'

/** Le curseur et l'estompe d'une commande : inerte, ou cliquable. */
export function compactButtonCursor(disabled: boolean): string {
  return disabled ? 'cursor-not-allowed opacity-40' : 'cursor-pointer'
}
