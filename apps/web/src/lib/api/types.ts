/**
 * Types de l'API LevelUp.
 *
 * Migration en cours (chantier types.ts → generated.ts) : les types qui ont un
 * schéma OpenAPI équivalent sont ré-exportés depuis `./generated` (source de
 * vérité = apps/go-api/api/openapi.yaml, régénéré via `npm run generate-types`).
 * Les types frontend-only (view models, sans schéma OpenAPI) restent définis ici.
 */
import type { components } from './generated'

// ---------------------------------------------------------------------------
// Communs
// ---------------------------------------------------------------------------

// Batch 1 (chantier migration) — ré-exports depuis le contrat OpenAPI.
export type PlayerSummary = components['schemas']['PlayerSummary']

export type CapabilityMap = components['schemas']['CapabilityMap']

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

export type FeatureFlags = components['schemas']['FeatureFlags']

export type SettingsExcerpt = components['schemas']['SettingsExcerpt']

export type HaloIdentitySummary = components['schemas']['HaloIdentitySummary']

export type TitleSummary = components['schemas']['TitleSummary']

// TODO(migration bucket B) : le schéma OpenAPI BootstrapResponse est INCOMPLET
// vs la réponse réelle du backend Go — il manque auth_mode, first_launch,
// current_username, current_title_slug, available_titles, registration_mode,
// is_admin, oauth_code_flow_enabled (+ le schéma a current_player?/privacy? en
// trop/optionnels). À réconcilier en complétant openapi.yaml puis régénérer,
// après quoi ce type pourra être shimé. Reste manuel d'ici là.
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
  auth_mode: 'none' | 'password' | 'xbox'
  /** Mode d'inscription ("invite" | "open" | "closed"). */
  registration_mode: 'invite' | 'open' | 'closed'
  /** Instance fermée : aucune nouvelle identité/BDD (register, SSO xuid inconnu, setup/players). */
  instance_locked?: boolean
  /** Joueur courant : refresh_token Microsoft mort → reconnexion Xbox requise (bannière). */
  reauth_required?: boolean
  /** Utilisateur connecté : a défini un mot de passe (opt-in re-login rapide). */
  has_password?: boolean
  /** True si l'utilisateur courant est admin. */
  is_admin: boolean
  /** Username connecté (si mode password et connecté). */
  current_username?: string | null
  /** True si aucun user n'est enregistré (premier lancement). */
  first_launch: boolean
  /** PR 4 — True si l'Authorization Code Flow OAuth est dispo (vraie UX redirect). */
  oauth_code_flow_enabled?: boolean
  /** Instance démo publique : settings figés, switch de langue client-side. */
  demo_mode?: boolean
}

export type PlayersListResponse = components['schemas']['PlayersListResponse']

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

export interface LabWaypointResponse {
  segment: string
  endpoint: string
  asset_id: string
  version_id: string
  lang: string
  resolved_ok: boolean
  asset_name?: string
  description?: string
  image_url?: string
  error?: string
  latency_ms: number
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

/** Réponse de POST /filters/match-ids : liste ordonnée (start_time DESC) des
 *  match_id de la sélection courante. Alimente le bouton "Voir les matchs". */
export interface FilterMatchIdsResponse {
  match_ids: string[]
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
  /** Slug joueur ("_all" pour les jobs serveur-wide, ex. cycle de sync forcé). */
  player_slug?: string
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

/**
 * IMPORTANT — clés en snake_case : le backend Go sérialise via les json tags
 * du struct domain.EngagementPoint / EngagementScoreResult (snake_case).
 * Les anciennes clés PascalCase (TimeMS, EngagementCurve…) étaient la cause
 * du chart d'engagement vide en Match View — bug pré-existant corrigé ici.
 */
export interface EngagementPointAPI {
  time_ms: number
  pace_joueur: number
  pace_team: number
  pace_attendu: number
  pace_lobby: number
  post_death_flag: boolean
  is_passive_death: boolean
}

export interface EngagementScoreResultAPI {
  engagement_score: number | null
  residual_brut: number
  engagement_curve: EngagementPointAPI[] | null
  match_intensity: number
  confidence: 'full' | 'partial' | 'insufficient_history'
  n_history_matches: number
  mean_pace_joueur?: number
  mean_pace_team?: number
  mean_pace_lobby?: number
  player_activity?: number
}

/** EngagementTimeseriesRequest — body de POST /engagement/timeseries.
 *  Aligné sur TimeseriesQueryRequest : `filters` honore period / cascade /
 *  sessions / match_context. `limit` borne le nombre de points retournés
 *  (défaut 50, max 500 côté Go). Quand le scope filtré dépasse `limit`, le
 *  backend bascule en granularité agrégée (session → week → month). */
export interface EngagementTimeseriesRequest {
  filters: FilterContextInput
  limit?: number
}

/** Granularité d'un point engagement timeseries. */
export type EngagementGranularity = 'match' | 'session' | 'week' | 'month'

/** EngagementMatchSummaryAPI — 1 point soit pour 1 match (granularité "match"),
 *  soit pour l'agrégat de N matchs (session/week/month). Pour les agrégats,
 *  `match_id` est vide et `match_count > 1`. */
export interface EngagementMatchSummaryAPI {
  match_id: string
  label: string
  map_name?: string | null
  started_at: string
  pace_joueur: number
  pace_team: number
  pace_attendu: number
  pace_lobby: number
  engagement_score: number | null
  /** Nombre de matchs représentés par ce point (1 pour granularité "match",
   *  >1 pour les agrégats). */
  match_count: number
}

/** EngagementTimeseriesResponse — réponse de POST /engagement/timeseries.
 *  - `granularity` : choisie automatiquement selon la densité filtrée.
 *  - `total_matches` : compte total filtré AVANT cap workCap.
 *  - `truncated_to_recent` : si non-null, signale que le compute a été borné
 *    aux N matchs les plus récents (perfs ; au-delà le binning agrège tout
 *    de même mais sur un sous-ensemble). */
export interface EngagementTimeseriesResponse {
  granularity: EngagementGranularity
  points: EngagementMatchSummaryAPI[]
  total_matches: number
  truncated_to_recent?: number | null
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
  /** Re-fetch CSR par-match via GetMatchSkill (RankRecap). Idempotent
   *  par défaut ; force_rescan=true → re-fetche tous les matchs ranked. */
  csr?: boolean
  engagement_scores?: boolean
  engagement_coefficients?: boolean
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
  // --- Rendement combat (OffensiveConversion) ---
  rendement_exclude_assists: boolean
  // --- Affichage Objectifs/Prestige ---
  show_progression: boolean
  // --- Coach proactif (pont coach → Prestige, cf. ADR 0020) ---
  coach_proactive_mode: boolean
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
  rank_image_url?: string | null
  next_rank_name_fr?: string
  next_rank_name_en?: string
  next_rank_image_url?: string | null
}

export interface HeroProgress {
  xp_total_required: number
  xp_remaining: number
  percentage: number
  current_rank: number
  total_ranks?: number
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
  current_xp: number
  xp_total: number
}

export interface FriendXPHistory {
  gamertag: string
  history: CareerHistoryPoint[]
}

export interface CareerLusrCheckpoint {
  recorded_at: string | null
  rating_type: string
  rating_value: number
  tier_label: string | null
  playlist_group: string | null
  playlist_name: string
  badge_image_url?: string | null
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
  friends_xp_history?: FriendXPHistory[]
  top_matches_preview: CareerTopMatch[]
  encounters_preview: CareerEncounter[]
}

export interface CareerTopMatchesResponse {
  items: CareerTopMatch[]
}

export interface CareerEncountersResponse {
  items: CareerEncounter[]
}

// Section "Matchs marquants" (page Carrière) : 15 best + 15 worst au format
// ExplorerMatchRow (mêmes 21 colonnes que la page Explorer) + cascade counts
// pour les filtres Expérience / Saisons.
export interface CareerHighlightMatchesResponse {
  best_matches: ExplorerMatchRow[]
  worst_matches: ExplorerMatchRow[]
  available_experience: HighlightExperienceCount[]
  available_seasons: HighlightSeasonCount[]
  available_modes: HighlightModeCount[]
  available_playlists: HighlightPlaylistCount[]
}

export interface HighlightExperienceCount {
  value: 'all' | 'ranked' | 'unranked'
  count: number
}

export interface HighlightSeasonCount {
  value: string // season ID, ex. "season6"
  count: number
}

export interface HighlightModeCount {
  value: string // pair_name = mode_ui
  count: number
}

export interface HighlightPlaylistCount {
  value: string // playlist_name
  count: number
}

// Filtres optionnels passés en query params à GET /pages/career/highlight-matches.
export interface CareerHighlightFilters {
  experience?: 'all' | 'ranked' | 'unranked'
  season_ids?: string[]      // multi-select
  mode_uis?: string[]        // multi-select (= pair_name / mode_ui)
  playlist_names?: string[]  // multi-select
}

// Section "Joueurs les plus croisés (hors amis)" : 10 lignes au format
// MatchEncounterRow (réutilise le tableau Match View > Historique de rencontre).
export interface CareerTopEncountersResponse {
  items: MatchEncounterRow[]
}

// Section "Top némésis" / "Top souffre-douleur" : top 10 chacun, ratio
// frags/deaths calculé côté backend.
export interface CareerRival {
  gamertag: string
  frags: number
  deaths: number
  ratio: number
  match_count: number
}

export interface CareerRivalsResponse {
  nemeses: CareerRival[]
  victims: CareerRival[]
}

export interface CareerCSRRank {
  value: number
  tier: string
  sub_tier: number
  measurement_matches_remaining: number
  badge_image_url?: string | null
  /** Seuil placement de la saison du snapshot (5 depuis S3, 10 historique).
   *  Toujours présent depuis Phase 6 du plan pipeline CSR. */
  placement_total: number
}

export interface CareerPlaylistCSR {
  playlist_id: string
  playlist_name: string
  queue: string
  input: string
  current: CareerCSRRank
  season: CareerCSRRank
  all_time: CareerCSRRank
}

/** Saison CSR sélectionnable dans le menu "Classements" (page Carrière).
 *  Une saison apparaît si le joueur y a des données classées + la saison courante. */
export interface CSRSeasonOption {
  season_id: string
  label: string
  is_current?: boolean
}

export interface CareerCSRResponse {
  playlists: CareerPlaylistCSR[]
  season_id: string
  /** Saisons proposables dans le menu déroulant (CSR uniquement ; LUSR est cumulatif). */
  available_seasons: CSRSeasonOption[]
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
// Historique des matchs (Slice 3)
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
  outcome_code: number
  score_label: string
  is_with_friends: boolean
  experience_type_label: string
  kills?: number | null
  deaths?: number | null
  assists?: number | null
  perf_score?: number | null
  perf_tier?: number
  delta_perf?: number | null
  skill_tier_label?: string | null
  /** "CSR" (classé officiel) ou "LUSR" (interne LevelUp). Nil si pas de skill rank (PvE, Custom). */
  rating_type?: string | null
  /** Gain/perte de rating du match. NON rendu par les colonnes propres d'ExplorerMatchesTable :
   *  porteur de données pour une colonne « Δ rang » INJECTÉE par un consommateur
   *  (vue session via extraColumns). Reste undefined côté page Explorer. */
  skill_rating_delta?: number | null
  /** Proba de victoire pré-match de l'équipe (LUSR v2, 0..1). Porteur de données
   *  pour une colonne « Pronostic » INJECTÉE par la vue session. Undefined côté Explorer. */
  expected_win_prob?: number | null
  /** Progression placement (X). Si défini avec placement_total, l'UI affiche "X/Y" dans la cellule Rang à la place du skill_tier_label. */
  placement_done?: number | null
  /** Seuil placement (Y). CSR : 5 ou 10 selon saison. LUSR : 10. */
  placement_total?: number | null
  delta_mmr?: number | null
  team_mmr?: number | null
  enemy_mmr?: number | null
  kda?: number | null
  duration_seconds?: number | null
  match_url?: string
  /** 0=none, 1=domination, 2=humiliation, 3=remontada, 4=débandade, 5=contre-remontada. */
  dominance_flag?: number
  /**
   * Indique qu'un coéquipier du joueur était un bot pendant le match.
   * Présent uniquement sur les best_matches de la page carrière (exclu côté
   * backend pour worst_matches afin d'isoler la responsabilité du joueur).
   * Le front affiche une pill "bot" sur la card du match.
   */
  had_bot_teammate?: boolean
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
  available_experience_types?: string[]
  available_playlists?: string[]
  available_maps?: string[]
  available_modes?: string[]
  /** Options Explorer-spécifiques avec count cascade-aware (sémantique OR au sein
   *  d'une dimension, AND entre dimensions). Les options à count=0 sont à griser.
   *  Le `value` est : code outcome (1..4) ou tier (1..5) en string, ou clé EN
   *  (Bronze..Onyx) pour skill_tier, ou "" / "ranked" / "unranked" pour ranked,
   *  ou "" / "solo" / "squad" pour squad_scope. */
  available_outcomes?: LabelValue[]
  available_perf_tiers?: LabelValue[]
  available_skill_tiers?: LabelValue[]
  available_ranked_contexts?: LabelValue[]
  available_squad_scopes?: LabelValue[]
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

export interface ExplorerMatchesQueryRequest {
  filters?: FilterContextInput
  pagination?: PaginationRequest
  sort_field?: string
  sort_dir?: string
  include_export_hint?: boolean
  perf_tiers?: number[]
  skill_tiers?: string[]
  ranked_context?: 'ranked' | 'unranked' | ''
  outcome_filter?: number[]
  // Explorer-specific match filters
  match_start_date?: string | null
  match_end_date?: string | null
  experience_types?: string[]
  playlists?: string[]
  map_names?: string[]
  mode_names?: string[]
  squad_scope?: 'solo' | 'squad' | ''
  match_id_search?: string
  /** Whitelist exacte de match_id (mode Joueur : matchs en commun). */
  match_ids?: string[]
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

/** Stats agrégées du couple (player_courant, target). Mirror partiel de
 *  MatchEncounterRow — les 7 colonnes du tableau MatchEncountersTable
 *  (sans is_ally car pas de match courant en Explorer). */
export interface ExplorerEncounterStats {
  count_together: number
  ally_count?: number | null
  enemy_count?: number | null
  winrate_as_ally?: number | null
  winrate_vs_enemy?: number | null
  kills_dealt?: number | null
  deaths_suffered?: number | null
  last_seen_at?: string | null
}

export interface ExplorerPlayerQueryResponse {
  target_gamertag: string
  target_xuid: string
  common_matches: ExplorerCommonMatchRow[]
  badges?: MatchEncounterBadge[]
  encounter_stats?: ExplorerEncounterStats
  total: number
  total_count: number
  wins_together: number
  losses_together: number
  page: number
  page_size: number
  /** Agrégat jour × heure des matchs communs (toutes pages confondues).
   *  Coloration UI pilotée par count (intensité d'activité commune). */
  activity_heatmap?: HeatmapCell[]
  /** Encart "Profil joueur cible" affiché en haut des résultats Explorer
   *  mode Joueur. 4 sous-blocs best-effort + flag auth_available. */
  target_profile?: ExplorerTargetProfile
}

/** Encart "Profil joueur cible" composite (4 sources fetch en parallèle).
 *  Toutes les sous-sections sont nullable : le front masque celles à null
 *  et affiche un hint "Connexion Halo requise" quand auth_available=false. */
export interface ExplorerTargetProfile {
  /** Identité Spartan : banner, emblem, service tag, rang carrière.
   *  Fetch live Halo en mode auth, fallback DB locale en mode no-tokens. */
  identity?: HomeSpartanIdentity | null
  /** Stats carrière entière du joueur (KDA / KDR / win rate / accuracy / ...).
   *  Fetch via halostats career-stats. null en mode no-tokens. */
  career_stats?: NormalizedPlayerStats | null
  /** Stats agrégées du target sur les matchs joués en commun avec le user.
   *  Toujours calculable depuis DuckDB, null seulement si common_matches=0. */
  sample_stats?: ExplorerTargetSampleStats | null
  /** Top médailles lifetime (triées par count, cap 20) — service record Waypoint
   *  + métadonnées locales. Le front affiche un top 5 + expander. */
  top_medals?: MedalDigestItem[] | null
  /** Classements CSR par playlist ranked engagée de la saison courante (live). */
  season_csrs?: CareerPlaylistCSR[] | null
  /** Nombre de matchs matchmade par saison (graphe). */
  matches_per_season?: SeasonMatchCount[] | null
  /** Avertissement privacy — conservé pour compat, toujours null (privacy non fetchée). */
  privacy_warning?: MatchPrivacyWarning | null
  /** ~20 derniers matchs PvP de la cible fetchés en LIVE (API Halo) — source
   *  AFFICHÉE PAR DÉFAUT pour les graphes "profil de combat". Vide si pas d'auth/live. */
  combat_profile?: ExplorerTargetRecentMatch[] | null
  /** Derniers matchs PvP de la cible présents en base locale (surtout matchs communs).
   *  Alimente le toggle "local" de la section profil de combat. */
  combat_profile_local?: ExplorerTargetRecentMatch[] | null
  /** true si le user connecté a des tokens OAuth Halo. Sert à rendre le hint
   *  "Connexion Halo requise" sur les sections en mode dégradé. */
  auth_available: boolean
}

/** ExplorerTargetRecentMatch — un match PvP récent du joueur cible, projeté pour
 *  les graphes profil de combat. Miroir exact du DTO Go (JSON snake_case).
 *  `rank` est null si DNF/non classé (trou dans la courbe placement — ne pas tracer 0). */
export interface ExplorerTargetRecentMatch {
  match_id: string
  start_time: string
  map_ui: string
  mode_ui: string
  /** 1=tie, 2=win, 3=loss, 4=DNF (convention produit). */
  outcome: number
  rank?: number | null
  kills: number
  deaths: number
  assists: number
  /** Ratio FDA pré-calculé (match_participants.kda). */
  kda: number
  /** personal_score (colonne "Score" du scoreboard). */
  score: number
  damage_dealt: number
  damage_taken: number
  max_killing_spree: number
  perfect_kills: number
}

/** Nombre de matchs matchmade joués par le joueur cible sur une saison.
 *  `matches` = total de la saison (mode live = service record filtré par saison ;
 *  mode dégradé sans auth = bucketing local). Le pic de rang CSR de la saison est
 *  exposé (tier + image du badge) quand disponible — rendu au-dessus de la barre. */
export interface SeasonMatchCount {
  season_id: string
  season_name: string
  matches: number
  csr_tier?: string
  csr_sub_tier?: number
  csr_badge_image_url?: string | null
}

/** Stats agrégées du joueur cible sur l'échantillon des matchs en commun.
 *  Ratios en nullable : null signifie "indisponible" (dénominateur nul). */
export interface ExplorerTargetSampleStats {
  sample_size: number
  // Totaux bruts.
  kills: number
  deaths: number
  assists: number
  wins: number
  losses: number
  draws: number
  shots_fired: number
  shots_hit: number
  damage_dealt: number
  damage_taken: number
  // Breakdown kill types (somme sur le sample).
  headshot_kills: number
  melee_kills: number
  power_weapon_kills: number
  grenade_kills: number
  // Médailles totales / types distincts.
  total_medals: number
  unique_medals: number
  // Ratios calculés (nullable si dénominateur nul).
  kda?: number | null
  kdr?: number | null
  win_rate?: number | null
  accuracy?: number | null
  headshot_rate?: number | null
  offensive_conversion?: number | null
  defensive_resistance?: number | null
  // Cadence par minute (frags/morts/assists ÷ minutes jouées). null si durée nulle.
  kills_per_min?: number | null
  deaths_per_min?: number | null
  assists_per_min?: number | null
  // Score Halo moyen par match (AVG personal_score). null si sample vide.
  avg_personal_score?: number | null
  // Frags parfaits (médaille Perfect) cumulés sur le sample.
  perfect_kills: number
  // Top armes (par kills) sur les matchs communs (nom + kills, sans icône).
  top_weapons?: ExplorerWeaponKill[]
}

/** Une arme du top armes (cible Explorer) : label localisé + kills. */
export interface ExplorerWeaponKill {
  weapon_id: number
  label_fr: string
  label_en: string
  kills: number
}

export interface ExplorerMatchesQueryResponse {
  summary: ExplorerMatchesQuerySummary
  table: PaginatedResponse<ExplorerMatchRow>
  export_hint?: ExportHint
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
  /** Dégâts moyens par frag / par mort (Σ dmg / Σ kills|deaths). Nil si dénominateur nul. */
  dmg_per_kill?: number | null
  dmg_per_death?: number | null
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
  /** True si la playlist est classée (CSR officiel). Source : match_registry.is_ranked. */
  is_ranked?: boolean | null
}

export interface MatchCitationSnippet {
  key: string
  name: string
  description?: string | null
  image_url?: string | null
  delta: number
  progress_pct: number
  is_newly_mastered?: boolean
  cumulative?: number
  tier_index?: number
  tier_count?: number
  next_tier_target?: number
}

export interface RecentMatchMedal {
  medal_id: number
  name: string
  count: number
  description?: string | null
  image_url: string
  difficulty?: string | null
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
  /** Sessions escouade : gamertags des coéquipiers les plus présents (amis
   *  configurés, top 3). Sert au deep-link card escouade → /squad. Absent/[] pour
   *  les sessions solo ou si la donnée n'est pas disponible. */
  teammates?: string[]
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
  /** XP de carrière cumulée — affichée au rang max (où current_xp vaut 0). */
  total_xp?: number
  progress_pct: number
  is_max_rank: boolean
}

export interface HomeSkillPeakSummary {
  rating_value: number
  tier_label?: string | null
  badge_image_url?: string | null
  /**
   * Matchs de placement restants (10 → 0). Présent côté backend depuis mai 2026
   * pour CSR (player_csr_snapshots.current_measurement_remaining) et LUSR
   * (10 matchs par playlist_group). Sémantique :
   *   - absent / null : champ non remonté (legacy)
   *   - 0             : phase de placement terminée → afficher rating + tier
   *   - >0            : afficher "En placement (X/N)", badge_image_url
   *                     pointe déjà sur unranked_X.png côté backend (mapping
   *                     proportionnel selon placement_total)
   */
  measurement_matches_remaining?: number | null
  /** Seuil placement de la saison (5 depuis S3 mars 2023, 10 historique).
   *  Phase 6 du plan pipeline CSR. nil → fallback front à 10 (back-compat
   *  payloads pré-Phase 6). */
  placement_total?: number | null
  /** Remplissage ORDINAL de la barre (0..100), via le sous-palier (n/6),
   *  indépendant de l'échelle CSR/LUSR. null hors phase matured → pas de barre. */
  tier_progress_pct?: number | null
  /** Libellé localisé du SOUS-PALIER suivant (extrémité droite de la barre, ex.
   *  "Or IV", "Platine I", "Onyx"). null pour Onyx (sommet). */
  next_tier_label?: string | null
}

export interface HomePlaylistRank {
  playlist_name: string
  is_ranked: boolean
  rating_type?: string | null   // "CSR" | "LUSR" — absent si aucun rang calculé
  rating_value?: number | null
  tier_label?: string | null
  badge_image_url?: string | null
  /** Matchs de placement restants (threshold→0). Présent uniquement pour CSR ranked en placement. */
  measurement_matches_remaining?: number | null
  /** Seuil placement de la saison du match (5 ou 10). nil → fallback front à 10. */
  placement_total?: number | null
  /** Remplissage ORDINAL de la barre (0..100) via le sous-palier (n/6), indépendant
   *  de l'échelle CSR/LUSR. Calculé par analysis.SkillTierBand (même bande que le
   *  skill peak). null hors phase matured (placement / sans rang) → pas de barre. */
  tier_progress_pct?: number | null
  /** Libellé localisé du SOUS-PALIER suivant (extrémité droite de la barre, ex.
   *  "Or V", "Platine I"). null pour Onyx (sommet). */
  next_tier_label?: string | null
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
  from_cache?: boolean
  /** RFC3339 — date du snapshot affiché (now en live, MAX(snapshot_at) en fallback cache). */
  snapshot_at?: string | null
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
  from_cache?: boolean
  /** RFC3339 — date du snapshot affiché (now en live, MAX(snapshot_at) en fallback cache). */
  snapshot_at?: string | null
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
  /** Même agrégat que `content` mais limité aux paliers PAS ENCORE atteints
   *  (rang > current_rank). null/absent au rang max. Pour l'overlay « restant »
   *  (XX/YY) accueil + page pass saisonnier. */
  remaining_content?: SeasonPassContentSummary | null
  /** RFC3339 — date du dernier `battlepass_snapshots` connu pour ce track. */
  snapshot_at?: string | null
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
  damage_dealt?: number
  damage_taken?: number
  rendement_offensif?: number
  resistance_defensive?: number
  team_mmr?: number
  skill_rating?: number
  skill_delta?: number
  skill_rating_type?: 'csr' | 'lusr'
  skill_playlist_group?: string | null
  skill_season_id?: string | null
  skill_measurement_remaining?: number | null
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
  /** Durée réelle de gameplay (countdown pré-match retranché). Préférée à
   *  duration_seconds pour l'affichage de la durée du match. */
  gameplay_duration_seconds?: number
  /** Taux de victoire historique du joueur sur cette carte (ratio 0..1). */
  win_rate_hist?: number
  /** Nombre total de matchs du joueur sur cette carte (dénominateur). */
  win_rate_hist_total?: number
  /** Proba de victoire pré-match de l'équipe (LUSR v2, 0..1). Colonne « Prob. vic. ». */
  expected_win_prob?: number | null
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
  /** Sessions où la composition EXACTE (joueur principal + tous les coéquipiers
   *  sélectionnés) a joué ensemble. Alimente le SessionMultiSelect + le
   *  ré-ancrage. Sans coéquipier : sessions squad du joueur principal. */
  composition_sessions?: SessionLabelEntry[]
  /** Label de la session la plus récente de la composition exacte (1re entrée de
   *  composition_sessions). Vide si la composition n'a jamais joué ensemble. */
  latest_composition_session?: string
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
  headshots_per_match: number | null
  deaths_per_min: number | null
  assists_per_min: number | null
  avg_max_killing_spree: number | null
  avg_damage_dealt: number | null
  avg_damage_taken: number | null
}

export interface ComparisonMetricItem {
  label: string
  solo_value: number
  squad_value: number
}

export interface SynthesisQueryRequest {
  period?: string
  start_date?: string | null
  end_date?: string | null
  filters?: FilterContextInput | null
}

export interface HeatmapCell {
  dow: number    // 0 = lundi … 6 = dimanche
  hour: number
  count: number
  wins?: number
  win_rate?: number
}

export interface TopWeekItem {
  week_label: string
  /** ISO date (YYYY-MM-DD) du lundi 00:00 UTC de la semaine. Pour tri chronologique. */
  week_start: string
  match_count: number
  wins: number
  win_rate: number
  kd_ratio: number | null
  avg_kills?: number
  avg_deaths?: number
}

export interface SynthesisDetailedStats {
  total_headshot_kills: number
  total_perfect_kills: number
  total_grenade_kills: number
  total_melee_kills: number
  total_power_weapon_kills: number
  max_killing_spree: number
  total_time_played_seconds?: number
  total_shots_fired: number
  total_shots_hit: number
  total_damage_dealt: number
  total_damage_taken: number
  total_betrayals: number
  total_suicides: number
  total_vehicles_destroyed: number
  total_hijacks: number
}

// PLAN_COMBAT_PROFILE_WIRING — types profil combat 3 axes.
export type CombatStyleOffensive = 'disperse' | 'irregulier' | 'equilibre' | 'precis' | 'chirurgical'
export type CombatStyleDefensive = 'fragile' | 'expose' | 'solide' | 'resistant' | 'inebranlable'
export type CombatStyleActivity = 'passif' | 'discret' | 'mesure' | 'actif' | 'agressif'

export interface CombatProfileBlock {
  avg_oc: number
  avg_dr: number
  match_count: number
  avg_pace_ratio?: number | null
  /** Dégâts moyens par frag / par mort (agrégés). Nil si dénominateur nul. */
  dmg_per_kill?: number | null
  dmg_per_death?: number | null
  style_offensive?: CombatStyleOffensive | null
  style_defensive?: CombatStyleDefensive | null
  style_activity?: CombatStyleActivity | null
}

export interface SynthesisPageResponse {
  period: string
  total_matches: number
  solo_kpis: SynthesisKPIs
  squad_kpis: SynthesisKPIs
  comparison_metrics: ComparisonMetricItem[]
  heatmap_data: HeatmapCell[]
  top_weeks: TopWeekItem[]
  // Sprint 55 D5/D7 — D6 (rivalries) retiré le 2026-05-27 (section
  // "Relations de jeu" supprimée de la page Synthesis ; les encounters
  // restent exposés par la page palmares/relations).
  highlights_preview?: SynthesisHighlightsPreview
  breakdowns?: SynthesisBreakdowns
  // Sprint 55 D9 — scope + overview
  scope?: SynthesisScope
  overview?: SynthesisOverview
  // P9 — detailed stats par catégories
  detailed_stats?: SynthesisDetailedStats
  // Top frags par arme (label résolu, weapon ID non-résolu exclus)
  top_weapon_kills?: SynthesisWeaponKillEntry[]
  // PLAN_COMBAT_PROFILE_WIRING Phase 1
  combat_profile?: CombatProfileBlock | null
}

export interface SynthesisWeaponKillEntry {
  label: string
  kills: number
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
  total_ties: number
  total_dnf: number
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
  // Refs cliquables vers le match record pour chaque métrique (2026-05-27).
  // Permet l'ouverture du match depuis les cartes "Top X" / "Meilleur X".
  best_kills_ref?: BestMatchRef | null
  best_kda_ref?: BestMatchRef | null
  best_perf_ref?: BestMatchRef | null
  best_accuracy_ref?: BestMatchRef | null
  best_damage_ref?: BestMatchRef | null
  best_killing_spree_ref?: BestMatchRef | null
  best_headshots_ref?: BestMatchRef | null
  best_personal_score_ref?: BestMatchRef | null
}

export interface BestMatchRef {
  match_id: string
  value: number
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

// Sprint 55 D7 — Breakdowns
export interface SynthesisMapEntry {
  map_name: string
  match_count: number
  wins: number
  losses: number
  ties: number
  unfinished: number
  win_rate: number
}

export interface SynthesisModeEntry {
  mode_name: string
  match_count: number
  wins: number
  losses: number
  ties: number
  unfinished: number
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
  unassigned_only?: boolean | null
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
  is_bot?: boolean
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
  own_score?: number | null
  enemy_score?: number | null
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
  /** True si la playlist du match est classée (CSR officiel). Désactive le bouton "Exclure". */
  is_ranked: boolean
  /** V7 : durée jouable réelle en secondes */
  playable_duration_seconds?: number | null
  /** V7 : lien Waypoint vers la replay */
  waypoint_url?: string | null
  /** URL de l'image de la map (TitleAssetURLAdapter). Null si capability absente. */
  map_image_url?: string | null
  /** True si le joueur a marqué ce match comme favori (table match_favorites). */
  is_favorite: boolean
}

export interface MatchViewRank {
  rating_type: string
  tier_label: string | null
  numeric_value: number | null
  delta_value: number | null
  icon_url: string | null
  /** Position dans le sous-tier courant (0.0–1.0). Nil pour Onyx. */
  progress_pct?: number | null
}

export interface MatchMedal {
  medal_name_id: number
  name: string
  count: number
  description: string | null
  image_url?: string | null
  difficulty?: string | null
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
  team_mmr?: number | null
  enemy_mmr?: number | null
  delta_mmr?: number | null
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
  /** Proba de victoire pré-match de l'équipe du joueur (LUSR v2, 0..1). Absente pré-v2. */
  expected_win_prob?: number | null
  /** V7 — moyennes historiques sur le mode */
  has_hist_avg?: boolean
  hist_avg_kills?: number | null
  hist_avg_deaths?: number | null
  hist_avg_assists?: number | null
  hist_avg_spree?: number | null
  hist_avg_headshot_kills?: number | null
  hist_avg_perfect_kills?: number | null
  hist_match_count?: number
  hist_mode_category?: string | null
}

export interface MatchSummaryTab {
  kpis: MatchSummaryKpis
  personal_result: MatchPersonalResult
  medals: MatchMedal[]
  citations: MatchCitationSnippet[]
  expected_stats: MatchExpectedStats
}

export interface MatchWeaponKill {
  weapon_id: number
  weapon_label: string
  effective_weapon_id: number | null
  kill_count: number
}

export interface PlayerWeaponKillRow {
  weapon_id: number
  kills: number
  label?: string
  /** URL absolue (ou relative au domaine) de l'icône de l'arme — composée backend via static.URL. */
  image_url?: string
}

export interface PlayerMedalRow {
  medal_id: number
  count: number
  label?: string
  /** URL absolue (ou relative au domaine) de l'icône de la médaille. */
  image_url?: string
  /** Normal | Heroic | Legendary | Mythic — pour l'effet glow dans le scoreboard. */
  difficulty?: string | null
}

export interface MatchHighlightEvent {
  event_time_ms: number | null
  event_type: string
  actor_xuid: string | null
  /**
   * Gamertag déjà résolu côté backend via `v_gamertag_lookup`
   * (bots → "343 Bot N", cascade alias/participants/raw fallback).
   * Préférer ce champ sur `actor_xuid` pour l'affichage. Optionnel : absent
   * du JSON quand le backend tourne sur une version pré-RC6, ou null si
   * l'xuid est totalement orphelin (jamais en DB).
   */
  actor_gamertag?: string | null
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
  /** True si participant détecté comme bot (xuid au format "bid(N.0)"). */
  is_bot?: boolean
  rank: number | null
  score: number | null
  kills: number | null
  deaths: number | null
  assists: number | null
  kda?: number | null
  shots_fired: number | null
  shots_hit: number | null
  accuracy: number | null
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
  expected_kills?: number | null
  expected_deaths?: number | null
  expected_assists?: number | null
  weapon_kills?: PlayerWeaponKillRow[]
  /** Médailles gagnées par CE joueur dans ce match (expander scoreboard). */
  medals?: PlayerMedalRow[]
  /** Performance score 0..100 — uniquement pour les joueurs trackés (main + amis). */
  performance_score?: number | null
  /** True si bot dans l'équipe du joueur — uniquement pour les joueurs trackés. */
  had_bot_teammate?: boolean
  /** Skill rank (CSR/LUSR) pour ce match — uniquement pour les joueurs trackés. */
  skill_rank?: MatchScoreboardSkillRank | null
}

export interface MatchScoreboardSkillRank {
  /** "CSR" ou "LUSR" */
  rating_type: string
  tier_label?: string | null
  rating_value?: number | null
  rating_delta?: number | null
  /** URL du badge image CSR/LUSR (résolu côté backend via TitleAssetURLAdapter). */
  icon_url?: string | null
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
  /** True si participant détecté comme bot (xuid au format "bid(N.0)"). */
  is_bot?: boolean
  /** Découpage de count_together en allié vs ennemi sur l'historique commun. */
  ally_count?: number | null
  enemy_count?: number | null
  /** Winrates ratio 0..1 sur l'historique commun. null si W+L == 0. */
  winrate_as_ally?: number | null
  winrate_vs_enemy?: number | null
  /** K/D croisé : kills par moi sur ce joueur / morts subies par moi causées par lui. */
  kills_dealt?: number | null
  deaths_suffered?: number | null
  /** Date ISO du dernier match commun (toutes occurrences). */
  last_seen_at?: string | null
  /** Phase 1 MV4.C / MV4.C' : badges narratifs typés (ordinal / ally_plus / tough_enemy). */
  badges?: MatchEncounterBadge[]
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
  /** Type brut DB ('video' | 'image'), normalisé front via normalizeMediaKind. */
  kind: string
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
  /**
   * RC6 — true quand le match a son `match_registry` peuplé mais qu'au moins
   * une source secondaire critique est vide (scoreboard / events / stats /
   * medals). Le 404 strict reste pour les match_id totalement absents en DB.
   * Le front affiche un bandeau dégradé au lieu de l'écran d'erreur.
   */
  is_partial?: boolean
  /**
   * Codes stables des raisons de la partialité. Utilisés pour traduire en
   * messages i18n côté front (« sync incomplet », « film expiré », etc.).
   * Codes possibles : "scoreboard_empty", "events_empty",
   * "player_stats_empty", "medals_empty".
   */
  partial_reasons?: string[]
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

/** Une citation enrichie avec sa progression par paliers. */
export interface CitationItem {
  name_norm: string
  name_display: string
  category: string
  total: number
  tier_count: number
  earned_tiers: number
  next_tier_target: number  // 0 si maîtrisé
  mastery_pct: number       // 0..100
  image_url?: string
  description?: string
}

/** Groupe de citations par catégorie. */
export interface CitationCategoryGroup {
  category: string
  items: CitationItem[]
  total: number
  completed: number
}

export interface CitationsQueryRequest {
  filters: FilterContextInput
}

export interface CitationsPageResponse {
  citations: CitationItem[]
  citations_by_category: CitationCategoryGroup[]
  categories: string[]
  total_count: number
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
  /** Métriques timeseries.16 (Spree + Headshots + Perfect kills). */
  max_killing_spree?: number | null
  headshot_kills?: number | null
  perfect_kills?: number | null
  /** Nom de carte pour étiquettes X compactes (timeseries.14 "Stats par minute"). */
  map_name?: string | null
  map_name_fr?: string | null
  /** Skill rank (CSR/LUSR) — rating brut + type + contexte playlist/saison. */
  skill_rating_value?: number | null
  skill_rating_type?: string | null
  skill_playlist_group?: string | null
  skill_season_id?: string | null
  skill_measurement_remaining?: number | null
  /** Session de rattachement (label sync). */
  session_label?: string | null
  /** MMR équipe — pour l'agrégat MMR moyen par session. */
  team_mmr?: number | null
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
  life_buckets: DistributionBucket[]
  perf_score_buckets: DistributionBucket[]
  personal_score_buckets: DistributionBucket[]
  max_killing_spree_buckets: DistributionBucket[]
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
  performance_score?: number | null
  avg_offensive_conversion?: number | null
  avg_defensive_resistance?: number | null
  // PLAN_COMBAT_PROFILE_WIRING Phase 2
  combat_profile?: CombatProfileBlock | null
  outcomes: { wins: number; losses: number; ties: number; dnf: number }
}

export interface TimeseriesWeaponKill {
  weapon_id: number
  label: string
  kills: number
}

export interface OutcomesPeriodPoint {
  period_label: string
  start_date: string
  wins: number
  losses: number
  ties: number
  dnf: number
}

export interface FirstEventBucket {
  lower_seconds: number
  upper_seconds: number
  first_kills: number
  first_deaths: number
}

export interface FirstEventDistribution {
  buckets: FirstEventBucket[]
  mean_first_kill_seconds?: number | null
  mean_first_death_seconds?: number | null
}

/** Ligne du heatmap d'intensité solo (1 match × 10 phases normalisées). */
export interface IntensityMatchRow {
  match_id: string
  label: string
  phases: number[]
}

/** Agrégat par session/semaine/mois (chart "Performance solo par session"). */
export interface SoloSessionPerfPoint {
  session_label: string
  started_at_utc: string
  match_count: number
  wins: number
  win_rate: number
  perf_avg?: number | null
  team_mmr_avg?: number | null
}

/** Bloc avec granularité auto-adaptative + points. */
export interface SoloSessionPerfBlock {
  /** "session" | "week" | "month" — choisie côté serveur selon densité. */
  granularity: 'session' | 'week' | 'month'
  points: SoloSessionPerfPoint[]
}

export interface TimeseriesPageResponse {
  total_matches: number
  match_rows: TimeseriesMatchRow[]
  summary_tab: TimeseriesSummaryTab
  cumul_tab: TimeseriesCumulTab
  intensity_tab: TimeseriesIntensityTab
  distributions_tab: TimeseriesDistributionsTab
  top_weapons: TimeseriesWeaponKill[]
  outcomes_over_time: OutcomesPeriodPoint[]
  map_breakdown: MapBreakdownRow[]
  first_events?: FirstEventDistribution | null
  intensity_rows?: IntensityMatchRow[] | null
  solo_session_perf?: SoloSessionPerfBlock | null
  /** Alimente <SessionBriefing> en haut de la page (mode solo). Nil si aucun match. */
  briefing_kpis?: KPIStats
}

// ---------------------------------------------------------------------------
// Session Compare (Slice 3C)
// ---------------------------------------------------------------------------

/** Point de données par match pour les charts de progression (K/D, cumul, précision). */
export interface SessionMatchPoint {
  index: number
  kd: number
  cumulative: number
  accuracy: number | null
  perf_score?: number | null
  skill_rating?: number | null
  engagement_score?: number | null
}

/** Ligne du tableau par carte. */
export interface SessionCompareMapRow {
  map_name: string
  a_matches: number
  a_wins: number
  a_losses: number
  b_matches: number
  b_wins: number
  b_losses: number
}

/** Ligne du tableau par mode. */
export interface SessionCompareModeRow {
  mode_name: string
  a_matches: number
  a_wins: number
  b_matches: number
  b_wins: number
}

/** Axe du profil de participation 6 axes, normalisé 0..100. */
export interface SessionParticipationAxis {
  name: string  // "combat" | "survival" | "support" | "score" | "objective" | "impact"
  value: number // 0..100
}

export interface SessionCompareEntry {
  session_label: string
  start_time: string | null
  end_time: string | null
  total_matches: number
  wins: number
  losses: number
  kda: number | null
  performance_score: number | null
  // Métriques dérivées (mêmes helpers backend que compare_metrics).
  win_rate: number
  kdr: number
  kills_per_match: number
  // Stats du radar de frags — AGRÉGATS DE SESSION : max spree atteint + totaux session.
  max_killing_spree?: number | null
  total_headshot_kills?: number | null
  total_perfect_kills?: number | null
  with_friends: boolean
  dominant_category: string | null
  // PLAN_COMBAT_PROFILE_WIRING Phase 3
  avg_oc?: number | null
  avg_dr?: number | null
  /** Dégâts moyens par frag / par mort sur la session. Nil si dénominateur nul. */
  dmg_per_kill?: number | null
  dmg_per_death?: number | null
  // Engagement absolu moyen (pace_joueur/pace_lobby ; 1.0 = rythme lobby).
  avg_pace_ratio?: number | null
  /** Série de points par match pour les charts de progression. */
  match_series?: SessionMatchPoint[]
  /** Dernier skill rating de la session (LUSR ou CSR). */
  last_skill_rating?: number | null
  skill_rating_type?: string | null   // "csr" | "lusr" | ""
  skill_rating_delta?: number | null  // last − first
  /** MMR moyen de la session. */
  avg_team_mmr?: number | null
  avg_enemy_mmr?: number | null
  /** Durée de vie moyenne sur la session (secondes). */
  avg_life_seconds?: number | null
  /** Précision moyenne de la session (0..1) — multipliée par 100 à l'affichage. */
  avg_accuracy?: number | null
  /** Profil de participation 6 axes (0..100). */
  participation?: SessionParticipationAxis[]
  /** Historique des matchs de la session (chronologique). */
  matches?: SessionDetailMatchRow[]
  /** Meilleur et pire match par performance score. */
  best_match?: SessionDetailMatchRow | null
  worst_match?: SessionDetailMatchRow | null
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
  maps_table: SessionCompareMapRow[]
  modes_table: SessionCompareModeRow[]
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
  // Dégâts infligés / subis du match (barre composite par match).
  damage_dealt?: number | null
  damage_taken?: number | null
  // Placement (rang API = "Rang" du scoreboard) + taille du lobby à la fin
  // (present_at_completion, bots inclus) — pour le breakdown des placements.
  placement?: number | null
  lobby_size?: number | null
  // Champs enrichis (Phase 3) pour le tableau détail.
  map_name?: string
  duration_seconds?: number | null
  team_mmr?: number | null
  enemy_mmr?: number | null
  delta_mmr?: number | null
  perf_tier?: number
  skill_rating_type?: string
  skill_rating_value?: number | null
  skill_rating_delta?: number | null
  /** Proba de victoire pré-match de l'équipe (LUSR v2, 0..1). Absente pré-v2 / non-LUSR. */
  expected_win_prob?: number | null
  /** Libellé du palier ("Or III", "Diamant V"…), construit côté backend comme l'Explorer.
   *  La colonne "Rang" affiche ça (pas la valeur brute). Nil si non rankée / placement. */
  skill_tier_label?: string | null
  /** Progression de placement (X/Y). Si présents, la colonne "Rang" affiche "X/Y"
   *  à la place du palier (comme l'Explorer). */
  placement_done?: number | null
  placement_total?: number | null
  /** Mode normalisé + traduit (comme l'Explorer). */
  mode_ui?: string
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
  /** Locale ("fr" | "en") pour la résolution FR/EN des cartes/modes/playlists. */
  locale?: string
}

export interface SessionPageResponse {
  current_session: SessionCompareEntry | null
  available_sessions: string[]
  matches: SessionDetailMatchRow[]
  suggested_compare: SessionCompareSuggestion | null
  compare_enabled: boolean
  compare_session: SessionCompareEntry | null
  // Champs P3 (drawer compare) : peuvent être absents en cas de payload legacy.
  compare_matches?: SessionDetailMatchRow[]
  compare_metrics: SessionCompareMetricRow[]
  previous_session_label?: string | null
  next_session_label?: string | null
}

// ─── Sprint 54-C : Compare joueur vs joueur ───────────────────────────────────

export interface NormalizedPlayerStats {
  xuid: string
  gamertag: string
  title_slug: string
  is_local: boolean
  matches: number
  win_rate: number
  kda: number
  kdr: number
  kills_per_game: number
  deaths_per_game: number
  assists_per_game: number
  accuracy: number
  damage_per_game: number
  // Phase 2 — métriques enrichies
  damage_taken_per_game: number
  perfect_kills_per_game: number
  max_killing_spree: number
  avg_life_secs: number
  headshot_kills_per_game: number
  perf_ath: number
  lusr_ath: number
  career_rank: number
  /** Titre localisé du rang carrière ("Général Platine VI"). Vide si rang inconnu. */
  career_rank_label?: string
  /** Meilleur CSR courant (saison en cours), récupéré en live. 0 si non classé. */
  highest_csr?: number
  /** Libellé tier du meilleur CSR courant ("Platine IV"). */
  highest_csr_label?: string
  /** Meilleur CSR de tous les temps. 0 si aucun. */
  highest_csr_all_time?: number
  /** Libellé tier du meilleur CSR all-time ("Onyx"). */
  highest_csr_all_time_label?: string
  /** Temps de jeu cumulé lifetime (s) depuis le service record Waypoint. */
  time_played_seconds?: number
  extended?: Record<string, unknown>
}

export interface CompareMetricRow {
  metric: string
  label_fr: string
  value_a: number
  value_b: number
  /** false = donnée non disponible côté joueur A (ex. ATH non calculé). À afficher en N/A. */
  value_a_available?: boolean
  /** false = donnée non disponible côté joueur B (ex. joueur remote sur une métrique locale-only). À afficher en N/A. */
  value_b_available?: boolean
  delta: number | null
  winner: 'a' | 'b' | 'tie' | null
  /** true = valeur basse meilleure (deaths_per_game, rendement, damage_taken_per_game). Sert au calcul du top des 3 en mode miroir. */
  less_is_better?: boolean
  sample_size_b?: number
  /** Libellé d'affichage prêt-à-rendre (rang carrière → "Général Platine VI", CSR → "Or III"). Sinon formate value_a/value_b. */
  display_a?: string | null
  display_b?: string | null
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
  /** Badges narratifs ally_plus / tough_enemy / ordinal pour joueur B (best-effort). */
  encounter_badges?: MatchEncounterBadge[]
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
  title_slug?: string
  season?: string
  playlist?: string
  csr_value: number
  tier: string
  sub_tier: number
  is_local: boolean
  // Catégories de stats (vides pour csr-world)
  category?: string
  value?: number
  value_formatted?: string
  unit?: string
  matches_played?: number
  // Enrichissement stats mondiales CSR (Phase C/D) — nil tant que le joueur
  // n'est pas backfillé (compteurs bruts + ratios dérivés + comparaison inter-saison).
  match_count?: number | null
  /** Matchs cumulés sur la playlist, toutes saisons jusqu'à celle affichée (colonne "Matchs"). */
  cumulative_match_count?: number | null
  win_count?: number | null
  loss_count?: number | null
  tie_count?: number | null
  dnf_count?: number | null
  kills?: number | null
  deaths?: number | null
  assists?: number | null
  playtime_seconds?: number | null
  medal_count?: number | null
  win_rate?: number | null
  kda?: number | null // somme native brute du KDA Halo
  accuracy?: number | null // somme native brute de l'Accuracy (%)
  damage_dealt?: number | null // somme des dégâts infligés
  damage_taken?: number | null // somme des dégâts subis
  kills_per_min?: number | null
  prev_season_id?: string | null
  prev_win_rate?: number | null
  prev_kda?: number | null
  kda_trend?: 'up' | 'down' | 'stable' | null
  win_rate_trend?: 'up' | 'down' | 'stable' | null
  rank_delta?: number | null
}

export interface LeaderboardRequest {
  category?: string
  season_id?: string
  playlist_id?: string
  limit?: number
}

export interface LeaderboardResponse {
  entries: LeaderboardEntry[]
  category: string
  season_id: string
  playlist_id: string
  title_slug: string
  total: number
}

export interface LeaderboardCatalogRef {
  id: string
  display_name: string
  /** Saison: stats détaillées disponibles (false = classement CSR seul, saison archivée). Toujours false pour les playlists. */
  enriched: boolean
}

export interface LeaderboardCatalog {
  seasons: LeaderboardCatalogRef[]
  playlists: LeaderboardCatalogRef[]
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
  group_id?: string
}

// ---------------------------------------------------------------------------
// Groupes / familles (accès mutuel aux données)
// ---------------------------------------------------------------------------

export type GroupRole = 'owner' | 'member'

export interface GroupMember {
  xuid: string
  gamertag: string
  role: GroupRole
  joined_at: string
}

export interface Group {
  id: string
  name: string
  owner_xuid: string
  members: GroupMember[]
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// Watcher présence Xbox RTA
// ---------------------------------------------------------------------------

export interface WatcherLastSeen {
  /** Timestamp RFC3339 UTC (ex: "2026-05-25T20:00:36Z") */
  timestamp: string
  /** Nom du jeu (ex: "Halo Infinite") */
  title_name: string
  /** Title ID Xbox (optionnel) */
  title_id?: string
}

export interface WatcherPlayerStatus {
  gamertag: string
  xuid: string
  /** État FSM watcher : "Idle" / "Watching" / "Syncing" / "Cooling" */
  state: string
  /** État Xbox brut : "Online" / "Away" / "Offline" — vide si aucun event reçu */
  presence_state?: string
  in_game: boolean
  state_since: string
  state_duration: string
  cooldown_left?: string
  subscribe_error?: string
  /** Dernière activité connue Xbox (snapshot Offline). Renseigné par le REST poll. */
  last_seen?: WatcherLastSeen
  /** RFC3339 UTC du dernier event présence reçu (chaque poll REST réussi). Vivacité du flux. */
  last_event_at?: string
}

export interface WatcherStatusResponse {
  daemon_running: boolean
  rta_connected: boolean
  token_valid: boolean
  token_expires_at?: string
  token_gamertag?: string
  subscribed_players: string[]
  players: WatcherPlayerStatus[]
  /** RFC3339 UTC du dernier event reçu tous joueurs confondus. Témoin global de vivacité du daemon. */
  last_event_at?: string
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
  /** Identifiant Xbox numérique du titre source (ex: "1144039928" pour Halo Infinite). Vide pour l'ancien data. */
  xbox_title_id?: string
  /** Catégorie produit issue du mapping statique par titre. Absent si le titre n'a pas de mapping. */
  category?: 'multiplayer' | 'campaign' | 'other'
}

export interface AchievementsPageResponse {
  summary: AchievementsSummary
  achievements: AchievementEntry[]
}

// ---------------------------------------------------------------------------
// Backup (pkg/duckdbbackup)
// ---------------------------------------------------------------------------

export interface IntegrityResult {
  ok: boolean
  detail?: string      // premier message d'erreur si !ok
  checked_at: string   // ISO 8601 UTC
}

export interface BackupConfig {
  interval: string
  keep_daily: number
  keep_weekly: number
  keep_monthly: number
}

export interface BackupStatusResponse {
  enabled: boolean
  available: boolean
  last_backup_at?: string   // ISO 8601, absent si jamais sauvegardé
  last_snapshot_id?: string
  last_exported?: string[]
  last_duration_ms?: number
  integrity_checks?: Record<string, IntegrityResult>
  config?: BackupConfig     // absent quand le scheduler est nil (backup non configuré)
}

export interface BackupRunResult {
  snapshot_id?: string
  skipped: boolean
  exported?: string[]
  duration_ms?: number
}

// ─── Admin — Intégrité des données (invariants sync) ─────────────────────────
// Miroir de domain.AdminInvariantsResponse (GET /admin/invariants).

export interface AdminInvariantViolation {
  key: string
  severity: 'fail' | 'warn'
  count: number
  sample: string[] | null
  description: string
}

export interface AdminPlayerInvariantsReport {
  player_slug: string
  gamertag: string
  xuid: string
  check_error?: string
  violations: AdminInvariantViolation[]
  fail_count: number
  warn_count: number
}

export interface AdminInvariantsResponse {
  title_slug: string
  generated_at: string
  reports: AdminPlayerInvariantsReport[]
  /** Invariants globaux (shared DB) — exécutés une fois par run, pas par joueur. */
  shared_violations: AdminInvariantViolation[]
  shared_fail_count: number
  shared_warn_count: number
  shared_check_error?: string
}

// ─── Admin — Contention DB (B-swap shared) ───────────────────────────────────
// Miroir de domain.DBContentionResponse (GET /admin/db-contention).

export interface DBContentionResponse {
  generated_at: string
  state: string
  swaps: number
  avg_acquire_ms: number
  avg_release_ms: number
  drain_ms_total: number
  reads_rejected: number
  readers_in_use: number
  swap_failures: number
  /** Blocage lecteurs (drain + maintien RW + reopen) — la durée la plus représentative du stall. */
  avg_blocked_ms: number
  max_blocked_ms: number
}

// ─── Admin — Santé des tokens (MSAL / XSTS / Refresh) ────────────────────────
// Miroir de domain.TokenHealthResponse (GET /admin/token-health).

export type TokenStatus = 'ok' | 'expiring' | 'expired' | 'absent' | 'reauth'

export interface PlayerTokenHealth {
  player_slug: string
  gamertag: string
  xuid: string
  refresh: TokenStatus
  msal: TokenStatus
  xsts: TokenStatus
  xsts_expires_at?: string
  oauth_expires_at?: string
  updated_at?: string
  load_error?: string
  /** Dernier échec OAuth permanent ("config" | "revoked"), vide si aucun. */
  last_auth_error_class?: 'config' | 'revoked' | 'transient' | ''
  last_auth_error?: string
  /** RFC3339 */
  last_auth_error_at?: string
  /** Source de credentials au dernier scan du pool (watcher_* = store canonique, sinon dette ADR-0023). */
  credential_source?: string
}

export interface TokenHealthResponse {
  generated_at: string
  players: PlayerTokenHealth[]
  store_unavailable?: boolean
}

// ─── Admin — Dashboard monitoring ─────────────────────────────────────────────
// Miroirs de domain.AdminMonitoringOverview (GET /admin/monitoring/overview),
// de la réponse scheduler (GET /admin/monitoring/scheduler — types du package
// Go scheduler) et de la liste de jobs (GET /admin/monitoring/jobs).

export interface MonitoringSchedulerSummary {
  available: boolean
  /** RFC3339, absent si aucun cycle depuis le boot. */
  last_cycle_at?: string
  interval_minutes?: number
  pool_size?: number
  last_total: number
  last_synced: number
  last_skipped: number
  last_failed: number
  last_duration_ms: number
  /** Joueurs ayant atteint le seuil d'alerte consecutive_zero_inserts. */
  zero_insert_alerts: number
  /** Syncs en vol toutes sources (watcher/HTTP/scheduler) vus par le SyncGate. */
  in_flight_claims: number
}

export interface MonitoringJobsSummary {
  active_count: number
  recent: AsyncJobStatus[]
}

export interface MonitoringDataHealth {
  /** RFC3339 — horodatage du dernier audit complet. */
  ran_at: string
  uuids_raw_count: number
  lying_bits_events: number
  lying_bits_weapon_kills: number
  orphan_xuids: number
  garbage_banner_urls: number
  warnings_total: number
  duration_ms: number
}

export interface MonitoringTokensSummary {
  players: number
  ok: number
  expiring: number
  expired: number
  absent: number
  reauth: number
  with_auth_error: number
}

export interface MonitoringInvariantsSummary {
  /** 0 = jamais couru depuis le boot (gauges expvar). */
  runs_total: number
  fail_last: number
  warn_last: number
}

export interface MonitoringServerInfo {
  uptime_s: number
  started_at: string
  version: string
}

export interface AdminMonitoringOverview {
  title_slug: string
  generated_at: string
  server: MonitoringServerInfo
  scheduler: MonitoringSchedulerSummary
  jobs: MonitoringJobsSummary
  data_health?: MonitoringDataHealth
  tokens?: MonitoringTokensSummary
  tokens_error?: string
  invariants: MonitoringInvariantsSummary
}

/** Durée d'une étape du pipeline post-sync (timeline monitoring P4). */
export interface PostSyncStepTiming {
  step: string
  duration_ms: number
  items: number
}

/** Compteurs du pipeline post-sync (miroir domain.PostSyncResult). */
export interface PostSyncCounters {
  perf_scores_computed: number
  lusr_updated: number
  career_synced: boolean
  views_refreshed: number
  achievements_synced: boolean
  matches_promoted_friends: number
  engagement_scores_computed: number
  engagement_coefs_updated: number
  sessions_assigned: number
  weapon_kills_processed: number
  weapon_kills_no_film: number
  citations_computed: number
  dominance_flags_computed: number
  /** Rattrapés par la convergence (étapes 1.54 / 1.56). */
  converged_events: number
  converged_psa: number
  /** Durée totale du pipeline + détail par étape (timeline). */
  duration_ms: number
  step_timings?: PostSyncStepTiming[]
  fatal_errors?: string[]
}

export type SchedulerOutcome = 'ok' | 'skipped' | 'failed'

export interface SchedulerPlayerOutcome {
  gamertag: string
  xuid: string
  outcome: SchedulerOutcome | ''
  reason: string
  attempted_at: string
  duration_ms: number
  matches_inserted?: number
  matches_skipped?: number
  medals_inserted?: number
  sync_status?: string
  error_count?: number
  first_error?: string
  consecutive_zero_inserts?: number
  post_sync?: PostSyncCounters
  /** Durées post-sync (ms) des derniers cycles de ce joueur (ancien → récent), sparkline de tendance. */
  post_sync_history_ms?: number[]
}

export interface SchedulerGateClaim {
  gamertag: string
  source: string
  age_ms: number
  stale: boolean
}

export interface SchedulerGateSnapshot {
  inflight_watcher: number
  inflight_gate: number
  granted_total: number
  coalesced_total: number
  stale_count: number
  claims?: SchedulerGateClaim[]
}

export interface SchedulerSnapshot {
  last_cycle_at: string
  last_cycle_result?: {
    total: number
    synced: number
    skipped: number
    failed: number
    duration_ns: number
  }
  interval_minutes: number
  pool_size: number
  players: SchedulerPlayerOutcome[]
  gate: SchedulerGateSnapshot
}

export interface SchedulerCycleRecord {
  /** RFC3339 */
  at: string
  trigger: 'tick' | 'manual'
  total: number
  synced: number
  skipped: number
  failed: number
  duration_ms: number
  /** Fenêtre cumulée d'INDISPONIBILITÉ des lectures shared pendant ce cycle
   *  (drain + maintien RW + reopen, B-swap). */
  blocked_ms: number
  /** Swaps RO→RW complets pendant ce cycle. */
  swap_count: number
  /** Lectures rejetées en 503 pendant ce cycle. */
  reads_rejected: number
  /** Temps cumulé d'appels API Halo pendant ce cycle (toutes goroutines). */
  api_ms: number
  /** Temps cumulé d'écriture persist (shared + player) pendant ce cycle. */
  persist_write_ms: number
}

export interface AdminSchedulerStatusResponse {
  available: boolean
  snapshot?: SchedulerSnapshot
  /** Plus récent en premier — ring mémoire, perdu au restart serveur. */
  history: SchedulerCycleRecord[]
  history_since_boot: boolean
  zero_insert_warn_threshold: number
}

export interface AdminJobsResponse {
  generated_at: string
  jobs: AsyncJobStatus[]
}

// ─── Admin — Convergence (backlog d'enrichissement par joueur) ────────────────
// Miroir de domain.AdminConvergenceReport (GET /admin/monitoring/convergence).

export interface PlayerConvergenceReport {
  player_slug: string
  gamertag: string
  xuid: string
  /** Non plafonné (diff complet shared vs player DB). */
  missing_enrichment: number
  /** Plafonnés à `horizon` — afficher « N+ » quand count == horizon. */
  missing_psa: number
  missing_events: number
  missing_weapons: number
  check_error?: string
}

export interface ConvergenceTotalsSinceBoot {
  events_processed: number
  weapons_processed: number
  psa_processed: number
  aliases_upserted: number
}

export interface AdminConvergenceReport {
  title_slug: string
  generated_at: string
  horizon: number
  players: PlayerConvergenceReport[]
  /** Travail rattrapé par la convergence depuis le boot (perdu au restart). */
  totals_since_boot: ConvergenceTotalsSinceBoot
}

// ─── Admin — Performance (agrégats expvar depuis le boot) ─────────────────────
// Miroir de domain.AdminPerfStats (GET /admin/monitoring/perf).

export interface PerfCallStats {
  name: string
  count: number
  sum_ms: number
  avg_ms: number
  max_ms: number
  errors?: number
}

export interface PerfAPIBuckets {
  rate_limited_429: number
  auth: number
  server_5xx: number
  network: number
  other: number
}

export interface AdminPerfStats {
  generated_at: string
  api_calls: PerfCallStats[]
  api_buckets: PerfAPIBuckets
  persist_phases: PerfCallStats[]
  postsync_steps: PerfCallStats[]
  postsync_total: PerfCallStats
  /** Fenêtre d'indispo des lectures shared par swap (count = swaps, sum = indispo cumulée). */
  blocked_window: PerfCallStats
  /** Breakdown des appels API attribuables (match_history, career_rank, csrs) par joueur. */
  api_by_player: PerfPlayerCallStats[]
}

/** Agrégat d'un appel API Halo attribué à un joueur. Miroir de domain.PerfPlayerCallStats. */
export interface PerfPlayerCallStats {
  player: string
  call: string
  count: number
  avg_ms: number
  max_ms: number
  errors: number
}

/** Miroir de domain.AdminErrorStats — logs WARN/ERROR agrégés depuis le boot. */
export interface AdminErrorStats {
  generated_at: string
  buckets: AdminErrorBucket[]
}

/** Une erreur agrégée par (niveau, message). Miroir de domain.AdminErrorBucket. */
export interface AdminErrorBucket {
  level: string
  module?: string
  message: string
  count: number
  first_seen: string
  last_seen: string
  last_detail?: string
}

// NB : les types Watcher (WatcherStatusResponse, WatcherPlayerStatus) existent
// déjà plus haut dans ce fichier (section watcher historique) — le dashboard
// monitoring les réutilise via features/settings/watcher-queries.ts.

// ─── Admin — Qualité données (inconnus + actions de résolution) ───────────────
// Miroirs de domain.AdminDataQualityCounts / Issues / actions.

export interface AdminDataQualityCounts {
  title_slug: string
  generated_at: string
  raw_uuid_playlists: number
  raw_uuid_maps: number
  raw_uuid_pairs: number
  raw_uuid_variants: number
  raw_uuid_total: number
  untranslated_modes: number
  orphan_playlists: number
  orphan_xuids: number
  lying_bits_events: number
  lying_bits_weapons: number
}

export type DataQualityIssueKind =
  | 'raw_uuids'
  | 'untranslated_modes'
  | 'orphan_playlists'
  | 'orphan_xuids'

export interface AdminDataQualityIssue {
  kind: string
  asset_kind?: string
  id: string
  label?: string
  occurrences: number
  last_seen?: string
}

export interface AdminDataQualityIssues {
  title_slug: string
  generated_at: string
  kind: string
  items: AdminDataQualityIssue[]
}

export interface RegistryNamesBackfillResult {
  dry_run: boolean
  playlists_scanned: number
  playlists_fixed: number
  maps_scanned: number
  maps_fixed: number
  pairs_scanned: number
  pairs_fixed: number
  variants_scanned: number
  variants_fixed: number
  total_fixed: number
}

export interface ResolveResult {
  action: 'created' | 'updated' | string
  mode_en?: string
  langs?: string[]
}

export interface AssetTranslationRequest {
  asset_kind: string // playlist | map | pair | game_variant
  asset_id: string
  name_en?: string
  name_fr?: string
}

export interface CatalogRefreshResult {
  playlists: number
  pairs: number
  maps: number
  game_variants: number
}

/** Miroir de domain.LyingBitsResetResult — reset des bits backfill_completed menteurs. */
export interface LyingBitsResetResult {
  dry_run: boolean
  events_bits_cleared: number
  weapons_bits_cleared: number
  events_loaded_cleared: number
  total: number
}

// ─── Admin — Viewer de logs ───────────────────────────────────────────────────
// Miroirs de domain.AdminLogModules / AdminLogTail.

export interface AdminLogModule {
  module: string
  size_bytes: number
  modified_at: string
}

export interface AdminLogModules {
  generated_at: string
  modules: AdminLogModule[]
}

export interface AdminLogEntry {
  time?: string
  level: string
  msg?: string
  module?: string
  request_id?: string
  event_id?: string
  err?: string
  source?: string
  fields?: Record<string, unknown>
  raw?: string
}

export interface AdminLogTail {
  module: string
  generated_at: string
  /** Du plus récent au plus ancien. */
  entries: AdminLogEntry[]
  scanned_bytes: number
  /** Budget de scan épuisé — affiner les filtres pour voir plus ancien. */
  truncated: boolean
}
