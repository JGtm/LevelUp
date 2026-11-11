/**
 * replayPressGuards — CE QU'UN BOUTON DU REJEU FAIT POUR QUE LA BARRE D'ESPACE RESTE « LECTURE ».
 *
 * Les lignes de frag du fil, les fiches de joueur et les boutons du panneau de joueur sont des
 * boutons posés sur une page dont la touche Espace lance et arrête la lecture. Deux gestes, écrits
 * UNE fois :
 *  - la SOURIS ne prend pas le focus (`mousedown` annulé) : après un clic, Espace est encore le
 *    raccourci global de lecture et ne rebascule pas le bouton ;
 *  - au CLAVIER, Espace active le bouton focalisé et ne remonte pas au raccourci de lecture.
 *
 * Quatre copies du même couple allaient exister (CLAUDE.md n°6) : `replayPressGuards.guard.test.ts`
 * interdit l'ancien littéral hors de ce fichier.
 */
import type { KeyboardEvent, MouseEvent } from 'react'

export const PRESS_GUARDS = {
  onMouseDown: (e: MouseEvent) => e.preventDefault(),
  onKeyDown: (e: KeyboardEvent) => {
    if (e.key === ' ') e.stopPropagation()
  },
}
