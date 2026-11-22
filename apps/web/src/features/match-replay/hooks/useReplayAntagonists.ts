/**
 * useReplayAntagonists — LA FABRIQUE DES ANTAGONISTES de la page de rejeu, mémoïsée.
 *
 * Ce hook ne calcule rien : la jointure vit dans `model/replayAntagonists`, pure et testée sans
 * React. Il est appelé UNE fois par la page, avec le fil et le roster du modèle
 * (`useReplayAntagonists(model.feed, model.players)`), et la fabrique descend en paramètre aux
 * surfaces qui lisent un titre (canevas, frise, fiche) : aucune ne refait le rapprochement des
 * éliminations. Le fil et le roster ne changent qu'au chargement ou au changement de match : la
 * fabrique, donc ses index par sujet, n'est pas reconstruite à chaque image.
 */
import { useMemo } from 'react'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { createReplayAntagonists, type ReplayAntagonists } from '../model/replayAntagonists'

export function useReplayAntagonists(
  feed: readonly ReplayFeedEntry[],
  players: readonly ReplayPlayer[],
): ReplayAntagonists {
  return useMemo(() => createReplayAntagonists(feed, players), [feed, players])
}
