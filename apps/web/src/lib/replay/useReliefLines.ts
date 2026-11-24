/**
 * useReliefLines — L'INTERRUPTEUR « TRAITS DU RELIEF », retenu PAR FOND, pour les trois vues 2D (rejeu, onglet
 * Tactique, « Occupation du terrain ») : un OUI/NON stocké sous `replay-relief-lines.<fond>`, une seule clé par fond
 * lue et écrite des trois côtés — allumer les traits sur le relief au rejeu les allume partout. Allumés d'office sur le
 * fond Relief, éteints sur le Plan et le Rendu ; le stockage ne retient que ce que l'utilisateur a fait, absent ou refusé
 * -> le défaut du fond.
 *
 * Même patron que `usePersistedFlag`, avec une clé qui dépend du fond : changer de fond relit la valeur
 * de CE fond (le défaut s'il n'a jamais été touché : allumé sur le relief, éteint ailleurs), et l'abonnement
 * fait bouger ensemble tous les composants qui lisent la clé. La persistance vient APRÈS la mise à jour
 * d'état, depuis le gestionnaire d'événement — jamais depuis un updater (cf. `usePersistedFlag`).
 */
import { useCallback, useEffect, useState } from 'react'

import type { MapBackgroundVariant } from './mapBackgroundVariant'
import { persistPreference, readStoredFlag, subscribePreference } from './preferenceStore'

const RELIEF_LINES_KEY_PREFIX = 'replay-relief-lines.'

/** La clé de stockage des traits du relief pour un fond. */
export function reliefLinesKey(fond: MapBackgroundVariant): string {
  return `${RELIEF_LINES_KEY_PREFIX}${fond}`
}

/** Le défaut des traits pour un fond : allumés sur le relief seulement. */
export function reliefLinesDefault(fond: MapBackgroundVariant): boolean {
  return fond === 'relief'
}

/** Les traits du relief retenus pour ce fond, ou son défaut. */
export function readReliefLines(fond: MapBackgroundVariant): boolean {
  return readStoredFlag(reliefLinesKey(fond), reliefLinesDefault(fond))
}

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
