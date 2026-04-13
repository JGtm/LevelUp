/**
 * Store Zustand — état shell de l'application (joueur courant, locale, etc.)
 *
 * Ce store est hydraté depuis la réponse de /bootstrap au démarrage.
 * Il est la source de vérité locale pour le shell React — pas localStorage.
 *
 * Le contenu de la session serveur est géré côté backend (cookie httpOnly).
 * Ce store ne contient que l'état UI dérivé nécessaire au rendu immédiat.
 */

import { create } from 'zustand'
import type { BootstrapResponse, CapabilityMap, PlayerSummary } from '@/lib/api/types'

interface AppShellState {
  // Joueur courant
  currentPlayer: PlayerSummary | null
  availablePlayers: PlayerSummary[]

  // Configuration
  locale: 'fr' | 'en'
  hintsVisible: boolean
  capabilities: CapabilityMap | null

  // État app
  setupRequired: boolean
  authState: 'missing' | 'partial' | 'ready'
  isBootstrapped: boolean

  // Actions
  hydrateFromBootstrap: (data: BootstrapResponse) => void
  setCurrentPlayer: (player: PlayerSummary) => void
  setLocale: (locale: 'fr' | 'en') => void
  setHintsVisible: (visible: boolean) => void
}

const DEFAULT_CAPABILITIES: CapabilityMap = {
  can_read_local_data: false,
  can_run_sync: false,
  can_use_live_halo: false,
  can_manage_settings: false,
  can_reset_media_index: false,
  can_view_media: false,
}

export const useAppShellStore = create<AppShellState>((set) => ({
  currentPlayer: null,
  availablePlayers: [],
  locale: 'fr',
  hintsVisible: true,
  capabilities: null,
  setupRequired: false,
  authState: 'missing',
  isBootstrapped: false,

  hydrateFromBootstrap: (data: BootstrapResponse) =>
    set({
      currentPlayer: data.current_player,
      availablePlayers: data.available_players,
      locale: (data.locale as 'fr' | 'en') ?? 'fr',
      hintsVisible: data.hints_visible_default,
      capabilities: data.capabilities ?? DEFAULT_CAPABILITIES,
      setupRequired: data.setup_required,
      authState: data.auth_state,
      isBootstrapped: true,
    }),

  setCurrentPlayer: (player) => set({ currentPlayer: player }),
  setLocale: (locale) => set({ locale }),
  setHintsVisible: (visible) => set({ hintsVisible: visible }),
}))
