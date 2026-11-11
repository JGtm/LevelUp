/**
 * feedRow.ts — LA RANGÉE D'UNE LIGNE DU FIL : la classe partagée par TOUTES les formes de ligne
 * (frag, mort neutre, médaille seule, entrée/sortie, objectif, prise, montée) et la largeur de
 * la gouttière d'horloge qui les ouvre. Constantes seulement : un fichier de composant n'exporte
 * que des composants (react-refresh/only-export-components).
 *
 * `flex-nowrap` + `overflow-hidden` SONT LA RÈGLE, pas de la mise en forme : c'est ce couple qui
 * interdit le retour à la ligne (demande du 2026-08-18, « tout sur une même ligne »). Ce qui
 * déborde est TRONQUÉ, jamais renvoyé à la ligne suivante ; les pictogrammes sont `shrink-0`
 * (une icône rognée ne se lit plus) et le nom cède le premier, l'infobulle le rendant en entier.
 *
 * `shrink-0` SUR LA RANGÉE rend aux lignes leur hauteur naturelle : la liste est une colonne
 * flex bornée par la rangée de la page, et `overflow-hidden` met la taille minimale automatique
 * des `li` à zéro — sans `shrink-0` elles s'écrasent jusqu'à tenir dans la hauteur disponible et
 * le fil ne défile plus (régression C1 du 2026-08-18). Le filet séparateur
 * (`replay-feed-row`, globals.css) structure la liste sans boîte ; le retrait à gauche est SUR
 * la ligne, pas sur la liste, pour que le filet garde sa largeur pleine.
 */

/** La rangée d'une ligne du fil. */
export const FEED_ROW =
  'replay-feed-row flex shrink-0 flex-nowrap items-center gap-2 overflow-hidden py-[5px] pl-2.5 pr-2 text-xs'

/** Largeur de la GOUTTIÈRE D'HORLOGE : fixe, les lignes s'alignent dessus. */
export const FEED_CLOCK_W = 42
