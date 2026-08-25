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

// Diagnostic apparence Spartan ID (admin, volet 2) — re-exports du contrat
// OpenAPI (source unique), pas de mirror manuel (Lot G, règle G4).
export type AppearanceDiagnosisResponse = components['schemas']['AppearanceDiagnosisResponse']
export type AppearanceComponentDiagnosis = components['schemas']['AppearanceComponentDiagnosis']

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

export type FeatureFlags = components['schemas']['FeatureFlags']

export type SettingsExcerpt = components['schemas']['SettingsExcerpt']

export type HaloIdentitySummary = components['schemas']['HaloIdentitySummary']

export type TitleSummary = components['schemas']['TitleSummary']

// Migré (réconciliation bucket B 2026-06-18) : openapi.yaml BootstrapResponse
// complété pour matcher la réponse réelle du domaine Go (12 champs ajoutés :
// current_title_slug, available_titles, auth_mode, registration_mode,
// instance_locked, reauth_required, has_password, is_admin, current_username,
// first_launch, oauth_code_flow_enabled, demo_mode). Source de vérité = contrat.
export type BootstrapResponse = components['schemas']['BootstrapResponse']

export type PlayersListResponse = components['schemas']['PlayersListResponse']

// Présence en jeu (GET /presence) — sélecteur de joueur du shell.
// `players` est typé `PlayerPresence[] | null` par le contrat généré (toute
// tranche Go se traduit ainsi) : les consommateurs comblent à la frontière.
export type PresenceSnapshot = components['schemas']['PresenceSnapshot']
export type PlayerPresence = components['schemas']['PlayerPresence']

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

export type LabelValue = components['schemas']['LabelValue']

export type AvailableOptions = components['schemas']['AvailableOptions']

export type SessionOption = components['schemas']['SessionOption']

export type SessionOptions = components['schemas']['SessionOptions']

export type FilterCounts = components['schemas']['FilterCounts']

export type PeriodPresetCount = components['schemas']['PeriodPresetCount']

/** Compte cascade-aware par saison du catalog (kind="season" du TOML).
 *  Sert au folding "+N saisons sans matchs ▾" dans SaisonPill. */
export type SeasonCount = components['schemas']['SeasonCount']

export type FilterContextResolved = components['schemas']['FilterContextResolved']

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

// Le backend a unifié le schéma d'erreur riche « ApiErrorSchema » avec l'ApiError
// auto-émis (Huma H4/H5) : le schéma OpenAPI s'appelle désormais « ApiError ». On
// conserve l'alias local ApiErrorSchema (surface stable pour les consommateurs) en
// le repointant vers le schéma renommé — mêmes champs (code/message/retryable/…).
export type ApiErrorSchema = components['schemas']['ApiError']

export type DeviceFlowStatusResponse = components['schemas']['DeviceFlowStatusResponse']

export interface CreatePlayerProfileRequest {
  gamertag: string
  xuid?: string | null
  profile_mode?: 'xbox' | 'azure_manual'
  // Multi-titre : cible le titre (prime sur le header) + nb de matchs initiaux.
  title_slug?: string
  initial_max_matches?: number
}

export type CreatePlayerProfileResponse = components['schemas']['CreatePlayerProfileResponse']

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
  // Multi-titre : cible la bonne DB (un gamertag peut exister sous plusieurs titres).
  title_slug?: string
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
  /** Base de l'attendu (modèle lobby-anchored v2). `cold_start` → la série
   *  « Joueur attendu » est masquée (pas d'historique exploitable). */
  expected_basis: 'bin' | 'global' | 'cold_start'
  /** Libellé du bin d'intensité quand `expected_basis === 'bin'`
   *  (calme | standard | chaotique). Vide sinon. */
  intensity_bin: string
  /** 2e porte F7 : statut de calibration des coefficients du titre.
   *  `provisional` → mention « calibration provisoire » (badge discret) ; absent
   *  ou `validated` → confiance pleine (rien affiché). */
  calibration?: 'validated' | 'provisional'
  /** 1re porte F7 : suffisance du vecteur de signaux (`full` | `partial`).
   *  Absent si non calculé. */
  signal_basis?: 'full' | 'partial'
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
  /** Durée du match (granularité "match") ou SOMME des durées du bucket
   *  (session/week/month), en secondes. Pondère l'écart d'engagement cumulé. */
  duration_seconds: number
}

/** EngagementTimeseriesResponse — réponse de POST /engagement/timeseries.
 *  - `granularity` : choisie automatiquement selon la densité filtrée.
 *  - `total_matches` : compte total filtré AVANT cap workCap.
 *  - `truncated_to_recent` : si non-null, signale que le compute a été borné
 *    aux N matchs les plus récents (perfs ; au-delà le binning agrège tout
 *    de même mais sur un sous-ensemble). */
export type EngagementTimeseriesResponse = components['schemas']['EngagementTimeseriesResponse']

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
  /** Durée en secondes de chaque match, alignée sur `labels`. Pondère l'écart
   *  d'engagement cumulé par joueur (résidu × durée). */
  durations_seconds: number[]
  players: SquadPlayerEngagementAPI[]
}

/** Un bin d'intensité (tercile de pace_lobby) avec son coef de réponse. */
export interface EngagementIntensityBinAPI {
  bin: string
  lower_bound: number
  upper_bound: number
  coef_lobby: number
  n_matches: number
}

/** Profil engagement par catégorie de mode (modèle lobby-anchored v2). Exposé
 *  par GET /engagement_profile. coef_team_share n'est plus exposé (D5). */
export interface EngagementProfileAPI {
  xuid: string
  gamertag?: string
  mode_category: string
  coef_lobby_share: number
  n_matches: number
  last_updated: string
  bins: EngagementIntensityBinAPI[] | null
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
  // Valeur effective résolue (env > réglage > défaut prod). PATCH persiste le réglage.
  media_delete_source_after_transcode: boolean
  media_tolerance_minutes: number
  media_watcher_enabled: boolean
  media_watcher_debounce_seconds: number
  discord_notifications_enabled: boolean
  discord_webhook_url_present: boolean
  discord_notify_sync: boolean
  discord_notify_backfill: boolean
  discord_notify_new_version: boolean
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
  // --- Rejeu 2D : fenêtre de rétention des artefacts (0 = illimité) ---
  replay_retention_months: number
  // --- Rejeu 2D : où se construit un rejeu. '' = défaut de l'instance
  // (ouvrier en production, ce serveur en développement). ---
  replay_build_location: '' | 'local' | 'worker' | 'off'
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
  // --- Sons d'armes du rejeu 2D (réglages d'instance, page admin) ---
  // Les .wav extraits du jeu sont purs : ces deux pourcentages rejouent côté app ce que
  // le moteur fait à chaque coup. Variation 100 = fourchettes du jeu telles quelles ;
  // distance 0 = son pur, aucun traitement dans le chemin du signal.
  replay_sound_variation_percent: number
  replay_sound_distance_percent: number
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

export type HeroProgress = components['schemas']['HeroProgress']

export type CareerProjections = components['schemas']['CareerProjections']

export interface CareerHistoryPoint {
  recorded_at: string
  rank: number
  current_xp: number
  xp_total: number
}

export type FriendXPHistory = components['schemas']['FriendXPHistory']

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

// V8b (2026-07-07) — les DTO réels servis par les endpoints Career. Le contrat Go
// NE renvoie PAS de `top_matches_preview`/`encounters_preview` sur /pages/career :
// les top matches et encounters sont fournis par leurs endpoints dédiés
// (/top-matches, /encounters), consommés en direct par la page. Les schémas
// canoniques CareerTopMatch/CareerEncounter existaient en surplus dans le contrat
// mais N'étaient PAS les shapes de réponse — d'où des `undefined` silencieux.
export type TopMatchDTO = components['schemas']['TopMatchDTO']

export type EncounterDTO = components['schemas']['EncounterDTO']

// CareerPageResponse reste une interface manuelle : les sous-types view-model
// (CareerSummary, CareerLusrSection, CareerHistoryPoint) sont hand-written et NE
// portent PAS les mêmes noms de schéma que le contrat généré (CareerRankSummary,
// LUSRSummary, XPHistoryPoint) — un ré-export cru romprait les consommateurs LUSR/
// résumé/xp. Le fix V8b se limite à retirer les DEUX champs fantômes
// (top_matches_preview / encounters_preview) absents du CareerPageResponse Go.
export interface CareerPageResponse {
  summary: CareerSummary | null
  hero_progress: HeroProgress | null
  projections: CareerProjections | null
  xp_history: CareerHistoryPoint[]
  lusr: CareerLusrSection | null
  friends_xp_history?: FriendXPHistory[]
}

export type CareerTopMatchesResponse = components['schemas']['CareerTopMatchesResponse']

export type CareerEncountersResponse = components['schemas']['CareerEncountersResponse']

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

export type HighlightExperienceCount = components['schemas']['HighlightExperienceCount']

export type HighlightSeasonCount = components['schemas']['HighlightSeasonCount']

export type HighlightModeCount = components['schemas']['HighlightModeCount']

export type HighlightPlaylistCount = components['schemas']['HighlightPlaylistCount']

// Filtres optionnels passés en query params à GET /pages/career/highlight-matches.
export interface CareerHighlightFilters {
  experience?: 'all' | 'ranked' | 'unranked'
  season_ids?: string[]      // multi-select
  mode_uis?: string[]        // multi-select (= pair_name / mode_ui)
  playlist_names?: string[]  // multi-select
}

// Section "Joueurs les plus croisés (hors amis)" : 10 lignes au format
// MatchEncounterRow (réutilise le tableau Match View > Historique de rencontre).
export type CareerTopEncountersResponse = components['schemas']['CareerTopEncountersResponse']

// Section "Top némésis" / "Top souffre-douleur" : top 10 chacun, ratio
// frags/deaths calculé côté backend.
export type CareerRival = components['schemas']['CareerRival']

export type CareerRivalsResponse = components['schemas']['CareerRivalsResponse']

export type CareerCSRRank = components['schemas']['CareerCSRRank']

export type CareerPlaylistCSR = components['schemas']['CareerPlaylistCSR']

/** Saison CSR sélectionnable dans le menu "Classements" (page Carrière).
 *  Une saison apparaît si le joueur y a des données classées + la saison courante. */
export type CSRSeasonOption = components['schemas']['CSRSeasonOption']

export type CareerCSRResponse = components['schemas']['CareerCSRResponse']

// Page Médailles (sous-page Carrière) — catalogue complet du titre + compteur
// obtenu par joueur (count=0 = jamais obtenue). Ré-exports du contrat OpenAPI
// (source unique) : liste plate triée + groupes par catégorie. Cf. Lot A2.
export type MedalsPageResponse = components['schemas']['MedalsPageResponse']
export type MedalSummaryItem = components['schemas']['MedalSummaryItem']
export type MedalCategoryGroup = components['schemas']['MedalCategoryGroup']

// ---------------------------------------------------------------------------
// Pagination — commun (Slices 3+)
// ---------------------------------------------------------------------------

export type SortSpec = components['schemas']['SortSpec']

export interface PaginationRequest {
  page?: number
  page_size?: number
}

export type PaginationMeta = components['schemas']['PaginationMeta']

export type FreshnessInfo = components['schemas']['FreshnessInfo']

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

export type MatchHistoryQuerySummary = components['schemas']['MatchHistoryQuerySummary']

export type ExportHint = components['schemas']['ExportHint']

export type MatchHistoryPageResponse = components['schemas']['MatchHistoryPageResponse']

export interface MatchHistoryQueryRequest {
  filters?: FilterContextInput
  pagination?: PaginationRequest
  columns?: string[] | null
  include_export_hint?: boolean
  /** §5 plan Squad/Sessions : filtre multi-sessions solo. */
  picked_solo_session_labels?: string[]
}

export type FileTokenResponse = components['schemas']['FileTokenResponse']

// ---------------------------------------------------------------------------
// Explorer (Slice 4)
// ---------------------------------------------------------------------------

export type GamertagSuggestion = components['schemas']['GamertagSuggestion']

export type GamertagSearchResponse = components['schemas']['GamertagSearchResponse']

/**
 * VIEW-MODEL du tableau de matchs — PAS un ré-export du contrat.
 *
 * ExplorerMatchesTable est réutilisé par trois surfaces (Explorer, vue Session,
 * « Matchs marquants » de la Carrière) alimentées par des schémas OpenAPI
 * DIFFÉRENTS. Ce type est leur dénominateur commun côté front ; il n'est donc
 * volontairement PAS dérivé de `components['schemas']['ExplorerMatchesRow']`
 * (vérifié le 2026-08-04, dérivation fidèle impossible) :
 *
 * - `skill_rating_delta` n'appartient pas à `ExplorerMatchesRow` : il vient de
 *   `SessionDetailMatchRow` et n'alimente qu'une colonne « Δ rang » INJECTÉE par
 *   la vue session via `extraColumns`. Undefined côté Explorer.
 * - `map_ui` / `mode_ui` / `playlist_label` sont `string | null` au contrat et
 *   `string` ici : les trois producteurs backend garantissent un libellé résolu
 *   (fallback appliqué côté service), le tableau n'a pas de branche « null ».
 * - les champs optionnels admettent `| null` en plus de `undefined` : le JSON
 *   servi porte `null` là où le générateur ne modélise que l'absence.
 *
 * Toute nouvelle colonne servie par l'API doit être ajoutée AUSSI au schéma
 * correspondant côté Go — ce type ne dispense pas du contrat, il l'unifie.
 */
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
  /** URL de l'image du badge de palier, résolue par l'adaptateur d'assets du TITRE
   *  côté backend. Absente quand le titre n'expose pas de badge, que le match est en
   *  placement ou que le palier est inconnu → la colonne « Rang » retombe alors sur
   *  `skill_tier_label` (texte localisé). Jamais construite côté front. */
  skill_rank_image_url?: string | null
  /** Score PERSONNEL du joueur sur le match. À ne pas confondre avec `score_label`,
   *  qui porte le score d'ÉQUIPE (« 50 - 30 »). */
  personal_score?: number | null
  /** "CSR" (classé officiel) ou "LUSR" (interne LevelUp). Nil si pas de skill rank (PvE, Custom). */
  rating_type?: string | null
  /** Gain/perte de rating du match. NON rendu par les colonnes propres d'ExplorerMatchesTable :
   *  porteur de données pour une colonne « Δ rang » INJECTÉE par un consommateur
   *  (vue session via extraColumns). Reste undefined côté page Explorer. */
  skill_rating_delta?: number | null
  /** Proba de victoire pré-match de l'équipe (LUSR v2, 0..1). Porteur de données
   *  pour une colonne « Pronostic » INJECTÉE par la vue session. Undefined côté Explorer. */
  expected_win_prob?: number | null
  /** Progression placement (X). Si défini avec placement_total, l'UI affiche "X/Y" dans la
   *  cellule Rang à la place du skill_tier_label, ET un badge « En placement » (V72-32,
   *  cf. ExplorerMatchesTable.placement.tsx) dans les cellules Perf/ΔPerf/Note à la place
   *  du "-" quand perf_score/delta_perf/rating_type sont nuls pour la même raison. */
  placement_done?: number | null
  /** Seuil placement (Y). CSR : 5 ou 10 selon saison. LUSR : 10. */
  placement_total?: number | null
  /** Placement de la chaîne de PERFORMANCE (X/Y) — signal DÉDIÉ aux colonnes Perf/ΔPerf,
   *  distinct de placement_done/total (colonne Note/Rang). Renseigné quand la chaîne perf
   *  du match compte moins de Y matchs éligibles → perf_score structurellement absent.
   *  Un match peut avoir une Note LUSR établie ET être en placement perf (chaîne perf
   *  jeune, cas BTB). Cf. ExplorerMatchesTable.placement.tsx. */
  perf_placement_done?: number | null
  perf_placement_total?: number | null
  delta_mmr?: number | null
  team_mmr?: number | null
  enemy_mmr?: number | null
  kda?: number | null
  duration_seconds?: number | null
  /** Match parti en PROLONGATION (durée de jeu au-delà du temps réglementaire de
   *  la variante + marge mesurée). Absent/false si le titre n'a pas de table
   *  réglementaire, si la variante y est inconnue ou si la durée n'est pas
   *  estimable — la pastille n'est alors simplement pas rendue. */
  is_overtime?: boolean
  /** Dépassement réel du temps réglementaire en secondes (tooltip « Prolongation : +X »). */
  overtime_seconds?: number
  match_url?: string
  /** Un artefact de rejeu 2D existe pour ce match → la ligne porte un lien vers la
   *  page de rejeu. Absent/false = rien n'est rendu (jamais de lien vers un 404). */
  has_replay?: boolean
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

export type ExplorerEncounterRow = components['schemas']['ExplorerEncounterRow']

export interface ExplorerMatchesQuerySummary {
  total_matches: number
  selected_match_id: string | null
  // available_experience_types : LabelValue (Label localisé backend, Value FR = clé
  // de filtre intacte). GH6-1, miroir GH5-2 Omnibar.
  available_experience_types?: LabelValue[]
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

export type ExplorerPlayerTarget = components['schemas']['ExplorerPlayerTarget']

export type ExplorerPlayerSummary = components['schemas']['ExplorerPlayerSummary']

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
  /** Présence d'un rejeu 2D : '' (tous) | 'with' | 'without'. Filtré côté Go. */
  replay_scope?: 'with' | 'without' | ''
  match_id_search?: string
  /** Whitelist exacte de match_id (mode Joueur : matchs en commun). */
  match_ids?: string[]
  /** Opt-in du bandeau de briefing (mode Matchs) : le mode Matchs l'envoie à true. */
  include_briefing?: boolean
}

export interface ExplorerPlayerQueryRequest {
  target_gamertag: string
  /** xuid optionnel : court-circuite la résolution gamertag→xuid locale (joueur du Classement). */
  target_xuid?: string
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
export type ExplorerEncounterStats = components['schemas']['ExplorerEncounterStats']

/** Un point de la courbe « écart de frags cumulé » de l'encart adversaire
 *  (cumul directionnel frags − morts + issue du duel). Miroir des cartes
 *  revanche du hub Relations (CumulativeFragGapChart). */
export type ExplorerFragGapPoint = components['schemas']['ExplorerFragGapPoint']

export type ExplorerPlayerQueryResponse = components['schemas']['ExplorerPlayerQueryResponse']

/** Encart "Profil joueur cible" composite (4 sources fetch en parallèle).
 *  Toutes les sous-sections sont nullable : le front masque celles à null
 *  et affiche un hint "Connexion Halo requise" quand auth_available=false. */
export type ExplorerTargetProfile = components['schemas']['ExplorerTargetProfile']

/** Statut par section live de l'encart "Profil joueur cible" (Lot A3, fin de la
 *  dégradation muette) — pourquoi une section est vide/partielle plutôt que
 *  silencieuse. Cf. ExplorerLiveStatusBadge (features/explorer). */
export type ExplorerLiveStatus = components['schemas']['ExplorerLiveStatus']

/** Un statut de section individuel ("ok" | "failed" | "no_auth" | "local_partial"). */
export type ExplorerLiveSectionStatus = ExplorerLiveStatus['identity']

/** ExplorerTargetRecentMatch — un match PvP récent du joueur cible, projeté pour
 *  les graphes profil de combat. Miroir exact du DTO Go (JSON snake_case).
 *  `rank` est null si DNF/non classé (trou dans la courbe placement — ne pas tracer 0). */
export type ExplorerTargetRecentMatch = components['schemas']['ExplorerTargetRecentMatch']

/** Nombre de matchs matchmade joués par le joueur cible sur une saison.
 *  `matches` = total de la saison (mode live = service record filtré par saison ;
 *  mode dégradé sans auth = bucketing local). Le pic de rang CSR de la saison est
 *  exposé (tier + image du badge) quand disponible — rendu au-dessus de la barre. */
export type SeasonMatchCount = components['schemas']['SeasonMatchCount']

/** Stats agrégées du joueur cible sur l'échantillon des matchs en commun.
 *  Ratios en nullable : null signifie "indisponible" (dénominateur nul). */
export type ExplorerTargetSampleStats = components['schemas']['ExplorerTargetSampleStats']

/** Une arme du top armes (cible Explorer) : label localisé + kills. */
export interface ExplorerWeaponKill {
  weapon_id: number
  label_fr: string
  label_en: string
  kills: number
}

export type ExplorerMatchesQueryResponse = components['schemas']['ExplorerMatchesQueryResponse']

// ─── Bandeau de briefing (mode Matchs) — alias des schémas auto-dérivés ────────
export type ExplorerBriefing = components['schemas']['ExplorerBriefing']
export type ExplorerBriefingScope = components['schemas']['ExplorerBriefingScope']
export type ExplorerBriefingPeakRank = components['schemas']['ExplorerBriefingPeakRank']
export type ExplorerBriefingBaseline = components['schemas']['ExplorerBriefingBaseline']
export type ExplorerBriefingDimension = components['schemas']['ExplorerBriefingDimension']
export type ExplorerBriefingDimensionEntry = components['schemas']['ExplorerBriefingDimensionEntry']
export type ExplorerBriefingRanked = components['schemas']['ExplorerBriefingRanked']
export type ExplorerBriefingRankedKind = components['schemas']['ExplorerBriefingRankedKind']
export type ExplorerBriefingContextSplit = components['schemas']['ExplorerBriefingContextSplit']
export type ExplorerBriefingContextGroup = components['schemas']['ExplorerBriefingContextGroup']
export type ExplorerBriefingStreaks = components['schemas']['ExplorerBriefingStreaks']
export type ExplorerBriefingDominance = components['schemas']['ExplorerBriefingDominance']

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
  /** Placement de la chaîne de PERFORMANCE (X/Y) : renseigné (avec
   *  performance_score_relative nul) quand la chaîne du match compte moins de Y matchs
   *  éligibles → aucune perf calculable. La tuile affiche « En placement (X/Y) » au lieu
   *  d'un vide. JAMAIS un 0 fabriqué pour une perf absente. Cf. PlacementPendingNote. */
  perf_placement_done?: number | null
  perf_placement_total?: number | null
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

export type MatchCitationSnippet = components['schemas']['MatchCitationSnippet']

export interface RecentMatchMedal {
  medal_id: number
  name: string
  count: number
  description?: string | null
  image_url: string
  difficulty?: string | null
  // Champs sprite (médailles Halo 5). Absents pour Halo Infinite (PNG image_url).
  sprite_sheet?: string
  sprite_left?: number
  sprite_top?: number
  sprite_width?: number
  sprite_height?: number
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

export type HomeCareerRankSummary = components['schemas']['HomeCareerRankSummary']

export type HomeSkillPeakSummary = components['schemas']['HomeSkillPeakSummary']

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

export type HomeSpartanIdentity = components['schemas']['HomeSpartanIdentity']

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

export type ChallengeItem = components['schemas']['ChallengeItem']

export type ChallengesResponse = components['schemas']['ChallengesResponse']

export type SeasonPassStatus = 'active' | 'in_progress' | 'completed' | 'not_started'

export type SeasonPassItemSummary = components['schemas']['SeasonPassItemSummary']

export type SeasonPassTierSummary = components['schemas']['SeasonPassTierSummary']

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

export type SeasonPassTrackSummary = components['schemas']['SeasonPassTrackSummary']

export type SeasonPassPageResponse = components['schemas']['SeasonPassPageResponse']

// Hub Communauté > Relations — types DÉRIVÉS du contrat OpenAPI (source unique :
// generated.ts), pas d'interface manuelle (garde-fou lint-contract-ratchet).
// Note : les tableaux (relations, badges) sont nullable côté contrat (slices Go) ;
// le service garantit non-nil, les consommateurs appliquent `?? []` / `?.`.
// category = string côté contrat ("ally" | "enemy" | "mixed" en pratique).
export type RelationBadge = components['schemas']['RelationBadge']
export type RelationInsight = components['schemas']['RelationInsight']
export type RelationRef = components['schemas']['RelationRef']
// RelationCSR : snapshot CSR courant de la bête noire (lot relations-G, best-effort).
export type RelationCSR = components['schemas']['RelationCSR']
export type RelationsOverview = components['schemas']['RelationsOverview']
export type RelationsPageResponse = components['schemas']['RelationsPageResponse']

// ── Phase 3a : Moments & Rivalités (sous-endpoint /relations/moments) ────────

// RelationHeatmapCell : une cellule du heatmap agrégé « Quand tu les croises »
// (une relation × une heure). hour : 0..23 (fuseau utilisateur).
export type RelationHeatmapCell = components['schemas']['RelationHeatmapCell']

// RelationDuelEntry : un duel (match commun en ennemi) de la frise revanche.
// outcome : "win" | "loss" | "other".
export type RelationDuelEntry = components['schemas']['RelationDuelEntry']

// RelationRivalry : une carte revanche (bête noire + autres rivaux).
export type RelationRivalry = components['schemas']['RelationRivalry']

// RelationsMomentsResponse : réponse du sous-endpoint Moments & Rivalités.
export type RelationsMomentsResponse = components['schemas']['RelationsMomentsResponse']

// ---------------------------------------------------------------------------
// Escouade / Coéquipiers (Slice 6)
// ---------------------------------------------------------------------------

export type TeammateOption = components['schemas']['TeammateOption']

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

export type TeammateRow = components['schemas']['TeammateRow']

export type TeammatesQueryRequest = components['schemas']['TeammatesQueryRequest']

export type SessionLabelEntry = components['schemas']['SessionLabelEntry']

export type SessionLabelsList = components['schemas']['SessionLabelsList']

export interface SquadTimeseriesPoint {
  period_label: string
  match_count: number
  wins: number
  win_rate: number
  avg_performance: number | null
  avg_mmr: number | null
}

export type MapBreakdownRow = components['schemas']['MapBreakdownRow']

export type SquadMatchSeriesPoint = components['schemas']['SquadMatchSeriesPoint']

/** Une ligne du chart kills par arme teammates.09. */
export interface SquadWeaponBar {
  weapon_id: number
  label: string
  /** Classe d'arme du registre (shoulder/sidearm/heavy/melee/grenade/…). Absente si non
   *  résolue (dont les sentinels grenade/mêlée). Sert au split gun/non-gun (buildSquadFragTools). */
  class?: string
  is_grenade_melee?: boolean
  /** gamertag → kills (joueurs absents = 0). */
  kills_by_player: Record<string, number>
  total_squad: number
}

/** Données du chart teammates.09 — players ordonnés (main puis teammates),
 *  bars triées par TotalSquad ASC (peu utilisées en haut). */
export type SquadWeaponKills = components['schemas']['SquadWeaponKills']

/** Une ligne du comparatif « Précision par rôle » (Escouade) : précision + tirs par joueur,
 *  agrégés PAR RÔLE d'arme (precision/automatic/sniper/…). Shim du schéma OpenAPI (contrat
 *  fidèle : role + accuracy_by_player + shots_fired_by_player + total_shots_squad). */
export type SquadWeaponAccuracyBar = components['schemas']['SquadWeaponAccuracyBar']

/** Données du comparatif « Précision par rôle » multi-joueurs (barres groupées horizontales).
 *  Précision NATIVE Halo 5 ; absent sur Infinite (capability weapon_accuracy). */
export type SquadWeaponAccuracy = components['schemas']['SquadWeaponAccuracy']

/** Breakdown mécaniques natives Halo 5 par coéquipier (barres empilées, Escouade). */
export type SquadKillMechanics = components['schemas']['SquadKillMechanics']
export type SquadKillMechanicBar = components['schemas']['SquadKillMechanicBar']

/** Point d'une série performance (1 par match × joueur) pour teammates.16. */
export type SquadPerformanceSeriesPoint = components['schemas']['SquadPerformanceSeriesPoint']

/** Un axe du radar synergie teammates.06 (value normalisé 0..100, raw debug). */
export type SquadSynergyRadarAxis = components['schemas']['SquadSynergyRadarAxis']

/** Profil radar (1 par joueur, sur les matchs partagés). */
export type SquadSynergyRadarSeries = components['schemas']['SquadSynergyRadarSeries']

/** Ligne du heatmap d'intensité teammates.15. Phases est une matrice 1×10. */
export type SquadIntensityMatchRow = components['schemas']['SquadIntensityMatchRow']

export type SquadIntensityOption = components['schemas']['SquadIntensityOption']

export interface SquadIntensityProfile {
  options: SquadIntensityOption[]
  rows: Record<string, SquadIntensityMatchRow[]>
}

/** Agrégat par joueur pour le chart per-minute teammates.14. */
export type SquadPerMinuteEntry = components['schemas']['SquadPerMinuteEntry']

/** Point agrégé par session pour le chart timeline teammates.04. */
export type SquadSessionPoint = components['schemas']['SquadSessionPoint']

/** Cellule (joueur, carte, perf_avg) du heatmap teammates.03. */
export type SquadMapHeatmapCell = components['schemas']['SquadMapHeatmapCell']

export type SquadMapHeatmap = components['schemas']['SquadMapHeatmap']

export type SquadImpactBadgeCount = components['schemas']['SquadImpactBadgeCount']

export type SquadImpactPlayerSummary = components['schemas']['SquadImpactPlayerSummary']

export type SquadImpactMatchHeader = components['schemas']['SquadImpactMatchHeader']

export type SquadImpactCell = components['schemas']['SquadImpactCell']

/** Données du scoreboard impact teammates.07. */
export type SquadImpactMatrix = components['schemas']['SquadImpactMatrix']

/**
 * Ligne du tableau historique escouade (teammates.11). Une ligne par match
 * unique sur le scope filtré, triée serveur-side par start_time DESC.
 * Pagination assurée côté client (TanStack Table, 20/page).
 */
export type SquadMatchHistoryRow = components['schemas']['SquadMatchHistoryRow']

// Champs sprite (médailles Halo 5) ajoutés à la main en attendant la régénération
// OpenAPI (le Go porte déjà les tags json sprite_*). Optionnels : Halo Infinite
// sert un PNG via image_url et n'a pas de sprite. Cf. AssetMeta (même shim).
export type MedalDigestItem = components['schemas']['MedalDigestItem'] & {
  sprite_sheet?: string
  sprite_left?: number
  sprite_top?: number
  sprite_width?: number
  sprite_height?: number
}

export type MedalDigestEntry = components['schemas']['MedalDigestEntry']

/** Ligne du tableau « qui assiste qui » de l'escouade (assistant → tueur assisté). */
export type SquadAssistPair = components['schemas']['SquadAssistPair']

/**
 * Bloc « assistances » de l'escouade : les paires internes ET la couverture de la
 * mesure (`matches_measured` / `matches_total`). Ré-export DIRECT du contrat, sans
 * réécrire le `pairs: […] | null` : le tableau nullable est la forme réelle du fil.
 */
export type SquadAssistPairs = components['schemas']['SquadAssistPairs']

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
  /** Répartition des frags PAR CLASSE (D8) par gamertag — barres empilées du
   *  sous-chart « Répartition des frags » de teammates.16. */
  frag_classes?: Record<string, FragClassEntry[]>
  weapon_kills?: SquadWeaponKills
  /** Comparatif « Précision par arme » multi-joueurs (barres groupées horizontales).
   *  Précision native Halo 5 ; absent sur Infinite (capability weapon_accuracy). */
  weapon_accuracy?: SquadWeaponAccuracy
  native_kill_mechanics?: SquadKillMechanics
  /** Premiers frag/mort PAR MATCH, une série par joueur de l'escouade (onglet Dynamique). */
  first_blood?: FirstBloodPlayerSeriesDTO[]
  /** Paires (assistant → tueur assisté) INTERNES à l'escouade + couverture de la
   *  mesure. Absent quand aucun match de la sélection n'a d'assistance mesurée. */
  assist_pairs?: SquadAssistPairs
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
  /** Chargements best-effort qui ont échoué : les nombres affichés sont partiels.
   *  Non vide => l'UI doit le signaler (fin des chiffres non reproductibles). */
  data_issues?: DataIssue[]
}

/** Dégradation d'un chargement best-effort. `code` est une clé stable traduite
 *  côté front ; `detail` (ex. gamertag) sert au diagnostic. */
export type DataIssue = components['schemas']['DataIssue']

// ---------------------------------------------------------------------------
// Synthèse (Slice 7)
// ---------------------------------------------------------------------------

export type SynthesisKPIs = components['schemas']['SynthesisKPIs']

export type ComparisonMetricItem = components['schemas']['ComparisonMetricItem']

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

export type SynthesisDetailedStats = components['schemas']['SynthesisDetailedStats']

// PLAN_COMBAT_PROFILE_WIRING — types profil combat 3 axes.
export type CombatStyleOffensive = 'disperse' | 'irregulier' | 'equilibre' | 'precis' | 'chirurgical'
export type CombatStyleDefensive = 'fragile' | 'expose' | 'solide' | 'resistant' | 'inebranlable'
export type CombatStyleActivity = 'passif' | 'discret' | 'mesure' | 'actif' | 'agressif'

// Base = schéma OpenAPI généré (source unique). On raffine les 3 styles en unions
// typées (PLAN_COMBAT_PROFILE_WIRING) et on conserve avg_pace_ratio (engagement
// absolu, live-fetch) tant que l'OpenAPI ne l'a pas régénéré.
//
// Les 3 styles restent `?` (optionnels) SANS `| null` : le DTO Go les émet en
// `*string,omitempty` (jamais sérialisés à null — omitempty droppe nil), donc le
// contrat est `string | undefined`. Garder `| null` ici rendait ce type non
// assignable au KPIStats généré (style_*: string), et les consommateurs testent
// déjà l'absence via `!= null` (couvre null ET undefined).
export type CombatProfileBlock = Omit<
  components['schemas']['CombatProfileBlock'],
  'style_offensive' | 'style_defensive' | 'style_activity'
> & {
  avg_pace_ratio?: number | null
  style_offensive?: CombatStyleOffensive
  style_defensive?: CombatStyleDefensive
  style_activity?: CombatStyleActivity
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
  // Répartition hiérarchique des frags v2 (sunburst classe→rôle) — title-agnostic.
  frag_distribution?: FragDistribution

  // Précision par arme (Halo 5 natif) — toutes les armes tirées, accuracy 0..1.
  // Omis pour les titres qui ne peuplent pas weapon_accuracy (Infinite).
  weapon_accuracy?: SynthesisWeaponAccuracyEntry[]
  // PLAN_COMBAT_PROFILE_WIRING Phase 1
  combat_profile?: CombatProfileBlock | null
  // KPI objectifs (cumul CTF/Zones/Oddball sur le scope) — omis pour un titre sans
  // capability match.objective.stats (Halo 5) ou un scope sans match à objectif.
  objective_stats?: ObjectiveAggregate | null
}

// Cumul des stats objectifs (CTF/Zones/Oddball) sur un scope — partagé Synthèse/Escouade.
export type ObjectiveAggregate = components['schemas']['ObjectiveAggregate']

export type SynthesisWeaponKillEntry = components['schemas']['SynthesisWeaponKillEntry']

// Précision par arme — accuracy en unité 0..1 (le composant multiplie par 100).
export type SynthesisWeaponAccuracyEntry = components['schemas']['SynthesisWeaponAccuracyEntry']

// Répartition hiérarchique des frags v2 (sunburst classe→rôle) — title-agnostic,
// partagé par Synthesis/Match view/Timeseries/Sessions. Cf. domain/frag_distribution.go.
export type FragDistribution = components['schemas']['FragDistribution']
export type FragClassEntry = components['schemas']['FragClassEntry']
export type FragRoleEntry = components['schemas']['FragRoleEntry']

// Sprint 55 D9 — Scope
export type SynthesisScope = components['schemas']['SynthesisScope']

// Sprint 55 D9 — Overview
export type SynthesisOverview = components['schemas']['SynthesisOverview']

export type BestMatchRef = components['schemas']['BestMatchRef']

// Sprint 55 D5 — Highlights
export type SynthesisMatchHighlight = components['schemas']['SynthesisMatchHighlight']

export type SynthesisHighlightsPreview = components['schemas']['SynthesisHighlightsPreview']

// Sprint 55 D7 — Breakdowns
export type SynthesisMapEntry = components['schemas']['SynthesisMapEntry']

export type SynthesisModeEntry = components['schemas']['SynthesisModeEntry']

export type SynthesisBreakdowns = components['schemas']['SynthesisBreakdowns']

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

// Phase D — re-shim : le contrat OpenAPI couvre désormais ces types média (schémas
// auto-dérivés des structs Go au moment de la migration Huma /media/*). Données pures
// (zéro champ calculé côté client) → ré-export direct du contrat.
export type MediaAuthor = components['schemas']['MediaAuthor']
export type MediaMatchLobbyEntry = components['schemas']['MediaMatchLobbyEntry']
export type MediaMatchCandidate = components['schemas']['MediaMatchCandidate']
export type MediaMatchCandidatesResponse = components['schemas']['MediaMatchCandidatesResponse']

export interface MediaAssociateRequest {
  file_path: string
  match_id: string
}

export type MediaAssociateResponse = components['schemas']['MediaAssociateResponse']
export type MediaAuthorsResponse = components['schemas']['MediaAuthorsResponse']
/** Suppression définitive d'un média (item 3.1) — dérivé du contrat. */
export type MediaDeleteResponse = components['schemas']['MediaDeleteResponse']

/**
 * PATCH /media/likes — dérivés du contrat (2026-08-03). Les versions écrites à
 * la main divergeaient du serveur sur deux points : `total_likers` y était
 * optionnel alors que le DTO Go l'émet toujours (pas d'`omitempty`), et la
 * requête portait un `liker_slug` que le front n'a jamais envoyé — le liker est
 * résolu côté serveur depuis la session (handlers.resolveLikerIdentity).
 */
export type MediaLikeRequest = components['schemas']['MediaLikeRequest']
export type MediaLikeResponse = components['schemas']['MediaLikeResponse']

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

export type MatchViewHeader = components['schemas']['MatchViewHeader']

export type MatchViewRank = components['schemas']['MatchViewRank']

// Canonical MatchEvents (PLAN_CANONICAL_MATCH_EVENTS) — timeline d'events
// on-demand (kill-feed / timeline). Schémas auto-dérivés (canonical.MatchEvent*).
export type MatchEventTimeline = components['schemas']['MatchEventTimeline']

export type MatchEvent = components['schemas']['MatchEvent']

export type Vec3 = components['schemas']['Vec3']

// Champs sprite (médailles Halo 5) — shim manuel comme AssetMeta / MedalDigestItem.
export type MatchMedal = components['schemas']['MatchMedal'] & {
  sprite_sheet?: string
  sprite_left?: number
  sprite_top?: number
  sprite_width?: number
  sprite_height?: number
}

export type MatchCitation = components['schemas']['MatchCitation']

export type MatchSummaryKpis = components['schemas']['MatchSummaryKpis']

export type MatchPersonalResult = components['schemas']['MatchPersonalResult']

export type MatchExpectedStats = components['schemas']['MatchExpectedStats']

export type MatchSummaryTab = components['schemas']['MatchSummaryTab']

export interface MatchWeaponKill {
  weapon_id: number
  weapon_label: string
  effective_weapon_id: number | null
  kill_count: number
  /** Axe manipulation de l'arme (registre) — recolore le breakdown par classe (sunburst v2). */
  class?: string
}

export type PlayerWeaponKillRow = components['schemas']['PlayerWeaponKillRow']

// Champs sprite (médailles Halo 5) — shim manuel comme MatchMedal / MedalDigestItem.
// Sans eux, le drawer scoreboard affichait les médailles H5 vides (GH-5a) faute de PNG.
export type PlayerMedalRow = components['schemas']['PlayerMedalRow'] & {
  sprite_sheet?: string
  sprite_left?: number
  sprite_top?: number
  sprite_width?: number
  sprite_height?: number
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
  /**
   * Équipe de l'acteur (le TUEUR sur un event `kill`), résolue côté backend depuis le
   * scoreboard. Absent si l'acteur n'y figure pas. Sert à colorer le nom et l'icône avec
   * la couleur d'IDENTITÉ de l'équipe (Eagle bleu / Cobra rouge), la même que l'en-tête
   * du scoreboard — pas un allié/ennemi binaire.
   */
  actor_team_id?: number | null
  /**
   * L'ARME DU KILL. Peuplée seulement quand la source de dégât du kill est connue ET
   * identifiée sans ambiguïté par le backend. Absente sinon : le feed affiche le kill
   * sans icône, et c'est le repli assumé — jamais l'icône d'une autre arme.
   *
   * `weapon_label` est un nom PROPRE (BR75, Needler), pas un libellé traduit : il ne
   * passe pas par i18n.ts. Vide pour les sources sans nom propre (mêlée, grenade), qui
   * gardent leur icône.
   *
   * `weapon_image_tinted` dit que l'image est un MASQUE à teindre (cf. WeaponIcon) —
   * ne jamais le déduire de la forme de l'URL.
   */
  weapon_key?: string | null
  weapon_label?: string | null
  weapon_image_url?: string | null
  weapon_image_tinted?: boolean | null
  /**
   * L'ASSISTANCE du kill, lue du film — TROIS états qui ne se confondent JAMAIS :
   * absent/'' = ON NE SAIT PAS (aucun kill-event apparié) ; 'none' = MESURÉ, pas
   * d'assistant ; 'named' = assistant nommé (+ parts de dégâts quand elles sont lues).
   * Ne jamais traiter l'absence comme « pas d'assistant » : c'est le mensonge que cette
   * énumération existe pour empêcher.
   *
   * Les parts sont des % ENTIERS, NON bornés à 100 (mesures jusqu'à 228 — dégât
   * excédentaire, hypothèse non établie). Absentes = non mesurées, jamais 0.
   */
  assist_state?: string | null
  assist_gamertag?: string | null
  assist_team_id?: number | null
  killer_damage_pct?: number | null
  assist_damage_pct?: number | null
  /**
   * La VICTIME du kill, jointe côté backend depuis killer_victim_pairs par la clé
   * (tueur, instant) avec garde d'unanimité : deux victimes distinctes sur la même
   * clé (double kill au même millisecond) n'en nomment AUCUNE. Absents quand la
   * paire manque — jamais une victime au hasard.
   */
  victim_xuid?: string | null
  victim_gamertag?: string | null
  victim_team_id?: number | null
  /**
   * L'IDENTITÉ DE LA MÉDAILLE (events `medal` uniquement). medal_name est le nom
   * ANGLAIS lu dans le film (quantité mesurée) ; label/description sont résolus
   * locale-aware côté backend (medal_definitions), medal_image_url est le visuel du
   * référentiel. Résolution absente → seul medal_name voyage : le front l'écrit en
   * toutes lettres, jamais le visuel d'une autre médaille.
   */
  medal_name?: string | null
  medal_name_id?: number | null
  medal_label?: string | null
  medal_description?: string | null
  medal_image_url?: string | null
}

export type MatchTugOfWarBin = components['schemas']['MatchTugOfWarBin']

export type MatchImpactBadge = components['schemas']['MatchImpactBadge']

export type MatchKDTimelinePoint = components['schemas']['MatchKDTimelinePoint']

/** Paire killer→victim agrégée pour le chart match_view.18 (antagonistes). */
export type MatchKillerVictimPair = components['schemas']['MatchKillerVictimPair']

/** Paire (assistant → tueur assisté) agrégée sur le match. */
export type MatchAssistPair = components['schemas']['MatchAssistPair']

/**
 * Bloc « assistances » : les paires ET la portée de leur mesure.
 *
 * Ré-export DIRECT du contrat, sans réécriture du `pairs: […] | null` : le tableau
 * nullable est la forme réelle du fil (huma sérialise toute tranche Go ainsi) et le
 * masquer ferait porter au composant un `undefined` silencieux. `measured_deaths` à 0
 * = « non mesuré » ; `pairs` vide avec `measured_deaths` > 0 = « aucune assistance ».
 */
export type MatchAssistPairs = components['schemas']['MatchAssistPairs']

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
  /**
   * Paires (assistant → tueur assisté) + portée de la mesure. ABSENT quand le match
   * n'a aucune ligne de film : l'UI ne rend alors rien.
   */
  assist_pairs?: MatchAssistPairs
  /** Phase 1 MV2 : 8 rôles narratifs typés via narrative.IdentifyImpactRoles. */
  impact_roles?: MatchViewImpactRole[]
  /** Phase 1 MV2 : cadence intra-match (ChartSeries<ChartPointStacked>). */
  cadence?: MatchViewCadence | null
  /**
   * Répartition hiérarchique des frags v2 (sunburst classe→rôle) du viewer pour ce
   * match. Nil si le viewer n'a aucun kill (le front rend null). Cf. P3.
   */
  frag_distribution?: FragDistribution
}

/** MV2 : rôle narratif attribué (1 entrée par joueur × rôle). */
export type MatchViewImpactRole = components['schemas']['MatchViewImpactRole']

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
  /** Libellé d'équipe localisé fourni par le backend (Halo 5 : « Rouge »/« Red »
   *  depuis team_colors). Absent/vide pour les titres sans référentiel d'équipes
   *  (Halo Infinite) → le front retombe sur resolveTeamName (Eagle/Cobra). */
  team_name?: string | null
  /** Couleur d'identité d'équipe (#RRGGBB) fournie par le backend (Halo 5 : depuis
   *  team_colors). Absente pour Halo Infinite → le front retombe sur la map
   *  TEAM_COLORS_HALO_INFINITE (par team_id), puis sur le token ally/enemy. */
  team_color?: string | null
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
  grenade_kills?: number | null
  // Mécaniques de kill natives Halo 5 (assassinats + compétences spartiate) — null hors h5.
  assassination_kills?: number | null
  ground_pound_kills?: number | null
  shoulder_bash_kills?: number | null
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
  /** True si les expected K/D viennent du modèle local (Halo 5), pas de l'API skill. */
  locally_estimated?: boolean
  weapon_kills?: PlayerWeaponKillRow[]
  /** Médailles gagnées par CE joueur dans ce match (expander scoreboard). */
  medals?: PlayerMedalRow[]
  /** Performance score 0..100 — uniquement pour les joueurs trackés (main + amis). */
  performance_score?: number | null
  /** True si bot dans l'équipe du joueur — uniquement pour les joueurs trackés. */
  had_bot_teammate?: boolean
  /** Skill rank (CSR/LUSR) pour ce match — uniquement pour les joueurs trackés. */
  skill_rank?: MatchScoreboardSkillRank | null
  /** Stats objectifs (CTF/Zones/Oddball) — null hors mode à objectif ou titre non
   *  supporté (capability objective_stats). Seuls les champs du mode joué sont non-nil. */
  objective?: MatchScoreboardObjective | null
}

export type MatchScoreboardSkillRank = components['schemas']['MatchScoreboardSkillRank']

export type MatchScoreboardObjective = components['schemas']['MatchScoreboardObjective']

export type MatchRosterRow = components['schemas']['MatchRosterRow']

export type MatchNemesisRow = components['schemas']['MatchNemesisRow']

export type MatchEncounterRow = components['schemas']['MatchEncounterRow']

export interface MatchTeamTab {
  roster: MatchRosterRow[]
  scoreboard: MatchScoreboardRow[]
  nemesis: MatchNemesisRow[]
  encounters: MatchEncounterRow[]
}

export type AssociatedMediaItem = components['schemas']['AssociatedMediaItem']

export type MatchMediaTab = components['schemas']['MatchMediaTab']

export type MatchCitationsTab = components['schemas']['MatchCitationsTab']

/** Commendation NATIVE (Halo 5) gagnée sur un match — affichée telle quelle. */
export type MatchNativeCommendation = components['schemas']['MatchNativeCommendation']

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
export type MatchNeighbors = components['schemas']['MatchNeighbors']

// ---------------------------------------------------------------------------
// Citations (Slice 2B)
// ---------------------------------------------------------------------------

/** Une citation enrichie avec sa progression par paliers. */
export type CitationItem = components['schemas']['CitationItem']

/** Groupe de citations par catégorie. */
export type CitationCategoryGroup = components['schemas']['CitationCategoryGroup']

export interface CitationsQueryRequest {
  filters: FilterContextInput
}

export type CitationsPageResponse = components['schemas']['CitationsPageResponse']

// Totaux à vie des commendations natives (Halo 5, AXE B).
export type NativeCommendationTotal = components['schemas']['NativeCommendationTotal']
export type NativeCommendationCategoryGroup = components['schemas']['NativeCommendationCategoryGroup']
export type NativeCommendationsTotalsResponse =
  components['schemas']['NativeCommendationsTotalsResponse']

// ---------------------------------------------------------------------------
// Timeseries (Slice 3B)
// ---------------------------------------------------------------------------

/** Point d'une courbe cumulative ou glissante indexée sur les matchs. */
export type CumulativePoint = components['schemas']['CumulativePoint']

/** Bucket d'un histogramme de distribution. */
export type DistributionBucket = components['schemas']['DistributionBucket']

/** Paire (x, y) pour un scatter plot de corrélation. */
export type CorrelationDataPair = components['schemas']['CorrelationDataPair']

/** Point de la heatmap intensité (jour × heure). */
export type IntensityHeatmapPoint = components['schemas']['IntensityHeatmapPoint']

/** Ligne brute par match pour les charts timeline côté frontend. */
export type TimeseriesMatchRow = components['schemas']['TimeseriesMatchRow']

export type TimeseriesSummaryTab = components['schemas']['TimeseriesSummaryTab']

export type TimeseriesCumulTab = components['schemas']['TimeseriesCumulTab']

export type TimeseriesIntensityTab = components['schemas']['TimeseriesIntensityTab']

export type TimeseriesDistributionsTab = components['schemas']['TimeseriesDistributionsTab']

export interface TimeseriesQueryRequest {
  filters: FilterContextInput
}

/** RankDelta — delta de skill rating sur le scope. Miroir Go domain.RankDelta.
 *  Kind = "csr" (classé) ou "lusr" (non classé) ; value = somme signée des
 *  per-match deltas ; count = nb matchs du Kind retenu dans le scope. */
export type RankDelta = components['schemas']['RankDelta']

/** KPIStats — agreges du joueur sur le scope filtre. Miroir Go domain.KPIStats. */
export type KPIStats = components['schemas']['KPIStats']

export type TimeseriesWeaponKill = components['schemas']['TimeseriesWeaponKill']

/** Répartition des frags par type sur la période (donut 1er onglet, Halo 5). */
export type TimeseriesKillTypes = components['schemas']['TimeseriesKillTypes']

export type OutcomesPeriodPoint = components['schemas']['OutcomesPeriodPoint']

/** Premiers frag/mort d'un joueur sur UN match (secondes, null = événement absent). */
export type FirstBloodMatchPointDTO = components['schemas']['FirstBloodMatchPoint']

/** Série « premier frag / première mort » d'un joueur — payload des 3 surfaces
 *  (Escouade/Dynamique, Timeseries, Session). Le suffixe DTO évite la collision
 *  avec le type de props du composant `FirstBloodLanes`. */
export type FirstBloodPlayerSeriesDTO = components['schemas']['FirstBloodPlayerSeries']

/** Ligne du heatmap d'intensité solo (1 match × 10 phases normalisées). */
export type IntensityMatchRow = components['schemas']['IntensityMatchRow']

/** Agrégat par session/semaine/mois (chart "Performance solo par session"). */
export type SoloSessionPerfPoint = components['schemas']['SoloSessionPerfPoint']

/** Bloc avec granularité auto-adaptative + points. */
export type SoloSessionPerfBlock = components['schemas']['SoloSessionPerfBlock']

export type TimeseriesPageResponse = components['schemas']['TimeseriesPageResponse']

// ---------------------------------------------------------------------------
// Session Compare (Slice 3C)
// ---------------------------------------------------------------------------

/** Point de données par match pour les charts de progression (K/D, cumul, précision). */
export type SessionMatchPoint = components['schemas']['SessionMatchPoint']

/** Axe du profil de participation 6 axes, normalisé 0..100. */
export type SessionParticipationAxis = components['schemas']['SessionParticipationAxis']

// Base = schéma OpenAPI généré (source unique). On conserve avg_pace_ratio
// (engagement absolu moyen, pace_joueur/pace_lobby ; 1.0 = rythme lobby) ajouté par
// live-fetch et consommé par SessionCompareEngagement, tant que l'OpenAPI ne l'a pas
// régénéré (le généré porte encore avg_residual_brut).
export type SessionCompareEntry = components['schemas']['SessionCompareEntry'] & {
  avg_pace_ratio?: number | null
}

export type SessionCompareMetricRow = components['schemas']['SessionCompareMetricRow']

export type SessionDetailMatchRow = components['schemas']['SessionDetailMatchRow']

export type SessionCompareSuggestion = components['schemas']['SessionCompareSuggestion']

export interface SessionPageRequest {
  filters?: FilterContextInput
  session_label?: string | null
  compare_session_label?: string | null
  enable_compare?: boolean
  /** Locale ("fr" | "en") pour la résolution FR/EN des cartes/modes/playlists. */
  locale?: string
}

export type SessionPageResponse = components['schemas']['SessionPageResponse']

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

export type CompareMetricRow = components['schemas']['CompareMetricRow']

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

export type MatchPrivacyInfo = components['schemas']['MatchPrivacyInfo']

export type MatchPrivacyWarning = components['schemas']['MatchPrivacyWarning']

// ─── Sprint 54-E : Leaderboard ───────────────────────────────────────────────

export type LeaderboardEntry = components['schemas']['LeaderboardEntry']

export interface LeaderboardRequest {
  category?: string
  season_id?: string
  playlist_id?: string
  limit?: number
}

export type LeaderboardResponse = components['schemas']['LeaderboardResponse']

export type LeaderboardCatalogRef = components['schemas']['LeaderboardCatalogRef']

export type LeaderboardCatalog = components['schemas']['LeaderboardCatalog']

// ---------------------------------------------------------------------------
// Auth locale
// ---------------------------------------------------------------------------

export interface LoginRequest {
  username: string
  password: string
}

export type LoginResponse = components['schemas']['LoginResponse']

export interface RegisterRequest {
  username: string
  password: string
  invite_code?: string
}

export type RegisterResponse = components['schemas']['RegisterResponse']

export type AdminUserSummary = components['schemas']['AdminUserSummary']

export type AdminInviteSummary = components['schemas']['AdminInviteSummary']

// Base = schéma OpenAPI généré (source unique : code/created_by/created_at/
// expires_at/used_at/used_by). On ajoute group_id (rattachement à un groupe,
// live-fetch) tant que l'OpenAPI ne l'a pas régénéré.
export type InviteCode = components['schemas']['InviteCode'] & {
  group_id?: string
}

// ---------------------------------------------------------------------------
// Groupes / familles (accès mutuel aux données)
// ---------------------------------------------------------------------------
// Contrat GÉNÉRÉ depuis Huma (V72-01 / H5 puis H7) : les routes /groups sont typées côté
// Go, avec `role` en enum (owner|member) et `members` non nullable → ré-export direct,
// plus de type manuel à maintenir en parallèle.

export type GroupMember = components['schemas']['GroupMember']
export type Group = components['schemas']['Group']
/** Union dérivée du contrat (pas de littéral en dur : l'enum vit dans domain/group.go). */
export type GroupRole = GroupMember['role']

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

export type WatcherStatusResponse = components['schemas']['WatcherStatusResponse']

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

// AssetMeta — type contractuel régénéré. Les champs sprite (médailles Halo 5) et
// description / description_fr sont désormais portés par le schéma OpenAPI (optionnels :
// vides pour maps/armes), plus besoin de shim manuel.
export type AssetMeta = components['schemas']['AssetMeta']

// ---------------------------------------------------------------------------
// Achievements Xbox (bilingues EN/FR)
// ---------------------------------------------------------------------------

export type AchievementsSummary = components['schemas']['AchievementsSummary']

export type AchievementEntry = components['schemas']['AchievementEntry']

export type AchievementsPageResponse = components['schemas']['AchievementsPageResponse']

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

export type BackupRunResult = components['schemas']['BackupRunResult']

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

export type AdminInvariantsResponse = components['schemas']['AdminInvariantsResponse']

// ─── Admin — Contention DB (B-swap shared) ───────────────────────────────────
// Miroir de domain.DBContentionResponse (GET /admin/db-contention).

export type DBContentionResponse = components['schemas']['DBContentionResponse']

// ─── Admin — Santé des tokens (MSAL / XSTS / Refresh) ────────────────────────
// Miroir de domain.TokenHealthResponse (GET /admin/token-health).

export type TokenStatus = 'ok' | 'expiring' | 'expired' | 'absent' | 'reauth'

export type PlayerTokenHealth = components['schemas']['PlayerTokenHealth']

export type TokenHealthResponse = components['schemas']['TokenHealthResponse']

// ─── Admin — Dashboard monitoring ─────────────────────────────────────────────
// Miroirs de domain.AdminMonitoringOverview (GET /admin/monitoring/overview),
// de la réponse scheduler (GET /admin/monitoring/scheduler — types du package
// Go scheduler) et de la liste de jobs (GET /admin/monitoring/jobs).

export type MonitoringSchedulerSummary = components['schemas']['MonitoringSchedulerSummary']

export type MonitoringJobsSummary = components['schemas']['MonitoringJobsSummary']

export type MonitoringDataHealth = components['schemas']['MonitoringDataHealth']

export type MonitoringTokensSummary = components['schemas']['MonitoringTokensSummary']

export type MonitoringInvariantsSummary = components['schemas']['MonitoringInvariantsSummary']

export type MonitoringServerInfo = components['schemas']['MonitoringServerInfo']

export type AdminMonitoringOverview = components['schemas']['AdminMonitoringOverview']

/** Durée d'une étape du pipeline post-sync (timeline monitoring P4). */
export type PostSyncStepTiming = components['schemas']['PostSyncStepTiming']

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
  /** true = un cycle a tourné depuis ce boot ; false = snapshot réhydraté du
   *  disque (dernier cycle d'avant redémarrage). Combiné à last_cycle_at (vide/
   *  zéro = aucune donnée connue) pour distinguer les trois états côté UI (C1). */
  since_boot: boolean
}

/** Journal des actions globales admin (GET /admin/actions/journal, survit au reboot). */
export type AdminActionJournalEntry = components['schemas']['AdminActionJournalEntry']
export type AdminActionJournalResponse = components['schemas']['AdminActionJournalResponse']

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

export type PlayerConvergenceReport = components['schemas']['PlayerConvergenceReport']

export type ConvergenceTotalsSinceBoot = components['schemas']['ConvergenceTotalsSinceBoot']

export type AdminConvergenceReport = components['schemas']['AdminConvergenceReport']

// ─── Admin — Performance (agrégats expvar depuis le boot) ─────────────────────
// Miroir de domain.AdminPerfStats (GET /admin/monitoring/perf).

export type PerfCallStats = components['schemas']['PerfCallStats']

export type PerfAPIBuckets = components['schemas']['PerfAPIBuckets']

export type AdminPerfStats = components['schemas']['AdminPerfStats']

/** Agrégat d'un appel API Halo attribué à un joueur. Miroir de domain.PerfPlayerCallStats. */
export type PerfPlayerCallStats = components['schemas']['PerfPlayerCallStats']

/** Miroir de domain.AdminErrorStats — logs WARN/ERROR agrégés depuis le boot. */
export type AdminErrorStats = components['schemas']['AdminErrorStats']

/** Une erreur agrégée par (niveau, message). Miroir de domain.AdminErrorBucket. */
export type AdminErrorBucket = components['schemas']['AdminErrorBucket']

/** Détection persistée avec cycle de vie. Miroir de domain.MonitoringDetection. */
export type MonitoringDetection = components['schemas']['MonitoringDetection']

/** Réponse GET /admin/monitoring/detections. Miroir de domain.AdminDetectionsResponse. */
export type AdminDetectionsResponse = components['schemas']['AdminDetectionsResponse']

/** Fraîcheur des données d'un joueur suivi. Miroir de domain.PlayerFreshness. */
export type PlayerFreshness = components['schemas']['PlayerFreshness']

/** Fraîcheur par titre actif. Miroir de domain.TitleFreshnessReport. */
export type TitleFreshnessReport = components['schemas']['TitleFreshnessReport']

/** Réponse GET /admin/monitoring/freshness. Miroir de domain.AdminFreshnessResponse. */
export type AdminFreshnessResponse = components['schemas']['AdminFreshnessResponse']

/** Taille d'une base DuckDB + WAL. Miroir de domain.ResourceDBFile. */
export type ResourceDBFile = components['schemas']['ResourceDBFile']

/** Réponse GET /admin/monitoring/resources. Miroir de domain.AdminResourcesResponse. */
export type AdminResourcesResponse = components['schemas']['AdminResourcesResponse']

/** Statut d'un cron. Miroir de domain.CronStatusEntry. */
export type CronStatusEntry = components['schemas']['CronStatusEntry']

/** Liveness d'une feature. Miroir de domain.FeatureHeartbeat. */
export type FeatureHeartbeat = components['schemas']['FeatureHeartbeat']

/** Réponse GET /admin/monitoring/crons. Miroir de domain.AdminCronsResponse. */
export type AdminCronsResponse = components['schemas']['AdminCronsResponse']

// ─── Admin — File de construction des rejeux + ouvriers ───────────────────────
// Miroirs de domain.BuildQueueJob / BuildQueueWorker / AdminBuildQueueResponse
// (GET /admin/monitoring/build-queue). L'état vit côté serveur : cette vue est
// complète même quand l'ouvrier tourne sur une autre machine.

/** Un job de la file durable de construction. */
export type BuildQueueJob = components['schemas']['BuildQueueJob']

/** Un ouvrier connu de la file (dernier battement, en ligne ou non). */
export type BuildQueueWorker = components['schemas']['BuildQueueWorker']

/** Réponse GET /admin/monitoring/build-queue. */
export type AdminBuildQueueResponse = components['schemas']['AdminBuildQueueResponse']

// NB : les types Watcher (WatcherStatusResponse, WatcherPlayerStatus) existent
// déjà plus haut dans ce fichier (section watcher historique) — le dashboard
// monitoring les réutilise via features/settings/watcher-queries.ts.

// ─── Admin — Qualité données (inconnus + actions de résolution) ───────────────
// Miroirs de domain.AdminDataQualityCounts / Issues / actions.

export type AdminDataQualityCounts = components['schemas']['AdminDataQualityCounts']

export type DataQualityIssueKind =
  | 'raw_uuids'
  | 'untranslated_modes'
  | 'orphan_playlists'
  | 'orphan_xuids'

export type AdminDataQualityIssue = components['schemas']['AdminDataQualityIssue']

export type AdminDataQualityIssues = components['schemas']['AdminDataQualityIssues']

export type RegistryNamesBackfillResult = components['schemas']['RegistryNamesBackfillResult']

export type ResolveResult = components['schemas']['ResolveResult']

export interface AssetTranslationRequest {
  asset_kind: string // playlist | map | pair | game_variant
  asset_id: string
  name_en?: string
  name_fr?: string
}

export type CatalogRefreshResult = components['schemas']['CatalogRefreshResult']

/** Miroir de domain.LyingBitsResetResult — reset des bits backfill_completed menteurs. */
export type LyingBitsResetResult = components['schemas']['LyingBitsResetResult']

// ─── Admin — Viewer de logs ───────────────────────────────────────────────────
// Miroirs de domain.AdminLogModules / AdminLogTail.

export type AdminLogModule = components['schemas']['AdminLogModule']

export type AdminLogModules = components['schemas']['AdminLogModules']

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

export type AdminLogTail = components['schemas']['AdminLogTail']

/**
 * Joueur impliqué dans un événement d'objectif (ex. le scorer d'une capture).
 * Miroir exact du DTO `objective-events` (clés camelCase).
 */
export interface MatchObjectiveEventPlayer {
  xuid: string
  role: string
}

/**
 * Événement d'objectif filmé (CTF flag capture, etc.).
 *
 * Source : `GET /players/{slug}/matches/{matchId}/objective-events`. Miroir
 * exact du DTO backend (camelCase, champs nullables omis du JSON). Pour CTF :
 * objectiveType='flag', eventType='capture', value=1, teamId=0|1 (même
 * numérotation que scoreboard.team_side), players[0].xuid = le scorer.
 */
export interface MatchObjectiveEvent {
  matchId: string
  seq: number
  timeMs?: number
  objectiveType: string
  eventType: string
  teamId?: number
  value?: number
  source: string
  confidence: string
  players: MatchObjectiveEventPlayer[]
}

/**
 * Position joueur keyframe v3 décodée du film (match-level — §N).
 * Miroir exact du DTO `positions` (clés camelCase). x/y/z sont des coordonnées
 * monde Halo ; team vaut -1 (inconnu) quand le clustering spatial n'a pas pu
 * l'attribuer, 0/1 sinon (best-effort, pas d'attribution xuid en v1).
 */
export interface MatchPlayerPosition {
  timeMs: number
  x: number
  y: number
  z: number
  team: number
}


// --- Rejeu 2D (GET /players/{slug}/matches/{matchId}/replay) ---
//
// CES TYPES NE SONT PLUS ÉCRITS À LA MAIN. Depuis que l'endpoint est déclaré en Huma, le
// document de rejeu a un schéma dans `api/openapi.yaml`, donc une définition générée dans
// `generated.ts`. En garder une seconde copie manuscrite, c'est se donner deux vérités qui
// divergeront au premier champ ajouté côté Go — exactement ce que le ratchet
// `tools/lint-contract-ratchet.mjs` interdit.
//
// Les alias ci-dessous existent quand même, et ce n'est pas de la cosmétique : les noms du
// contrat sont génériques (`Track`, `Shot`, `Bounds`…) parce qu'ils vivent dans un espace de
// noms plat partagé par toute l'API. Le préfixe `Replay` dit de quel document ils sont les
// pièces, et évite qu'un `Point` du rejeu soit confondu avec un point de série temporelle.
//
// Artefact pré-construit hors ligne (`cmd/replay-build`). Positions en MÈTRES MONDE : le
// build exige les bornes de la carte (`-map`) et refuse de produire un artefact sans elles,
// ce qui rend le fond de carte figé superposable au rejeu. Le client auto-cadre via `bounds`.
// `points[].t` = index de pas de temps ∈ [0, frameCount).
export type ReplayPoint = components['schemas']['Point']
export type ReplayTrack = components['schemas']['Track']
export type ReplayBounds = components['schemas']['Bounds']
export type ReplayMapObject = components['schemas']['MapObject']
export type ReplaySurface = components['schemas']['Surface']
export type ReplayShot = components['schemas']['Shot']
export type ReplayGrenade = components['schemas']['Grenade']
export type ReplayProjectile = components['schemas']['Projectile']
export type ReplayLoadout = components['schemas']['Loadout']
export type ReplayAmmoSlot = components['schemas']['AmmoSlot']
export type ReplayInventory = components['schemas']['Inventory']
// L'état ACTIF d'un équipement (schéma 7) : épisodes datés par vie, deux familles
// mesurées (`camo`, `overshield`) — cf. equipmentFx.ts pour la lecture côté rendu.
export type ReplayEquipmentEpisode = components['schemas']['EquipmentEpisode']
// La TRACTION de grappin (schéma 8) : fenêtre mesurée [t0, t1] par vie + point
// d'accroche en coordonnées monde — cf. grappleLayer.ts pour le tracé.
export type ReplayGrappleLine = components['schemas']['GrappleLine']
// Une POSE d'équipement (schéma 9) : mur, capteur, ou objet dont la nature n'est pas
// établie (`family: other`). `owner` est le SLOT du poseur (-1 si aucun bipède
// contemporain à moins de 3 m) et `h` son cap de VISÉE à l'instant de la pose, en degrés
// [0,360[ — la même convention que `Point.h`, et JAMAIS l'orientation de l'objet, que le
// film ne porte pas. Cf. equipmentPlacementsLayer.ts pour le rendu.
export type ReplayEquipmentPlacement = components['schemas']['EquipmentPlacement']
// Un SOCLE D'ARME (schéma 11) : la position où une arme de la même famille réapparaît, ses
// apparitions, ses intervalles de présence et son cycle de réapparition QUAND IL EST ÉTABLI
// (`cycle` vaut `null` sinon — jamais un chiffre instable). Donnée de MATCH et non de carte :
// le socle appartient à la carte, l'arme qui y apparaît appartient au match.
export type ReplayWeaponPad = components['schemas']['WeaponPad']
export type ReplayPadPresence = components['schemas']['PadPresence']
export type ReplayPadCycle = components['schemas']['PadCycle']
// Une occupation de socle ACHEVÉE (schéma 11) : le socle s'est vidé quelque part dans
// [tLow, tHigh]. C'est un INTERVALLE et non un instant, et `xuid` vaut TOUJOURS `null` — le
// ramasseur n'est pas publié (oracle mesuré à 79,7 %, contre 90 % exigé).
export type ReplayPadPickup = components['schemas']['PadPickup']
// LE SCORE DANS LE TEMPS (schéma 12) : la courbe des deux équipes et les compteurs des
// joueurs, décodés du film et publiés AUX CHANGEMENTS SEULEMENT — la donnée est une suite de
// PALIERS, pas un échantillonnage régulier. `t` est une frame du document (la même grille que
// `tracks`), `v` la valeur atteinte à cette frame.
//
// DEUX NIVEAUX DE LECTURE, ET ILS NE SE CONFONDENT PAS. `rounds` porte la valeur DANS la
// manche (elle repart de zéro à chaque manche d'un Oddball), `total` le cumul du match.
// Un mode à une seule manche a donc `rounds` de longueur 1 et un `total` qui lui est égal.
//
// UNE ÉQUIPE PEUT N'AVOIR AUCUNE SÉRIE, et c'est une mesure : le camp qui n'a jamais marqué
// n'émet rien (temoin CTF 3-0 : une seule série publiée). Son score vaut zéro partout —
// jamais « inconnu » (cf. teamSeriesFor, lib/replay/scoreTimeline.ts).
export type ReplayScoreTick = components['schemas']['ScoreTick']
export type ReplayScoreRound = components['schemas']['ScoreRound']
export type ReplayScoreSeries = components['schemas']['ScoreSeries']
export type ReplayTeamScore = components['schemas']['TeamScore']
export type ReplayPlayerScore = components['schemas']['PlayerScore']
export type ReplayScoreTimeline = components['schemas']['ScoreTimeline']
// La COUVERTURE du calque de score : par quelle voie l'identité des équipes a été résolue
// (`teamIdentity` : a | b | unresolved), si le mode porte le compteur, si la lecture a été
// tronquée, et le nombre de points publiés. `oracle` dit à quelle grandeur le décodage a été
// confronté (`displayed` = le score affiché en jeu).
export type ReplayScoreCoverage = components['schemas']['ScoreCoverage']
// LA VIE D'UN DRAPEAU de CTF (schéma 14) : une entrée par OBJET (deux drapeaux en CTF), et une
// suite d'intervalles d'état contigus. `team` est l'équipe PROPRIÉTAIRE du drapeau (-1 = carte
// hors du catalogue d'objectifs).
//
// QUATRE ÉTATS, ET LE QUATRIÈME PORTE UN DOUTE MESURÉ. `carried` = un joueur le porte et un fait
// DATÉ a mis fin au portage ; `carried_open` = un joueur l'a pris et RIEN dans le film ne dit
// qu'il l'a lâché (l'intervalle court jusqu'à la fin de l'axe : c'est une borne haute, pas une
// mesure — le contrôle indépendant du marqueur confirme 37/37 des portages fermés et 0/5 des
// ouverts) ; `dropped` = au sol, à l'endroit du dernier porteur ; `home` = sur son socle.
//
// `xuid` est renseigné pour les deux états portés, `null` pour les deux autres. `x`/`y` sont en
// coordonnées monde : pour un état porté c'est le POINT DE PRISE, et la suite se lit sur la
// piste du porteur — le drapeau porté est à la position de son porteur, et republier sa
// trajectoire serait republier celle du joueur.
export type ReplayFlagCarry = components['schemas']['FlagCarry']
export type ReplayFlagSpan = components['schemas']['FlagSpan']
// L'ÉTAT DE CHAQUE ZONE du mode (schéma 16) : qui la tient, depuis quand, et jusqu'à quel niveau
// de jauge elle a été contestée. `zoneRef` est un INDEX dans `mapObjectives.zones` — le calque
// statique servi avec le document —, jamais un nom : la lettre A/B/C affichée en jeu n'existe
// dans aucune donnée décodée.
//
// `owner` vaut `null` quand PERSONNE ne tient la zone, et c'est une mesure (la valeur neutre du
// canal), pas une absence de donnée. `active` marque la zone ACTIVE d'un mode à colline ;
// `progress` est le sommet de la jauge atteint pendant l'intervalle, ramené à [0, 1].
//
// `gauge` (schéma 18) est LA JAUGE DE CAPTURE EN DIRECT : la série datée `[{t, v}]` de la valeur
// de la jauge PENDANT ses rampes (allégée : un point par variation >= 0,02 ou par seconde de
// rampe, rien hors rampe, chaque rampe fermée par son retour à zéro), sur la même échelle que
// `progress`, sur les modes à zones SIMULTANÉES seulement (jamais sur une colline de KOTH, où le
// canal est un compteur de transfert). Le rendu la lit en escalier — la dernière valeur tient
// jusqu'au point suivant, une seconde après le dernier de la série. Absente sur un artefact de schéma
// <= 17 — et le rendu ne dessine alors AUCUN arc : le sommet statique se lisait comme une jauge.
export type ReplayZoneState = components['schemas']['ZoneState']
export type ReplayZoneSpan = components['schemas']['ZoneSpan']
export type ReplayGaugePoint = components['schemas']['GaugePoint']
// La COUVERTURE du calque du drapeau : le verdict de mode et les trois signaux du film qui le
// fondent, les prises de l'oracle, les portages publiés partagés en fermés / ouverts, les rejets
// par cause, le contrôle du marqueur (sur les FERMÉS ; les ouverts ont leur propre compte) et
// les incohérences. Absente = personne n'a lu le film pour ce calque.
export type ReplayFlagCarriesCoverage = components['schemas']['FlagCarriesCoverage']

export type ReplayDocument = components['schemas']['ReplayDocument']

// Le FOND DE CARTE : l'image vue du dessus d'une carte, et le calage qui la pose dans le
// repère monde du rejeu. Le calage voyage AVEC l'image parce qu'une image dont on ignore où
// elle se pose ne se superpose à rien — leçon de la première carte reconstruite, dont le
// calage a dû être retrouvé à la main sur des trajectoires de joueur.
export type ReplayMapBackground = components['schemas']['MapBackground']
export type ReplayMapBackgroundCalibration = components['schemas']['MapBackgroundCalibration']

// Les ZONES NOMMÉES (« callouts ») de la carte du match : polygones monde + libellés
// FR/EN officiels, servies à part du document (résolution par carte au service, comme le
// fond). Absentes = la carte n'en a pas (cas Forge, par construction) — pas un dégradé.
export type ReplayMapCallouts = components['schemas']['MapCalloutsEntry']
export type ReplayCalloutZone = components['schemas']['CalloutZone']

// Les OBJECTIFS STATIQUES du mode joué : zones (boîtes orientées, cylindres) et
// marqueurs ponctuels (apparitions, livraisons, socles), joints par map_id au catalogue
// versionné et servis AVEC le document (`mapObjectives` — rempli à la requête, jamais
// écrit dans l'artefact). Absents = mode sans objectifs statiques (Slayer), carte hors
// catalogue ou map_id vide — jamais un dégradé. `team` = index d'équipe À AFFICHER,
// -1 = neutre (les modes à possession dynamique arrivent déjà neutralisés).
export type ReplayMapObjectives = components['schemas']['MapObjectives']
export type ReplayObjectiveZone = components['schemas']['ObjectiveZoneDTO']
export type ReplayObjectiveMarker = components['schemas']['ObjectiveMarkerDTO']

// Les EMPLACEMENTS DE SOCLE de la carte, croisés avec le match et servis AVEC le document
// (`mapWeaponPads` — rempli à la requête comme `mapObjectives`, jamais écrit dans
// l'artefact). Chaque entrée porte la position du SPAWNER telle que le fichier de carte la
// pose, au centimètre, et `pad` : l'index du socle de `weaponPads` qui la CONFIRME.
//
// SEULS LES EMPLACEMENTS ALLUMÉS ARRIVENT, et c'est une décision produit : le fichier de
// carte pose les socles, le mode les allume (Cliffhanger en porte dix-sept, dix en CTF et
// zéro en Super Fiesta). `catalogN` dit combien la carte en porte au total — ce que le
// calque n'affiche donc pas. Absent = carte hors catalogue, ou aucun socle confirmé : le
// calque retombe alors sur les socles du film seuls.
export type ReplayMapWeaponPads = components['schemas']['MapWeaponPads']
export type ReplayMapWeaponPad = components['schemas']['MapWeaponPadDTO']

// La table d'appariement du film : xuid ET index de slot.
//
// LES DEUX CHAMPS NE SONT PAS INTERCHANGEABLES : le xuid IDENTIFIE, l'index ORDONNE et n'a de
// sens qu'à l'intérieur de ce film. Les événements du film désignent leur auteur par index ;
// c'est cette table qui permet de le traduire en identité sans jamais confondre les deux.
// `name` est le gamertag TEL QUE LE FILM L'ÉCRIT — ce n'est pas une résolution, rien n'est
// allé le chercher ailleurs, donc rien ne peut l'avoir mal apparié.
export type ReplayRosterEntry = components['schemas']['RosterEntry']
