/**
 * replayCardPress — LA LIGNE ET LA TUILE D'UN JOUEUR SONT DES BOUTONS, au même contrat : ce que la page
 * leur donne pour qu'un clic ouvre la fiche du joueur, et les attributs que ce bouton porte. Écrit une
 * seule fois pour `ReplayPlayerCard` (tuile) et `ReplayRosterRow` (ligne).
 */
import type { CSSProperties, KeyboardEvent } from 'react'

import { COTES_FICHE_OUVERTE } from '../model/cotesFenetreFiche'
import { PRESS_GUARDS } from './replayPressGuards'

/**
 * LA FICHE EN BOUTON (panneau de joueur, lot 4) : ce que la page donne à une fiche pour qu'un clic
 * ouvre le panneau du joueur. `selected` = son panneau est ouvert (`aria-pressed`), `label` = son
 * nom accessible. SANS `press`, la tuile reste une boîte inerte, nœud pour nœud : les attributs
 * de bouton ne s'ajoutent que quand la page le demande.
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
 * (`PRESS_GUARDS`). La tuile choisie prend l'habit de `selectedChrome`.
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

/**
 * L'HABIT DE LA FICHE OUVERTE : le fond plus clair et la bordure claire, par jetons. Il se pose sur le `chrome` de
 * la tuile (fond et bordure inline) : la part de premier plan est une couche posée AU-DESSUS du fond d'origine, qui
 * garde sa teinte (une fiche morte reste rouge sombre). Sans `chrome` (une ligne, fond transparent), il ne reste
 * que la couche claire. Aucune cote ne bouge.
 */
export function selectedChrome(chrome: CSSProperties = {}): CSSProperties {
  const couche = `color-mix(in srgb, var(--foreground) ${COTES_FICHE_OUVERTE.fondPct}%, transparent)`
  const dessous = chrome.background === undefined ? '' : `, ${chrome.background}`
  return { ...chrome, borderColor: 'var(--foreground)', background: `linear-gradient(${couche}, ${couche})${dessous}` }
}
