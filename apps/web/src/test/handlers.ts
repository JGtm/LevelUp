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
  charts: { rank_progress_gauge: null, hero_progress_gauge: null, xp_history_figure: null, lusr_rating_figure: null },
  xp_history: [],
  lusr: null,
  top_matches_preview: [],
  encounters_preview: [],
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
  distribution_chart: null,
}

const emptyPagination = { total: 0, page: 1, page_size: 20, has_next: false, has_prev: false }

const matchHistoryFixture = {
  summary: { total_matches_scoped: 0, total_matches_unfiltered: 0, period_label: null, active_filter_mode: 'none' },
  table: { items: [], pagination: emptyPagination, freshness: null },
  available_sort_fields: [],
  export_hint: null,
}

const homeFixture = {
  hero: {
    player_name: 'TestPlayer',
    kpis: { win_rate: 55.0, global_ratio: 1.2, avg_accuracy: 42.0, total_matches: 120, wins: 66, losses: 54 },
    trend: null,
  },
  highlights: [],
  recent_matches: [],
  recent_media: [],
  solo_session: null,
  squad_session: null,
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
  discord_notify_new_media: false,
  spnkr_refresh_with_backfill: false,
  spnkr_refresh_backfill_medals: false,
  spnkr_refresh_backfill_skill: false,
  spnkr_refresh_backfill_aliases: false,
  spnkr_refresh_backfill_personal_scores: false,
  spnkr_refresh_backfill_performance_scores: false,
  spnkr_refresh_backfill_lusr: false,
  spnkr_refresh_backfill_events: false,
  spnkr_refresh_backfill_weapons: false,
}

const emptyKPIs = {
  match_count: 0, wins: 0, kd_ratio: null, win_rate: 0.0,
  accuracy: null, kills_per_game: null, assists_per_game: null,
}

const teammatesFixture = {
  options: [],
  teammates: [],
  solo_reference: null,
  total_matches: 0,
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
  rivalries_preview: {
    top_teammates: [
      { gamertag: 'AllyOne', shared_matches: 8, win_rate_together: 0.75 },
    ],
    top_enemies: [
      { gamertag: 'RivalOne', shared_matches: 6, win_rate_together: 0.33 },
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

const labResourcesFixture = {
  title_slug: 'halo_infinite',
  metadata_db_path: 'data/titles/halo_infinite/warehouse/metadata.duckdb',
  current_season: {
    title_id: 'halo_infinite',
    season_id: 'season-5',
    version: 'v5',
    name: 'Echoes Within',
    start_date: '2026-01-01T00:00:00Z',
    end_date: '2026-06-01T00:00:00Z',
    fetched_at: '2026-04-19T10:00:00Z',
    content_hash: 'season-hash',
    etag: 'etag-season',
    source_url: 'https://waypoint.example/seasons',
  },
  seasons: [],
  csr_seasons: [],
  snapshots: [
    {
      resource_key: 'season_calendar',
      version: 'v5',
      fetched_at: '2026-04-19T10:00:00Z',
      content_hash: 'season-hash',
      etag: 'etag-season',
      source_url: 'https://waypoint.example/seasons',
      payload_size: 2450,
    },
  ],
  selected_snapshot: {
    resource_key: 'season_calendar',
    version: 'v5',
    fetched_at: '2026-04-19T10:00:00Z',
    content_hash: 'season-hash',
    etag: 'etag-season',
    source_url: 'https://waypoint.example/seasons',
    payload: '{"seasons":[{"id":"season-5","name":"Echoes Within"}]}',
  },
  assets: {
    total: 2,
    search: '',
    items: [
      {
        asset_id: 'asset-aquarius',
        asset_type: 'map',
        version_id: '1',
        name: 'Aquarius',
        fetched_at: '2026-04-19T10:00:00Z',
      },
    ],
    selected: {
      asset_id: 'asset-aquarius',
      asset_type: 'map',
      version_id: '1',
      name: 'Aquarius',
      description: 'Arena map',
      fetched_at: '2026-04-19T10:00:00Z',
      content_hash: 'asset-hash',
      raw_json: '{"id":"asset-aquarius","kind":"map"}',
    },
  },
  medals: {
    total: 2,
    search: '',
    items: [
      {
        medal_id: 101,
        name_id: 'DoubleKill',
        description_id: 'Earned for two kills',
        medal_type: 'multikill',
        difficulty: 'normal',
        sprite_index: 3,
        fetched_at: '2026-04-19T10:00:00Z',
      },
    ],
    selected: {
      medal_id: 101,
      name_id: 'DoubleKill',
      description_id: 'Earned for two kills',
      medal_type: 'multikill',
      difficulty: 'normal',
      sprite_index: 3,
      personal_score: 100,
      fetched_at: '2026-04-19T10:00:00Z',
      content_hash: 'medal-hash',
      raw_json: '{"id":101,"name":"DoubleKill"}',
    },
  },
}

const labContractsFixture = {
  go_openapi: {
    path: 'apps/go-api/api/openapi.yaml',
    exists: true,
    size_bytes: 128000,
    modified_at: '2026-04-19T10:00:00Z',
  },
  fastapi_reference: {
    path: 'apps/go-api/api/openapi_fastapi_reference.yaml',
    exists: true,
    size_bytes: 120000,
    modified_at: '2026-04-18T10:00:00Z',
  },
  summary: {
    fastapi_route_count: 24,
    go_route_count: 27,
    missing_in_go: 0,
    extra_in_go: 3,
    method_mismatches: 1,
    status: 'DIVERGENCES',
  },
  missing_in_go: [],
  extra_in_go: [
    { path: '/lab/resources', methods: ['get'] },
    { path: '/lab/contracts', methods: ['get'] },
    { path: '/lab/diagnostics', methods: ['get'] },
  ],
  method_mismatches: [
    {
      fastapi_path: '/players/{player_slug}/pages/media',
      go_path: '/players/{player_slug}/pages/media',
      fastapi_methods: ['get'],
      go_methods: ['post'],
      missing_methods: ['get'],
      extra_methods: ['post'],
    },
  ],
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

  // Instance lab
  http.get(p('/lab/resources'), () => HttpResponse.json(labResourcesFixture)),
  http.get(p('/lab/contracts'), () => HttpResponse.json(labContractsFixture)),
  http.get(p('/lab/diagnostics'), () => HttpResponse.json(labDiagnosticsFixture)),

  // Career
  http.get(p(`/players/${SLUG}/pages/career`), () => HttpResponse.json(careerFixture)),
  http.get(p(`/players/${SLUG}/pages/career/top-matches`), () => HttpResponse.json({ items: [] })),
  http.get(p(`/players/${SLUG}/pages/career/encounters`), () => HttpResponse.json({ items: [] })),
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
  http.get(p(`/players/${SLUG}/battlepass`), () =>
    HttpResponse.json({ available: false, rank: null, reward_track: null, progress: null, error_hint: null }),
  ),
  http.get(p(`/players/${SLUG}/challenges`), () =>
    HttpResponse.json({ available: false, total: null, completed: null, xp_available: null, next_expiry: null, items: [], error_hint: null }),
  ),

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

  // Palmares — Relations
  http.get(p(`/players/${SLUG}/pages/palmares/relations`), () =>
    HttpResponse.json({
      overview: { distinct_players: 42, frequent_allies: 5, repeat_rivals: 3, closed_circle: 2 },
      frequent_allies: [
        { xuid: '1', gamertag: 'DuoAlpha', total_matches: 80, teammate_matches: 60, teammate_wins: 40, enemy_matches: 20, enemy_wins: 10 },
        { xuid: '2', gamertag: 'QueueGhost', total_matches: 50, teammate_matches: 45, teammate_wins: 30, enemy_matches: 5, enemy_wins: 2 },
      ],
      best_synergies: [
        { xuid: '3', gamertag: 'SynergyOne', total_matches: 30, teammate_matches: 30, teammate_wins: 25, enemy_matches: 0, enemy_wins: 0 },
      ],
      nemeses: [
        { xuid: '4', gamertag: 'NemesisBravo', total_matches: 40, teammate_matches: 5, teammate_wins: 3, enemy_matches: 35, enemy_wins: 28 },
      ],
      favorite_victims: [],
      closed_circle: [],
    }),
  ),

  // Squad / Teammates
  http.post(p(`/players/${SLUG}/pages/teammates`), () => HttpResponse.json(teammatesFixture)),

  // Synthesis
  http.post(p(`/players/${SLUG}/pages/synthesis`), () => HttpResponse.json(synthesisFixture)),

  // Media
  http.post(p(`/players/${SLUG}/pages/media`), () => HttpResponse.json(mediaFixture)),
  http.get(p('/media/feed-version'), () => HttpResponse.json({ version: 1 })),
]

// Ré-export de emptyKPIs pour usage éventuel dans les tests unitaires
export { emptyKPIs }
