/**
 * useReplayAntagonistMarks : LE CÂBLAGE DES TITRES D'ANTAGONISTE AU CANEVAS, en un seul point.
 *
 * Même parti que les autres hooks `useReplay*` : `ReplayCanvas.tsx` porte un seuil de taille, un
 * calque de plus s'y branche par un import, un appel et une ligne de scène. Le hook rend ce que le
 * calque des pions lit (`MarkerStyle.antagonists`) :
 *  - `titlesOfSlot` : les titres que porte chaque slot à chaque image (`createSlotTitles`, qui lit
 *    l'index du point de vue une fois par image) ;
 *  - `icons` : les deux PNG noirs chargés une fois (`loadedImage`), teintés à la demande par
 *    (rôle, couleur, densité) ; le repeint passe par `redraw` quand une image arrive.
 *
 * Ni index ni titre à porter dans tout le match (`changes` vide) : `undefined`, rien n'est chargé et
 * le calque des pions se dessine exactement comme sans titres. Le point de vue qui change donne un
 * autre index, donc de nouveaux titres, sans recharger les images.
 */
import { useEffect, useMemo } from 'react'

import type { Antagonists } from '@/lib/antagonists/antagonists'
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

import { REPLAY_TIMING_MS } from '../hooks/useReplayTiming'
import { createSlotTitles } from '../model/antagonistTitles'
import { createTitleIcons, TITLE_ICON_URL, type AntagonistMarks } from './antagonistMarks'
import { withLoadedImage } from './loadedImage'

export interface AntagonistMarksInput {
  doc: ReplayDocumentReady
  /** L'index des antagonistes du point de vue ; absent = aucun titre. */
  antagonists: Antagonists | undefined
  /** La clé du film du propriétaire du slot à l'image (`useSlotIdentity`). */
  ownerKeyOfSlot: (slot: number, frame: number) => string | null
  reducedMotion: boolean
  /** Repeindre la scène : les images arrivent après coup. */
  redraw: () => void
}

export function useReplayAntagonistMarks({
  doc,
  antagonists,
  ownerKeyOfSlot,
  reducedMotion,
  redraw,
}: AntagonistMarksInput): AntagonistMarks | undefined {
  const icons = useMemo(() => createTitleIcons(), [])
  const active = !!antagonists && antagonists.changes.length > 0

  useEffect(() => {
    if (!active) return
    let stale = false
    for (const role of ['nemesis', 'victim'] as const) {
      withLoadedImage(TITLE_ICON_URL[role], (image) => {
        if (stale) return
        icons.setSource(role, image)
        redraw()
      })
    }
    return () => {
      stale = true
    }
  }, [active, icons, redraw])

  return useMemo(() => {
    if (!antagonists || !active) return undefined
    const titlesOfSlot = createSlotTitles({
      antagonists,
      ownerKeyOfSlot,
      msOfFrame: (frame) => frameToMs(frame, doc),
      holdMs: REPLAY_TIMING_MS.event,
      reducedMotion,
    })
    return { titlesOfSlot, icons }
  }, [antagonists, active, ownerKeyOfSlot, doc, reducedMotion, icons])
}
