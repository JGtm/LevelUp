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
import type { Locale } from '@/lib/i18n/locale'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'

interface AppShellState {
  // Joueur courant
  currentPlayer: PlayerSummary | null
  availablePlayers: PlayerSummary[]

  // Sprint 44 : Titre courant
  currentTitleSlug: string
  availableTitles: TitleSummary[]
  isTitleSwitching: boolean

  // Configuration
  locale: Locale
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
  /** Joueur courant : refresh_token Microsoft mort → reconnexion Xbox requise. */
  reauthRequired: boolean
  /** Utilisateur connecté : a défini un mot de passe (opt-in re-login rapide). */
  hasPassword: boolean
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
  setLocale: (locale: Locale) => void
  setHintsVisible: (visible: boolean) => void
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

export const useAppShellStore = create<AppShellState>((set) => ({
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
  reauthRequired: false,
  hasPassword: false,
  isAdmin: false,
  currentUsername: null,
  firstLaunch: false,
  oauthCodeFlowEnabled: false,
  demoMode: false,

  hydrateFromBootstrap: (data: BootstrapResponse) => {
    const titleSlug = data.current_title_slug ?? 'halo_infinite'
    setApiTitleSlug(titleSlug)
    const locale: Locale = (data.locale as Locale) ?? 'fr'
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
      reauthRequired: data.reauth_required ?? false,
      hasPassword: data.has_password ?? false,
      isAdmin: data.is_admin ?? false,
      currentUsername: data.current_username ?? null,
      firstLaunch: data.first_launch ?? false,
      oauthCodeFlowEnabled: data.oauth_code_flow_enabled ?? false,
      demoMode: data.demo_mode ?? false,
    })
    // Garde deep-link (fresh-load / bookmark) : si un `?f=` (share-link) a hydraté
    // le store solo avec un filtre généré pour un AUTRE titre, le reset. Le reset
    // au switch de titre (applyActiveTitle) ne couvre PAS ce chemin — au fresh-load,
    // seul le bootstrap connaît le titre actif réel. setApiTitleSlug(titleSlug) a
    // déjà été appelé ci-dessus → un éventuel resetFilters ré-estampille l'URL au
    // bon titre. Squad n'a pas de deep-link (urlEnabled=false) → no-op inoffensif.
    useSoloFilterStore.getState().reconcileActiveTitle(titleSlug)
    useSquadFilterStore.getState().reconcileActiveTitle(titleSlug)
  },

  setCurrentPlayer: (player) => {
    set({ currentPlayer: player })
    // Persister le choix côté serveur (fire-and-forget).
    api.post('/session/context', { player_slug: player.player_slug }).catch(() => {})
  },

  setLocale: (locale) => {
    setApiLocale(locale)
    set({ locale })
  },
  setHintsVisible: (visible) => set({ hintsVisible: visible }),
  setActiveSyncJobId: (id) => set({ activeSyncJobId: id }),
}))

/**
 * Entrée du title-switcher dérivée d'`availableTitles` (MT-22 / PMT-8).
 *
 * Le backend (`BuildAvailableTitles`) liste désormais les titres `active` ET
 * `coming_soon` (exclut `archived`), en conservant leur `status`. Le front
 * consomme ce signal : un titre `coming_soon` est listé mais `disabled`
 * (affichage « bientôt disponible ») ; un titre `archived` est exclu (garde-fou
 * défensif, le backend ne le renvoie déjà plus). L'entrée du titre courant est
 * marquée `isCurrent`.
 */
export interface TitleSwitcherEntry {
  slug: string
  name: string
  status: string
  disabled: boolean
  isCurrent: boolean
}

export function buildTitleSwitcherEntries(
  titles: TitleSummary[],
  currentSlug: string,
): TitleSwitcherEntry[] {
  return titles
    .filter((t) => t.status !== 'archived')
    .map((t) => ({
      slug: t.slug,
      name: t.name,
      status: t.status ?? 'active',
      disabled: (t.status ?? 'active') !== 'active',
      isCurrent: t.slug === currentSlug,
    }))
}
