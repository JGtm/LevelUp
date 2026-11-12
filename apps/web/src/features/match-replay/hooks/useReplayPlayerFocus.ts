/**
 * useReplayPlayerFocus — CE QUE LE PANNEAU DE JOUEUR FAIT À LA CARTE, câblé au canvas en un point
 * (lot 4 du plan « maquettes du rejeu »).
 *
 * Trois actions, deux effets. SEUL et MASQUER filtrent le dessin : le hook rend le prédicat
 * `shownAtSlot` que le canvas passe aux calques des éléments propres au joueur (pistes, marques de
 * tir, éclairs de bouche, sillage du propulseur, fins de vol de grenade) et `shownProjectile` pour
 * les vols de projectile ; `undefined` quand rien n'est filtré, pour que les calques gardent
 * exactement leur dessin d'avant. SUIVRE cadre la carte sur le joueur à chaque
 * image (`useReplayFollow`).
 *
 * LE POINT DE VUE N'EST PAS TOUCHÉ (`useReplayViewpoint`) : ces actions ne décident ni de l'encre
 * de camp, ni du roster, ni de la frise — elles ne changent que ce que la carte dessine et où elle
 * regarde. Le joueur choisi n'est lu que pour ses vies.
 */
import { useMemo } from 'react'

import { shownSlotPredicate, type PlayerFocus } from '../model/playerPanelLogic'
import { projectileOwners } from '../model/projectileOwners'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { useReplayFollow } from './useReplayFollow'
import type { ReplayZoom } from './useReplayZoom'

export interface ReplayPlayerFocusOptions {
  doc: ReplayDocumentReady
  /** Le joueur du panneau ouvert et ses actions ; `null` = panneau fermé. */
  focus: PlayerFocus | null
  /** La cellule de dessin du canvas (l'image montrée) : lue par Suivre, jamais écrite. */
  frameRef: { readonly current: number }
  zoom: ReplayZoom
}

export interface ReplayPlayerFocusLayers {
  /** Faux = ce slot, à cette image, est retiré de la carte (Seul, Masquer). `undefined` = rien n'est filtré. */
  shownAtSlot: ((slot: number, frame: number) => boolean) | undefined
  /**
   * Faux = le projectile d'index `i` de `doc.projectiles` (vol de grenade ou de tir) est retiré :
   * son auteur est écarté. Seul ne garde que ce qui est PROUVÉ au joueur (un projectile sans auteur
   * connu est retiré) ; Masquer ne retire que ce qui est PROUVÉ à lui (sans auteur connu, il reste).
   * Les véhicules ne passent pas par là : ils peuvent porter d'autres joueurs. `undefined` = rien
   * n'est filtré.
   */
  shownProjectile: ((index: number) => boolean) | undefined
}

export function useReplayPlayerFocus({ doc, focus, frameRef, zoom }: ReplayPlayerFocusOptions): ReplayPlayerFocusLayers {
  const shownAtSlot = useMemo(() => shownSlotPredicate(focus) ?? undefined, [focus])
  useReplayFollow({ doc, player: focus?.player ?? null, enabled: focus?.follow ?? false, frameRef, zoom })
  const owners = useMemo(() => (shownAtSlot ? projectileOwners(doc) : []), [doc, shownAtSlot])
  const solo = focus?.solo ?? false
  const shownProjectile = useMemo(() => {
    if (!shownAtSlot) return undefined
    return (index: number) => {
      const owner = owners[index]
      return owner ? shownAtSlot(owner.slot, owner.frame) : !solo
    }
  }, [shownAtSlot, owners, solo])
  return { shownAtSlot, shownProjectile }
}
