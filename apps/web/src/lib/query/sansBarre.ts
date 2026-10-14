/**
 * sansBarre.ts — drapeau `meta` des requêtes qui ne doivent pas allumer la barre de
 * chargement de page (`TopProgressBar`).
 *
 * La barre suit les requêtes `pending` (sans données en cache) : c'est juste pour le
 * premier chargement d'une page, faux pour une interaction locale (clic sur une zone,
 * préchargement d'une lecture voisine) qui n'est pas un changement de page.
 */
import type { Query } from '@tanstack/react-query'

/** Clé du drapeau dans `meta` ; une constante, jamais un littéral à la main. */
export const META_SANS_BARRE = 'sansBarre'

/** `meta` à poser sur une requête qui ne doit pas allumer la barre de chargement. */
export const SANS_BARRE_META = { [META_SANS_BARRE]: true } as const

/** Prédicat de la barre : requête sans données en cache, et non marquée « sans barre ». */
export function compteDansLaBarre(query: Query): boolean {
  return query.state.status === 'pending' && query.meta?.[META_SANS_BARRE] !== true
}
