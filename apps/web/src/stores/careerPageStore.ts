/**
 * Store de la page Carrière.
 *
 * Gère l'état UI de la page Carrière : panneaux ouverts, onglet actif, etc.
 * Pas de persistance : état réinitialisé à chaque navigation.
 */

import { create } from 'zustand'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface CareerPageState {
  /** Identifiants des panneaux expansibles ouverts */
  expandedPanels: string[]

  /** Onglet actif dans la section top matches ("best" | "worst") */
  selectedTopMatchTab: 'best' | 'worst'

  /** Ouvre / ferme un panneau */
  togglePanel: (panelId: string) => void

  /** Sélectionne un onglet top match */
  setTopMatchTab: (tab: 'best' | 'worst') => void

  /** Réinitialise l'état au montage de la page */
  reset: () => void
}

// ---------------------------------------------------------------------------
// État initial
// ---------------------------------------------------------------------------

const initialState = {
  expandedPanels: ['summary', 'hero-progress'],
  selectedTopMatchTab: 'best' as const,
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useCareerPageStore = create<CareerPageState>()((set) => ({
  ...initialState,

  togglePanel: (panelId) =>
    set((state) => ({
      expandedPanels: state.expandedPanels.includes(panelId)
        ? state.expandedPanels.filter((id) => id !== panelId)
        : [...state.expandedPanels, panelId],
    })),

  setTopMatchTab: (tab) => set({ selectedTopMatchTab: tab }),

  reset: () => set(initialState),
}))
