/**
 * Handlers MSW — mocks API pour les tests Vitest.
 * Couvre les endpoints utilisés par les features frontend.
 * URLs alignées sur /api/v1 (base URL du client HTTP).
 */
import { http, HttpResponse } from 'msw'

const SLUG = ':playerSlug'
const p = (path: string) => `/api/v1${path}`

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const playerFixture = {
  player_slug: 'test-player',
  gamertag: 'TestPlayer',
  xuid: '0000000000000001',
  waypoint_player: 'TestPlayer',
  is_demo: false,
}

const bootstrapFixture = {
  setup_required: false,
  auth_state: 'ready',
  setup_state: 'ready',
  current_player: playerFixture,
  available_players: [playerFixture],
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
}

const careerFixture = {
  summary: {
    rank_number: 12,
    rank_label: 'Gold 3',
    rank_name_raw: 'Gold',
    rank_tier: 'Gold',
    current_xp: 5000,
    xp_for_next_rank: 10000,
    xp_total: 50000,
    progress_pct: 50,
    is_max_rank: false,
    recorded_at: null,
  },
  hero_progress: { xp_total_required: 100000, xp_remaining: 50000, percentage: 50, current_rank: 12 },
  projections: null,
  xp_history: [],
  lusr: null,
}

const citationsFixture = {
  commendations: [
    {
      key: 'double-kill-mastery',
      label: 'Double Kill',
      category: 'Multikill',
      current_value: 42,
      color: '#7c3aed',
      icon_path: null,
      tier_label: 'Or',
      mastery_pct: 84.0,
    },
  ],
  medals_summary: [
    {
      medal_name_id: 101,
      name: 'Double Kill',
      count_filtered: 12,
      count_total: 42,
      description: 'Deux kills rapides',
    },
  ],
  deltas: {
    filtered_total: 12,
    unfiltered_total: 42,
    delta_count: -30,
  },
}

const emptyPagination = { total: 0, page: 1, page_size: 20, has_next: false, has_prev: false }

const matchHistoryFixture = {
  summary: { total_matches_scoped: 0, total_matches_unfiltered: 0, period_label: null, active_filter_mode: 'none' },
  table: { items: [], pagination: emptyPagination, freshness: null },
  available_sort_fields: [],
  export_hint: null,
  session_labels: { solo: [], squad: [] },
}

const homeFixture = {
  hero: {
    player_name: 'TestPlayer',
    kpis: { win_rate: 55.0, global_ratio: 1.2, avg_kda: 1.5, avg_accuracy: 42.0, total_matches: 120, wins: 66, draws: 2, dnfs: 0, losses: 52, total_playtime_secs: 180000, favorite_weapon_name: 'BR75', favorite_weapon_kills: 320, avg_offensive_conversion: 1.1, avg_defensive_resistance: 1.4 },
    trend: null,
  },
  highlights: [],
  recent_matches: [],
  recent_media: [],
  solo_session: null,
  squad_session: null,
}

const seasonPassFixture = {
  title_slug: 'halo_infinite',
  available: false,
  error_hint: null,
  active_track_path: null,
  challenges: {
    available: false,
    total: null,
    completed: null,
    xp_available: null,
    next_expiry: null,
    items: [],
    error_hint: null,
  },
  passes: [],
}

// setupStatusFixture supprimé (sprint 29) : GET /setup/status est un artefact mort.
// L'état setup est porté par BootstrapResponse.setup_state.

const settingsFixture = {
  lang: 'fr',
  discord_lang: 'fr',
  user_timezone: 'UTC',
  normalize_mode_labels: true,
  show_records: true,
  refresh_clears_caches: false,
  career_top_exclude_btb: false,
  media_captures_base_dir: '',
  media_tolerance_minutes: 5,
  media_watcher_enabled: false,
  media_watcher_debounce_seconds: 5,
  discord_notifications_enabled: false,
  discord_webhook_url_present: false,
  discord_notify_sync: false,
  discord_notify_backfill: false,
  discord_notify_new_version: false,
  discord_notify_friends: false,
  spnkr_auto_sync_enabled: false,
  spnkr_auto_sync_interval_hours: 0,
  spnkr_auto_sync_interval_minutes: 360,
  watcher_presence_enabled: false,
  watcher_subscribed_players: [],
  friend_gamertags: [],
}

const emptyKPIs = {
  match_count: 0, wins: 0, kd_ratio: null, win_rate: 0.0,
  accuracy: null, kills_per_game: null, assists_per_game: null,
}

const teammatesFixture = {
  options: [],
  teammates: [],
  total_matches: 0,
  session_labels: { solo: [], squad: [] },
  friends_count: 0,
  composition_sessions: [],
  latest_composition_session: '',
}

const synthesisKPIs = {
  match_count: 0, wins: 0, kd_ratio: null, win_rate: 0.0,
  accuracy: null, kills_per_min: null, avg_life_seconds: null, performance_score: null,
}

const synthesisFixture = {
  period: 'all',
  total_matches: 5,
  solo_kpis: synthesisKPIs,
  squad_kpis: synthesisKPIs,
  comparison_metrics: [],
  heatmap_data: [],
  top_weeks: [],
  scope: {
    period: 'all',
    match_count: 5,
    filters_applied: [],
    filters_ignored: [],
    description: '5 matchs — toutes périodes',
    computed_at: '2025-01-01T00:00:00Z',
  },
  overview: {
    total_matches: 5,
    total_wins: 3,
    total_losses: 2,
    total_kills: 30,
    total_deaths: 20,
    total_assists: 10,
    win_rate: 0.6,
  },
  highlights_preview: {
    top_by_kills: [
      { match_id: 'aaaabbbbcccc1111', kills: 12, deaths: 3, kda: 4.0, outcome: 2, perf_score: 220 },
    ],
    top_by_kda: [],
    worst_by_deaths: [
      { match_id: 'ddddeeeeffffg222', kills: 2, deaths: 15, kda: 0.13, outcome: 3, perf_score: 55 },
    ],
  },
  breakdowns: {
    top_maps: [
      { map_name: 'Aquarius', match_count: 4, win_rate: 0.75 },
      { map_name: 'Bazaar', match_count: 1, win_rate: 0.0 },
    ],
    top_modes: [
      { mode_name: 'Slayer', match_count: 3, win_rate: 0.67 },
      { mode_name: 'CTF', match_count: 2, win_rate: 0.5 },
    ],
  },
}

const mediaFixture = {
  items: { items: [], pagination: { ...emptyPagination, page_size: 24 }, freshness: null },
  total_mine: 0,
  total_teammates: 0,
  total_unassigned: 0,
  available_filters: {
    maps: [{ label: 'Aquarius', value: 'Aquarius' }],
    modes: [{ label: 'Slayer', value: 'Slayer' }],
  },
}

const labDiagnosticsFixture = {
  title_slug: 'halo_infinite',
  parity_report_file: {
    path: 'apps/go-api/tests/fixtures/parity_report.json',
    exists: true,
    size_bytes: 4500,
    modified_at: '2026-04-19T11:00:00Z',
  },
  parity_report: {
    generated_at: '2026-04-19T11:00:00Z',
    go_url: 'http://localhost:8000',
    player: 'TestPlayer',
    summary: {
      total: 24,
      passed: 20,
      failed: 2,
      skipped: 2,
    },
    results: [
      { name: 'health', status: 'passed', http_status: 200 },
      { name: 'media', status: 'failed', http_status: 500, error: 'fixture mismatch' },
    ],
  },
  medal_guards: {
    entry_count: 2,
    cardinality: { passed: true, reason: 'cardinalité OK', details: [] },
    required_fields: { passed: true, reason: 'Tous les champs requis sont présents', details: [] },
    images: { passed: true, reason: '0 images — import sans assets visuels (accepté)', details: [] },
    overall: { passed: true, reason: 'tous les garde-fous passent (2 entrées)', details: [] },
  },
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

export const handlers = [
  // Bootstrap & Health
  http.get(p('/bootstrap'), () => HttpResponse.json(bootstrapFixture)),
  http.get(p('/health'), () => HttpResponse.json({ status: 'ok', version: '0.0.0-test', uptime_seconds: 0, checks: {} })),

  // Players list
  http.get(p('/players'), () => HttpResponse.json({ items: [playerFixture], default_player_slug: 'test-player' })),

  // Setup
  // GET /setup/status supprimé (sprint 29) — artefact mort
  http.post(p('/setup/players'), () => HttpResponse.json({ player: playerFixture, db_created: true, warnings: [] })),
  http.post(p('/setup/smoke-test'), () =>
    HttpResponse.json({ job_id: 'job-1', job_type: 'smoke_test', status: 'queued', progress_pct: null, current_step: null, started_at: null, finished_at: null, result: null, error: null }),
  ),

  // Auth device flow
  http.post(p('/auth/device-flow/start'), () =>
    HttpResponse.json({ attempt_id: 'attempt-1', user_code: 'ABCD-1234', verification_uri: 'https://microsoft.com/link', verification_uri_complete: null, expires_in_seconds: 900, poll_interval_seconds: 5 }),
  ),
  http.get(p('/auth/device-flow/:attemptId'), () =>
    HttpResponse.json({ attempt_id: 'attempt-1', status: 'pending', gamertag: null, xuid: null, error: null }),
  ),

  // Jobs
  http.get(p('/jobs/:jobId'), () =>
    HttpResponse.json({ job_id: 'job-1', job_type: 'smoke_test', status: 'succeeded', progress_pct: 100, current_step: null, started_at: null, finished_at: null, result: {}, error: null }),
  ),

  // Settings
  http.get(p('/settings'), () => HttpResponse.json(settingsFixture)),
  http.patch(p('/settings'), async ({ request }) => {
    const body = await request.json()
    return HttpResponse.json({ ...settingsFixture, ...(body as Record<string, unknown>) })
  }),

  // Diagnostic d'instance (ex-Lab, panneau Donnees)
  http.get(p('/lab/diagnostics'), () => HttpResponse.json(labDiagnosticsFixture)),

  // Career
  http.get(p(`/players/${SLUG}/pages/career`), () => HttpResponse.json(careerFixture)),
  http.get(p(`/players/${SLUG}/pages/career/top-matches`), () =>
    HttpResponse.json({ best_matches: [], worst_matches: [] })),
  http.get(p(`/players/${SLUG}/pages/career/encounters`), () =>
    HttpResponse.json({
      teammates: [
        { xuid: '1', gamertag: 'DuoAlpha', match_count: 80, as_teammate: 60, as_enemy: 20, avg_kda: 1.6 },
        { xuid: '2', gamertag: 'QueueGhost', match_count: 50, as_teammate: 45, as_enemy: 5, avg_kda: 1.3 },
        { xuid: '3', gamertag: 'SynergyOne', match_count: 30, as_teammate: 30, as_enemy: 0, avg_kda: 2.1 },
      ],
      enemies: [
        { xuid: '4', gamertag: 'NemesisBravo', match_count: 40, as_teammate: 5, as_enemy: 35, avg_kda: 0.8 },
      ],
      total: 4,
    }),
  ),
  http.post(p(`/players/${SLUG}/pages/citations`), () => HttpResponse.json(citationsFixture)),

  // Match History
  http.post(p(`/players/${SLUG}/pages/match-history/query`), () => HttpResponse.json(matchHistoryFixture)),
  http.post(p(`/players/${SLUG}/pages/match-history/export`), () =>
    HttpResponse.json({ file_token: 'tok-1', file_name: 'export.csv', content_type: 'text/csv', download_path: '/dl/tok-1', expires_at: '', estimated_rows: null }),
  ),

  // Explorer
  http.get(p('/directory/gamertags/search'), () => HttpResponse.json({ query: '', items: [] })),
  http.post(p(`/players/${SLUG}/pages/explorer/matches-query`), () =>
    HttpResponse.json({ summary: { total_matches: 0, selected_match_id: null }, table: { items: [], pagination: emptyPagination, freshness: null } }),
  ),
  http.post(p(`/players/${SLUG}/pages/explorer/player-query`), () =>
    HttpResponse.json({ target: { gamertag: '', xuid: null }, summary: { matches_together: 0, wins_together: 0, losses_together: 0, last_seen_at: null }, allies_table: [], enemies_table: [], common_matches: [] }),
  ),

  // Home
  http.get(p(`/players/${SLUG}/pages/home`), () => HttpResponse.json(homeFixture)),
  http.get(p(`/players/${SLUG}/pages/palmares/season-pass`), () => HttpResponse.json(seasonPassFixture)),

  // Leaderboard
  http.get(p(`/players/${SLUG}/pages/leaderboard`), ({ request }) => {
    const url = new URL(request.url)
    const season = url.searchParams.get('season') ?? 'Season5'
    const playlist = url.searchParams.get('playlist') ?? 'Ranked'
    return HttpResponse.json({
      entries: [
        { rank: 1, xuid: '1000', gamertag: 'LocalAce', title_slug: 'halo_infinite', season_id: season, playlist_id: playlist, csr_value: 1850, tier: 'Onyx', sub_tier: 0, is_local: true },
        { rank: 2, xuid: '2000', gamertag: 'RemoteRival', title_slug: 'halo_infinite', season_id: season, playlist_id: playlist, csr_value: 1720, tier: 'Diamond', sub_tier: 6, is_local: false },
        { rank: 3, xuid: '3000', gamertag: 'LocalVet', title_slug: 'halo_infinite', season_id: season, playlist_id: playlist, csr_value: 1600, tier: 'Diamond', sub_tier: 3, is_local: true },
      ],
      season_id: season,
      playlist_id: playlist,
      title_slug: 'halo_infinite',
      total: 3,
    })
  }),

  // Palmares — Relations (hub Communauté > Relations, forme {overview, relations[]}).
  // Phase 2 : POST avec FilterContextInput en body (segmentation serveur).
  http.post(p(`/players/${SLUG}/pages/palmares/relations`), () =>
    HttpResponse.json({
      overview: {
        distinct_players: 42,
        allies_count: 30,
        rivals_count: 25,
        core_count: 2,
        top_ally: { gamertag: 'DuoAlpha', win_rate: 0.67, matches: 60 },
        top_nemesis: { gamertag: 'NemesisBravo', win_rate: 0.2, matches: 35 },
      },
      relations: [
        {
          xuid: '1',
          gamertag: 'DuoAlpha',
          total_matches: 80,
          teammate_matches: 60,
          teammate_wins: 40,
          teammate_win_rate: 0.67,
          enemy_matches: 20,
          enemy_wins: 10,
          enemy_win_rate: 0.5,
          avg_kda_with: 1.6,
          avg_kda_against: 1.1,
          kills_dealt: 30,
          deaths_suffered: 18,
          duel_ratio: 1.67,
          first_seen_at: '2024-01-02T10:00:00Z',
          last_seen_at: '2026-06-20T12:00:00Z',
          category: 'mixed',
          badges: [
            { label_key: 'narrative.encounter.ordinal', color_token: 'narrative-encounter-ordinal', style: 'tinted', detail: { ordinal: 79 } },
            { label_key: 'narrative.encounter.ally_plus', color_token: 'narrative-encounter-ally-plus', style: 'tinted', detail: null },
            { label_key: 'narrative.encounter.duo_gagnant', color_token: 'narrative-encounter-duo-gagnant', style: 'solid', detail: { teammate_win_rate: 0.67 } },
            { label_key: 'narrative.encounter.cross_game', color_token: 'narrative-encounter-cameleon', style: 'solid', detail: { game: 'Halo 5', matches_together: 7 } },
          ],
        },
        {
          xuid: '2',
          gamertag: 'QueueGhost',
          total_matches: 50,
          teammate_matches: 45,
          teammate_wins: 30,
          teammate_win_rate: 0.67,
          enemy_matches: 5,
          enemy_wins: 2,
          enemy_win_rate: 0.4,
          avg_kda_with: 1.3,
          avg_kda_against: 0.9,
          kills_dealt: 8,
          deaths_suffered: 6,
          duel_ratio: 1.33,
          first_seen_at: '2025-12-01T10:00:00Z',
          last_seen_at: '2026-06-25T12:00:00Z',
          category: 'mixed',
          badges: [],
        },
        {
          xuid: '4',
          gamertag: 'NemesisBravo',
          total_matches: 40,
          teammate_matches: 5,
          teammate_wins: 3,
          teammate_win_rate: 0.6,
          enemy_matches: 35,
          enemy_wins: 28,
          enemy_win_rate: 0.2,
          avg_kda_with: 1.4,
          avg_kda_against: 0.8,
          kills_dealt: 12,
          deaths_suffered: 40,
          duel_ratio: 0.3,
          first_seen_at: '2024-03-10T10:00:00Z',
          last_seen_at: '2026-05-01T12:00:00Z',
          category: 'mixed',
          badges: [
            { label_key: 'narrative.encounter.coriace', color_token: 'narrative-encounter-coriace', style: 'tinted', detail: null },
          ],
        },
      ],
    }),
  ),

  // Palmares — Relations > Moments & Rivalités (Phase 3a, sous-endpoint).
  http.post(p(`/players/${SLUG}/pages/palmares/relations/moments`), () =>
    HttpResponse.json({
      top_relations: 8,
      heatmap: [
        { xuid: '4', gamertag: 'NemesisBravo', hour: 19, count: 12 },
        { xuid: '4', gamertag: 'NemesisBravo', hour: 2, count: 3 },
        { xuid: '1', gamertag: 'DuoAlpha', hour: 14, count: 9 },
        { xuid: '1', gamertag: 'DuoAlpha', hour: 9, count: 4 },
      ],
      rivalries: [
        {
          xuid: '4',
          gamertag: 'NemesisBravo',
          enemy_matches: 35,
          duels: [
            { match_id: 'd1', started_at: '2026-05-01T19:00:00Z', outcome: 'loss', kills_on_rival: 2, deaths_by_rival: 7 },
            { match_id: 'd2', started_at: '2026-05-02T20:00:00Z', outcome: 'loss', kills_on_rival: 3, deaths_by_rival: 6 },
            { match_id: 'd3', started_at: '2026-05-03T19:00:00Z', outcome: 'win', kills_on_rival: 8, deaths_by_rival: 4 },
          ],
          rolling_win_rate: [0, 0, 0.33],
          rolling_window: 5,
          recent_win_rate: 0.33,
          global_win_rate: 0.33,
          current_streak: 1,
          frag_gap: -4,
        },
      ],
    }),
  ),

  // Squad / Teammates
  http.post(p(`/players/${SLUG}/pages/teammates`), () => HttpResponse.json(teammatesFixture)),

  // Synthesis
  http.post(p(`/players/${SLUG}/pages/synthesis`), () => HttpResponse.json(synthesisFixture)),

  // Media
  http.post(p(`/players/${SLUG}/pages/media`), () => HttpResponse.json(mediaFixture)),
  http.get(p('/media/feed-version'), () => HttpResponse.json({ version: 1 })),
  http.get(p(`/players/${SLUG}/media/authors`), () => HttpResponse.json({ items: [] })),

  // Notifications
  http.get(p(`/players/${SLUG}/notifications/unread-count`), () =>
    HttpResponse.json({ count: 0, by_category: {} }),
  ),

  // Field mappings i18n (réutilisé par toutes les pages via useFieldMappings)
  http.get(p('/titles/:slug/field-mappings'), ({ params, request }) => {
    const url = new URL(request.url)
    return HttpResponse.json({
      title_slug: params.slug,
      schema_version: 1,
      locale: url.searchParams.get('locale') ?? 'fr',
      fields: {},
      assets: {},
      outcomes: {},
    })
  }),

  // Filters resolve (NavL2 / FilterOmnibar / SessionNavBar via useFiltersResolve)
  http.post(p(`/players/${SLUG}/filters/resolve`), () =>
    HttpResponse.json({
      effective: {
        filter_mode: 'period',
        period: { start_date: null, end_date: null },
        sessions: { picked_sessions: [], gap_minutes: 120 },
        cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
      },
      available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
      session_options: { all_sessions: [], solo_labels: [], squad_labels: [] },
      counts: { total_matches_before_filters: 0, total_matches_after_filters: 0 },
    }),
  ),
]

// Ré-export de emptyKPIs pour usage éventuel dans les tests unitaires
export { emptyKPIs }
