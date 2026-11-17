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
