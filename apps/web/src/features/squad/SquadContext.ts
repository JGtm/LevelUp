/**
 * SquadContext.ts — Context React + hook d'accès partagé entre SquadLayout
 * et les pages enfants (Synergies / Contributions).
 *
 * Extrait de SquadLayout.tsx pour respecter la règle eslint
 * `react-refresh/only-export-components` : un fichier .tsx ne doit
 * exporter que des composants, sinon HMR casse.
 */
import { createContext, useContext } from 'react'
import type { TeammateRow, TeammatesPageResponse } from '@/lib/api/types'

export interface SquadContextValue {
  selectedRows: TeammateRow[]
  confirmedGamertags: string[]
  pageData: TeammatesPageResponse | null
  /** Slug du joueur principal (URL param) — utilisé par les enfants pour navigation. */
  playerSlug: string
}

export const SquadContext = createContext<SquadContextValue>({
  selectedRows: [],
  confirmedGamertags: [],
  pageData: null,
  playerSlug: '',
})

export function useSquadContext(): SquadContextValue {
  return useContext(SquadContext)
}
