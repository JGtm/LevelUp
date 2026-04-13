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
import type { BootstrapResponse, CapabilityMap, HaloIdentitySummary, PlayerSummary } from '@/lib/api/types'

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
  setupState: 'no_halo_link' | 'halo_linked_no_profile' | 'profile_ready_no_sync' | 'ready'
  isBootstrapped: boolean

  /** Identité Halo liée à cette session (gamertag + xuid). Null si auth pas encore faite. */
  linkedHaloIdentity: HaloIdentitySummary | null
  /** ID de job de sync initiale actif pour cette session. Null si aucun. */
  activeSyncJobId: string | null

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
  can_self_provision: false,
  can_start_initial_sync: false,
  can_manage_instance: false,
}

export const useAppShellStore = create<AppShellState>((set) => ({
  currentPlayer: null,
  availablePlayers: [],
  locale: 'fr',
  hintsVisible: true,
  capabilities: null,
  setupRequired: false,
  authState: 'missing',
  setupState: 'no_halo_link',
  isBootstrapped: false,
  linkedHaloIdentity: null,
  activeSyncJobId: null,

  hydrateFromBootstrap: (data: BootstrapResponse) =>
    set({
      currentPlayer: data.current_player,
      availablePlayers: data.available_players,
      locale: (data.locale as 'fr' | 'en') ?? 'fr',
      hintsVisible: data.hints_visible_default,
      capabilities: data.capabilities ?? DEFAULT_CAPABILITIES,
      setupRequired: data.setup_required,
      authState: data.auth_state,
      setupState: data.setup_state ?? 'no_halo_link',
      isBootstrapped: true,
      linkedHaloIdentity: data.linked_halo_identity ?? null,
      activeSyncJobId: data.active_sync_job_id ?? null,
    }),

  setCurrentPlayer: (player) => set({ currentPlayer: player }),
  setLocale: (locale) => set({ locale }),
  setHintsVisible: (visible) => set({ hintsVisible: visible }),
}))
