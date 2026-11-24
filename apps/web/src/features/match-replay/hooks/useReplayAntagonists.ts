/**
 * useReplayAntagonists — LA FABRIQUE DES ANTAGONISTES de la page de rejeu, mémoïsée.
 *
 * Ce hook ne calcule rien : la jointure vit dans `model/replayAntagonists`, pure et testée sans
 * React. Il est appelé UNE fois par la page, avec le fil et le roster du modèle
 * (`useReplayAntagonists(model.feed, model.players, data)`), et la fabrique descend en paramètre aux
 * surfaces qui lisent un titre (canevas, frise, fiche) : aucune ne refait le rapprochement des
 * éliminations. Les éliminations dont un bot est l'auteur ne sont pas dans le fil : elles se lisent dans le document
 * (`botKillsOf`). Le fil, le roster et le document ne changent qu'au chargement ou au changement de match : la
 * fabrique, donc ses index par sujet, n'est pas reconstruite à chaque image.
 */
import { useMemo } from 'react'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayFeedEntry } from '../model/killFeedLogic'
import {
  botKillsOf,
  createReplayAntagonists,
  type BotKillsDocument,
  type ReplayAntagonists,
} from '../model/replayAntagonists'

export function useReplayAntagonists(
  feed: readonly ReplayFeedEntry[],
  players: readonly ReplayPlayer[],
  doc: BotKillsDocument | undefined,
): ReplayAntagonists {
  return useMemo(
    () => createReplayAntagonists(feed, players, doc ? botKillsOf(doc, players) : []),
    [feed, players, doc],
  )
}
