/**
 * useReplayPlayerFocus — CE QUE LE PANNEAU DE JOUEUR FAIT À LA CARTE, câblé au canvas en un point
 * (lot 4 du plan « maquettes du rejeu »).
 *
 * Deux actions et un repère : la fiche ouverte cerne son pion d'un anneau (`openPlayer`). Les JOUEURS MASQUÉS (l'œil
 * d'une tuile, « Seul », qui masque tous les autres) valent pour TOUS les calques : ce qui appartient à un joueur
 * (pion, tirs, croix et effets de mort, poses, véhicules occupés, objets portés…) suit ; ce qui n'est à
 * personne reste. Le hook rend le prédicat `shownAtSlot` que le canvas passe aux calques de pions (pistes,
 * marques de tir, éclairs de bouche, sillage du propulseur, fins de vol de grenade), `shownProjectile` pour
 * les vols de projectile et `shown` (un prédicat par genre d'élément) pour le reste ; `undefined` quand rien
 * n'est filtré, pour que les calques gardent exactement leur dessin d'avant. SUIVRE cadre la carte sur le
 * joueur à chaque image (`useReplayFollow`) : le joueur suivi est celui de la CAMÉRA (`followPlayer`), que
 * la fiche soit ouverte ou fermée, indépendant de la fiche et des joueurs masqués.
 *
 * LE POINT DE VUE N'EST PAS TOUCHÉ (`useReplayViewpoint`) : ces actions ne décident ni de l'encre
 * de camp, ni du roster, ni de la frise — elles ne changent que ce que la carte dessine et où elle
 * regarde. Les joueurs masqués ne sont lus que pour leurs vies.
 */
import { useMemo } from 'react'

import { buildFocusFilter, type FocusFilter } from '../model/focusFilter'
import { shownSlotPredicate, type HiddenPlayers, type PlayerFocus } from '../model/playerPanelLogic'
import { projectileOwners } from '../model/projectileOwners'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { OpenPlayerRing } from '../layers/openPlayerRing'
import { useReplayOpenPlayer } from '../layers/useReplayOpenPlayer'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { QuarterTurns } from '@/lib/replay/orientation'
import { useReplayFollow } from './useReplayFollow'
import type { ReplayZoom } from './useReplayZoom'

export interface ReplayPlayerFocusOptions {
  doc: ReplayDocumentReady
  /** Le joueur du panneau ouvert (l'anneau de son pion) ; `null` = panneau fermé. */
  focus: PlayerFocus | null
  /** Les joueurs masqués, retirés de tous les calques ; `null` = personne n'est masqué, rien n'est filtré. */
  hidden: HiddenPlayers | null
  /** Le joueur que la caméra suit, fiche ouverte ou non ; `null` = la carte ne suit personne. */
  followPlayer: ReplayPlayer | null
  /** La cellule de dessin du canvas (l'image montrée) : lue par Suivre, jamais écrite. */
  frameRef: { readonly current: number }
  zoom: ReplayZoom
  /** L'orientation de la carte : un changement fait recadrer Suivre sur la position courante. */
  turns?: QuarterTurns
}

export interface ReplayPlayerFocusLayers {
  /** Faux = ce slot, à cette image, est retiré de la carte (joueur masqué). `undefined` = rien n'est filtré. */
  shownAtSlot: ((slot: number, frame: number) => boolean) | undefined
  /**
   * Faux = le projectile d'index `i` de `doc.projectiles` (vol de grenade ou de tir) est retiré :
   * son auteur est masqué. Un projectile dont l'auteur ne se lit pas n'est à personne : il reste.
   * `undefined` = rien n'est filtré.
   */
  shownProjectile: ((index: number) => boolean) | undefined
  /**
   * Les joueurs masqués pour tout ce qui n'est PAS un pion : un prédicat par genre d'élément (effets de mort,
   * poses, véhicules, objets portés…), tous issus de la même règle (`ownershipShown`). `undefined` = rien
   * n'est filtré.
   */
  shown: FocusFilter | undefined
  /**
   * Le pion cerclé du joueur de la fiche ouverte : sa position à l'image et les encres de l'anneau. `undefined` =
   * aucune fiche ouverte, ou joueur masqué.
   */
  openPlayer: OpenPlayerRing | undefined
}

export function useReplayPlayerFocus({ doc, focus, hidden, followPlayer, frameRef, zoom, turns }: ReplayPlayerFocusOptions): ReplayPlayerFocusLayers {
  const shownAtSlot = useMemo(() => shownSlotPredicate(hidden) ?? undefined, [hidden])
  useReplayFollow({ doc, player: followPlayer, enabled: followPlayer !== null, frameRef, zoom, turns })
  const owners = useMemo(() => (shownAtSlot ? projectileOwners(doc) : []), [doc, shownAtSlot])
  const shownProjectile = useMemo(() => {
    if (!shownAtSlot) return undefined
    return (index: number) => {
      const owner = owners[index]
      return owner ? shownAtSlot(owner.slot, owner.frame) : true
    }
  }, [shownAtSlot, owners])
  const shown = useMemo(() => buildFocusFilter(doc, hidden) ?? undefined, [doc, hidden])
  const openPlayer = useReplayOpenPlayer(doc, focus)
  return { shownAtSlot, shownProjectile, shown, openPlayer }
}
