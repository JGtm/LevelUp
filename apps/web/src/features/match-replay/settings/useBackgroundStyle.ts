/**
 * useBackgroundStyle — LE STYLE DU FOND DE CARTE (rendu ou plan), préférence globale persistée.
 *
 * Même patron que `usePersistedFlag` pour un choix à deux valeurs : l'état initial vient du
 * stockage, l'abonnement fait bouger ensemble tous les composants qui lisent la clé (le panneau
 * « Vue » écrit, la page du rejeu et le canvas lisent), et la persistance vient APRÈS la mise à
 * jour d'état, depuis le gestionnaire d'événement — jamais depuis un updater (cf.
 * `usePersistedFlag`).
 */
import { useCallback, useEffect, useState } from 'react'

import { normalizeBackgroundStyle, type MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import {
  BACKGROUND_STYLE_KEY,
  persistBackgroundStyle,
  readBackgroundStyle,
  subscribePreference,
} from './replayPreferences'

/** Le style de fond retenu, et la fonction qui le change et le retient. */
export function useBackgroundStyle(): [MapBackgroundVariant, (style: MapBackgroundVariant) => void] {
  const [style, setStyle] = useState<MapBackgroundVariant>(readBackgroundStyle)
  useEffect(() => subscribePreference(BACKGROUND_STYLE_KEY, (raw) => setStyle(normalizeBackgroundStyle(raw))), [])
  const select = useCallback((next: MapBackgroundVariant) => {
    setStyle(next)
    persistBackgroundStyle(next)
  }, [])
  return [style, select]
}
