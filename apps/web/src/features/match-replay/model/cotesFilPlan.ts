/**
 * cotesFilPlan.ts — LES COTES DU FIL DES ÉLIMINATIONS POSÉ SUR LE PLAN (Cinéma, Plein cadre) : son nombre de
 * lignes, sa largeur, ses marges dans le bloc de la toile, sa couche et l'estompage de ses lignes les plus
 * anciennes. Source unique : le rendu (`ReplayFeedOverlay`) et la règle d'estompage (`planFeed.ts`) les lisent
 * ici, jamais un nombre répété. Les colonnes et la hauteur de ligne sont celles de la ligne du fil
 * (`cotesFeedLine.ts`).
 */

export const COTES_FIL_PLAN = Object.freeze({
  /** Nombre de lignes affichées au plus : les dernières éliminations passées. */
  lignes: 5,
  /** Largeur du fil, bord à bord. */
  largeurPx: 350,
  /** Distance du bord gauche du bloc de la toile. */
  margeGauchePx: 16,
  /** Distance du bord bas du bloc de la toile. */
  margeBasPx: 12,
  /**
   * Couche du fil : sous les surcouches de la carte (écran de fin, message inter-manche, compte à rebours de
   * la bombe, toutes à la couche 10) et au-dessus de la toile.
   */
  couche: 2,
  /**
   * Opacité des deux lignes les plus anciennes quand le fil est plein, de la plus ancienne à la suivante. Un
   * fil de moins de `lignes` lignes n'estompe rien : aucune ligne n'est encore sur le point de sortir.
   */
  estompage: Object.freeze([0.7, 0.85]),
})
