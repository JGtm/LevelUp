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
import type { BootstrapResponse, CapabilityMap, HaloIdentitySummary, PlayerSummary, TitleSummary } from '@/lib/api/types'
import { api, setApiTitleSlug, setApiLocale } from '@/lib/api/client'

interface AppShellState {
  // Joueur courant
  currentPlayer: PlayerSummary | null
  availablePlayers: PlayerSummary[]

  // Sprint 44 : Titre courant
  currentTitleSlug: string
  availableTitles: TitleSummary[]
  isTitleSwitching: boolean

  // Configuration
  locale: 'fr' | 'en'
  userTimezone: string
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

  // Auth locale
  authMode: 'none' | 'password' | 'xbox'
  registrationMode: 'invite' | 'open' | 'closed'
  /** Instance fermée : aucune nouvelle inscription / identité / BDD. */
  instanceLocked: boolean
  isAdmin: boolean
  currentUsername: string | null
  firstLaunch: boolean
  /** PR 4 — Auth Code Flow OAuth dispo (true si cfg.OAuthRedirectURI configuré). */
  oauthCodeFlowEnabled: boolean
  /** Instance démo : settings figés (read-only sauf langue/accessibilité), switch
   * de langue client-side (le PATCH /settings est refusé en démo). */
  demoMode: boolean

  // Actions
  hydrateFromBootstrap: (data: BootstrapResponse) => void
  setCurrentPlayer: (player: PlayerSummary) => void
  setCurrentTitle: (titleSlug: string) => void
  /** Switch de titre complet : POST /session/context + re-bootstrap + flush stores */
  switchTitle: (titleSlug: string) => Promise<void>
  setLocale: (locale: 'fr' | 'en') => void
  setHintsVisible: (visible: boolean) => void
  /** Reset des données joueur (appelé lors d'un switch titre) */
  resetPlayerData: () => void
  /** Met à jour l'ID du job de sync actif (null = aucun sync en cours). */
  setActiveSyncJobId: (id: string | null) => void
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

export const useAppShellStore = create<AppShellState>((set, get) => ({
  currentPlayer: null,
  availablePlayers: [],
  currentTitleSlug: 'halo_infinite',
  availableTitles: [],
  isTitleSwitching: false,
  locale: 'fr',
  userTimezone: 'Europe/Paris',
  hintsVisible: true,
  capabilities: null,
  setupRequired: false,
  authState: 'missing',
  setupState: 'no_halo_link',
  isBootstrapped: false,
  linkedHaloIdentity: null,
  activeSyncJobId: null,
  authMode: 'none',
  registrationMode: 'invite',
  instanceLocked: false,
  isAdmin: false,
  currentUsername: null,
  firstLaunch: false,
  oauthCodeFlowEnabled: false,
  demoMode: false,

  hydrateFromBootstrap: (data: BootstrapResponse) => {
    const titleSlug = data.current_title_slug ?? 'halo_infinite'
    setApiTitleSlug(titleSlug)
    const locale: 'fr' | 'en' = (data.locale as 'fr' | 'en') ?? 'fr'
    setApiLocale(locale)
    set({
      currentPlayer: data.current_player,
      availablePlayers: data.available_players,
      currentTitleSlug: titleSlug,
      availableTitles: data.available_titles ?? [],
      locale,
      userTimezone: data.settings_excerpt?.user_timezone || 'Europe/Paris',
      hintsVisible: data.hints_visible_default,
      capabilities: data.capabilities ?? DEFAULT_CAPABILITIES,
      setupRequired: data.setup_required,
      authState: data.auth_state,
      setupState: data.setup_state ?? 'no_halo_link',
      isBootstrapped: true,
      linkedHaloIdentity: data.linked_halo_identity ?? null,
      activeSyncJobId: data.active_sync_job_id ?? null,
      authMode: data.auth_mode ?? 'none',
      registrationMode: data.registration_mode ?? 'invite',
      instanceLocked: data.instance_locked ?? false,
      isAdmin: data.is_admin ?? false,
      currentUsername: data.current_username ?? null,
      firstLaunch: data.first_launch ?? false,
      oauthCodeFlowEnabled: data.oauth_code_flow_enabled ?? false,
      demoMode: data.demo_mode ?? false,
    })
  },

  setCurrentPlayer: (player) => {
    set({ currentPlayer: player })
    // Persister le choix côté serveur (fire-and-forget).
    api.post('/session/context', { player_slug: player.player_slug }).catch(() => {})
  },
  setCurrentTitle: (titleSlug) => {
    setApiTitleSlug(titleSlug)
    set({ currentTitleSlug: titleSlug })
  },

  switchTitle: async (titleSlug) => {
    const current = get().currentTitleSlug
    if (titleSlug === current) return

    set({ isTitleSwitching: true })
    try {
      // 1. Informer le backend du changement de titre
      await api.post('/session/context', { title_slug: titleSlug })
      // 2. Mettre à jour le client API
      setApiTitleSlug(titleSlug)
      // 3. Re-bootstrap pour obtenir les données du nouveau titre
      const bootstrap = await api.get<BootstrapResponse>('/bootstrap')
      // 4. Réhydrater le store avec les nouvelles données
      get().hydrateFromBootstrap(bootstrap)
      // 5. Reset des données joueur en cache
      get().resetPlayerData()
    } catch {
      // Rollback silencieux : restaurer l'ancien titre
      setApiTitleSlug(current)
      set({ currentTitleSlug: current })
    } finally {
      set({ isTitleSwitching: false })
    }
  },

  setLocale: (locale) => {
    setApiLocale(locale)
    set({ locale })
  },
  setHintsVisible: (visible) => set({ hintsVisible: visible }),
  setActiveSyncJobId: (id) => set({ activeSyncJobId: id }),

  resetPlayerData: () => {
    set({
      currentPlayer: null,
      availablePlayers: [],
    })
  },
}))
