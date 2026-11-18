/**
 * useReliefLines — L'INTERRUPTEUR « TRAITS DU RELIEF », retenu PAR FOND (cf. `RELIEF_LINES_KEY_PREFIX`).
 *
 * Même patron que `usePersistedFlag`, avec une clé qui dépend du fond : changer de fond relit la valeur
 * de CE fond (le défaut s'il n'a jamais été touché : allumé sur le relief, éteint ailleurs), et l'abonnement
 * fait bouger ensemble tous les composants qui lisent la clé. La persistance vient APRÈS la mise à jour
 * d'état, depuis le gestionnaire d'événement — jamais depuis un updater (cf. `usePersistedFlag`).
 */
import { useCallback, useEffect, useState } from 'react'

import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import { persistPreference, readReliefLines, reliefLinesKey, subscribePreference } from './replayPreferences'

/** L'état des traits du relief pour `fond`, et sa bascule. */
export function useReliefLines(fond: MapBackgroundVariant): [boolean, () => void] {
  const [seen, setSeen] = useState(fond)
  const [value, setValue] = useState(() => readReliefLines(fond))
  // UN AUTRE FOND RELIT SA VALEUR, pendant le rendu : une image peinte avec les traits de l'ancien fond serait fausse.
  if (seen !== fond) {
    setSeen(fond)
    setValue(readReliefLines(fond))
  }
  useEffect(() => subscribePreference(reliefLinesKey(fond), (raw) => setValue(raw === 'true')), [fond])
  const toggle = useCallback(() => {
    const next = !value
    setValue(next)
    persistPreference(reliefLinesKey(fond), String(next))
  }, [fond, value])
  return [value, toggle]
}
