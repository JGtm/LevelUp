/**
 * useReplayPlayerFocus — CE QUE LE PANNEAU DE JOUEUR FAIT À LA CARTE, câblé au canvas en un point
 * (lot 4 du plan « maquettes du rejeu »).
 *
 * Trois actions, deux effets. SEUL et MASQUER valent pour TOUS les calques : ce qui appartient à un joueur
 * (pion, tirs, croix et effets de mort, poses, véhicules occupés, objets portés…) suit ; ce qui n'est à
 * personne reste. Le hook rend le prédicat `shownAtSlot` que le canvas passe aux calques de pions (pistes,
 * marques de tir, éclairs de bouche, sillage du propulseur, fins de vol de grenade), `shownProjectile` pour
 * les vols de projectile et `shown` (un prédicat par genre d'élément) pour le reste ; `undefined` quand rien
 * n'est filtré, pour que les calques gardent exactement leur dessin d'avant. SUIVRE cadre la carte sur le
 * joueur à chaque image (`useReplayFollow`).
 *
 * LE POINT DE VUE N'EST PAS TOUCHÉ (`useReplayViewpoint`) : ces actions ne décident ni de l'encre
 * de camp, ni du roster, ni de la frise — elles ne changent que ce que la carte dessine et où elle
 * regarde. Le joueur choisi n'est lu que pour ses vies.
 */
import { useMemo } from 'react'

import { buildFocusFilter, type FocusFilter } from '../model/focusFilter'
import { shownSlotPredicate, type PlayerFocus } from '../model/playerPanelLogic'
import { projectileOwners } from '../model/projectileOwners'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { QuarterTurns } from '@/lib/replay/orientation'
import { useReplayFollow } from './useReplayFollow'
import type { ReplayZoom } from './useReplayZoom'

export interface ReplayPlayerFocusOptions {
  doc: ReplayDocumentReady
  /** Le joueur du panneau ouvert et ses actions ; `null` = panneau fermé. */
  focus: PlayerFocus | null
  /** La cellule de dessin du canvas (l'image montrée) : lue par Suivre, jamais écrite. */
  frameRef: { readonly current: number }
  zoom: ReplayZoom
  /** L'orientation de la carte : un changement fait recadrer Suivre sur la position courante. */
  turns?: QuarterTurns
}

export interface ReplayPlayerFocusLayers {
  /** Faux = ce slot, à cette image, est retiré de la carte (Seul, Masquer). `undefined` = rien n'est filtré. */
  shownAtSlot: ((slot: number, frame: number) => boolean) | undefined
  /**
   * Faux = le projectile d'index `i` de `doc.projectiles` (vol de grenade ou de tir) est retiré :
   * son auteur est écarté. Un projectile dont l'auteur ne se lit pas n'est à personne : il reste, sous Seul
   * comme sous Masquer. `undefined` = rien n'est filtré.
   */
  shownProjectile: ((index: number) => boolean) | undefined
  /**
   * Seul et Masquer pour tout ce qui n'est PAS un pion : un prédicat par genre d'élément (effets de mort,
   * poses, véhicules, objets portés…), tous issus de la même règle (`ownershipShown`). `undefined` = rien
   * n'est filtré.
   */
  shown: FocusFilter | undefined
}

export function useReplayPlayerFocus({ doc, focus, frameRef, zoom, turns }: ReplayPlayerFocusOptions): ReplayPlayerFocusLayers {
  const shownAtSlot = useMemo(() => shownSlotPredicate(focus) ?? undefined, [focus])
  useReplayFollow({ doc, player: focus?.player ?? null, enabled: focus?.follow ?? false, frameRef, zoom, turns })
  const owners = useMemo(() => (shownAtSlot ? projectileOwners(doc) : []), [doc, shownAtSlot])
  const shownProjectile = useMemo(() => {
    if (!shownAtSlot) return undefined
    return (index: number) => {
      const owner = owners[index]
      return owner ? shownAtSlot(owner.slot, owner.frame) : true
    }
  }, [shownAtSlot, owners])
  const shown = useMemo(() => buildFocusFilter(doc, focus) ?? undefined, [doc, focus])
  return { shownAtSlot, shownProjectile, shown }
}
