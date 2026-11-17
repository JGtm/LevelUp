/**
 * cotesBandeauScore.ts — LES COTES DU BANDEAU DE SCORE (`ui/ReplayScoreBanner.tsx`), PAR EMPLACEMENT.
 *
 * Le bandeau se pose à trois endroits, et chacun a ses proportions : au-dessus de la toile (`scene`),
 * dans la cellule centrale de l'en-tête du Standard (`entete`), au centre de la rangée de fiches du
 * Cinéma (`rangee`). Le composant reçoit son emplacement et lit CE tableau ; il n'écrit aucun nombre.
 * La lecture du bandeau (camps, scores, manche, garde de colline) ne dépend jamais de l'emplacement :
 * seules les dimensions changent (`model/scoreBannerLogic.ts` ne les connaît pas).
 *
 * Toutes les cotes s'appliquent en `style` (nombres en pixels) : un littéral Tailwind arbitraire ne
 * naît que s'il est écrit en clair dans une classe, une interpolation ne produit aucune règle.
 */
import { COTES_RANGEE } from './cardGabarit'

/** Où le bandeau est posé. */
export type EmplacementScore = 'scene' | 'entete' | 'rangee'

export interface CotesBandeauScore {
  /** Largeur de la cellule du bandeau, px ; `null` : il remplit la largeur que son parent lui laisse. */
  largeurPx: number | null
  /** Marge sous le bandeau. */
  margeBassePx: number
  /** Hauteur, rayon et liseré extérieur d'une barre de camp. */
  barreHauteurPx: number
  barreRayonPx: number
  lisereBordPx: number
  /** Écart entre une barre et l'horloge. */
  ecartPx: number
  /** Taille du nombre écrit dans la barre et marge latérale de ce nombre. */
  scorePx: number
  scorePadXPx: number
  /** Horloge : taille du texte, largeur minimale de sa cellule, marge latérale ; taille et marge haute du numéro de manche. */
  horlogePx: number
  horlogeMinW: number
  horlogePadXPx: number
  manchePx: number
  mancheMargeHautePx: number
  /** Pastilles de manche : côté, écart entre deux pastilles, marge sous la rangée. */
  pastillePx: number
  pastillesEcartPx: number
  pastillesMargeBassePx: number
  /** Jauge de garde de colline : hauteur, et écart entre la barre et elle. */
  gardeHauteurPx: number
  gardeEcartPx: number
}

/** Le bandeau tel qu'il coiffe la toile en Cinéma et en Plein cadre : barres de 28 px, toute la largeur. */
const COTES_SCENE: CotesBandeauScore = Object.freeze({
  largeurPx: null,
  margeBassePx: 8,
  barreHauteurPx: 28,
  barreRayonPx: 2,
  lisereBordPx: 3,
  ecartPx: 8,
  scorePx: 15,
  scorePadXPx: 8,
  horlogePx: 14,
  horlogeMinW: 0,
  horlogePadXPx: 4,
  manchePx: 11,
  mancheMargeHautePx: 2,
  pastillePx: 10,
  pastillesEcartPx: 6,
  pastillesMargeBassePx: 4,
  gardeHauteurPx: 4,
  gardeEcartPx: 2,
} as const)

/** Les proportions des tableaux A (en-tête du Standard) et K (rangée) : barres de 24 px, horloge de 44 px. */
const COTES_COMPACTES = Object.freeze({
  ...COTES_SCENE,
  margeBassePx: 0,
  barreHauteurPx: 24,
  barreRayonPx: 3,
  scorePx: 14,
  horlogePx: 13,
  horlogeMinW: 44,
  horlogePadXPx: 0,
} as const)

/** La largeur de la cellule du score dans un en-tête : 440 px (tableau A, retenue sur J), manches et garde comprises. */
export const LARGEUR_CELLULE_SCORE_ENTETE_PX = 440

/** La cellule centrale de l'en-tête (Standard ; Cinéma et Plein cadre quand les équipes sont dans la colonne). */
const COTES_ENTETE: CotesBandeauScore = Object.freeze({ ...COTES_COMPACTES, largeurPx: LARGEUR_CELLULE_SCORE_ENTETE_PX } as const)

/** La cellule du score au centre de la rangée : la cote de `COTES_RANGEE`, jamais recopiée. */
const COTES_RANGEE_SCORE: CotesBandeauScore = Object.freeze({
  ...COTES_COMPACTES,
  largeurPx: COTES_RANGEE.scoreCellW,
} as const)

/** Les cotes du bandeau de score, par emplacement : la table est fermée, un emplacement ajouté la complète. */
export const COTES_BANDEAU_SCORE: Readonly<Record<EmplacementScore, CotesBandeauScore>> = Object.freeze({
  scene: COTES_SCENE,
  entete: COTES_ENTETE,
  rangee: COTES_RANGEE_SCORE,
})
