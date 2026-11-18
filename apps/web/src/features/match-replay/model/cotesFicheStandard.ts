/**
 * cotesFicheStandard.ts — LES COTES DE LA FICHE DU JOUEUR DANS LA COLONNE DU STANDARD : la zone qui défile
 * (retour, sélecteur de joueurs, tête, actions, sections) et le pied « Éliminations » fixé sous elle.
 *
 * La fiche remplace toute la colonne : elle reprend la marge intérieure de la colonne
 * (`COTES_COLONNE_STANDARD`, source unique) et ne redéfinit que ce qui lui est propre. La hauteur d'une
 * ligne de l'extrait est celle d'une ligne du fil (`COTES_LIGNE_FIL`, source unique).
 */

export const COTES_FICHE_STANDARD = Object.freeze({
  /** Écart entre les blocs de la zone qui défile. */
  ecartBlocsPx: 14,
  /** Marge à droite de la zone qui défile : la barre de défilement ne mord pas sur le contenu. */
  margeBarrePx: 6,
  retour: Object.freeze({
    /** Hauteur du bouton de retour, écart entre son chevron et son texte, marge intérieure de gauche et de droite. */
    hauteurPx: 26,
    ecartPx: 6,
    margeGauchePx: 4,
    margeDroitePx: 8,
    chevronPx: 14,
    taillePx: 12,
  }),
  capsule: Object.freeze({
    /** La capsule « Échap » : largeur minimale, hauteur, marge intérieure, corps du texte. */
    largeurMinPx: 20,
    hauteurPx: 18,
    margeIntPx: 5,
    taillePx: 10.5,
  }),
  selecteur: Object.freeze({
    /** Un rectangle par joueur : sa largeur, sa hauteur, son rayon et le corps de ses initiales. */
    largeurPx: 34,
    hauteurPx: 22,
    rayonPx: 5,
    taillePx: 10.5,
    /** Nombre de lettres des initiales. */
    initiales: 2,
    /** Écart entre deux rectangles d'un camp, d'une rangée à l'autre, et entre deux camps. */
    ecartPx: 5,
    ecartRangeesPx: 6,
    ecartCampsPx: 18,
    /** L'estompe d'un joueur mort à l'image lue. */
    mortOpacite: 0.35,
    /** Le double anneau du joueur ouvert : le liseré de fond, puis l'anneau de premier plan. */
    anneauFondPx: 2,
    anneauPx: 2,
  }),
  pied: Object.freeze({
    /** Marge intérieure au-dessus du titre (sous le filet), écart sous le titre, nombre de lignes de l'extrait. */
    margeHautePx: 10,
    ecartTitrePx: 4,
    lignes: 2,
    /** Corps du lien « Fil complet ». */
    tailleLienPx: 11,
  }),
})

/** La marge que le double anneau dépasse du rectangle : le sélecteur la réserve pour que la zone qui défile ne le rogne pas. */
export const SELECTEUR_ANNEAU_TOTAL_PX =
  COTES_FICHE_STANDARD.selecteur.anneauFondPx + COTES_FICHE_STANDARD.selecteur.anneauPx
