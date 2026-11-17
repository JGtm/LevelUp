/**
 * rowLayout — LES ÉQUIPES SE RANGENT-ELLES EN RANGÉE, ET LAQUELLE ? (règle D-12, pure).
 *
 * La rangée affiche `[fiches du camp A] [score] [fiches du camp B]`. Elle n'est retenue que si le
 * match a exactement deux camps et si la largeur du corps loge les fiches à leur largeur minimale, les
 * écarts, le score et les marges. Deux formes, deux règles de tenue qui ne se mélangent pas :
 *  - la rangée ordinaire (`rowFits`) : une fiche par place, `PLACES_PAR_CAMP_MAX_RANGEE` places par camp au plus ;
 *  - la rangée de Grande équipe (`rowFitsGrandeEquipe`) : des emplacements de deux fiches compactes empilées,
 *    de 7 à `PLACES_PAR_CAMP_MAX_GRANDE_EQUIPE` places par camp.
 * `dispositionEquipes` choisit entre les deux et la colonne ; sinon le cadre garde la colonne.
 *
 * Les cotes viennent toutes de `COTES_RANGEE` et `COTES_RANGEE_GRANDE_EQUIPE` (cardGabarit.ts) : aucun littéral
 * de largeur ici. `NB_CAMPS_RANGEE` compte aussi les côtés (une marge par côté, un camp par côté).
 *
 * Pas de React, pas de DOM : géométrie pure, testée dans `rowLayout.test.ts`.
 */
import type { EmplacementScore } from './cotesBandeauScore'
import { COTES_RANGEE, COTES_RANGEE_GRANDE_EQUIPE } from './cardGabarit'
import type { ReplayFrame } from './replayFrame'

/** Le nombre de camps d'une rangée ; il compte aussi ses deux côtés (marges, camps). */
export const NB_CAMPS_RANGEE = 2

/** Le nombre de places par camp au-delà duquel la rangée ne se rend pas (Grande équipe exclue). */
export const PLACES_PAR_CAMP_MAX_RANGEE = 5

/**
 * grandeEquipe rend vrai quand un match à deux camps a, dans l'un d'eux, plus de
 * `PLACES_PAR_CAMP_MAX_RANGEE` places : c'est le seuil de la forme COMPACTE des fiches (la colonne les
 * range par paires), le même que celui au-delà duquel la rangée de fiches ordinaires ne se rend pas.
 * Elle lit le nombre de places, jamais un nom de mode ni un libellé.
 */
export function grandeEquipe(placesParCamp: readonly number[]): boolean {
  return placesParCamp.length === NB_CAMPS_RANGEE && Math.max(...placesParCamp) > PLACES_PAR_CAMP_MAX_RANGEE
}

/** Ce que la règle lit d'un match : ses camps, les places de chaque camp, la largeur du corps. */
export interface RowFitInput {
  camps: number
  placesParCamp: readonly number[]
  largeurCorps: number
}

/**
 * largeurRequiseRangee rend la largeur minimale du corps pour `places` fiches par camp : deux camps
 * de fiches, un écart par fiche, le score entre les deux, et une marge de chaque côté.
 */
export function largeurRequiseRangee(places: number): number {
  const { ficheMinW, ecartFiches, scoreCellW, margeCorps } = COTES_RANGEE
  const fiches = NB_CAMPS_RANGEE * places * (ficheMinW + ecartFiches)
  return fiches + scoreCellW + NB_CAMPS_RANGEE * margeCorps
}

/**
 * rowFits rend vrai quand la rangée se rend : deux camps exactement, une liste de places à deux
 * entrées, chaque camp au plus `PLACES_PAR_CAMP_MAX_RANGEE` places (la rangée est symétrique : on
 * compte le plus grand), et une largeur de corps égale ou supérieure à `largeurRequiseRangee`.
 */
export function rowFits({ camps, placesParCamp, largeurCorps }: RowFitInput): boolean {
  if (camps !== NB_CAMPS_RANGEE) return false
  if (placesParCamp.length !== NB_CAMPS_RANGEE) return false
  const plusGrand = Math.max(...placesParCamp)
  if (plusGrand > PLACES_PAR_CAMP_MAX_RANGEE) return false
  return largeurCorps >= largeurRequiseRangee(plusGrand)
}

/** Le nombre de places par camp au-delà duquel la rangée de Grande équipe ne se rend pas : emplacements fois fiches. */
export const PLACES_PAR_CAMP_MAX_GRANDE_EQUIPE =
  COTES_RANGEE_GRANDE_EQUIPE.emplacementsParCamp * COTES_RANGEE_GRANDE_EQUIPE.fichesParEmplacement

/**
 * largeurRequiseRangeeGrandeEquipe rend la largeur minimale du corps pour la rangée de Grande équipe : le
 * même assemblage que `largeurRequiseRangee` (un écart par colonne de fiches, le score entre les deux camps,
 * une marge de chaque côté), avec `emplacementsParCamp` colonnes de la largeur minimale d'une fiche compacte.
 */
export function largeurRequiseRangeeGrandeEquipe(): number {
  const { ecartFiches, scoreCellW, margeCorps } = COTES_RANGEE
  const { emplacementsParCamp, ficheMinW } = COTES_RANGEE_GRANDE_EQUIPE
  return NB_CAMPS_RANGEE * emplacementsParCamp * (ficheMinW + ecartFiches) + scoreCellW + NB_CAMPS_RANGEE * margeCorps
}

/**
 * rowFitsGrandeEquipe rend vrai quand la rangée de Grande équipe se rend : deux camps exactement, une
 * Grande équipe (`grandeEquipe`), chaque camp au plus `PLACES_PAR_CAMP_MAX_GRANDE_EQUIPE` places, et une largeur
 * de corps égale ou supérieure à `largeurRequiseRangeeGrandeEquipe`. La largeur demandée est celle du pire cas
 * (tous les emplacements d'un camp pleins), quel que soit l'effectif réel.
 */
export function rowFitsGrandeEquipe({ camps, placesParCamp, largeurCorps }: RowFitInput): boolean {
  if (camps !== NB_CAMPS_RANGEE) return false
  if (!grandeEquipe(placesParCamp)) return false
  if (Math.max(...placesParCamp) > PLACES_PAR_CAMP_MAX_GRANDE_EQUIPE) return false
  return largeurCorps >= largeurRequiseRangeeGrandeEquipe()
}

/**
 * colonnesGrandeEquipe rend le nombre d'emplacements que la rangée de Grande équipe dessine par camp : ceux du
 * plus grand camp, à `fichesParEmplacement` fiches chacun (le dernier peut n'en porter qu'une). Au moins un.
 */
export function colonnesGrandeEquipe(placesParCamp: readonly number[]): number {
  const plusGrand = placesParCamp.length === 0 ? 0 : Math.max(...placesParCamp)
  return Math.max(1, Math.ceil(plusGrand / COTES_RANGEE_GRANDE_EQUIPE.fichesParEmplacement))
}

/** Comment les équipes se rangent : la rangée ordinaire, la rangée de Grande équipe, ou la colonne. */
export type DispositionEquipes = 'rangee' | 'rangee-grande-equipe' | 'colonne'

/**
 * dispositionEquipes est LE CHOIX, fonction pure des camps, des places et de la largeur du corps : la rangée
 * ordinaire quand elle tient, sinon la rangée de Grande équipe quand elle tient, sinon la colonne (aucune
 * largeur mesurée, chacun pour soi, un camp trop grand, une fenêtre trop étroite).
 */
export function dispositionEquipes(entree: RowFitInput): DispositionEquipes {
  if (rowFits(entree)) return 'rangee'
  if (rowFitsGrandeEquipe(entree)) return 'rangee-grande-equipe'
  return 'colonne'
}

/**
 * emplacementScoreDe dit où le bandeau de score est posé : dans le Standard, dans son en-tête (il n'est pas
 * sur la toile) ; au Cinéma et au Plein cadre, au centre de la rangée quand elle est posée (il n'est pas non
 * plus sur la toile), sur la toile quand les équipes sont dans la colonne.
 */
export function emplacementScoreDe(frame: ReplayFrame, disposition: DispositionEquipes): EmplacementScore | null {
  if (frame === 'standard') return null
  return disposition === 'colonne' ? 'scene' : null
}

/**
 * alliesAGauche range les deux camps de la rangée pour que le camp du joueur regardé soit à gauche, comme
 * la barre alliée du bandeau de score : l'ordre des désignateurs du film ne décide que quand aucun camp n'est
 * allié (référence absente du film). `ofTeam` est l'allégeance d'un camp (`FilmAllegiance.ofTeam`).
 */
export function alliesAGauche<T extends { team: number }>(
  camps: readonly T[],
  ofTeam: (team: number) => boolean | null,
): T[] {
  if (camps.length === NB_CAMPS_RANGEE && ofTeam(camps[0].team) !== true && ofTeam(camps[1].team) === true) {
    return [camps[1], camps[0]]
  }
  return [...camps]
}
