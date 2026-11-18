/**
 * cotesStandard.ts — LES COTES DE LA PAGE DU CADRE STANDARD : la largeur bornée du contenu et l'en-tête
 * en trois cellules (retour et titre à gauche, bandeau de score au centre, commandes à droite), puis la
 * colonne des équipes et du fil.
 *
 * La largeur de la cellule du centre n'est PAS ici : c'est celle du bandeau de score à l'emplacement
 * `entete` (`model/cotesBandeauScore.ts`), qui en reste l'unique source. Le Cinéma et le Plein cadre
 * ne lisent aucune de ces cotes : ils prennent la largeur du corps.
 */
import { COTES_LIGNE_FIL } from './cotesFeedLine'

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

/**
 * LES COTES DE LA COLONNE du Standard, à droite de la carte (équipes puis fil) : sa largeur fixe, sa marge
 * intérieure (verticale puis latérale), l'écart entre ses deux blocs, et ce que le fil garde TOUJOURS.
 */
export const COTES_COLONNE_STANDARD = Object.freeze({
  /** Largeur de la colonne, filet compris ; la carte prend le reste de la page. */
  largeurPx: 360,
  margeVerticalePx: 12,
  margeLateralePx: 14,
  /** Écart entre le bloc des équipes et celui du fil. */
  ecartBlocsPx: 14,
  /** Nombre de lignes du fil toujours visibles : quand la place manque, ce sont les équipes qui défilent. */
  filLignesMin: 5,
  /**
   * Ce que le fil dépense avant sa première ligne : le titre (menu de filtre : deux fois 4 px de marge et une
   * ligne de 15 px) et la marge de 8 px sous lui. Le fil ne publie pas la hauteur de sa liste ; cette réserve
   * est à relire si l'on change le titre.
   */
  filEnteteReservePx: 23 + 8,
} as const)

/**
 * La hauteur minimale de la cellule du fil : sa réserve de titre, puis `filLignesMin` lignes de la hauteur
 * d'une ligne du fil (`COTES_LIGNE_FIL`, source unique).
 */
export const FIL_STANDARD_HAUTEUR_MIN_PX =
  COTES_COLONNE_STANDARD.filEnteteReservePx + COTES_COLONNE_STANDARD.filLignesMin * COTES_LIGNE_FIL.hauteurPx
