/**
 * PersonalStatsContext.ts — Context React + hook d'accès partagé entre
 * PersonalStatsLayout et les pages enfants (Résumé / Cartes & Modes /
 * Distributions / Progression / Avancé).
 *
 * Extrait du layout pour respecter la règle eslint
 * `react-refresh/only-export-components` : un fichier .tsx ne doit
 * exporter que des composants, sinon HMR casse.
 *
 * Pattern aligné avec features/squad/SquadContext.ts.
 */
import { createContext, useContext } from 'react'
import type { SynthesisPageResponse } from '@/lib/api/types'

export interface PersonalStatsContextValue {
  /** Réponse de useSynthesisPage — alimente la SessionBriefing solo + onglets enfants. */
  pageData: SynthesisPageResponse | null
  /** Slug du joueur principal (URL param) — utilisé par les enfants pour navigation. */
  playerSlug: string
}

export const PersonalStatsContext = createContext<PersonalStatsContextValue>({
  pageData: null,
  playerSlug: '',
})

export function usePersonalStatsContext(): PersonalStatsContextValue {
  return useContext(PersonalStatsContext)
}
