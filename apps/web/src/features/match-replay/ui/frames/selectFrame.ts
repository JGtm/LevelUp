/**
 * selectFrame — LE GESTE DU BOUTON D'UN CADRE : mener la page au cadre demandé avec les seuls
 * événements que le modèle connaît (`toggleCinema`, `toggleFullFrame`, `exit`). Aucune transition
 * n'est redite ici ; ce module ne fait que les enchaîner.
 *
 * Seul cas à deux gestes : le Standard demandé depuis un Plein cadre ouvert depuis le Cinéma. La
 * sortie rend le Cinéma (le cadre d'où l'on vient), et une seconde bascule Cinéma mène au Standard.
 * L'état entre les deux se lit par `peek`, qui rend la vérité du geste précédent sans attendre un rendu.
 *
 * Pendant un export vidéo rien ne bascule (`disabled`) : le modèle l'ignore déjà pour les deux
 * bascules, et la sortie du Plein cadre, qu'il admet, ne se demande pas par ce bouton.
 */
import type { ReplayFrameControl } from '../../hooks/useReplayFrame'
import type { ReplayFullscreen } from '../../hooks/useReplayFullscreen'
import type { ReplayFrame } from '../../model/replayFrame'

export type FrameSelector = Pick<ReplayFrameControl, 'disabled' | 'peek' | 'toggleCinema' | 'toggleFullFrame' | 'exit'>

export function selectFrame(control: FrameSelector, target: ReplayFrame): void {
  if (control.disabled || control.peek().frame === target) return
  switch (target) {
    case 'cinema':
      control.toggleCinema()
      return
    case 'plein-cadre':
      control.toggleFullFrame()
      return
    case 'standard':
      if (control.peek().frame === 'plein-cadre') control.exit()
      if (control.peek().frame === 'cinema') control.toggleCinema()
  }
}

/**
 * leaveFullFrame — LA SORTIE DU PLEIN CADRE par un geste de l'utilisateur (le bouton « Quitter » de l'en-tête,
 * l'icône du Plein cadre courant) : rend le cadre d'où l'on vient et, comme Échap, referme le plein écran du
 * système s'il est atteint. Un seul chemin pour les deux gestes : une sortie qui ne referme que le cadre
 * laisserait la page en plein écran natif sans cadre qui le motive.
 *
 * La sortie reste admise pendant un export vidéo (le modèle l'admet) ; le plein écran du système, lui, est
 * inerte alors (`fullscreen.toggle` ne fait rien si `disabled`) : le cadre se referme, le natif attend la fin.
 */
export function leaveFullFrame(
  control: Pick<ReplayFrameControl, 'exit'>,
  fullscreen: Pick<ReplayFullscreen, 'active' | 'toggle'>,
): void {
  control.exit()
  if (fullscreen.active) fullscreen.toggle()
}
