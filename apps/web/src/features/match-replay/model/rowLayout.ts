/**
 * rowLayout — LA RANGÉE DE FICHES TIENT-ELLE DANS LA LARGEUR DU CORPS ? (règle D-12, pure).
 *
 * La rangée affiche `[fiches du camp A] [score] [fiches du camp B]`. Elle n'est retenue que si le
 * match a exactement deux camps, chacun de `PLACES_PAR_CAMP_MAX_RANGEE` places au plus, et si la
 * largeur du corps loge toutes les fiches à leur largeur minimale, les écarts, le score et les marges.
 * Sinon le cadre garde la colonne du Standard.
 *
 * Les cotes viennent toutes de `COTES_RANGEE` (cardGabarit.ts) : aucun littéral de largeur ici.
 * `NB_CAMPS_RANGEE` compte aussi les côtés (un marge par côté, un camp par côté).
 *
 * Pas de React, pas de DOM : géométrie pure, testée dans `rowLayout.test.ts`.
 */
import { COTES_RANGEE } from './cardGabarit'

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
