/**
 * useChatterVoices — LES VOIX DES RÉPLIQUES prêtes pour la piste sonore : les répliques qui passent les filtres des réglages
 * (familles, fréquence) et le masquage des joueurs, et qui ont une voix dans la langue de l'interface (cf. `chatterSound.ts`).
 *
 * LA CATÉGORIE « VOIX » ET LE SON ACTIVÉ ne se décident pas ici : `useReplaySound` retire la catégorie coupée à la construction
 * de la piste, et le lecteur ne joue rien tant que le son est coupé. Cette liste dit seulement ce que la piste PEUT jouer.
 */
import { useMemo } from 'react'

import type { Locale } from '@/lib/i18n/locale'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { chatterShownFor, filterChatterLines } from '../model/chatterLines'
import type { HiddenPlayers } from '../model/playerPanelLogic'
import { chatterVoiceEvents } from '../sound/chatterSound'
import type { ReplaySoundEvent } from '../sound/replaySoundVariants'
import { useChatterFilter, useChatterLines } from './useChatterLines'

const NO_VOICES: readonly ReplaySoundEvent[] = []

export function useChatterVoices(
  doc: ReplayDocumentReady,
  locale: Locale,
  hidden: HiddenPlayers | null,
): readonly ReplaySoundEvent[] {
  const lines = useChatterLines(doc, locale)
  const filter = useChatterFilter()
  return useMemo(() => {
    if (lines.length === 0) return NO_VOICES
    const shown = chatterShownFor(hidden)
    const kept = filterChatterLines(lines, filter).filter((l) => shown === null || shown(l))
    return chatterVoiceEvents(kept, doc)
  }, [lines, filter, hidden, doc])
}
