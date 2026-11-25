/**
 * useReplayChatter — LES RÉPLIQUES SUR LA CARTE, câblées au canvas : les répliques affichables du match, filtrées par les
 * familles et la fréquence des réglages et par les joueurs masqués, et le peintre que la scène ordonne au-dessus des pions.
 *
 * `available` dit au panneau ce que le film porte (jamais ce que les interrupteurs montrent) : une réplique affichable existe,
 * et l'une d'elles a une voix dans la langue de l'interface. L'interrupteur du calque est « bulle OU icône » : éteints tous les
 * deux, le calque ne s'ouvre pas.
 *
 * SEUL / MASQUER : une réplique est à son joueur (`focusShown.author`), comme ses traces et ses effets.
 */
import { useMemo } from 'react'

import type { Locale } from '@/lib/i18n/locale'

import { durationFrameMs, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { liveChatter } from '../model/chatterBubbles'
import type { ChatterLine } from '../model/chatterLines'
import type { CanvasView } from '../model/replayView'
import { useChatterFilter, useChatterLines, useFilteredChatterLines } from '../hooks/useChatterLines'
import { useCarrierPosAt } from '../model/carrierPosition'
import { drawChatter, type ChatterInk } from './chatterLayer'
import type { NamedLayerPainter } from './replayCompose'
import type { SlotIdentity } from './useSlotIdentity'

export interface ChatterOptions {
  doc: ReplayDocumentReady
  locale: Locale
  view: CanvasView
  /** Les deux interrupteurs d'affichage des réglages. */
  showBubbles: boolean
  showIcon: boolean
  ink: ChatterInk
  /** La couleur de repli d'un joueur dont l'équipe n'est pas lue. */
  fallbackColor: string
  /** Les lectures d'identité du canvas : la couleur d'un joueur. */
  identity: Pick<SlotIdentity, 'colorOfXuid'>
  /** « Seul » et « Masquer » : la réplique d'un joueur masqué ne se dessine pas. Absent = rien n'est filtré. */
  shownAuthor?: (xuid: string) => boolean
  reducedMotion: boolean
}

export interface ReplayChatterLayer {
  /** Le calque, à passer à `bindPainters`. */
  layers: readonly [NamedLayerPainter<'repliques'>]
  /** L'un des deux affichages est allumé : sinon le calque ne s'ouvre pas. */
  show: boolean
  /** Ce que le film porte : au moins une réplique affichable, et l'une d'elles a une voix dans la langue. */
  available: { count: number; withVoice: number }
}

export function useReplayChatter(o: ChatterOptions): ReplayChatterLayer {
  const { doc, locale, view, showBubbles, showIcon, ink, fallbackColor, identity, shownAuthor, reducedMotion } = o
  const all = useChatterLines(doc, locale)
  const filter = useChatterFilter()
  const filtered = useFilteredChatterLines(all, filter)
  const posOf = useCarrierPosAt(doc)
  const frameMs = useMemo(() => durationFrameMs(doc), [doc])
  const { colorOfXuid } = identity
  const visible = useMemo<readonly ChatterLine[]>(
    () => (shownAuthor ? filtered.filter((l) => shownAuthor(l.xuid)) : filtered),
    [filtered, shownAuthor],
  )
  const layers = useMemo(
    () =>
      [
        {
          id: 'repliques',
          paint: (ctx: CanvasRenderingContext2D, frame: number, k: number) =>
            drawChatter(ctx, view, {
              live: liveChatter(visible, frame, frameMs, reducedMotion),
              posOf: (xuid: string, f: number): XY | null => posOf(xuid, f),
              colorOf: colorOfXuid,
              frame,
              showBubble: showBubbles,
              showIcon,
              ink,
              fallbackColor,
              k,
            }),
        },
      ] as const satisfies ReplayChatterLayer['layers'],
    [view, visible, frameMs, reducedMotion, posOf, colorOfXuid, showBubbles, showIcon, ink, fallbackColor],
  )
  const available = useMemo(
    () => ({ count: all.length, withVoice: all.filter((l) => l.voice !== null).length }),
    [all],
  )
  const show = showBubbles || showIcon
  return useMemo(() => ({ layers, show, available }), [layers, show, available])
}
