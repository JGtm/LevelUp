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

const setupStatusFixture = {
  setup_required: false,
  has_players: true,
  has_credentials: true,
}

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
  total_matches: 0,
  solo_kpis: synthesisKPIs,
  squad_kpis: synthesisKPIs,
  comparison_metrics: [],
}

const mediaFixture = {
  items: { items: [], pagination: { ...emptyPagination, page_size: 24 }, freshness: null },
  total_mine: 0,
  total_teammates: 0,
  total_unassigned: 0,
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

export const handlers = [
  // Bootstrap & Health
  http.get(p('/bootstrap'), () => HttpResponse.json(bootstrapFixture)),
  http.get(p('/health'), () => HttpResponse.json({ status: 'ok', version: '0.0.0-test', uptime_seconds: 0, checks: {} })),

  // Players list
  http.get(p('/players'), () => HttpResponse.json({ items: [playerFixture], default_player_slug: 'test-player' })),

  // Setup
  http.get(p('/setup/status'), () => HttpResponse.json(setupStatusFixture)),
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

  // Career
  http.get(p(`/players/${SLUG}/pages/career`), () => HttpResponse.json(careerFixture)),
  http.get(p(`/players/${SLUG}/pages/career/top-matches`), () => HttpResponse.json({ items: [] })),
  http.get(p(`/players/${SLUG}/pages/career/encounters`), () => HttpResponse.json({ items: [] })),

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
    HttpResponse.json({ available: false, total: null, completed: null, xp_available: null, next_expiry: null, error_hint: null }),
  ),

  // Squad / Teammates
  http.post(p(`/players/${SLUG}/pages/teammates`), () => HttpResponse.json(teammatesFixture)),

  // Synthesis
  http.post(p(`/players/${SLUG}/pages/synthesis`), () => HttpResponse.json(synthesisFixture)),

  // Media
  http.post(p(`/players/${SLUG}/pages/media`), () => HttpResponse.json(mediaFixture)),
]

// Ré-export de emptyKPIs pour usage éventuel dans les tests unitaires
export { emptyKPIs }
