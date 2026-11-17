/**
 * cotesEnteteScore.ts — L'EN-TÊTE DU CINÉMA ET DU PLEIN CADRE QUAND IL PORTE LE SCORE : trois cellules (retour
 * et titre, bandeau de score, droite), comme celui du Standard, quand les équipes sont dans la colonne.
 *
 * La largeur de la cellule du score est celle du bandeau à l'emplacement `entete` (`cotesBandeauScore`,
 * source unique) ; la marge latérale du corps est celle de la rangée (`COTES_RANGEE.margeCorps`).
 */
import { COTES_RANGEE } from './cardGabarit'
import { LARGEUR_CELLULE_SCORE_ENTETE_PX } from './cotesBandeauScore'

export const COTES_ENTETE_SCORE = Object.freeze({
  /** Hauteur de l'en-tête, et écart entre ses trois cellules. */
  hauteurPx: 56,
  ecartPx: 16,
  /** Largeur minimale de chacune des deux cellules latérales : le retour, un titre tronqué, ou le lien vers la fiche. */
  celluleLateraleMinPx: 150,
})

/**
 * La largeur du corps du cadre, bord à bord, en dessous de laquelle l'en-tête n'a plus la place de la cellule du
 * score : les deux marges, les deux écarts, les deux cellules latérales minimales et la cellule du score. Le
 * score reste alors sur la toile.
 */
export const LARGEUR_REQUISE_ENTETE_SCORE_PX =
  2 * COTES_RANGEE.margeCorps +
  2 * COTES_ENTETE_SCORE.ecartPx +
  2 * COTES_ENTETE_SCORE.celluleLateraleMinPx +
  LARGEUR_CELLULE_SCORE_ENTETE_PX
