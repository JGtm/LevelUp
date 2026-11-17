/**
 * cotesFeedLine.ts — LES COTES DE LA LIGNE DU FIL : la hauteur d'une ligne, ses colonnes fixes et les
 * tailles de ce qu'elles portent. Source unique : le rendu (grille de la ligne d'élimination, gouttière
 * d'instant des autres formes, pictogrammes) et la classe de rangée (`feedRow.ts`, dont un test tient
 * l'accord) les lisent ici, jamais un nombre répété dans le JSX.
 *
 * L'ordre des colonnes d'une ligne d'élimination est : instant, lieu de la victime, tueur, arme,
 * victime, puis les médailles dans le reste de la largeur. La colonne du lieu n'existe que si la carte
 * a des zones nommées ; les autres gardent alors leurs largeurs et leur alignement.
 */

export const COTES_LIGNE_FIL = Object.freeze({
  /** Hauteur d'une ligne, quelle que soit sa forme. */
  hauteurPx: 28,
  /** Écart entre deux colonnes (et entre deux éléments d'une ligne d'une autre forme). */
  ecartPx: 5,
  /** Corps du texte secondaire : instant et lieu. */
  tailleSecondairePx: 11.5,
  colonnes: Object.freeze({
    instantPx: 30,
    lieuPx: 70,
    tueurPx: 70,
    armePx: 18,
    victimePx: 70,
  }),
  /** Gabarit du pictogramme d'une arme dans sa colonne. */
  arme: Object.freeze({ largeurPx: 18, hauteurPx: 9 }),
  medailles: Object.freeze({
    /** Côté d'un pictogramme de médaille. */
    pictoPx: 13,
    /** Au-delà, un « +N » dit le nombre de médailles de plus ; leurs noms sont en infobulle. */
    pictosMax: 2,
    ecartPx: 3,
    /** Corps du « +N ». */
    tailleResteDemiPx: 10,
    /** Largeur d'une médaille sans visuel : son texte est tronqué à cette largeur. */
    texteMaxPx: 36,
  }),
})

/**
 * Le gabarit de colonnes (`grid-template-columns`) d'une ligne d'élimination. `avecLieu` : la carte a
 * des zones nommées, la colonne du lieu est réservée sur CHAQUE ligne (vide quand la zone est inconnue).
 * Les médailles prennent le reste ; `minmax(0, 1fr)` pour que rien ne force la largeur de la ligne.
 */
export function feedLineColumns(avecLieu: boolean): string {
  const c = COTES_LIGNE_FIL.colonnes
  const fixes = [c.instantPx, ...(avecLieu ? [c.lieuPx] : []), c.tueurPx, c.armePx, c.victimePx]
  return `${fixes.map((px) => `${px}px`).join(' ')} minmax(0, 1fr)`
}
