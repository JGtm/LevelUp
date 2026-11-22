/**
 * usePlayerInstant — LA FICHE D'UN JOUEUR À L'IMAGE LUE : ce qu'il a fait du début du match jusqu'à elle.
 *
 * LES SÉRIES SE LISENT UNE FOIS, jamais par image. Ce qui ne dépend que du document et du lobby (score et rang,
 * tirs, équipement, sprint et sauts) est lu dans un `useMemo` sur le document et ses joueurs ; ce qui dépend du joueur
 * (compteurs, éliminations, lancers, médailles) dans un second, sur le joueur ouvert. L'image lue ne fait
 * qu'interroger ces séries (`playerInstantViewAt`, une dichotomie par série) : la fiche suit la lecture sans rien
 * relire du document.
 */
import { useMemo } from 'react'

import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayLocale } from '../i18n/i18n'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import type { WeaponIconsByKey } from '../model/killSourceIcons'
import { playerInstantViewAt, type PlayerInstantView } from '../model/playerInstantView'
import { readInstantDocSeries, readPlayerInstantSources } from '../model/playerInstantSources'

/** Ce que la lecture demande : le document, le joueur et son lobby, le fil, la langue. */
export interface PlayerInstantInput {
  doc: ReplayDocumentReady
  player: ReplayPlayer
  /** Le lobby du match (`ReplayModel.players`). */
  players: readonly ReplayPlayer[]
  /** Le fil du rejeu (`ReplayModel.feed`). */
  feed: readonly ReplayFeedEntry[]
  /** Première image du match (`ReplayWindowBounds.startFrame`), ou `null`. */
  matchStartFrame: number | null
  weaponIcons: WeaponIconsByKey
  locale: ReplayLocale
}

export function usePlayerInstant(input: PlayerInstantInput, frame: number): PlayerInstantView {
  const { doc, player, players, feed, matchStartFrame, weaponIcons, locale } = input
  const scoreTimeline = useMemo(() => scoreTimelineOf(doc), [doc])
  const docSeries = useMemo(() => readInstantDocSeries(doc, players), [doc, players])
  const sources = useMemo(
    () =>
      readPlayerInstantSources({
        doc, player, players, feed, scoreTimeline, matchStartFrame, weaponIcons, locale, docSeries,
      }),
    [doc, player, players, feed, scoreTimeline, matchStartFrame, weaponIcons, locale, docSeries],
  )
  return useMemo(() => playerInstantViewAt(sources, frame), [sources, frame])
}
