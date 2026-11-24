/**
 * useFeedFilter — le filtre du fil des éliminations et son choix retenu dans le navigateur
 * (`replayPreferences`). Rend le filtre MÉMORISÉ tel quel : c'est `effectiveFeedFilter` qui dit
 * lequel s'applique à ce match, sans jamais réécrire la préférence.
 *
 * DEUX COMMANDES LE RÉGLENT, la colonne du fil (au Standard) et la rubrique « Fil » du panneau « Vue » (aussi au
 * Cinéma et au Plein cadre, où la colonne est cachée) : l'abonnement par clé les fait bouger ensemble.
 */
import { useCallback, useEffect, useState } from 'react'

import { DEFAULT_FEED_FILTER, FEED_FILTER_IDS, type FeedFilterId } from '../model/feedFilterLogic'
import { FEED_FILTER_KEY, persistFeedFilter, readFeedFilter, subscribePreference } from './replayPreferences'

function isFeedFilter(raw: string): raw is FeedFilterId {
  return (FEED_FILTER_IDS as readonly string[]).includes(raw)
}

export function useFeedFilter(): readonly [FeedFilterId, (id: FeedFilterId) => void] {
  const [stored, setStored] = useState<FeedFilterId>(readFeedFilter)
  useEffect(() => subscribePreference(FEED_FILTER_KEY, (raw) => setStored(isFeedFilter(raw) ? raw : DEFAULT_FEED_FILTER)), [])
  const choose = useCallback((id: FeedFilterId) => {
    persistFeedFilter(id)
    setStored(id)
  }, [])
  return [stored, choose]
}
