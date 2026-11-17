/**
 * cotesFicheInstant.ts — LES COTES DES SECTIONS DE LA FICHE DU JOUEUR « À L'INSTANT » : l'identité, les lignes
 * libellé / valeur, la barre empilée et le tableau des éliminations, le tableau de l'équipement, la rangée
 * des médailles, les groupes séparés d'un filet. Source unique : les composants de `ui/fiche/` les lisent
 * ici, jamais un nombre écrit dans le rendu.
 *
 * Elles sont relevées sur la maquette de la fiche enrichie. Le bloc « À l'instant » (`ReplayPlayerNow`) garde
 * les siennes (`COTES_BLOC_INSTANT`) ; l'écart entre les sections de la fiche est celui de la colonne du
 * Standard et de la fenêtre (`COTES_FICHE_STANDARD`, `COTES_FENETRE_FICHE`), source unique de chaque surface.
 */
import { COTES_FENETRE_FICHE } from './cotesFenetreFiche'
import { COTES_FICHE_STANDARD } from './cotesFicheStandard'

/** Qui fait défiler la fiche : la colonne du Standard (`outer`) ou la fenêtre du Cinéma et du Plein cadre (`internal`). */
export type FicheSurface = 'outer' | 'internal'

export const COTES_FICHE_INSTANT = Object.freeze({
  /** L'écart entre deux sections du corps, selon la surface qui le porte. */
  ecartSectionsPx: { outer: COTES_FICHE_STANDARD.ecartBlocsPx, internal: COTES_FENETRE_FICHE.ecartBlocsPx },
  /** L'identité : pastille de camp, nom, camp, frags / morts / assistances. */
  identite: Object.freeze({
    ecartPx: 8,
    pastillePx: 8,
    pastilleRayonPx: 2,
    /** Corps du nom : 17 px dans la colonne, 16 px dans la fenêtre. */
    nomPx: { outer: 17, internal: 16 },
    campPx: 12,
    triplePx: 15,
    /** Marge de part et d'autre d'une barre oblique du triplet. */
    separateurMargePx: 2,
  }),
  /** Une section : l'écart entre son titre et ses lignes, et entre ses lignes ; corps de son titre. */
  section: Object.freeze({
    ecartPx: 8,
    titrePx: 11.5,
    titreGroupePx: 11,
    /** Un groupe séparé d'un filet (« Totaux du match », « Gestes ») : l'espace entre le filet et son titre. */
    groupeMargeHautePx: 12,
    /** L'écart entre le titre et la capsule « estimation » qui le suit. */
    titreEcartPx: 8,
  }),
  /** La capsule « estimation » : hauteur, marge intérieure, rayon, corps du texte. */
  capsule: Object.freeze({ hauteurPx: 16, margeIntPx: 5, rayonPx: 4, taillePx: 10.5 }),
  /** Une rangée libellé / valeur : écart entre les rangées, entre le libellé et la valeur ; corps du texte. */
  ligne: Object.freeze({ ecartRangeesPx: 5, ecartColonnesPx: 10, taillePx: 12, ecartLibellePx: 6 }),
  /** La barre empilée des éliminations par classe : hauteur, rayon, écart entre segments. */
  barre: Object.freeze({ hauteurPx: 8, rayonPx: 4, ecartPx: 2 }),
  /** Le tableau des éliminations : colonnes (libellé souple, à l'instant, part, match), écart, corps, pastille, retrait d'une arme. */
  tableauKills: Object.freeze({
    colonnes: 'minmax(0, 1fr) 62px 40px 40px',
    ecartPx: 8,
    taillePx: 12,
    tailleEntetePx: 10.5,
    tailleArmePx: 11.5,
    pastillePx: 9,
    pastilleRayonPx: 2,
    pastilleEcartPx: 7,
    armeRetraitPx: 28,
  }),
  /** Le tableau de l'équipement : colonnes (famille souple, pris, utilisé, perdu), écart, corps du texte et de l'en-tête. */
  tableauEquipement: Object.freeze({
    colonnes: 'minmax(0, 1fr) 44px 44px 44px',
    ecartPx: 8,
    taillePx: 12,
    tailleEntetePx: 10.5,
  }),
  /** Les médailles : écart entre deux, marge haute ; largeur d'une médaille, côté de l'icône, corps du nom et du compte. */
  medailles: Object.freeze({
    ecartPx: 14,
    margeHautePx: 2,
    largeurPx: 44,
    iconePx: 32,
    ecartVerticalPx: 2,
    taillePx: 9,
  }),
})
