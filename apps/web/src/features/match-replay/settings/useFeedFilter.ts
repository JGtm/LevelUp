/**
 * useFeedFilter — le filtre du fil des éliminations et son choix retenu dans le navigateur
 * (`replayPreferences`). Rend le filtre MÉMORISÉ tel quel : c'est `effectiveFeedFilter` qui dit
 * lequel s'applique à ce match, sans jamais réécrire la préférence.
 */
import { useCallback, useState } from 'react'

import type { FeedFilterId } from '../model/feedFilterLogic'
import { persistFeedFilter, readFeedFilter } from './replayPreferences'

export function useFeedFilter(): readonly [FeedFilterId, (id: FeedFilterId) => void] {
  const [stored, setStored] = useState<FeedFilterId>(readFeedFilter)
  const choose = useCallback((id: FeedFilterId) => {
    persistFeedFilter(id)
    setStored(id)
  }, [])
  return [stored, choose]
}
