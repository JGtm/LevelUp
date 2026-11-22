/**
 * cotesFenetreFiche.ts — LES COTES DE LA FICHE DU JOUEUR EN FENÊTRE (Cinéma et Plein cadre) : la fenêtre
 * alignée sur la fiche cliquée et posée sous le haut du plan, son contenu, et le pion cerclé du joueur ouvert sur le plan.
 *
 * La capsule « Échap » est celle de la fiche du Standard (`COTES_FICHE_STANDARD.capsule`, source unique).
 * L'écart du panneau à son déclencheur et la hauteur minimale sont ceux de `settings/useAnchoredPanel`.
 */
import { COTES_FICHE_STANDARD } from './cotesFicheStandard'

export const COTES_FENETRE_FICHE = Object.freeze({
  /** Largeur de la fenêtre, px. */
  largeurPx: 330,
  /** Écart entre le haut du bloc du plan et le haut de la fenêtre. */
  margeHautPlanPx: 8,
  /** Rayon de la fenêtre. */
  rayonPx: 12,
  /** Marge intérieure verticale puis latérale, et écart entre les trois blocs de la fiche. */
  margeVerticalePx: 12,
  margeLateralePx: 14,
  ecartBlocsPx: 12,
  /** Marge à droite de la zone qui défile : la barre de défilement ne mord pas sur le contenu. */
  margeBarrePx: 4,
  /** La capsule « Échap » de la tête : la même que celle de la fiche du Standard. */
  capsule: COTES_FICHE_STANDARD.capsule,
  /** La croix de fermeture : côté du bouton, rayon, côté de la croix dessinée, épaisseur de son trait. */
  croix: Object.freeze({ cotePx: 26, rayonPx: 6, tracePx: 12, traitPx: 2 }),
})

/**
 * LE PION CERCLÉ du joueur dont la fiche est ouverte, en pixels d'écran (multipliés par l'échelle de la
 * toile) : un anneau clair posé à `rayonPx` du centre du pion, cerné de part et d'autre d'un liseré à
 * l'encre du fond pour garder un bord franc sur un fond de carte clair. Le rayon dépasse le double
 * contour du joueur de la page et son halo (10,6 px) : les deux repères se lisent ensemble.
 */
export const COTES_PION_CERCLE = Object.freeze({
  rayonPx: 12,
  /** Épaisseur de l'anneau clair, et surépaisseur du liseré sombre de chaque côté. */
  anneauPx: 2,
  lisereExtraPx: 1.2,
})

/**
 * LA FICHE OUVERTE dans la rangée ou dans la colonne : son fond s'éclaircit d'une part du premier plan du thème
 * (posée par-dessus le fond de la tuile, mort comprise) et sa bordure devient celle du premier plan. Aucune cote
 * ne bouge : la bordure existe déjà.
 */
export const COTES_FICHE_OUVERTE = Object.freeze({
  fondPct: 10,
})
