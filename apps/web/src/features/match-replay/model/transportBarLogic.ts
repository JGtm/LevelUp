/**
 * transportBarLogic — DEUX DÉCISIONS PURES DE LA BARRE DE LECTURE D'UNE LIGNE (Cinéma, Plein cadre) : sur
 * combien de lignes elle se range, et sous quelle forme elle se montre au Plein cadre. Pas de React, pas de DOM :
 * le composant mesure et écoute, ces fonctions décident (`transportBarLogic.test.ts`).
 *
 * LES LIGNES. La barre tient sur UNE ligne tant que la frise y garde sa largeur minimale ; sinon la frise passe
 * seule sur la première ligne, toute la largeur, et les commandes sur la seconde. La frise ne descend jamais sous
 * `friseLargeurMinPx` : c'est le critère du passage sur deux lignes, pas une valeur qu'elle atteindrait ensuite.
 * Une marge de bascule (`MARGE_BASCULE_PX`, celle des dispositions de la rangée) évite le battement : on quitte
 * la ligne unique dès que la largeur requise manque, on n'y revient qu'avec la marge en plus.
 *
 * LA FORME. Au Plein cadre, avec la préférence allumée, la barre est posée sur le bas du plan (`surimpression`) et
 * FINE pendant la lecture. Elle est PLEINE dès qu'on s'en sert : pause, survol, focus clavier, geste récent
 * (une touche, un survol tout juste quitté). Hors Plein cadre, ou préférence éteinte, elle reste fixe, sous le plan,
 * et pleine.
 */
import { COTES_BARRE_COMPACTE as C } from './cotesBarreCompacte'
import { usesCompactBar, type ReplayFrame } from './replayFrame'
import { MARGE_BASCULE_PX } from './rowLayout'

/** Le nombre de lignes de la barre. */
export type BarLines = 1 | 2

/** Ce que la règle des lignes lit : la largeur de la barre et celle de chaque groupe de commandes hors frise. */
export interface BarLinesInput {
  /** La marge latérale de la barre, de chaque côté (`barMarginX`) ; absente : celle d'une barre dans le bloc. */
  marginX?: number
  /** La largeur utile de la barre, marges comprises ; 0 tant qu'elle n'est pas mesurée. */
  availableWidth: number
  /** La largeur de chaque groupe de commandes visible, hors frise, avec ses marges propres. */
  groupWidths: readonly number[]
}

/**
 * requiredWidthOneLine rend la largeur minimale pour tenir sur une ligne : les groupes, un écart entre deux
 * éléments (la frise est un élément : autant d'écarts que de groupes), les deux marges de la barre, et la frise à sa
 * largeur minimale avec ses deux marges.
 */
export function requiredWidthOneLine(groupWidths: readonly number[], marginX: number = C.margeXPx): number {
  const groups = groupWidths.reduce((sum, w) => sum + w, 0)
  return groups + C.ecartPx * groupWidths.length + 2 * marginX + C.friseLargeurMinPx + 2 * C.friseMargePx
}

/**
 * barLines est LE CHOIX, avec la décision précédente en mémoire (`null` = aucune). Une largeur ou des groupes pas
 * encore mesurés donnent une ligne. Sur une ligne, on la garde tant que la largeur requise tient ; sur deux, on ne
 * revient à une ligne qu'à la largeur requise plus `MARGE_BASCULE_PX`. Pure : même entrée et même précédente,
 * même réponse.
 */
export function barLines(input: BarLinesInput, previous: BarLines | null): BarLines {
  if (input.availableWidth <= 0 || input.groupWidths.length === 0) return 1
  const required = requiredWidthOneLine(input.groupWidths, input.marginX)
  if (previous === 2) return input.availableWidth >= required + MARGE_BASCULE_PX ? 1 : 2
  return input.availableWidth >= required ? 1 : 2
}

/** La forme de la barre : la forme fine n'existe qu'en surimpression. */
export type BarForm = 'fine' | 'pleine'

/**
 * overlaysTransportBar dit si la barre est posée SUR le plan : au Plein cadre seulement, et seulement si la
 * préférence « barre fine » est allumée. Sinon, elle est fixe sous le plan.
 */
export function overlaysTransportBar(frame: ReplayFrame, thinPreferred: boolean): boolean {
  return frame === 'plein-cadre' && thinPreferred
}

/**
 * LA PLACE DE LA BARRE par rapport au bloc du plan : `in-block` (Standard : la barre complète est dans le bloc,
 * que le cadre pose autour de la carte et de sa colonne), `overlay` (posée sur le bas du plan) ou `below` (sous
 * le bloc du plan, hors de lui, sur le fond de la page : Cinéma, et Plein cadre quand la barre fine est éteinte).
 */
export type BarPlacement = 'in-block' | 'overlay' | 'below'

/**
 * barPlacement décide de la place de la barre : la barre complète est toujours dans le bloc ; la barre d'une ligne
 * est sur le plan quand `overlaysTransportBar`, sinon sous le bloc.
 */
export function barPlacement(frame: ReplayFrame, thinPreferred: boolean): BarPlacement {
  if (!usesCompactBar(frame)) return 'in-block'
  return overlaysTransportBar(frame, thinPreferred) ? 'overlay' : 'below'
}

/** Ce que la règle de forme lit : le cadre et la préférence, la lecture, et ce que l'utilisateur fait de la barre. */
export interface BarFormInput {
  frame: ReplayFrame
  /** La préférence « barre fine au Plein cadre ». */
  thinPreferred: boolean
  /** La lecture tourne (faux = pause). */
  playing: boolean
  /** Le pointeur est sur la barre. */
  hovered: boolean
  /** Un élément de la barre a le focus du clavier. */
  focused: boolean
  /** Un geste est récent : une touche, ou un survol quitté depuis moins de `fineRetenuMs`. */
  recentGesture: boolean
}

/**
 * barForm est LA RÈGLE DE FORME : fine seulement en surimpression, en lecture, sans survol, sans focus clavier et
 * sans geste récent. Toute autre situation rend la barre pleine, et un élément qui a le focus n'est donc jamais
 * dans une barre réduite.
 */
export function barForm(input: BarFormInput): BarForm {
  if (!overlaysTransportBar(input.frame, input.thinPreferred)) return 'pleine'
  const inUse = !input.playing || input.hovered || input.focused || input.recentGesture
  return inUse ? 'pleine' : 'fine'
}

/**
 * barMarginX rend la marge latérale de la barre d'une ligne, de chaque côté : celle du socle quand elle est dans le
 * bloc du plan ou posée sur lui, la marge étroite quand elle est sous le bloc, sur le fond de la page (le bloc porte
 * déjà la sienne). La règle des lignes la compte dans la largeur requise.
 */
export function barMarginX(placement: BarPlacement): number {
  return placement === 'below' ? C.margeXHorsBlocPx : C.margeXPx
}
