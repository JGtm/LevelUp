// cross-feature-allow: la sélection d'escouade de la page Escouade (coéquipiers et composition
// stricte, persistées par joueur) — la même clé de stockage, donc la même sélection, dans les
// deux pages.
/**
 * useTendancesSquadSelection — la sélection d'escouade de la vue Escouade des Tendances : les
 * coéquipiers choisis et l'option « Composition stricte ». L'état initial est LU dans le
 * stockage de la page Escouade et chaque changement y est ÉCRIT : choisir une escouade ici ou là
 * donne la même sélection.
 */
import { useCallback, useState } from 'react'

import {
  readStoredExactComposition,
  writeStoredExactComposition,
} from '@/features/squad/exactComposition'
import { readStoredTeammates, writeStoredTeammates } from '@/features/squad/squadSelectionStorage'

export interface TendancesSquadSelection {
  squadGamertags: string[]
  setSquadGamertags: (next: string[]) => void
  exactComposition: boolean
  setExactComposition: (value: boolean) => void
}

export function useTendancesSquadSelection(playerSlug: string): TendancesSquadSelection {
  const [squadGamertags, setGamertags] = useState<string[]>(() => readStoredTeammates(playerSlug))
  const [exactComposition, setExact] = useState<boolean>(() =>
    readStoredExactComposition(playerSlug),
  )
  const setSquadGamertags = useCallback(
    (next: string[]) => {
      setGamertags(next)
      writeStoredTeammates(playerSlug, next)
    },
    [playerSlug],
  )
  const setExactComposition = useCallback(
    (value: boolean) => {
      setExact(value)
      writeStoredExactComposition(playerSlug, value)
    },
    [playerSlug],
  )
  return { squadGamertags, setSquadGamertags, exactComposition, setExactComposition }
}
