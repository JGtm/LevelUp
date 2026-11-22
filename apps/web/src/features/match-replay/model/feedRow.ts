/**
 * feedRow.ts — LA RANGÉE D'UNE LIGNE DU FIL : les classes partagées par TOUTES les formes de ligne.
 * Constantes seulement : un fichier de composant n'exporte que des composants
 * (react-refresh/only-export-components).
 *
 * UNE LIGNE FAIT 28 PX, TOUJOURS, ET SANS FOND NI BORDURE. La hauteur et l'écart viennent de
 * `cotesFeedLine.ts` ; les classes ci-dessous les écrivent en valeurs arbitraires (Tailwind ne lit
 * pas une valeur calculée) et `cotesFeedLine.test.ts` tient l'accord.
 *
 * L'élimination est une GRILLE à colonnes fixes (`FEED_GRID_ROW`, gabarit de `feedLineColumns`) ; les
 * autres formes (mort neutre, médaille seule, entrée ou sortie, objectif, prise, véhicule) sont une
 * rangée flexible (`FEED_ROW`) dont le premier élément est la gouttière d'instant, à la même largeur que
 * la première colonne de la grille : toutes les formes s'alignent sur l'instant.
 *
 * `flex-nowrap` / `whitespace-nowrap` + `overflow-hidden` SONT LA RÈGLE, pas de la mise en forme : c'est
 * ce couple qui interdit le retour à la ligne. Ce qui déborde est TRONQUÉ, jamais renvoyé à la ligne
 * suivante ; les pictogrammes sont `shrink-0` (une icône rognée ne se lit plus) et le nom cède le
 * premier, l'infobulle le rendant en entier.
 *
 * `shrink-0` SUR LA RANGÉE rend aux lignes leur hauteur : la liste est une colonne flex bornée par la
 * rangée de la page, et `overflow-hidden` met la taille minimale automatique des `li` à zéro — sans
 * `shrink-0` elles s'écrasent jusqu'à tenir dans la hauteur disponible et le fil ne défile plus.
 */

/** La rangée flexible d'une ligne du fil qui n'est pas une élimination. */
export const FEED_ROW =
  'flex h-[28px] shrink-0 flex-nowrap items-center gap-[5px] overflow-hidden text-xs'

/** La grille d'une ligne d'élimination en lecture seule ; le gabarit de colonnes est posé en style. */
export const FEED_GRID_ROW =
  'grid h-[28px] shrink-0 items-center gap-x-[5px] overflow-hidden whitespace-nowrap text-xs'

/**
 * L'ÉLIMINATION CLIQUABLE : le `li` ne porte que la hauteur, et c'est le BOUTON qui prend la grille
 * (mêmes règles : une ligne, rien ne passe à la ligne, ce qui déborde est tronqué), pour que toute la
 * ligne soit la cible du clic. Le focus clavier se voit par un anneau de jeton, jamais par une teinte seule.
 */
export const FEED_ROW_ITEM = 'shrink-0'
export const FEED_GRID_BUTTON =
  'grid h-[28px] w-full cursor-pointer items-center gap-x-[5px] overflow-hidden whitespace-nowrap text-left text-xs hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring'

/**
 * LA LIGNE QUI CONCERNE LE JOUEUR SUIVI par la caméra (`feedFollowed.ts`) : un fond léger à la couleur de suivi
 * (le jeton `ring`, celui de la bordure de la fiche suivie), posé par un calque sous le contenu. Un calque et non un
 * fond de la ligne : il se cumule avec le fond de la ligne choisie (`bg-muted`) et avec la teinte de camp d'un
 * bandeau d'objectif, qu'un second `background` effacerait. AUCUN liseré : le fond seul marque la ligne.
 */
export const FEED_FOLLOWED =
  'relative isolate before:pointer-events-none before:absolute before:inset-0 before:-z-10 before:bg-ring/15'

/** Les classes de la rangée `base`, marquée quand la ligne concerne le joueur suivi. */
export function feedRowClass(base: string, followed: boolean | undefined): string {
  return followed ? `${base} ${FEED_FOLLOWED}` : base
}

/** L'attribut qui dit le marquage (et que les tests lisent) : présent seulement sur une ligne marquée. */
export function followedAttr(followed: boolean | undefined): 'true' | undefined {
  return followed ? 'true' : undefined
}
