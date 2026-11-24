/**
 * ficheViewpoint — CE QUE LA FICHE D'UN JOUEUR DIT DU POINT DE VUE : « Prendre le point de vue » pose ce joueur comme
 * point de vue de la page (`useReplayViewpoint`), le même geste que le menu de la frise et celui de la caméra. Rien n'est
 * recopié : la valeur est celle du menu (`viewpointValueOf`), et la règle d'inertie aussi — un joueur sans ligne de tableau
 * de score est listé mais inerte, sa piste serait vide quoi qu'il arrive (cf. `model/viewpointOptions`).
 *
 * Il ne touche ni au temps, ni aux calques, ni au contrat : changer de joueur change ce qu'on voit, jamais où l'on en est.
 */
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayViewpoint } from '../../hooks/useReplayViewpoint'
import { viewpointValueOf } from '../../model/viewpointOptions'

export interface FicheViewpoint {
  /** Ce joueur est déjà le point de vue : le bouton n'a rien à faire. */
  current: boolean
  /** Sans ligne de tableau de score : le bouton est grisé, la raison en infobulle. */
  inert: boolean
  onTake: () => void
}

export function ficheViewpointOf(player: ReplayPlayer, viewpoint: ReplayViewpoint): FicheViewpoint {
  const value = viewpointValueOf(player)
  return {
    current: viewpoint.xuid === value,
    inert: !player.board,
    onTake: () => viewpoint.select(value),
  }
}
