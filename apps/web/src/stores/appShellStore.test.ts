/**
 * Tests unitaires — AppShellStore (Slice 0a).
 *
 * Vérifie que la hydration depuis bootstrap fonctionne correctement,
 * que les valeurs par défaut sont en place, et que les mutations d'état
 * passent bien.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { useAppShellStore } from '@/stores/appShellStore'

const PLAYER = {
  player_slug: 'test-player',
  gamertag: 'TestPlayer',
  xuid: '0000000000000001',
  waypoint_player: 'TestPlayer',
  is_demo: false,
}

const BOOTSTRAP_READY = {
  setup_required: false,
  auth_state: 'ready' as const,
  setup_state: 'ready' as const,
  current_player: PLAYER,
  available_players: [PLAYER],
  current_title_slug: 'halo_infinite',
  available_titles: [
    { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active' as const, capabilities: ['matchmaking'], is_default: true },
  ],
  locale: 'fr',
  hints_visible_default: true,
  feature_flags: {
    v7_enabled: true,
    media_enabled: true,
    demo_mode: false,
    discord_configured: false,
    tailscale_enabled: false,
  },
  capabilities: {
    can_read_local_data: true,
    can_run_sync: true,
    can_use_live_halo: true,
    can_manage_settings: true,
    can_reset_media_index: true,
    can_view_media: true,
    can_self_provision: true,
    can_start_initial_sync: true,
    can_manage_instance: true,
  },
  settings_excerpt: {
    lang: 'fr',
    user_timezone: 'UTC',
    show_records: true,
    normalize_mode_labels: true,
  },
  auth_mode: 'none' as const,
  registration_mode: 'closed' as const,
  is_admin: false,
  first_launch: false,
}

describe('AppShellStore', () => {
  beforeEach(() => {
    // Réinitialiser le store avant chaque test
    useAppShellStore.setState({
      currentPlayer: null,
      availablePlayers: [],
      currentTitleSlug: 'halo_infinite',
      availableTitles: [],
      locale: 'fr',
      hintsVisible: true,
      capabilities: null,
      setupRequired: false,
      authState: 'missing',
      setupState: 'no_halo_link',
      isBootstrapped: false,
    })
  })

  it('démarre avec isBootstrapped=false', () => {
    expect(useAppShellStore.getState().isBootstrapped).toBe(false)
  })

  it('démarre avec currentPlayer=null', () => {
    expect(useAppShellStore.getState().currentPlayer).toBeNull()
  })

  it('hydrate depuis bootstrap — setup_state ready', () => {
    useAppShellStore.getState().hydrateFromBootstrap(BOOTSTRAP_READY)
    const s = useAppShellStore.getState()
    expect(s.isBootstrapped).toBe(true)
    expect(s.currentPlayer?.player_slug).toBe('test-player')
    expect(s.setupState).toBe('ready')
    expect(s.setupRequired).toBe(false)
  })

  it('hydrate depuis bootstrap — setup_required=true', () => {
    useAppShellStore.getState().hydrateFromBootstrap({
      ...BOOTSTRAP_READY,
      setup_required: true,
      setup_state: 'no_halo_link',
      current_player: null,
      available_players: [],
    })
    const s = useAppShellStore.getState()
    expect(s.setupRequired).toBe(true)
    expect(s.currentPlayer).toBeNull()
  })

  it('setCurrentPlayer met à jour le joueur courant', () => {
    useAppShellStore.getState().hydrateFromBootstrap(BOOTSTRAP_READY)
    const newPlayer = { ...PLAYER, player_slug: 'other-player', gamertag: 'OtherPlayer' }
    useAppShellStore.getState().setCurrentPlayer(newPlayer)
    expect(useAppShellStore.getState().currentPlayer?.player_slug).toBe('other-player')
  })

  it('setLocale change la locale', () => {
    useAppShellStore.getState().setLocale('en')
    expect(useAppShellStore.getState().locale).toBe('en')
  })

  it('capabilities peuplées depuis bootstrap', () => {
    useAppShellStore.getState().hydrateFromBootstrap(BOOTSTRAP_READY)
    const caps = useAppShellStore.getState().capabilities
    expect(caps?.can_read_local_data).toBe(true)
    expect(caps?.can_manage_instance).toBe(true)
  })

  it('hydrate depuis bootstrap — titre courant', () => {
    useAppShellStore.getState().hydrateFromBootstrap(BOOTSTRAP_READY)
    const s = useAppShellStore.getState()
    expect(s.currentTitleSlug).toBe('halo_infinite')
    expect(s.availableTitles).toHaveLength(1)
    expect(s.availableTitles[0].slug).toBe('halo_infinite')
  })

  it('setCurrentTitle change le titre courant', () => {
    useAppShellStore.getState().setCurrentTitle('halo_mcc')
    expect(useAppShellStore.getState().currentTitleSlug).toBe('halo_mcc')
  })
})
