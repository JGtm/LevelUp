/**
 * replayCardPress — LA LIGNE ET LA TUILE D'UN JOUEUR SONT DES BOUTONS, au même contrat : ce que la page
 * leur donne pour qu'un clic ouvre la fiche du joueur, et les attributs que ce bouton porte. Écrit une
 * seule fois pour `ReplayPlayerCard` (tuile) et `ReplayRosterRow` (ligne).
 */
import type { KeyboardEvent } from 'react'

import { PRESS_GUARDS } from './replayPressGuards'

/**
 * LA FICHE EN BOUTON (panneau de joueur, lot 4) : ce que la page donne à une fiche pour qu'un clic
 * ouvre le panneau du joueur. `selected` = son panneau est ouvert (`aria-pressed`), `label` = son
 * nom accessible. SANS `press`, la tuile reste une boîte inerte, nœud pour nœud (fixation
 * `__fixtures__/replayTeams.4v4.html`) : les attributs de bouton ne s'ajoutent que quand la page
 * le demande.
 */
export interface TilePress {
  label: string
  selected: boolean
  onActivate: () => void
}

/**
 * LES ATTRIBUTS D'UNE TUILE-BOUTON : un `div` `role="button"` plutôt qu'un `<button>` — la tuile
 * contient des boîtes (jauges, rangées), qu'un bouton HTML n'admet pas. Entrée et Espace
 * activent ; la souris ne prend pas le focus et Espace ne remonte pas au raccourci de lecture
 * (`PRESS_GUARDS`). La tuile choisie porte un anneau (ombre, aucune cote ne bouge).
 */
export function pressProps(press: TilePress) {
  return {
    role: 'button' as const,
    // Repère de la fiche : le panneau de joueur lui rend le focus à sa fermeture.
    'data-replay-player-card': '',
    tabIndex: 0,
    'aria-pressed': press.selected,
    'aria-label': press.label,
    onClick: press.onActivate,
    onMouseDown: PRESS_GUARDS.onMouseDown,
    onKeyDown: (e: KeyboardEvent<HTMLDivElement>) => {
      PRESS_GUARDS.onKeyDown(e)
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        press.onActivate()
      }
    },
  }
}

export const PRESS_CLASS = 'cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring'
export const SELECTED_CLASS = 'ring-2 ring-ring'
