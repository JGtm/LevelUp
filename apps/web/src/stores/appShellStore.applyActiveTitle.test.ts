/**
 * Tests — applyActiveTitle : ordre load-bearing complet + reset filtres.
 *
 * MIGRÉ depuis appShellStore.switchTitle.test.ts (Phase 4a). Le wrapper
 * `switchTitle` a été SUPPRIMÉ du store (navigate-first D-6 : le TitleSwitcher
 * navigue, le layout `t/$titleSlug` exécute la bascule). Les assertions PR #59
 * (correction #10 : données du jeu précédent) sont PRÉSERVÉES telles quelles,
 * re-ciblées sur la fonction survivante `applyActiveTitle` (module title-routing,
 * appelée par le layout et le fallback sans-joueur du TitleSwitcher).
 *
 * Vérifie l'ORDRE load-bearing : le titre (header X-LevelUp-Title + store) est
 * committé AVANT le re-bootstrap, puis cancelQueries() + clear() purgent le cache
 * AVANT le re-bootstrap → aucune requête/cache de l'ancien titre ne survit.
 *
 * Les 2 derniers cas exercent hydrateFromBootstrap (garde deep-link `?f=`, PR #59) —
 * chemin fresh-load/bookmark qui SURVIT, distinct de la bascule applyActiveTitle.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'

const calls: string[] = []

vi.mock('@/lib/api/client', () => ({
  api: {
    post: vi.fn(async () => {
      calls.push('post')
    }),
    get: vi.fn(async () => {
      calls.push('get')
      return BOOTSTRAP_H5
    }),
  },
  setApiTitleSlug: vi.fn((slug: string) => {
    calls.push(`setApiTitleSlug:${slug}`)
  }),
  getApiTitleSlug: vi.fn(() => 'halo_infinite'),
  setApiLocale: vi.fn(),
}))

vi.mock('@/app/queryClient', () => ({
  queryClient: {
    cancelQueries: vi.fn(async () => {
      calls.push('cancelQueries')
    }),
    clear: vi.fn(() => {
      calls.push('clear')
    }),
  },
}))

import { useAppShellStore } from '@/stores/appShellStore'
import { applyActiveTitle } from '@/lib/title-routing/applyActiveTitle'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { DEFAULT_FILTER_CONTEXT } from '@/stores/createFilterStore'
import type { BootstrapResponse } from '@/lib/api/types'

const PLAYER = {
  player_slug: 'p1', gamertag: 'P1', xuid: '1', waypoint_player: 'P1', is_demo: false, sync_enabled: true,
}

const BOOTSTRAP_H5 = {
  setup_required: false, auth_state: 'ready', setup_state: 'ready',
  current_player: PLAYER, available_players: [PLAYER],
  current_title_slug: 'halo_5',
  available_titles: [
    { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', capabilities: [], is_default: true, effective_hp_to_kill: 225 },
    { slug: 'halo_5', name: 'Halo 5', status: 'active', capabilities: [], is_default: false, effective_hp_to_kill: 115 },
  ],
  locale: 'fr', hints_visible_default: true,
  feature_flags: { v7_enabled: true, media_enabled: true, demo_mode: false, discord_configured: false, tailscale_enabled: false },
  capabilities: {
    can_read_local_data: true, can_run_sync: true, can_use_live_halo: true, can_manage_settings: true,
    can_reset_media_index: true, can_view_media: true, can_self_provision: true, can_start_initial_sync: true, can_manage_instance: true,
  },
  settings_excerpt: { lang: 'fr', user_timezone: 'UTC', show_records: true, normalize_mode_labels: true },
  auth_mode: 'none', registration_mode: 'closed', is_admin: false, first_launch: false,
  instance_locked: false, reauth_required: false, has_password: false, current_username: null,
  oauth_code_flow_enabled: false, demo_mode: false,
} as const

describe('applyActiveTitle (ordre load-bearing — ex-switchTitle #10)', () => {
  beforeEach(() => {
    calls.length = 0
    vi.restoreAllMocks()
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', currentPlayer: PLAYER, availablePlayers: [PLAYER] })
    // Repartir de filtres propres avant chaque test.
    useSoloFilterStore.getState().resetFilters()
    useSquadFilterStore.getState().resetFilters()
  })

  it('committe le titre AVANT le clear, reset les filtres solo/squad, et purge le cache AVANT le re-bootstrap', async () => {
    // Instrumenter l'ordre du reset filtres (sans exécuter le vrai reset ici :
    // l'assertion de contenu est couverte par le test dédié ci-dessous).
    vi.spyOn(useSoloFilterStore.getState(), 'resetFilters').mockImplementation(() => {
      calls.push('resetSolo')
    })
    vi.spyOn(useSquadFilterStore.getState(), 'resetFilters').mockImplementation(() => {
      calls.push('resetSquad')
    })

    await applyActiveTitle('halo_5')

    // Le store reflète le nouveau titre.
    expect(useAppShellStore.getState().currentTitleSlug).toBe('halo_5')

    // Ordre load-bearing.
    const iPost = calls.indexOf('post')
    const iSet = calls.indexOf('setApiTitleSlug:halo_5')
    const iResetSolo = calls.indexOf('resetSolo')
    const iResetSquad = calls.indexOf('resetSquad')
    const iCancel = calls.indexOf('cancelQueries')
    const iClear = calls.indexOf('clear')
    const iGet = calls.indexOf('get')

    expect(iPost).toBeGreaterThanOrEqual(0)
    expect(iSet).toBeGreaterThan(iPost) // header committé après le POST session
    expect(iResetSolo).toBeGreaterThan(iSet) // filtres reset APRÈS le commit du titre
    expect(iResetSquad).toBeGreaterThan(iSet)
    expect(iCancel).toBeGreaterThan(iResetSolo) // cancel/clear APRÈS le reset filtres
    expect(iCancel).toBeGreaterThan(iResetSquad)
    expect(iClear).toBeGreaterThan(iCancel)
    expect(iGet).toBeGreaterThan(iClear) // re-bootstrap APRÈS le clear (pas avant)
  })

  it('réinitialise les filtres solo/squad pollués au switch', async () => {
    // Polluer les deux stores avec un contexte propre au titre courant.
    useSoloFilterStore.getState().setSessions({ picked_sessions: ['28/06/2026 (5)'], gap_minutes: 120 })
    useSquadFilterStore.getState().setCascade({ experience_types: [], playlists: [], modes: ['m-infinite'], maps: [] })
    expect(useSoloFilterStore.getState().filterContext).not.toEqual(DEFAULT_FILTER_CONTEXT)
    expect(useSquadFilterStore.getState().filterContext).not.toEqual(DEFAULT_FILTER_CONTEXT)

    await applyActiveTitle('halo_5')

    expect(useSoloFilterStore.getState().filterContext).toEqual(DEFAULT_FILTER_CONTEXT)
    expect(useSquadFilterStore.getState().filterContext).toEqual(DEFAULT_FILTER_CONTEXT)
  })

  it('no-op si le titre est déjà courant (filtres intacts)', async () => {
    useSoloFilterStore.getState().setSessions({ picked_sessions: ['garde-moi'], gap_minutes: 120 })
    const polluted = useSoloFilterStore.getState().filterContext

    await applyActiveTitle('halo_infinite')

    expect(calls).toHaveLength(0)
    expect(useSoloFilterStore.getState().filterContext).toEqual(polluted)
  })

  // Garde deep-link : le fresh-load / bookmark passe par hydrateFromBootstrap (pas
  // par applyActiveTitle). Un ?f= généré pour un AUTRE titre a hydraté le store solo
  // (urlHydratedTitleSlug posé par onRehydrateStorage) → le bootstrap doit le reset.
  it('hydrateFromBootstrap reset le filtre solo hydraté depuis un deep-link d’un autre titre', () => {
    useSoloFilterStore.getState().setSessions({ picked_sessions: ['sess-infinite'], gap_minutes: 120 })
    // Simule l'état posé par onRehydrateStorage au fresh-load d'un deep-link Infinite.
    useSoloFilterStore.setState({ urlHydratedTitleSlug: 'halo_infinite' })
    expect(useSoloFilterStore.getState().filterContext).not.toEqual(DEFAULT_FILTER_CONTEXT)

    // BOOTSTRAP_H5.current_title_slug === 'halo_5' → mismatch → reset.
    useAppShellStore.getState().hydrateFromBootstrap(BOOTSTRAP_H5 as unknown as BootstrapResponse)

    expect(useSoloFilterStore.getState().filterContext).toEqual(DEFAULT_FILTER_CONTEXT)
    expect(useSoloFilterStore.getState().urlHydratedTitleSlug).toBeNull()
  })

  it('hydrateFromBootstrap CONSERVE le filtre solo si le deep-link correspond au titre bootstrap', () => {
    const infiniteBootstrap = { ...BOOTSTRAP_H5, current_title_slug: 'halo_infinite' } as unknown as BootstrapResponse
    useSoloFilterStore.getState().setSessions({ picked_sessions: ['sess-infinite'], gap_minutes: 120 })
    useSoloFilterStore.setState({ urlHydratedTitleSlug: 'halo_infinite' })

    useAppShellStore.getState().hydrateFromBootstrap(infiniteBootstrap)

    expect(useSoloFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['sess-infinite'])
    expect(useSoloFilterStore.getState().urlHydratedTitleSlug).toBeNull()
  })
})
