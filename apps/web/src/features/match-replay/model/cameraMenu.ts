/**
 * cameraMenu — CE QUE LE MENU DE LA CAMÉRA DE LA BARRE COMPACTE LIT ET COMMANDE, hors de tout composant.
 *
 * LA CAMÉRA N'EST PAS LE POINT DE VUE. « Suivre » cadre la carte sur un joueur à chaque image (l'état
 * `followXuid` du panneau de joueur, indépendant de la fiche) ; le point de vue décide par les yeux de qui
 * se lit le match (camp de référence, frise, écran de fin). Le menu propose les deux, en deux listes, et ne
 * mêle jamais leurs gestes : choisir un joueur dans « Suivre » ne change pas le point de vue, et
 * inversement.
 *
 * `CameraFollow` est ce que la page remet à la toile : le joueur suivi (retrouvé au roster) et le geste
 * « Suivre ce joueur », le même que le bouton « Suivre » de la fiche. La toile cadre sur `player` que la
 * fiche soit ouverte ou fermée.
 */
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ViewpointOptionGroup } from './viewpointOptions'

export interface CameraFollow {
  /** Le joueur que la caméra suit, retrouvé au roster ; `null` = personne, ou joueur absent du roster. */
  player: ReplayPlayer | null
  /** Suivre ce joueur (clé de roster, `ReplayPlayer.xuid`) ; `null` = ne suivre personne. */
  onFollow: (xuid: string | null) => void
}

/** Sans page autour de la toile (un montage isolé) : personne n'est suivi, et le geste ne fait rien. */
export const NO_CAMERA_FOLLOW: CameraFollow = Object.freeze({ player: null, onFollow: () => {} })

/**
 * followedLabelOf rend le nom, dans la liste du menu, du joueur que la caméra suit (clé de roster) ;
 * `null` quand personne n'est suivi ou quand le joueur n'a aucune entrée (aucun nom lisible).
 */
export function followedLabelOf(
  groups: readonly ViewpointOptionGroup[],
  followedKey: string | null,
): string | null {
  if (followedKey === null) return null
  for (const group of groups) {
    const entry = group.options.find((o) => o.rosterKey === followedKey)
    if (entry) return entry.label
  }
  return null
}
