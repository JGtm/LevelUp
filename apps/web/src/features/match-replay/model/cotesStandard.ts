/**
 * cotesStandard.ts — LES COTES DE LA PAGE DU CADRE STANDARD : la largeur bornée du contenu et l'en-tête
 * en trois cellules (retour et titre à gauche, bandeau de score au centre, commandes à droite).
 *
 * La largeur de la cellule du centre n'est PAS ici : c'est celle du bandeau de score à l'emplacement
 * `entete` (`model/cotesBandeauScore.ts`), qui en reste l'unique source. Le Cinéma et le Plein cadre
 * ne lisent aucune de ces cotes : ils prennent la largeur du corps.
 */

export const COTES_PAGE_STANDARD = Object.freeze({
  /** Largeur maximale du contenu, px : au-delà, la page se centre et les marges grandissent. */
  contenuMaxPx: 1200,
  /** Marge latérale de la page, de chaque côté du contenu. */
  margeLateralePx: 24,
  /** Hauteur de l'en-tête, et écart entre ses trois cellules. */
  enteteHauteurPx: 64,
  enteteEcartPx: 16,
} as const)

/** La boîte de la page, marges latérales comprises : le contenu garde `contenuMaxPx` une fois les marges retirées. */
export const BOITE_PAGE_STANDARD_MAX_PX =
  COTES_PAGE_STANDARD.contenuMaxPx + 2 * COTES_PAGE_STANDARD.margeLateralePx
