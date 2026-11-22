/**
 * useReplayAntagonistMarks : LE CÂBLAGE DES TITRES D'ANTAGONISTE AU CANEVAS, en un seul point.
 *
 * Même parti que les autres hooks `useReplay*` : `ReplayCanvas.tsx` porte un seuil de taille, un
 * calque de plus s'y branche par un import, un appel et une ligne de scène. Le hook rend ce que le
 * calque des pions lit (`MarkerStyle.antagonists`) :
 *  - `titlesOfSlot` : les titres que porte chaque slot à chaque image (`createSlotTitles`, qui lit
 *    l'index du point de vue une fois par image) ;
 *  - `icons` : les deux PNG noirs chargés une fois (`loadedImage`), teintés à la demande par
 *    (rôle, couleur, densité) ; le repeint passe par `redraw` quand une image arrive ;
 *  - `advance` : à brancher sur les pas de la boucle de lecture (`onAdvance`), seul déclencheur d'une pulsation.
 *
 * L'HORLOGE DE LA PULSATION EST MURALE (1,4 s, la durée de rémanence d'un événement), comme celle de la mise en
 * valeur d'une élimination (`useReplayKillFocus`) : la lecture peut être en pause quand elle joue, ni l'image du
 * rejeu ni la boucle de lecture ne tournent alors. Une boucle d'images d'écran repeint donc la scène tant qu'une
 * pulsation est en cours, et s'arrête d'elle-même, après un dernier tracé qui pose l'état de repos. Elle ne repeint
 * pas quand un tracé de la LECTURE vient d'avoir lieu (elle repeint à chaque image : la doubler coûterait deux fois), et
 * pose toujours un dernier tracé de repos quand la pulsation est finie.
 *
 * Ni index ni titre à porter dans tout le match (`changes` vide) : `undefined`, rien n'est chargé et
 * le calque des pions se dessine exactement comme sans titres. Le point de vue qui change donne un
 * autre index, donc de nouveaux titres, sans recharger les images.
 */
import { useEffect, useMemo, useRef } from 'react'

import type { Antagonists } from '@/lib/antagonists/antagonists'
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

import { REPLAY_TIMING_MS } from '../hooks/useReplayTiming'
import { createSlotTitles, type SlotTitlesReader } from '../model/antagonistTitles'
import { createTitleIcons, TITLE_ICON_URL, type AntagonistMarks, type TitleIcons } from './antagonistMarks'
import { withLoadedImage } from './loadedImage'

/** En deçà de cet écart depuis le dernier tracé, la boucle de repeinte laisse la lecture peindre (un peu plus d'une image d'écran). */
const RECENT_PAINT_MS = 24

export interface AntagonistMarksInput {
  doc: ReplayDocumentReady
  /** L'index des antagonistes du point de vue ; absent = aucun titre. */
  antagonists: Antagonists | undefined
  /** La clé du film du propriétaire du slot à l'image (`useSlotIdentity`). */
  ownerKeyOfSlot: (slot: number, frame: number) => string | null
  reducedMotion: boolean
  /** Repeindre la scène : les images arrivent après coup, et la pulsation avance quand la lecture est en pause. */
  redraw: () => void
  /** L'horloge murale ; `performance.now` par défaut. */
  now?: () => number
}

/** Ce que le calque lit (`AntagonistMarks`), plus le pas de lecture qui lance une pulsation. */
export interface AntagonistLayer extends AntagonistMarks {
  /** La lecture vient de parcourir les images `from` à `to` (`ReplayPlaybackOptions.onAdvance`). */
  advance: (from: number, to: number) => void
}

const wallClock = (): number => performance.now()

export function useReplayAntagonistMarks({
  doc,
  antagonists,
  ownerKeyOfSlot,
  reducedMotion,
  redraw,
  now = wallClock,
}: AntagonistMarksInput): AntagonistLayer | undefined {
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

  const reader = useMemo(() => {
    if (!antagonists || !active) return undefined
    return createSlotTitles({
      antagonists,
      ownerKeyOfSlot,
      msOfFrame: (frame) => frameToMs(frame, doc),
      holdMs: REPLAY_TIMING_MS.event,
      reducedMotion,
      now,
    })
  }, [antagonists, active, ownerKeyOfSlot, doc, reducedMotion, now])

  // LA BOUCLE DE REPEINTE d'une pulsation : une seule à la fois, arrêtée au démontage ou au changement d'index.
  const raf = useRef(0)
  useEffect(
    () => () => {
      cancelAnimationFrame(raf.current)
      raf.current = 0
    },
    [reader],
  )

  return useMemo(
    () => (reader ? withRepaintLoop({ reader, icons, raf, now, redraw }) : undefined),
    [reader, icons, now, redraw],
  )
}

interface RepaintLoopInput {
  reader: SlotTitlesReader
  icons: TitleIcons
  /** L'identifiant de l'image d'animation en attente ; 0 = aucune boucle. */
  raf: { current: number }
  now: () => number
  redraw: () => void
}

/**
 * withRepaintLoop : le calque que lit le canevas, avec la boucle de repeinte qui mène une pulsation à son terme
 * quand la lecture est en pause (cf. l'en-tête).
 */
function withRepaintLoop({ reader, icons, raf, now, redraw }: RepaintLoopInput): AntagonistLayer {
  // LES TRACÉS DE LA LECTURE SEULS font taire la boucle : son propre repeint lit les titres lui aussi, il ne
  // doit pas lui faire croire qu'une image vient d'être peinte (elle sauterait une image sur deux, et le tracé
  // de repos de la fin).
  let paintedAt = Number.NEGATIVE_INFINITY
  let ownRepaint = false
  const touched = () => {
    if (!ownRepaint) paintedAt = now()
  }
  const repaint = () => {
    ownRepaint = true
    try {
      redraw()
    } finally {
      ownRepaint = false
    }
  }
  const loop = () => {
    if (reader.pulsing()) {
      if (now() - paintedAt > RECENT_PAINT_MS) repaint()
      raf.current = requestAnimationFrame(loop)
      return
    }
    // LA PULSATION EST FINIE : un dernier tracé, toujours, pose l'état de repos (l'ancien tenant n'a plus d'icône).
    raf.current = 0
    repaint()
  }
  const advance = (from: number, to: number) => {
    reader.advance(from, to)
    if (raf.current === 0 && reader.pulsing()) raf.current = requestAnimationFrame(loop)
  }
  return {
    titlesOfSlot: (slot: number, frame: number) => {
      touched()
      return reader.titlesOfSlot(slot, frame)
    },
    titlesOfKey: (key: string, frame: number) => {
      touched()
      return reader.titlesOfKey(key, frame)
    },
    icons,
    advance,
  }
}
