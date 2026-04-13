/**
 * Types TypeScript alignés sur les schémas Pydantic du backend.
 * Générés manuellement pour Slice 0a — seront remplacés par openapi-typescript
 * dès que le pipeline `make generate-types` est opérationnel.
 */

// ---------------------------------------------------------------------------
// Communs
// ---------------------------------------------------------------------------

export interface PlayerSummary {
  player_slug: string
  gamertag: string
  xuid: string
  waypoint_player: string
  is_demo: boolean
}

export interface CapabilityMap {
  can_read_local_data: boolean
  can_run_sync: boolean
  can_use_live_halo: boolean
  can_manage_settings: boolean
  can_reset_media_index: boolean
  can_view_media: boolean
  can_self_provision: boolean
  can_start_initial_sync: boolean
  can_manage_instance: boolean
}

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

export interface FeatureFlags {
  v7_enabled: boolean
  media_enabled: boolean
  demo_mode: boolean
  discord_configured: boolean
  tailscale_enabled: boolean
}

export interface SettingsExcerpt {
  lang: string
  user_timezone: string
  show_records: boolean
  normalize_mode_labels: boolean
}

export interface BootstrapResponse {
  setup_required: boolean
  auth_state: 'missing' | 'partial' | 'ready'
  setup_state: 'no_halo_link' | 'halo_linked_no_profile' | 'profile_ready_no_sync' | 'ready'
  current_player: PlayerSummary | null
  available_players: PlayerSummary[]
  locale: string
  hints_visible_default: boolean
  feature_flags: FeatureFlags
  capabilities: CapabilityMap
  settings_excerpt: SettingsExcerpt
}

export interface PlayersListResponse {
  items: PlayerSummary[]
  default_player_slug: string | null
}

export interface SessionContextRequest {
  player_slug?: string | null
  locale?: string | null
  hints_visible?: boolean | null
}

export interface SessionContextResponse {
  current_player: PlayerSummary | null
  locale: string
  hints_visible: boolean
  capabilities: CapabilityMap
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

export interface HealthResponse {
  status: 'ok' | 'degraded' | 'down'
  version: string
  uptime_seconds: number
  checks: Record<string, string>
}

// ---------------------------------------------------------------------------
// Filtres
// ---------------------------------------------------------------------------

export interface PeriodInput {
  start_date?: string | null
  end_date?: string | null
}

export interface SessionsInput {
  picked_session_label?: string | null
  picked_solo_session_label?: string | null
  picked_squad_session_label?: string | null
  picked_sessions?: string[]
  gap_minutes?: number
}

export interface CascadeInput {
  experience_types?: string[]
  playlists?: string[]
  modes?: string[]
  maps?: string[]
}

export interface FilterContextInput {
  filter_mode: 'period' | 'sessions'
  period?: PeriodInput
  sessions?: SessionsInput
  cascade?: CascadeInput
}

export interface LabelValue {
  label: string
  value: string
}

export interface AvailableOptions {
  experience_types: LabelValue[]
  playlists: LabelValue[]
  modes: LabelValue[]
  maps: LabelValue[]
}

export interface SessionOption {
  label: string
  session_id: string
  match_count: number
  is_squad: boolean
}

export interface SessionOptions {
  all_sessions: SessionOption[]
  solo_labels: string[]
  squad_labels: string[]
}

export interface FilterCounts {
  total_matches_before_filters: number
  total_matches_after_filters: number
}

export interface FilterContextResolved {
  effective: FilterContextInput
  available_options: AvailableOptions
  session_options: SessionOptions
  counts: FilterCounts
}

// ---------------------------------------------------------------------------
// Setup / Auth (Slice 1)
// ---------------------------------------------------------------------------

export interface SetupAuthInfo {
  has_client_id: boolean
  has_refresh_token: boolean
  has_msal_cache: boolean
  preferred_method: 'refresh_token' | 'device_code' | 'unknown'
}

export interface SetupPlayerInfo {
  has_any_profile: boolean
  default_player_slug: string | null
}

export type SetupNextStep = 'choose_mode' | 'auth' | 'player' | 'initial_sync' | 'smoke_test' | 'done'

export interface SetupStatusResponse {
  needs_setup: boolean
  auth: SetupAuthInfo
  player: SetupPlayerInfo
  next_blocking_step: SetupNextStep
}

export interface DeviceFlowStartResponse {
  attempt_id: string
  user_code: string
  verification_uri: string
  verification_uri_complete: string | null
  expires_in_seconds: number
  poll_interval_seconds: number
}

export type DeviceFlowStatus = 'pending' | 'authorized' | 'provisioned' | 'failed' | 'expired'

export interface ApiErrorSchema {
  code: string
  message: string
  retryable: boolean
  details?: Record<string, unknown> | null
}

export interface DeviceFlowStatusResponse {
  attempt_id: string
  status: DeviceFlowStatus
  gamertag: string | null
  xuid: string | null
  error: ApiErrorSchema | null
}

export interface CreatePlayerProfileRequest {
  gamertag: string
  xuid?: string | null
  profile_mode?: 'xbox' | 'azure_manual'
}

export interface CreatePlayerProfileResponse {
  player: PlayerSummary
  db_created: boolean
  warnings: string[]
}

export interface SmokeTestStartRequest {
  player_slug: string
  max_matches?: number
  run_backfill?: boolean
}

// ---------------------------------------------------------------------------
// Jobs asynchrones (Slice 1)
// ---------------------------------------------------------------------------

export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface AsyncJobStatus {
  job_id: string
  job_type: string
  status: JobStatus
  progress_pct: number | null
  current_step: string | null
  started_at: string | null
  finished_at: string | null
  result: Record<string, unknown> | null
  error: ApiErrorSchema | null
  // Champs enrichis sync initiale (Sprint 3)
  phase_key: string | null
  phase_label: string | null
  matches_done: number | null
  matches_total: number | null
  subtasks_done: number | null
  subtasks_total: number | null
  eta_seconds: number | null
  warnings: string[]
}

// ---------------------------------------------------------------------------
// Sync initiale (Sprint 3)
// ---------------------------------------------------------------------------

export interface InitialSyncStartRequest {
  player_slug: string
  max_matches?: number
}

// ---------------------------------------------------------------------------
// Settings (Slice 1)
// ---------------------------------------------------------------------------

export interface SettingsResponse {
  lang: string
  discord_lang: string
  user_timezone: string
  normalize_mode_labels: boolean
  show_records: boolean
  refresh_clears_caches: boolean
  career_top_exclude_btb: boolean
  media_captures_base_dir: string
  media_tolerance_minutes: number
  media_watcher_enabled: boolean
  media_watcher_debounce_seconds: number
  discord_notifications_enabled: boolean
  discord_webhook_url_present: boolean
  discord_notify_sync: boolean
  discord_notify_backfill: boolean
  discord_notify_new_version: boolean
  discord_notify_new_media: boolean
  spnkr_refresh_with_backfill: boolean
  spnkr_refresh_backfill_medals: boolean
  spnkr_refresh_backfill_skill: boolean
  spnkr_refresh_backfill_aliases: boolean
  spnkr_refresh_backfill_personal_scores: boolean
  spnkr_refresh_backfill_performance_scores: boolean
  spnkr_refresh_backfill_lusr: boolean
  spnkr_refresh_backfill_events: boolean
  spnkr_refresh_backfill_weapons: boolean
}

export type UpdateSettingsRequest = Partial<
  Omit<SettingsResponse, 'discord_webhook_url_present'> & {
    discord_webhook_url?: string
  }
>

export interface MediaResetRequest {
  confirm_destructive: boolean
  reindex_after_reset?: boolean
}

// ---------------------------------------------------------------------------
// Carrière (Slice 2)
// ---------------------------------------------------------------------------

export interface PlotlyFigurePayload {
  data: Record<string, unknown>[]
  layout: Record<string, unknown>
}

export interface CareerSummary {
  rank_number: number
  rank_label: string
  rank_name_raw: string
  rank_tier: string
  current_xp: number
  xp_for_next_rank: number
  xp_total: number
  progress_pct: number
  is_max_rank: boolean
  recorded_at: string | null
}

export interface HeroProgress {
  xp_total_required: number
  xp_remaining: number
  percentage: number
  current_rank: number
}

export interface CareerProjections {
  xp_per_day_active: number
  xp_per_day_fallback: number
  estimated_hero_date: string | null
  estimated_rank_cap_date: string | null
}

export interface CareerCharts {
  rank_progress_gauge: PlotlyFigurePayload | null
  hero_progress_gauge: PlotlyFigurePayload | null
  xp_history_figure: PlotlyFigurePayload | null
  lusr_rating_figure: PlotlyFigurePayload | null
}

export interface CareerHistoryPoint {
  recorded_at: string
  rank: number
  xp_total: number
}

export interface CareerLusrCheckpoint {
  recorded_at: string
  rating_value: number
  playlist_group: string
}

export interface CareerLusrSection {
  current_rating: number | null
  current_tier_label: string | null
  current_playlist_group: string | null
  trend_label: string | null
  checkpoints: CareerLusrCheckpoint[]
}

export interface CareerTopMatch {
  match_id: string
  start_time: string | null
  map_ui: string | null
  mode_ui: string | null
  playlist_label: string | null
  performance_score: number | null
  badge_type: string | null
  score_label: string | null
  outcome_label: string | null
}

export interface CareerEncounter {
  encounter_key: string
  opponent_gamertag: string
  count_matches: number
  wins: number
  losses: number
  last_seen_at: string | null
}

export interface CareerPageResponse {
  summary: CareerSummary | null
  hero_progress: HeroProgress | null
  projections: CareerProjections | null
  charts: CareerCharts
  xp_history: CareerHistoryPoint[]
  lusr: CareerLusrSection | null
  top_matches_preview: CareerTopMatch[]
  encounters_preview: CareerEncounter[]
}

export interface CareerTopMatchesResponse {
  items: CareerTopMatch[]
}

export interface CareerEncountersResponse {
  items: CareerEncounter[]
}

// ---------------------------------------------------------------------------
// Pagination — commun (Slices 3+)
// ---------------------------------------------------------------------------

export interface SortSpec {
  field: string
  direction: 'asc' | 'desc'
}

export interface PaginationRequest {
  page?: number
  page_size?: number
}

export interface PaginationMeta {
  total: number
  page: number
  page_size: number
  has_next: boolean
  has_prev: boolean
}

export interface FreshnessInfo {
  source: 'live' | 'cached' | 'mixed'
  sync_status: 'fresh' | 'stale' | 'unknown'
  warnings: string[]
}

export interface PaginatedResponse<T> {
  items: T[]
  pagination: PaginationMeta
  freshness?: FreshnessInfo | null
}

// ---------------------------------------------------------------------------
// Historique des parties (Slice 3)
// ---------------------------------------------------------------------------

export interface MatchHistoryRow {
  match_id: string
  start_time: string
  start_time_label: string
  outcome_code: number | null
  outcome_label: string
  score_label: string
  map_ui: string
  mode_ui: string
  playlist_label: string
  team_mmr: number | null
  enemy_mmr: number | null
  delta_mmr: number | null
  win_rate_hist: number | null
  win_rate_hist_total: number | null
  performance_score_relative: number | null
  average_life_mmss: string
  match_url: string
}

export interface MatchHistoryQuerySummary {
  total_matches_scoped: number
  total_matches_unfiltered: number
  period_label: string | null
  active_filter_mode: string
}

export interface ExportHint {
  file_name: string
  estimated_rows: number
  token: string | null
}

export interface MatchHistoryPageResponse {
  summary: MatchHistoryQuerySummary
  table: PaginatedResponse<MatchHistoryRow>
  available_sort_fields: string[]
  export_hint: ExportHint | null
}

export interface MatchHistoryQueryRequest {
  filters?: FilterContextInput
  pagination?: PaginationRequest
  columns?: string[] | null
  include_export_hint?: boolean
}

export interface FileTokenResponse {
  file_token: string
  file_name: string
  content_type: string
  download_path: string
  expires_at: string
  estimated_rows: number | null
}

// ---------------------------------------------------------------------------
// Explorer (Slice 4)
// ---------------------------------------------------------------------------

export interface GamertagSuggestion {
  gamertag: string
  xuid: string | null
  score: number
  exact_match: boolean
}

export interface GamertagSearchResponse {
  query: string
  items: GamertagSuggestion[]
}

export interface ExplorerMatchRow {
  match_id: string
  start_time: string
  start_time_label: string
  map_ui: string
  mode_ui: string
  playlist_label: string
  outcome_label: string
  score_label: string
  is_with_friends: boolean
  experience_type_label: string
}

export interface ExplorerEncounterRow {
  gamertag: string
  xuid: string | null
  count_matches: number
  wins: number
  losses: number
  last_seen_at: string | null
  same_team: boolean | null
}

export interface ExplorerMatchesQuerySummary {
  total_matches: number
  selected_match_id: string | null
}

export interface ExplorerPlayerTarget {
  gamertag: string
  xuid: string | null
}

export interface ExplorerPlayerSummary {
  matches_together: number
  wins_together: number
  losses_together: number
  last_seen_at: string | null
}

export interface ExplorerMatchFilters {
  selected_date?: string | null
  squad_scope?: 'all' | 'solo' | 'squad'
  experience_type?: string | null
  playlist?: string | null
  mode?: string | null
  map?: string | null
  selected_match_id?: string | null
}

export interface ExplorerMatchesQueryRequest {
  filters?: FilterContextInput
  match_filters?: ExplorerMatchFilters
  pagination?: PaginationRequest
}

export interface ExplorerPlayerQueryRequest {
  target_gamertag: string
  filters?: FilterContextInput | null
}

export interface ExplorerMatchesQueryResponse {
  summary: ExplorerMatchesQuerySummary
  table: PaginatedResponse<ExplorerMatchRow>
}

export interface ExplorerPlayerQueryResponse {
  target: ExplorerPlayerTarget
  summary: ExplorerPlayerSummary
  allies_table: ExplorerEncounterRow[]
  enemies_table: ExplorerEncounterRow[]
  common_matches: ExplorerMatchRow[]
}

// ---------------------------------------------------------------------------
// Accueil Mission Control (Slice 5)
// ---------------------------------------------------------------------------

export interface HeroKPIs {
  win_rate: number
  global_ratio: number | null
  avg_accuracy: number | null
  total_matches: number
  wins: number
  losses: number
}

export interface HeroTrend {
  ratio_delta: number | null
  accuracy_delta: number | null
  win_rate_delta: number | null
}

export interface HomeHeroCard {
  player_name: string
  kpis: HeroKPIs
  trend: HeroTrend | null
}

export interface HighlightItem {
  title: string
  value: string
  detail: string
}

export interface RecentMatchItem {
  match_id: string
  title: string
  detail: string
  started_at: string | null
  outcome_label: string
  outcome_tone: string
}

export interface SessionSummaryItem {
  session_label: string
  match_count: number
  win_rate: number
  global_ratio: number | null
  started_at: string | null
}

export interface RecentMediaItem {
  basename: string
  match_id: string | null
  match_start_time: string | null
}

export interface HomePageResponse {
  hero: HomeHeroCard
  highlights: HighlightItem[]
  recent_matches: RecentMatchItem[]
  recent_media: RecentMediaItem[]
  solo_session: SessionSummaryItem | null
  squad_session: SessionSummaryItem | null
}

export interface BattlePassResponse {
  available: boolean
  rank: number | null
  reward_track: string | null
  progress: number | null
  error_hint: string | null
}

export interface ChallengesResponse {
  available: boolean
  total: number | null
  completed: number | null
  xp_available: number | null
  next_expiry: string | null
  error_hint: string | null
}

// ---------------------------------------------------------------------------
// Escouade / Coéquipiers (Slice 6)
// ---------------------------------------------------------------------------

export interface TeammateOption {
  gamertag: string
  xuid: string | null
  encounter_count: number
  last_seen_at: string | null
}

export interface TeammateKPIs {
  match_count: number
  wins: number
  kd_ratio: number | null
  win_rate: number
  accuracy: number | null
  kills_per_game: number | null
  assists_per_game: number | null
}

export interface TeammateRow {
  gamertag: string
  xuid: string | null
  encounter_count: number
  last_seen_at: string | null
  with_kpis: TeammateKPIs
  without_kpis: TeammateKPIs | null
}

export interface TeammatesQueryRequest {
  selected_gamertags?: string[]
  filters?: FilterContextInput | null
}

export interface TeammatesPageResponse {
  options: TeammateOption[]
  teammates: TeammateRow[]
  solo_reference: TeammateKPIs | null
  total_matches: number
}

// ---------------------------------------------------------------------------
// Synthèse (Slice 7)
// ---------------------------------------------------------------------------

export interface SynthesisKPIs {
  match_count: number
  wins: number
  kd_ratio: number | null
  win_rate: number
  accuracy: number | null
  kills_per_min: number | null
  avg_life_seconds: number | null
  performance_score: number | null
}

export interface ComparisonMetricItem {
  label: string
  solo_value: number
  squad_value: number
  solo_text: string
  squad_text: string
}

export interface SynthesisQueryRequest {
  period?: string
  filters?: FilterContextInput | null
}

export interface SynthesisPageResponse {
  period: string
  total_matches: number
  solo_kpis: SynthesisKPIs
  squad_kpis: SynthesisKPIs
  comparison_metrics: ComparisonMetricItem[]
}

// ---------------------------------------------------------------------------
// Médias (Slice 8)
// ---------------------------------------------------------------------------

export interface MediaItemRow {
  basename: string
  file_path: string
  kind: string
  thumbnail_path: string | null
  match_id: string | null
  capture_end_utc: string | null
  match_start_time: string | null
  section: string
  owner_gamertag: string | null
  map_name: string | null
}

export interface MediaQueryRequest {
  sort?: string
  kind_filter?: string | null
  section_filter?: string | null
  pagination?: PaginationRequest
}

export interface MediaPageResponse {
  items: PaginatedResponse<MediaItemRow>
  total_mine: number
  total_teammates: number
  total_unassigned: number
}
