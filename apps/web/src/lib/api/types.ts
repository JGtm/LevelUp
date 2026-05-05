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
  /** Sprint 44 : titre associé (ex: "halo_infinite"). */
  title_slug?: string
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

export interface HaloIdentitySummary {
  gamertag: string
  xuid: string
}

/** Sprint 44 : résumé d'un titre supporté pour le frontend. */
export interface TitleSummary {
  slug: string
  name: string
  icon_url?: string
  status: 'active' | 'coming_soon' | 'archived'
  capabilities: string[]
  is_default: boolean
}

export interface BootstrapResponse {
  setup_required: boolean
  auth_state: 'missing' | 'partial' | 'ready'
  setup_state: 'no_halo_link' | 'halo_linked_no_profile' | 'profile_ready_no_sync' | 'ready'
  current_player: PlayerSummary | null
  available_players: PlayerSummary[]
  /** Sprint 44 : titre courant */
  current_title_slug: string
  /** Sprint 44 : titres disponibles */
  available_titles: TitleSummary[]
  locale: string
  hints_visible_default: boolean
  feature_flags: FeatureFlags
  capabilities: CapabilityMap
  settings_excerpt: SettingsExcerpt
  /** Identité Halo liée (gamertag + xuid) — absente si auth non complétée. */
  linked_halo_identity?: HaloIdentitySummary | null
  /** ID du job de sync initial actif pour cette session (null si aucun). */
  active_sync_job_id?: string | null
  /** Auth locale : mode d'authentification ("none" | "password"). */
  auth_mode: 'none' | 'password'
  /** Mode d'inscription ("invite" | "open" | "closed"). */
  registration_mode: 'invite' | 'open' | 'closed'
  /** True si l'utilisateur courant est admin. */
  is_admin: boolean
  /** Username connecté (si mode password et connecté). */
  current_username?: string | null
  /** True si aucun user n'est enregistré (premier lancement). */
  first_launch: boolean
}

export interface PlayersListResponse {
  items: PlayerSummary[]
  default_player_slug: string | null
}

export interface SessionContextRequest {
  player_slug?: string | null
  title_slug?: string | null  // Sprint 44 : switch titre
  locale?: string | null
  hints_visible?: boolean | null
}

export interface SessionContextResponse {
  current_player: PlayerSummary | null
  current_title_slug: string  // Sprint 44
  locale: string
  hints_visible: boolean
  capabilities: CapabilityMap
  /** Sprint 44 : titres disponibles pour le titre switcher. */
  available_titles?: TitleSummary[]
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
// Lab interne
// ---------------------------------------------------------------------------

export interface LabFileStatus {
  path: string
  exists: boolean
  size_bytes: number
  modified_at?: string | null
}

export interface LabSnapshotSummary {
  resource_key: string
  version: string
  fetched_at: string
  content_hash: string
  etag?: string | null
  source_url?: string | null
  payload_size: number
}

export interface LabSnapshotDetail {
  resource_key: string
  version: string
  fetched_at: string
  content_hash: string
  etag?: string | null
  source_url?: string | null
  payload: string
}

export interface LabAssetSummary {
  asset_id: string
  asset_type: string
  version_id: string
  name: string
  fetched_at: string
}

export interface LabAssetDetail {
  asset_id: string
  asset_type: string
  version_id: string
  name: string
  description: string
  fetched_at: string
  content_hash: string
  raw_json: string
}

export interface LabAssetExplorer {
  total: number
  search?: string | null
  items: LabAssetSummary[]
  selected?: LabAssetDetail | null
}

export interface LabMedalSummary {
  medal_id: number
  name_id: string
  description_id: string
  medal_type: string
  difficulty: string
  sprite_index: number
  fetched_at: string
}

export interface LabMedalDetail {
  medal_id: number
  name_id: string
  description_id: string
  medal_type: string
  difficulty: string
  sprite_index: number
  personal_score: number
  fetched_at: string
  content_hash: string
  raw_json: string
}

export interface LabMedalExplorer {
  total: number
  search?: string | null
  items: LabMedalSummary[]
  selected?: LabMedalDetail | null
}

export interface LabResourcesResponse {
  title_slug: string
  metadata_db_path: string
  current_season?: {
    title_id: string
    season_id: string
    version: string
    name: string
    start_date: string
    end_date?: string | null
    fetched_at: string
    content_hash: string
    etag?: string | null
    source_url?: string | null
  } | null
  seasons: Array<{
    title_id: string
    season_id: string
    version: string
    name: string
    start_date: string
    end_date?: string | null
    fetched_at: string
    content_hash: string
    etag?: string | null
    source_url?: string | null
  }>
  csr_seasons: Array<{
    title_id: string
    season_id: string
    version: string
    name: string
    start_date: string
    end_date?: string | null
    fetched_at: string
    content_hash: string
    etag?: string | null
    source_url?: string | null
  }>
  snapshots: LabSnapshotSummary[]
  selected_snapshot?: LabSnapshotDetail | null
  assets: LabAssetExplorer
  medals: LabMedalExplorer
}

export interface LabRouteMethods {
  path: string
  methods: string[]
}

export interface LabMethodMismatch {
  fastapi_path: string
  go_path: string
  fastapi_methods: string[]
  go_methods: string[]
  missing_methods: string[]
  extra_methods: string[]
}

export interface LabOpenAPISummary {
  fastapi_route_count: number
  go_route_count: number
  missing_in_go: number
  extra_in_go: number
  method_mismatches: number
  status: string
}

export interface LabContractsResponse {
  go_openapi: LabFileStatus
  fastapi_reference: LabFileStatus
  summary: LabOpenAPISummary
  missing_in_go: LabRouteMethods[]
  extra_in_go: LabRouteMethods[]
  method_mismatches: LabMethodMismatch[]
}

export interface LabGuardResult {
  passed: boolean
  reason: string
  details: string[]
}

export interface LabMedalGuardsReport {
  entry_count: number
  cardinality: LabGuardResult
  required_fields: LabGuardResult
  images: LabGuardResult
  overall: LabGuardResult
}

export interface LabParitySummary {
  total: number
  passed: number
  failed: number
  skipped: number
}

export interface LabParityResult {
  name: string
  status: string
  http_status?: number | null
  mode?: string | null
  reason?: string | null
  error?: string | null
  diffs?: Array<Record<string, unknown>>
}

export interface LabParityReport {
  generated_at: string
  go_url: string
  player: string
  summary: LabParitySummary
  results: LabParityResult[]
}

export interface LabDiagnosticsResponse {
  title_slug: string
  parity_report_file: LabFileStatus
  parity_report?: LabParityReport | null
  medal_guards?: LabMedalGuardsReport | null
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
  match_context?: 'solo' | 'squad' | 'all'
}

export interface LabelValue {
  label: string
  value: string
  /** Nombre de matchs si on AJOUTE cette option à la sélection courante de la
   *  catégorie (sémantique OR). Pour une option déjà cochée : total post-cascade.
   *  Pour une option non cochée : matchs après ajout. 0 = option à neutraliser
   *  (grisée dans la cascade, masquée pour sessions/presets). */
  count: number
  /** Optionnel : si présent, l'option est un enfant à grouper sous l'option racine
   *  dont la value est <parent>. Utilisé pour les <optgroup> hiérarchiques. */
  parent?: string
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
  /** Nombre de matchs de la session APRÈS application de la cascade et du
   *  match_context (sans filtre période, puisque period et sessions sont
   *  mutuellement exclusifs). 0 = session à masquer dans le dropdown. */
  match_count_filtered: number
  is_squad: boolean
  /** Timestamps début/fin de session (ISO UTC) — start_time du 1er et du
   *  dernier match. Permettent à l'UI de formater des labels localisés
   *  type « Session du 6 avril 2026 de 21:43 à 23:40 ». */
  started_at_utc?: string
  ended_at_utc?: string
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

export interface PeriodPresetCount {
  /** "7d" | "30d" | "90d" | "all" — doit rester aligné avec PERIOD_PRESETS
   *  côté frontend (apps/web/src/components/shell/_filter_pills/_hooks.ts). */
  preset_id: string
  /** 7, 30, 90, 0 (=all). Permet au frontend de croiser preset_id et days
   *  sans recompter. */
  days: number
  /** Matchs si l'utilisateur switchait en mode period sur ce preset, avec la
   *  cascade actuelle. 0 = preset à griser dans PeriodePill. */
  count: number
}

/** Compte cascade-aware par saison du catalog (kind="season" du TOML).
 *  Sert au folding "+N saisons sans matchs ▾" dans SaisonPill. */
export interface SeasonCount {
  /** Identifiant stable de la saison (ex: "season6", "season10_op1"). */
  season_id: string
  /** Matchs si l'utilisateur sélectionnait cette saison, avec la cascade
   *  actuelle. 0 = saison repliée sous le folding. */
  count: number
}

export interface FilterContextResolved {
  effective: FilterContextInput
  available_options: AvailableOptions
  session_options: SessionOptions
  counts: FilterCounts
  period_presets: PeriodPresetCount[]
  /** Optionnel : absent si le titre n'a pas de kind "season" dans son TOML. */
  season_counts?: SeasonCount[]
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

// @deprecated sprint 29 — GET /setup/status est un artefact mort (absent FastAPI + Go).
// Conserver temporairement pour ne pas casser les imports existants.
// À supprimer avec useSetupStatus() au Sprint 32.
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
  /** Durée de validité en secondes depuis l'émission. */
  expires_in: number
  /** @deprecated Alias de expires_in pour compatibilité backend. */
  expires_in_seconds?: number
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

export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'interrupted' | 'cancelled'

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
// Backfill (recalcul rétroactif)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Engagement (Phase 4 plan engagement)
// ---------------------------------------------------------------------------

export interface EngagementPointAPI {
  TimeMS: number
  PaceJoueur: number
  PaceTeam: number
  PaceAttendu: number
  PaceLobby: number
  PostDeathFlag: boolean
  IsPassiveDeath: boolean
}

export interface EngagementScoreResultAPI {
  EngagementScore: number | null
  ResidualBrut: number
  EngagementCurve: EngagementPointAPI[] | null
  MatchIntensity: number
  Confidence: 'full' | 'partial' | 'insufficient_history'
  NHistoryMatches: number
}

export interface EngagementMatchSummaryAPI {
  match_id: string
  label: string
  started_at: string
  pace_joueur: number
  pace_team: number
  pace_attendu: number
  pace_lobby: number
  engagement_score: number | null
}

export interface SquadPlayerEngagementAPI {
  xuid: string
  gamertag: string
  pace_observed: number[]
}

export interface SquadEngagementSessionAPI {
  labels: string[]
  map_names: string[]
  lobby_per_player: number[]
  team_expected: number[]
  team_observed: number[]
  players: SquadPlayerEngagementAPI[]
}

export interface EngagementCoefficientAPI {
  XUID: string
  ModeCategory: string
  CoefTeamShare: number
  CoefLobbyShare: number
  NMatches: number
  LastUpdated: string
}

export interface BackfillStartRequest {
  player_slug: string
  medals?: boolean
  events?: boolean
  skill?: boolean
  personal_scores?: boolean
  performance_scores?: boolean
  aliases?: boolean
  weapons?: boolean
  lusr?: boolean
  engagement_scores?: boolean
  all_data?: boolean
  max_matches?: number
  dry_run?: boolean
  force_rescan?: boolean
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
  discord_notify_friends: boolean
  spnkr_auto_sync_enabled: boolean
  spnkr_auto_sync_interval_hours: number
  spnkr_auto_sync_interval_minutes: number
  watcher_presence_enabled: boolean
  watcher_subscribed_players: string[]
  friend_gamertags: string[]
  // --- Règles de sessions ---
  session_gap_minutes: number
  session_split_on_ranked_change: boolean
  session_team_change_mode: 'ignore' | 'group' | 'friends'
  // --- Règles de badges narratifs ---
  outcome_exclude_bot_matches_from_badges: boolean
  outcome_exclude_bot_matches_from_records: boolean
  outcome_badge_sensitivity: 'relaxed' | 'standard' | 'strict'
  // --- Affichage Objectifs/Prestige ---
  show_progression: boolean
  // --- Fournisseur d'authentification (admin uniquement) ---
  auth_provider: string
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
  kills: number | null
  deaths: number | null
  assists: number | null
  kd_ratio: number | null
  variant: 'best' | 'worst' | null
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
  is_excluded?: boolean
  /** S56 — combat yield */
  kills?: number | null
  assists?: number | null
  deaths?: number | null
  damage_dealt?: number | null
  damage_taken?: number | null
  offensive_conversion?: number | null
  defensive_resistance?: number | null
  offensive_finishing?: number | null
  map_image_url?: string | null
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
  /** Sprint 54-B : avertissement privacy */
  privacy_warning?: MatchPrivacyWarning | null
  /** §5 plan Squad/Sessions : sessions dispo (split solo/squad). */
  session_labels: SessionLabelsList
  /** Alimente <SessionBriefing> en haut de la page Stats Solo (mode solo). */
  briefing_kpis?: KPIStats
}

export interface MatchHistoryQueryRequest {
  filters?: FilterContextInput
  pagination?: PaginationRequest
  columns?: string[] | null
  include_export_hint?: boolean
  /** §5 plan Squad/Sessions : filtre multi-sessions solo. */
  picked_solo_session_labels?: string[]
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
  favorites_only?: boolean
}

export interface ExplorerPlayerQueryRequest {
  target_gamertag: string
  page?: number
}

export interface ExplorerCommonMatchRow {
  match_id: string
  start_time: string
  map_ui: string
  mode_ui: string
  were_teammates: boolean
  player_outcome: number
  outcome_label: string
  kills: number
  deaths: number
  kda: number
}

export interface MatchEncounterBadge {
  kind: string
  label_key: string
  color_token: string
  detail?: Record<string, unknown>
}

export interface ExplorerPlayerQueryResponse {
  target_gamertag: string
  target_xuid: string
  common_matches: ExplorerCommonMatchRow[]
  badges?: MatchEncounterBadge[]
  total: number
  total_count: number
  wins_together: number
  losses_together: number
  page: number
  page_size: number
}

export interface ExplorerMatchesQueryResponse {
  summary: ExplorerMatchesQuerySummary
  table: PaginatedResponse<ExplorerMatchRow>
}

// ---------------------------------------------------------------------------
// Accueil Mission Control (Slice 5)
// ---------------------------------------------------------------------------

export interface HeroKPIs {
  win_rate: number
  global_ratio: number | null
  avg_kda: number | null
  avg_accuracy: number | null
  total_matches: number
  wins: number
  draws: number
  dnfs: number
  losses: number
  total_playtime_secs: number
  favorite_weapon_name: string
  favorite_weapon_kills: number
  favorite_playlist_name: string
  favorite_playlist_count: number
  avg_offensive_conversion: number | null
  avg_defensive_resistance: number | null
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

export type HighlightValueColor =
  | 'positive'
  | 'warning'
  | 'neutral'
  | 'negative'
  | 'perf-excellent'
  | 'perf-good'
  | 'perf-ok'
  | 'perf-low'
  | 'perf-bad'

export interface HighlightSlide {
  label_key?: string
  label?: string
  value: string
  detail?: string
  detail_key?: string
  detail_params?: Record<string, string | number>
  value_color?: HighlightValueColor
}

export interface HighlightItem {
  title_key?: string
  title?: string
  value: string
  detail?: string
  detail_key?: string
  detail_params?: Record<string, string | number>
  value_color?: HighlightValueColor
  slides?: HighlightSlide[]
}

export interface RecentMatchItem {
  match_id: string
  title: string
  detail: string
  started_at: string | null
  outcome_label: string
  outcome_tone: string
  score_label?: string | null
  narrative_badges?: string[]
  is_favorite: boolean
  /** S56 — champs enrichis pour MatchCard */
  map_ui?: string | null
  mode_ui?: string | null
  playlist_ui?: string | null
  kills?: number | null
  assists?: number | null
  deaths?: number | null
  performance_score_relative?: number | null
  offensive_conversion?: number | null
  defensive_resistance?: number | null
  offensive_finishing?: number | null
  damage_dealt?: number | null
  damage_taken?: number | null
  map_image_url?: string | null
  skill_rating_value?: number | null
  skill_rating_type?: string | null
  skill_tier_label?: string | null
  skill_rating_delta?: number | null
  skill_playlist_group?: string | null
  skill_rank_image_url?: string | null
  skill_progress_pct?: number | null
  skill_points_in_tier?: number | null
  kda?: number | null
  duration_secs?: number | null
  accuracy?: number | null
  avg_life_secs?: number | null
  team_mmr?: number | null
  enemy_mmr?: number | null
  delta_mmr?: number | null
  is_with_friends?: boolean | null
  rank_in_team?: number | null
  headshot_kills?: number | null
  perfect_kills?: number | null
  top_medals?: RecentMatchMedal[] | null
  top_citations?: MatchCitationSnippet[] | null
  /** Bug #6 — permet au front de filtrer l'OutcomeSequenceTape sur la dernière session. */
  session_label?: string | null
}

export interface MatchCitationSnippet {
  key: string
  name: string
  description?: string | null
  image_url?: string | null
  delta: number
  progress_pct: number
  is_newly_mastered?: boolean
}

export interface RecentMatchMedal {
  medal_id: number
  name: string
  count: number
  description?: string | null
  image_url: string
}

export interface SessionSummaryItem {
  session_label: string
  match_count: number
  win_rate: number
  global_ratio: number | null
  started_at: string | null
  ended_at: string | null
  wins: number
  losses: number
  draws: number
  dnfs: number
  avg_player_performance: number | null
  avg_team_performance: number | null
  avg_kda: number | null
  dominant_playlist: string | null
  dominant_mode: string | null
}

export interface RecentMediaItem {
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
  mode_name: string | null
  liked: boolean
  like_count: number
}

export interface HomeCareerRankSummary {
  rank_number: number
  rank_title: string
  next_rank_title?: string
  rank_image_url?: string | null
  adornment_image_url?: string | null
  current_xp: number
  xp_for_next_rank: number
  progress_pct: number
  is_max_rank: boolean
}

export interface HomeSkillPeakSummary {
  rating_value: number
  tier_label?: string | null
  badge_image_url?: string | null
}

export interface HomePlaylistRank {
  playlist_name: string
  is_ranked: boolean
  rating_type?: string | null   // "CSR" | "LUSR" — absent si aucun rang calculé
  rating_value?: number | null
  tier_label?: string | null
  badge_image_url?: string | null
}

export interface HomeSpartanIdentity {
  banner_image_url?: string | null
  spartan_id?: string | null
  emblem_image_url?: string | null
  backdrop_image_url?: string | null
  highest_csr?: HomeSkillPeakSummary | null
  highest_lusr?: HomeSkillPeakSummary | null
  career_rank?: HomeCareerRankSummary | null
}

export interface HomePageResponse {
  hero: HomeHeroCard
  spartan_identity?: HomeSpartanIdentity | null
  highlights: HighlightItem[]
  recent_matches: RecentMatchItem[]
  favorite_matches: RecentMatchItem[]
  recent_media: RecentMediaItem[]
  solo_session: SessionSummaryItem | null
  squad_session: SessionSummaryItem | null
  solo_sessions?: SessionSummaryItem[]
  squad_sessions?: SessionSummaryItem[]
  has_ranked_history?: boolean
  has_unranked_history?: boolean
  /** Sprint 54-B9 : signal discret si les données sont partielles (compte privé). */
  privacy_warning?: MatchPrivacyWarning | null
  recent_playlist_ranks?: HomePlaylistRank[]
}

export interface BattlePassResponse {
  available: boolean
  rank: number | null
  reward_track: string | null
  progress: number | null
  error_hint: string | null
}

export interface ChallengeItem {
  challenge_path: string
  tracking_id?: string | null
  title: string
  description?: string | null
  image_url?: string | null
  progress_current?: number | null
  progress_target?: number | null
  progress_percent?: number | null
  xp_reward?: number | null
  is_squad?: boolean | null
}

export interface ChallengesResponse {
  available: boolean
  total: number | null
  completed: number | null
  xp_available: number | null
  next_expiry: string | null
  items?: ChallengeItem[]
  error_hint: string | null
}

export type SeasonPassStatus = 'active' | 'in_progress' | 'completed' | 'not_started'

export interface SeasonPassItemSummary {
  title: string
  description?: string | null
  image_url?: string | null
  /** Rareté brute renvoyée par GameCMS : Common / Rare / Epic / Legendary / Mythic. */
  quality?: string | null
  /** Catégorie brute : ArmorCoating, WeaponCharm, SpartanEmblem… */
  item_type?: string | null
}

export interface SeasonPassTierSummary {
  rank: number
  title: string
  description?: string | null
  image_url?: string | null
  quality?: string | null
  item_type?: string | null
  is_obtained: boolean
  is_current: boolean
  is_premium: boolean
  free_rewards?: SeasonPassItemSummary[]
}

export interface SeasonPassContentSummary {
  total_tiers: number
  credits?: number | null
  spartan_points?: number | null
  xp_boosts?: number | null
  challenge_swaps?: number | null
  cosmetics_total?: number | null
  rarity_breakdown?: Record<string, number> | null
  type_breakdown?: Record<string, number> | null
}

export interface SeasonPassTrackSummary {
  reward_track_path: string
  name: string
  description?: string | null
  status: SeasonPassStatus
  is_active: boolean
  is_owned: boolean
  has_reached_max_rank: boolean
  current_rank: number
  partial_progress: number
  xp_per_rank?: number | null
  max_rank?: number | null
  completion_percent?: number | null
  active_tier_rank?: number | null
  active_tier_progress_percent?: number | null
  image_url?: string | null
  background_image_url?: string | null
  tiers?: SeasonPassTierSummary[]
  content?: SeasonPassContentSummary | null
}

export interface SeasonPassPageResponse {
  title_slug: string
  available: boolean
  error_hint?: string | null
  active_track_path?: string | null
  challenges: ChallengesResponse
  passes: SeasonPassTrackSummary[]
}

export interface RelationInsight {
  xuid: string
  gamertag: string
  total_matches: number
  teammate_matches: number
  teammate_wins: number
  teammate_win_rate?: number | null
  enemy_matches: number
  enemy_wins: number
  enemy_win_rate?: number | null
  avg_kda_with?: number | null
  avg_kda_against?: number | null
  last_seen_at?: string | null
}

export interface RelationsOverview {
  distinct_players: number
  frequent_allies: number
  repeat_rivals: number
  closed_circle: number
}

export interface RelationsPageResponse {
  overview: RelationsOverview
  frequent_allies: RelationInsight[]
  best_synergies: RelationInsight[]
  nemeses: RelationInsight[]
  favorite_victims: RelationInsight[]
  closed_circle: RelationInsight[]
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

export interface RadarAxes {
  objectives: number
  combat: number
  support: number
  score: number
  impact: number
  survival: number
}

export interface TeammateKPIs {
  match_count: number
  wins: number
  kd_ratio: number | null
  win_rate: number
  accuracy: number | null
  kills_per_game: number | null
  assists_per_game: number | null
  headshot_kills_per_game?: number | null
  perfect_kills_per_game?: number | null
  radar_axes?: RadarAxes | null
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
  picked_solo_session_labels?: string[]
  picked_squad_session_labels?: string[]
  locale?: string
}

export interface SessionLabelEntry {
  label: string
  started_at: string
  ended_at: string
  experiences?: string[]
  playlists?: string[]
}

export interface SessionLabelsList {
  solo: SessionLabelEntry[]
  squad: SessionLabelEntry[]
}

export interface SquadTimeseriesPoint {
  period_label: string
  match_count: number
  wins: number
  win_rate: number
  avg_performance: number | null
  avg_mmr: number | null
}

export interface MapBreakdownRow {
  map_ui: string
  match_count: number
  win_rate: number
  historical_win_rate?: number
  /** Moyenne du performance_score sur les matchs escouade filtrés (session). Nil si aucun score. */
  performance_avg?: number
  /** Moyenne du performance_score sur l'historique complet du joueur principal pour cette carte. */
  historical_performance_avg?: number
}

export interface SquadMatchSeriesPoint {
  match_id: string
  start_time: string
  outcome: number
  performance_score: number | null
  team_mmr_avg: number
  session_label: string | null
}

/** Une ligne (joueur) du butterfly first-events teammates.17. */
export interface SquadFirstEventsRow {
  player: string
  kill_counts: number[]
  death_counts: number[]
}

/** Données du chart teammates.17 — bins 15 s par défaut. */
export interface SquadFirstEvents {
  bin_size_seconds: number
  bin_labels: string[]
  rows: SquadFirstEventsRow[]
}

/** Une ligne du chart kills par arme teammates.09. */
export interface SquadWeaponBar {
  weapon_id: number
  label: string
  is_grenade_melee?: boolean
  /** gamertag → kills (joueurs absents = 0). */
  kills_by_player: Record<string, number>
  total_squad: number
}

/** Données du chart teammates.09 — players ordonnés (main puis teammates),
 *  bars triées par TotalSquad ASC (peu utilisées en haut). */
export interface SquadWeaponKills {
  players: string[]
  bars: SquadWeaponBar[]
}

/** Point d'une série performance (1 par match × joueur) pour teammates.16. */
export interface SquadPerformanceSeriesPoint {
  match_id: string
  start_time: string
  match_order: number
  map_name?: string
  kills: number
  deaths: number
  assists: number
  kda?: number
  accuracy?: number
  avg_life_seconds?: number
  performance_score?: number
  max_killing_spree?: number
  headshot_kills?: number
  perfect_kills?: number
  rendement_offensif?: number
  resistance_defensive?: number
  team_mmr?: number
  skill_rating?: number
  skill_delta?: number
  skill_rating_type?: 'csr' | 'lusr'
}

/** Un axe du radar synergie teammates.06 (value normalisé 0..100, raw debug). */
export interface SquadSynergyRadarAxis {
  axis: string
  value: number
  raw: number
}

/** Profil radar (1 par joueur, sur les matchs partagés). */
export interface SquadSynergyRadarSeries {
  player: string
  axes: SquadSynergyRadarAxis[]
  mode_family?: string
}

/** Ligne du heatmap d'intensité teammates.15. Phases est une matrice 1×10. */
export interface SquadIntensityMatchRow {
  match_id: string
  label: string
  phases: number[]
}

export interface SquadIntensityOption {
  key: string
  label: string
}

export interface SquadIntensityProfile {
  options: SquadIntensityOption[]
  rows: Record<string, SquadIntensityMatchRow[]>
}

/** Agrégat par joueur pour le chart per-minute teammates.14. */
export interface SquadPerMinuteEntry {
  player: string
  kills_per_minute: number
  deaths_per_minute: number
  assists_per_minute: number
  match_count: number
}

/** Point agrégé par session pour le chart timeline teammates.04. */
export interface SquadSessionPoint {
  session_label: string
  squad_perf: number
  match_count: number
  wins: number
  losses: number
  win_rate?: number
  team_mmr_avg?: number
}

/** Cellule (joueur, carte, perf_avg) du heatmap teammates.03. */
export interface SquadMapHeatmapCell {
  player: string
  map_ui: string
  perf_avg?: number
  match_count: number
}

export interface SquadMapHeatmap {
  players: string[]
  maps_topn: string[]
  cells: SquadMapHeatmapCell[]
}

export interface SquadImpactBadgeCount {
  badge_key: string
  count: number
}

export interface SquadImpactPlayerSummary {
  player: string
  counts: SquadImpactBadgeCount[]
  score: number
}

export interface SquadImpactMatchHeader {
  match_id: string
  outcome: number
}

export interface SquadImpactCell {
  player: string
  match_id: string
  badge_keys: string[]
}

/** Données du scoreboard impact teammates.07. */
export interface SquadImpactMatrix {
  matches: SquadImpactMatchHeader[]
  players: SquadImpactPlayerSummary[]
  cells: SquadImpactCell[]
  badge_ord: string[]
}

/**
 * Ligne du tableau historique escouade (teammates.11). Une ligne par match
 * unique sur le scope filtré, triée serveur-side par start_time DESC.
 * Pagination assurée côté client (TanStack Table, 20/page).
 */
export interface SquadMatchHistoryRow {
  match_id: string
  start_time: string
  map_ui: string
  playlist_name?: string
  pair_name?: string
  mode_ui?: string
  outcome: number
  kills: number
  deaths: number
  assists: number
  accuracy?: number
  performance_score?: number
  team_mmr_avg: number
  enemy_mmr_avg?: number
  delta_mmr?: number
  score_label?: string
  duration_seconds?: number
  /** Taux de victoire historique du joueur sur cette carte (ratio 0..1). */
  win_rate_hist?: number
  /** Nombre total de matchs du joueur sur cette carte (dénominateur). */
  win_rate_hist_total?: number
  session_label?: string | null
}

export interface MedalDigestItem {
  medal_id: number
  label?: string
  description?: string
  image_url?: string
  total_count: number
  match_count: number
  category?: string       // multikill | spree | skill | style | mode | proficiency
  difficulty?: string     // Normal | Heroic | Legendary | Mythic
  personal_score?: number // XP de carrière par médaille (0 ou absent = fallback difficulty)
}

export interface MedalDigestEntry {
  player: string
  emblem_url?: string
  distinct_types: number
  total_count: number
  avg_per_match: number
  peak_in_match: number
  top_medals: MedalDigestItem[]
  all_medals: MedalDigestItem[]
}

export interface TeammatesPageResponse {
  options: TeammateOption[]
  teammates: TeammateRow[]
  total_matches: number
  session_labels: SessionLabelsList
  /** Nombre total d'amis configurés (settings.friend_gamertags). Sert au label UI "parmi N amis". */
  friends_count: number
  timeseries?: SquadTimeseriesPoint[]
  map_breakdown?: MapBreakdownRow[]
  match_series?: Record<string, SquadMatchSeriesPoint[]>
  match_history?: SquadMatchHistoryRow[]
  session_timeline?: SquadSessionPoint[]
  map_heatmap?: SquadMapHeatmap
  impact_matrix?: SquadImpactMatrix
  per_minute_stats?: SquadPerMinuteEntry[]
  synergy_radar?: SquadSynergyRadarSeries[]
  intensity_profile?: SquadIntensityProfile
  performance_series?: Record<string, SquadPerformanceSeriesPoint[]>
  weapon_kills?: SquadWeaponKills
  first_events?: SquadFirstEvents
  /** Header alimente <SessionBriefing> (mode solo si pas de coéquipier sélectionné, mode squad sinon). */
  header?: import('@/features/squad/v2/types').SquadHeader
  /** Gamertag du joueur principal — sert à identifier le card "moi" dans header.player_cards. */
  main_player?: string
  /** MedalDigest alimente <MedalDigest> en bas de SquadSynergiesPage. */
  medal_digest?: MedalDigestEntry[]
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
}

export interface SynthesisQueryRequest {
  period?: string
  filters?: FilterContextInput | null
}

export interface HeatmapCell {
  dow: number   // 0 = lundi … 6 = dimanche
  hour: number
  count: number
}

export interface TopWeekItem {
  week_label: string
  match_count: number
  win_rate: number
  kd_ratio: number | null
}

export interface SynthesisPageResponse {
  period: string
  total_matches: number
  solo_kpis: SynthesisKPIs
  squad_kpis: SynthesisKPIs
  comparison_metrics: ComparisonMetricItem[]
  heatmap_data: HeatmapCell[]
  top_weeks: TopWeekItem[]
  // Sprint 55 D5/D6/D7
  highlights_preview?: SynthesisHighlightsPreview
  rivalries_preview?: SynthesisRivalriesPreview
  breakdowns?: SynthesisBreakdowns
  // Sprint 55 D9 — scope + overview
  scope?: SynthesisScope
  overview?: SynthesisOverview
}

// Sprint 55 D9 — Scope
export interface SynthesisScope {
  period: string
  match_count: number
  filters_applied?: string[]
  filters_ignored?: string[]
  description: string
  computed_at: string
}

// Sprint 55 D9 — Overview
export interface SynthesisOverview {
  total_matches: number
  total_wins: number
  total_losses: number
  total_kills: number
  total_deaths: number
  total_assists: number
  avg_kda?: number | null
  avg_kills?: number | null
  avg_deaths?: number | null
  win_rate: number
  avg_perf_score?: number | null
  // P2.5 (revue 2026-04-29 ADR 0006) : K/D agrege canonique sum/sum (analysis.KDR)
  // distinct du recompute front faux (sum/sum != avg(K/D)). Voir B3.
  total_kdr?: number | null
  best_kills_match?: number | null
  best_kda_match?: number | null
  longest_win_streak?: number
}

// Sprint 55 D5 — Highlights
export interface SynthesisMatchHighlight {
  match_id: string
  kills: number
  deaths: number
  kda: number | null
  outcome: number
  perf_score: number | null
}

export interface SynthesisHighlightsPreview {
  top_by_kills: SynthesisMatchHighlight[]
  top_by_kda: SynthesisMatchHighlight[]
  worst_by_deaths: SynthesisMatchHighlight[]
}

// Sprint 55 D6 — Rivalries
export interface SynthesisEncounterPreview {
  xuid: string
  gamertag: string
  match_count: number
  as_teammate: number
  as_enemy: number
  avg_kda: number | null
}

export interface SynthesisRivalriesPreview {
  top_teammates: SynthesisEncounterPreview[]
  top_enemies: SynthesisEncounterPreview[]
  total: number
}

// Sprint 55 D7 — Breakdowns
export interface SynthesisMapEntry {
  map_name: string
  match_count: number
  wins: number
  win_rate: number
}

export interface SynthesisModeEntry {
  mode_name: string
  match_count: number
  wins: number
  win_rate: number
}

export interface SynthesisBreakdowns {
  top_maps: SynthesisMapEntry[]
  top_modes: SynthesisModeEntry[]
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
  mode_name: string | null
  liked: boolean
  like_count: number
  /** Noms des 3 premiers likers (affiches inline) */
  likers?: string[]
  /** Nombre total de personnes ayant liké */
  total_likers?: number
}

export interface MediaAvailableFilters {
  playlists: LabelValue[]
  maps: LabelValue[]
  modes: LabelValue[]
}

export interface MediaQueryRequest {
  sort?: string
  kind_filter?: string | null
  section_filter?: string | null
  /** Whitelist explicite de player_slug ; prend le pas sur section_filter si non vide. */
  author_slugs?: string[] | null
  /** playlist_id (UUID) ou label brut. */
  playlist_filter?: string | null
  map_filter?: string | null
  mode_filter?: string | null
  group_by?: string | null
  liked_only?: boolean | null
  pagination?: PaginationRequest
}

export interface MediaPageResponse {
  items: PaginatedResponse<MediaItemRow>
  total_mine: number
  total_teammates: number
  total_unassigned: number
  available_filters: MediaAvailableFilters
}

export interface MediaAuthor {
  player_slug: string
  gamertag: string
  is_self: boolean
  media_count: number
}

export interface MediaMatchLobbyEntry {
  gamertag: string
  team_id?: number | null
  is_self: boolean
}

export interface MediaMatchCandidate {
  match_id: string
  start_time?: string | null
  end_time?: string | null
  map_name?: string | null
  map_image_url?: string | null
  mode_name?: string | null
  playlist_name?: string | null
  is_current: boolean
  delta_seconds?: number | null
  outcome?: number | null
  lobby?: MediaMatchLobbyEntry[]
}

export interface MediaMatchCandidatesResponse {
  file_path: string
  capture_utc?: string | null
  window_minutes: number
  candidates: MediaMatchCandidate[]
}

export interface MediaAssociateRequest {
  file_path: string
  match_id: string
}

export interface MediaAssociateResponse {
  file_path: string
  match_id: string
  map_name?: string | null
  mode_name?: string | null
}

export interface MediaAuthorsResponse {
  authors: MediaAuthor[]
}

export interface MediaLikeRequest {
  file_path: string
  liked: boolean
  /** Slug du joueur qui like (pour la table shared) */
  liker_slug?: string
}

export interface MediaLikeResponse {
  file_path: string
  liked: boolean
  like_count: number
  likers?: string[]
  total_likers?: number
}

export interface MediaUploadResponse {
  saved: number
  new_indexed: number
  associated: number
  thumbnails: number
  errors?: string[]
}

// ---------------------------------------------------------------------------
// Match View (Slice 4B)
// ---------------------------------------------------------------------------

export interface MatchViewHeader {
  match_id: string
  start_time: string | null
  start_time_label: string
  outcome_code: number | null
  outcome_label: string
  /** Hex legacy. Préférer outcome_color_token (chunk MV3 cleanup). */
  outcome_color: string
  /** Token sémantique outcome-{win,loss,draw,dnf} (Phase 1 MV3). */
  outcome_color_token?: string
  score_label: string
  /** Booléen legacy (true si un badge narratif s'applique). Préférer dominance_badge. */
  dominance_flag: boolean
  /** Badge narratif typé (chunk MV1). Nul si aucun badge ne s'applique. */
  dominance_badge?: {
    flag: number
    label_key: string
    color_token: string
  }
  had_bot_teammate: boolean
  map_ui: string
  map_id: string | null
  mode_ui: string
  playlist_label: string
  performance_display: string
  /** Hex legacy. Préférer performance_color_token. */
  performance_color: string | null
  /** Token sémantique perf-tier-1..5 (Phase 1 MV3). */
  performance_color_token?: string
  is_excluded: boolean
  /** V7 : durée jouable réelle en secondes */
  playable_duration_seconds?: number | null
  /** V7 : lien Waypoint vers la replay */
  waypoint_url?: string | null
}

export interface MatchViewRank {
  rating_type: string
  tier_label: string | null
  numeric_value: number | null
  delta_value: number | null
  icon_url: string | null
}

export interface MatchMedal {
  medal_name_id: number
  name: string
  count: number
  description: string | null
}

export interface MatchCitation {
  key: string
  label: string
  color: string | null
  value: number | null
}

export interface MatchSummaryKpis {
  kills: number | null
  deaths: number | null
  assists: number | null
  kda: number | null
  damage_dealt: number | null
  average_life: string | null
  /** V7 */
  headshot_kills?: number | null
  max_killing_spree?: number | null
  perfect_kills?: number | null
  accuracy?: number | null
  personal_score?: number | null
}

export interface MatchPersonalResult {
  outcome_label: string
  outcome_color: string
  score: number | null
  rank_in_team: number | null
}

export interface MatchExpectedStats {
  has_expected_data: boolean
  expected_kills: number | null
  expected_deaths: number | null
  expected_assists: number | null
  /** V7 — moyennes historiques sur le mode */
  has_hist_avg?: boolean
  hist_avg_kills?: number | null
  hist_avg_deaths?: number | null
  hist_avg_assists?: number | null
  hist_match_count?: number
  hist_mode_category?: string | null
}

export interface MatchSummaryTab {
  kpis: MatchSummaryKpis
  personal_result: MatchPersonalResult
  medals: MatchMedal[]
  citations: MatchCitation[]
  expected_stats: MatchExpectedStats
}

export interface MatchWeaponKill {
  weapon_id: number
  weapon_label: string
  effective_weapon_id: number | null
  kill_count: number
}

export interface MatchHighlightEvent {
  event_time_ms: number | null
  event_type: string
  actor_xuid: string | null
  target_xuid: string | null
  weapon_id: number | null
}

export interface MatchTugOfWarBin {
  bin_start: number
  bin_end: number
  team_kills: number
  enemy_kills: number
  net_kills: number
}

export interface MatchImpactBadge {
  key: string
  label: string
  value?: string | null
  player_xuid?: string
  /** Instant (ms depuis le début du match) pour les badges event-based.
   *  Nul/absent pour les badges stat-based (top_killer, silent_hero, false_brother). */
  time_ms?: number | null
}

export interface MatchKDTimelinePoint {
  time_seconds: number
  kills: number
  deaths: number
}

/** Paire killer→victim agrégée pour le chart match_view.18 (antagonistes). */
export interface MatchKillerVictimPair {
  killer_xuid: string
  killer_gamertag: string
  victim_xuid: string
  victim_gamertag: string
  kill_count: number
}

export interface MatchCombatTab {
  weapon_kills: MatchWeaponKill[]
  highlight_events: MatchHighlightEvent[]
  /** V7 */
  tug_of_war: MatchTugOfWarBin[]
  /** Deprecated. Préférer impact_roles (8 rôles narratifs typés). */
  impact_badges: MatchImpactBadge[]
  kd_timeline: MatchKDTimelinePoint[]
  nemesis_duels: MatchNemesisRow[]
  /** Paires killer→victim agrégées (match_view.18). Vide si killer_victim_pairs absent. */
  killer_victim?: MatchKillerVictimPair[]
  /** Phase 1 MV2 : 8 rôles narratifs typés via narrative.IdentifyImpactRoles. */
  impact_roles?: MatchViewImpactRole[]
  /** Phase 1 MV2 : cadence intra-match (ChartSeries<ChartPointStacked>). */
  cadence?: MatchViewCadence | null
}

/** MV2 : rôle narratif attribué (1 entrée par joueur × rôle). */
export interface MatchViewImpactRole {
  xuid: string
  role_key: string
  label_key: string
  color_token: string
  inverted?: boolean
}

/** MV2 : cadence chart (mirror domain.ChartSeries<ChartPointStacked>). */
export interface MatchViewCadence {
  key: string
  label_key?: string
  datapoints: Array<{
    category: string
    components: Record<string, number>
  }>
  meta?: Record<string, unknown>
}

/** MV4.B : série radar 6 axes par joueur. */
export interface MatchViewRadarSeries {
  xuid: string
  gamertag?: string
  axes: Array<{
    Axis: string
    Value: number
    Raw: number
  }>
  mode_family?: string
}

export interface MatchScoreboardRow {
  xuid: string
  gamertag: string
  team_side: string | null
  is_me: boolean
  rank: number | null
  score: number | null
  kills: number | null
  deaths: number | null
  assists: number | null
  kda?: number | null
  shots_fired: number | null
  shots_hit: number | null
  shots_accuracy: number | null
  damage_dealt: number | null
  damage_taken: number | null
  average_life: string | null
  avg_life_seconds?: number | null
  headshot_kills: number | null
  max_killing_spree: number | null
  perfect_kills: number | null
  power_weapon_kills: number | null
  melee_kills: number | null
  outcome_label: string
  /** V7 — combat yield */
  top_weapon_id?: number | null
  top_weapon_label?: string | null
  offensive_conversion?: number | null
  defensive_resistance?: number | null
  damage_per_kill?: number | null
  damage_per_death?: number | null
}

export interface MatchRosterRow {
  xuid: string
  gamertag: string
  team_side: string | null
  is_me: boolean
  is_bot: boolean
  kills: number | null
  deaths: number | null
  assists: number | null
  kda: number | null
  damage_dealt: number | null
  damage_taken: number | null
}

export interface MatchNemesisRow {
  xuid: string
  gamertag: string
  killed_me: number
  i_killed: number
}

export interface MatchEncounterRow {
  xuid: string
  gamertag: string
  count_together: number
  is_ally: boolean
  /** Phase 1 MV4.C : badges narratifs typés (ordinal aujourd'hui ; ally_plus + tough_enemy à venir). */
  badges?: Array<{
    kind: string
    label_key: string
    color_token: string
    detail?: Record<string, unknown>
  }>
}

export interface MatchTeamTab {
  roster: MatchRosterRow[]
  scoreboard: MatchScoreboardRow[]
  nemesis: MatchNemesisRow[]
  encounters: MatchEncounterRow[]
}

export interface AssociatedMediaItem {
  file_id: string
  file_name: string
  file_path: string
  thumbnail_url: string | null
  duration_seconds: number | null
  capture_time: string | null
  liked: boolean
}

export interface MatchMediaTab {
  media_items: AssociatedMediaItem[]
}

export interface MatchCitationsTab {
  commendations: MatchCitation[]
  medals: MatchMedal[]
}

export interface MatchViewResponse {
  header: MatchViewHeader
  rank: MatchViewRank
  summary_tab: MatchSummaryTab
  combat_tab: MatchCombatTab
  team_tab: MatchTeamTab
  media_tab: MatchMediaTab
  citations_tab: MatchCitationsTab
  /** Phase 1 MV4.B : radar 6 axes par joueur (vide si awards absents). */
  radar?: MatchViewRadarSeries[]
  /** Sprint 54-B : avertissement privacy */
  privacy_warning?: MatchPrivacyWarning | null
}

/** Navigation prev/next entre matchs adjacents d'un joueur (ordre chronologique). */
export interface MatchNeighbors {
  previous_match_id: string | null
  next_match_id: string | null
  current_index: number
  total_matches: number
}

// ---------------------------------------------------------------------------
// Citations (Slice 2B)
// ---------------------------------------------------------------------------

export interface CommendationSummary {
  key: string
  label: string
  category: string | null
  current_value: number
  color: string | null
  icon_path: string | null
  tier_label: string | null
  mastery_pct: number | null
}

export interface MedalSummary {
  medal_name_id: number
  name: string
  count_filtered: number
  count_total: number
  description: string | null
}

export interface CitationsDeltas {
  filtered_total: number
  unfiltered_total: number
  delta_count: number
}

export interface CitationsQueryRequest {
  filters: FilterContextInput
}

export interface CitationsPageResponse {
  commendations: CommendationSummary[]
  medals_summary: MedalSummary[]
  deltas: CitationsDeltas
}

// ---------------------------------------------------------------------------
// Timeseries (Slice 3B)
// ---------------------------------------------------------------------------

/** Point d'une courbe cumulative ou glissante indexée sur les matchs. */
export interface CumulativePoint {
  index: number
  start_time: string
  value: number
}

/** Bucket d'un histogramme de distribution. */
export interface DistributionBucket {
  bucket_lower: number
  bucket_upper: number
  count: number
}

/** Paire (x, y) pour un scatter plot de corrélation. */
export interface CorrelationDataPair {
  metric_x_key: string
  metric_y_key: string
  x_value: number
  y_value: number
  outcome: number | null
}

/** Point de la heatmap intensité (jour × heure). */
export interface IntensityHeatmapPoint {
  day_of_week: number // 0=Lundi … 6=Dimanche
  hour: number
  count: number
  avg_kd: number
}

/** Ligne brute par match pour les charts timeline côté frontend. */
export interface TimeseriesMatchRow {
  match_id: string
  index: number
  start_time: string
  kills: number
  deaths: number
  assists: number
  // P2.5 (revue 2026-04-29 ADR 0006) : KDA (sync) + KDRatio canonique calcule
  // par analysis.KDR(). Permet de supprimer le recompute K/D cote front (B3).
  kda?: number | null
  kd_ratio?: number | null
  accuracy: number | null
  outcome: number | null
  personal_score: number | null
  damage_dealt: number | null
  damage_taken: number | null
  perf_score: number | null
  rank: number | null
  playlist_name: string
  time_played_seconds: number | null
}

export interface TimeseriesKpiCard {
  key: string
  label: string
  value: string
  delta: string | null
}

export interface TimeseriesSummaryTab {
  kpi_cards: TimeseriesKpiCard[]
}

export interface TimeseriesCumulTab {
  cumulative_kd: CumulativePoint[]
  cumulative_net: CumulativePoint[]
  rolling_kd: CumulativePoint[]
}

export interface TimeseriesRegressionStats {
  kd_slope: number | null
  winrate_slope: number | null
  r_squared: number | null
  has_enough_for_trend: boolean
  trend: 'improving' | 'declining' | 'stable' | null
}

export interface TimeseriesFormTab {
  regression_stats: TimeseriesRegressionStats
  ewma_kd_points: CumulativePoint[]
}

export interface TimeseriesIntensityTab {
  heatmap_data: IntensityHeatmapPoint[]
  score_per_min_data: CumulativePoint[]
}

export interface TimeseriesDistributionsTab {
  kda_buckets: DistributionBucket[]
  kills_buckets: DistributionBucket[]
  accuracy_buckets: DistributionBucket[]
  score_per_min_buckets: DistributionBucket[]
  rolling_wr_buckets: DistributionBucket[]
  correlation_points: CorrelationDataPair[]
}

export interface TimeseriesQueryRequest {
  filters: FilterContextInput
}

/** RankDelta — delta de skill rating sur le scope. Miroir Go domain.RankDelta.
 *  Kind = "csr" (classé) ou "lusr" (non classé) ; value = somme signée des
 *  per-match deltas ; count = nb matchs du Kind retenu dans le scope. */
export interface RankDelta {
  kind: 'csr' | 'lusr'
  value: number
  count: number
}

/** KPIStats — agreges du joueur sur le scope filtre. Miroir Go domain.KPIStats. */
export interface KPIStats {
  matches_count: number
  total_play_seconds: number
  avg_match_seconds: number
  kills_per_game: number
  kills_per_minute: number
  deaths_per_game: number
  deaths_per_minute: number
  assists_per_game: number
  assists_per_minute: number
  avg_accuracy: number
  avg_life_seconds: number
  /** Delta de rang (CSR ou LUSR) sur le scope. Absent si aucun match
   *  classé/non-classé dans le scope. Couleur par signe (pos/neg/neutral). */
  rank_delta?: RankDelta
  outcomes: { wins: number; losses: number; ties: number; dnf: number }
}

export interface TimeseriesPageResponse {
  total_matches: number
  match_rows: TimeseriesMatchRow[]
  summary_tab: TimeseriesSummaryTab
  cumul_tab: TimeseriesCumulTab
  form_tab: TimeseriesFormTab
  intensity_tab: TimeseriesIntensityTab
  distributions_tab: TimeseriesDistributionsTab
  /** Alimente <SessionBriefing> en haut de la page (mode solo). Nil si aucun match. */
  briefing_kpis?: KPIStats
}

// ---------------------------------------------------------------------------
// Session Compare (Slice 3C)
// ---------------------------------------------------------------------------

export interface SessionCompareEntry {
  session_label: string
  start_time: string | null
  end_time: string | null
  total_matches: number
  wins: number
  losses: number
  kda: number | null
  performance_score: number | null
  with_friends: boolean
  dominant_category: string | null
}

export interface SessionCompareMetricRow {
  key: string
  label: string
  value_a: string
  value_b: string
  delta: string | null
  winner: string | null
}

export interface SessionCompareRequest {
  filters: FilterContextInput
  session_a?: string | null
  session_b?: string | null
}

export interface SessionCompareResponse {
  session_a: SessionCompareEntry | null
  session_b: SessionCompareEntry | null
  available_sessions: string[]
  metrics: SessionCompareMetricRow[]
  maps_table: Record<string, unknown>[]
  modes_table: Record<string, unknown>[]
}

export interface SessionDetailMatchRow {
  match_id: string
  start_time: string
  outcome: number | null
  playlist_name: string
  pair_name: string
  is_ranked: boolean
  kills: number
  deaths: number
  assists: number
  kda: number | null
  accuracy: number | null
  personal_score: number | null
  performance_score: number | null
  session_label: string | null
  dominant_category: string | null
  offensive_conversion: number | null
  defensive_resistance: number | null
}

export interface SessionCompareSuggestion {
  session_label: string
  strategy: string
  reason: string
}

export interface SessionPageRequest {
  filters?: FilterContextInput
  session_label?: string | null
  compare_session_label?: string | null
  enable_compare?: boolean
}

export interface SessionPageResponse {
  current_session: SessionCompareEntry | null
  available_sessions: string[]
  matches: SessionDetailMatchRow[]
  suggested_compare: SessionCompareSuggestion | null
  compare_enabled: boolean
  compare_session: SessionCompareEntry | null
  compare_metrics: SessionCompareMetricRow[]
}

// ─── Sprint 54-C : Compare joueur vs joueur ───────────────────────────────────

export interface NormalizedPlayerStats {
  xuid: string
  gamertag: string
  title_slug: string
  matches: number
  win_rate: number
  kda: number
  kdr: number
  kills_per_game: number
  deaths_per_game: number
  assists_per_game: number
  accuracy: number
  damage_per_game: number
  career_rank: number
  csr_current: number
  csr_best: number
  extended?: Record<string, unknown>
  is_local: boolean
}

export interface CompareMetricRow {
  metric: string
  label_fr: string
  value_a: string | number
  value_b: string | number
  delta: number | null
  winner: 'a' | 'b' | 'tie' | null
}

export interface CompareRequest {
  target_gamertag: string
  filters?: FilterContextInput
}

export interface CompareResponse {
  player_a: NormalizedPlayerStats
  player_b: NormalizedPlayerStats
  metrics: CompareMetricRow[]
  title_slug: string
  /** C3.6 : avertissement si joueur B est privé ou introuvable. */
  privacy_warning?: MatchPrivacyWarning | null
  /** C3.6 : indique si les données de joueur B sont partielles (champs null). */
  player_b_partial?: boolean
}

// ─── Sprint 54-B : Match Privacy ─────────────────────────────────────────────

export interface MatchPrivacyInfo {
  is_private: boolean
  is_partial: boolean
  hint: string
}

export interface MatchPrivacyWarning {
  level: 'none' | 'partial' | 'full'
  message: string
}

// ─── Sprint 54-E : Leaderboard ───────────────────────────────────────────────

export interface LeaderboardEntry {
  rank: number
  xuid: string
  gamertag: string
  title_slug: string
  season_id: string
  playlist_id: string
  csr_value: number
  tier: string
  sub_tier: number
  is_local: boolean
}

export interface LeaderboardRequest {
  season_id?: string
  playlist_id?: string
  limit?: number
}

export interface LeaderboardResponse {
  entries: LeaderboardEntry[]
  season_id: string
  playlist_id: string
  title_slug: string
  total: number
}

// ---------------------------------------------------------------------------
// Auth locale
// ---------------------------------------------------------------------------

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  username: string
  role: 'admin' | 'user'
  gamertag?: string
}

export interface RegisterRequest {
  username: string
  password: string
  invite_code?: string
}

export interface RegisterResponse {
  username: string
  role: 'admin' | 'user'
}

export interface AdminUserSummary {
  username: string
  role: 'admin' | 'user'
  gamertag?: string
  created_at: string
  last_login_at?: string
}

export interface AdminInviteSummary {
  code: string
  created_by: string
  created_at: string
  expires_at: string
  used_by?: string | null
  used_at?: string | null
  valid: boolean
}

export interface InviteCode {
  code: string
  created_by: string
  created_at: string
  expires_at: string
}

// ---------------------------------------------------------------------------
// Watcher présence Xbox RTA
// ---------------------------------------------------------------------------

export interface WatcherPlayerStatus {
  gamertag: string
  xuid: string
  state: string
  in_game: boolean
  state_since: string
  state_duration: string
  cooldown_left?: string
  subscribe_error?: string
}

export interface WatcherStatusResponse {
  daemon_running: boolean
  rta_connected: boolean
  token_valid: boolean
  token_expires_at?: string
  token_gamertag?: string
  subscribed_players: string[]
  players: WatcherPlayerStatus[]
}

export interface WatcherAuthAttempt {
  attempt_id: string
  user_code: string
  verification_url: string
  expires_in: number
}

export interface WatcherAuthStatus {
  status: 'pending' | 'authorized' | 'failed' | 'expired'
  error_code?: string
  gamertag?: string
  xuid?: string
}

// ---------------------------------------------------------------------------
// Asset Drawer (Phase 2)
// ---------------------------------------------------------------------------

export interface AssetMeta {
  id: string
  name_en: string
  name_fr: string
  image_url: string
}

// ---------------------------------------------------------------------------
// Achievements Xbox (bilingues EN/FR)
// ---------------------------------------------------------------------------

export interface AchievementsSummary {
  total_count: number
  unlocked_count: number
  total_gamerscore: number
  earned_gamerscore: number
  /** 0..100, arrondi à 0.1 */
  completion_pct: number
}

export interface AchievementEntry {
  achievement_id: string
  name_en: string
  name_fr: string
  description_en: string
  description_fr: string
  locked_desc_en?: string
  locked_desc_fr?: string
  gamerscore: number
  image_url?: string
  is_secret: boolean
  rarity_category?: string
  rarity_percent?: number
  unlocked: boolean
  /** ISO 8601 (ex: "2026-04-15T14:30:00Z") */
  unlocked_at?: string
  current_progress?: number
  target_progress?: number
}

export interface AchievementsPageResponse {
  summary: AchievementsSummary
  achievements: AchievementEntry[]
}
