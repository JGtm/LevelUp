/**
 * Tests unitaires — AppShellStore (Slice 0a).
 *
 * Vérifie que la hydration depuis bootstrap fonctionne correctement,
 * que les valeurs par défaut sont en place, et que les mutations d'état
 * passent bien.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { useAppShellStore, buildTitleSwitcherEntries } from '@/stores/appShellStore'

const PLAYER = {
  player_slug: 'test-player',
  gamertag: 'TestPlayer',
  xuid: '0000000000000001',
  waypoint_player: 'TestPlayer',
  is_demo: false,
  sync_enabled: true,
}

const BOOTSTRAP_READY = {
  setup_required: false,
  auth_state: 'ready' as const,
  setup_state: 'ready' as const,
  current_player: PLAYER,
  available_players: [PLAYER],
  current_title_slug: 'halo_infinite',
  available_titles: [
    { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active' as const, capabilities: ['matchmaking'], is_default: true, effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
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
  instance_locked: false,
  reauth_required: false,
  has_password: false,
  current_username: null,
  oauth_code_flow_enabled: false,
  demo_mode: false,
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

  // Note (Phase 4a) : le setter trivial setCurrentTitle a été SUPPRIMÉ (aucun caller
  // de production ; navigate-first D-6). Son assertion « le titre courant change »
  // est couverte par les happy-paths d'applyActiveTitle (currentTitleSlug converge)
  // — cf. lib/title-routing/applyActiveTitle.test.ts + appShellStore.applyActiveTitle.test.ts.
})

describe('buildTitleSwitcherEntries (MT-22 / PMT-8)', () => {
  const TITLES = [
    { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active' as const, capabilities: [], is_default: true, effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
    { slug: 'halo_mcc', name: 'Halo MCC', status: 'coming_soon' as const, capabilities: [], is_default: false, effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
    { slug: 'halo_5', name: 'Halo 5', status: 'archived' as const, capabilities: [], is_default: false, effective_hp_to_kill: 115, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
  ]

  it('garde active + coming_soon, exclut archived', () => {
    const entries = buildTitleSwitcherEntries(TITLES, 'halo_infinite')
    const slugs = entries.map((e) => e.slug)
    expect(slugs).toContain('halo_infinite')
    expect(slugs).toContain('halo_mcc')
    expect(slugs).not.toContain('halo_5')
  })

  it('désactive l\'entrée coming_soon, laisse l\'active activée', () => {
    const entries = buildTitleSwitcherEntries(TITLES, 'halo_infinite')
    const active = entries.find((e) => e.slug === 'halo_infinite')
    const soon = entries.find((e) => e.slug === 'halo_mcc')
    expect(active?.disabled).toBe(false)
    expect(soon?.disabled).toBe(true)
    expect(soon?.status).toBe('coming_soon')
  })

  it('marque isCurrent sur le titre courant', () => {
    const entries = buildTitleSwitcherEntries(TITLES, 'halo_infinite')
    expect(entries.find((e) => e.slug === 'halo_infinite')?.isCurrent).toBe(true)
    expect(entries.find((e) => e.slug === 'halo_mcc')?.isCurrent).toBe(false)
  })
})
