/**
 * Types + API client pour PlayerProfile V1 (Ascension).
 *
 * Miroir du JSON produit par `apps/go-api/internal/progression/profile/types.go`.
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4-§5.
 */
import { api } from './api/client'

// ─── Sections A1 ────────────────────────────────────────────────────────────

export interface ParticipationAxisValue {
  axis: string // 'combat' | 'survival' | 'support' | 'score' | 'objective' | 'impact'
  value: number // 0..100
  raw?: number
}

export interface RadarAxisInsight {
  axis: string
  value: number
  message?: string // i18n key
}

// ─── Section A2 ─────────────────────────────────────────────────────────────

export interface StyleSignature {
  first_kill_count: number
  first_death_count: number
  fkfd_ratio: number
  style_key?: string // 'opportunistic_finisher' | 'overextended' | 'hyper_engaged' | 'passive'
}

export interface EngagementSnapshot {
  score: number // 0..100
  tier: string // 'low' | 'regular' | 'high' | 'intense'
  matches_per_day_avg: number
  max_gap_days: number
  regularity_coach?: string // i18n key
}

// ─── Section B ──────────────────────────────────────────────────────────────

export interface SkillRatingSnapshot {
  tier_name: string
  tier_name_fr: string
  sub_tier: number
  label: string
  mu: number
  sigma: number
  next_tier_label?: string
  next_tier_mu?: number
  gap_to_next?: number
  progress_ratio?: number
}

export interface LUSRState {
  Mu?: number
  Sigma?: number
  MatchesCount?: number
  LastMatchAt?: string
}

export interface LUSRComponentBreakdown {
  name: string
  weight: number
  current_avg: number
  personal_top_20: number
  target_for_tier: number
  trend: number
}

export interface LOWESSTrend {
  Metric?: string
  Slope?: number
  Window?: number
}

// ─── Section C ──────────────────────────────────────────────────────────────

export interface ProgressionLeverage {
  component: string
  leverage_value: number
  narrative_axes: string[]
  coaching_message: string // i18n key
}

export interface SuggestedChallenge {
  template_id: string
  target_tier: string // 'normal' | 'heroic' | 'legendary' | 'mythic'
  historical_streak: number
  is_arc_step: boolean
  arc_id?: string
  // V2 §3 : hydratés côté backend depuis le template prestige. Permettent
  // à l'UI d'afficher un libellé humain sans charger le catalogue séparément.
  label_fr?: string
  label_en?: string
  description_fr?: string
  description_en?: string
}

// ─── PlayerProfile complet ──────────────────────────────────────────────────

export interface PlayerProfile {
  user_id: string
  title_slug: string
  updated_at: string
  has_enough_data: boolean
  matches_analyzed: number

  // A1
  dominant_role?: string
  secondary_role?: string
  radar_axes?: ParticipationAxisValue[]
  strengths?: RadarAxisInsight[]
  improvement_areas?: RadarAxisInsight[]

  // A2
  style_signature: StyleSignature
  engagement_snap: EngagementSnapshot

  // B
  lusr: LUSRState
  skill_rating: SkillRatingSnapshot
  lusr_components?: LUSRComponentBreakdown[]
  mu_trend: LOWESSTrend

  // C
  leverages?: ProgressionLeverage[]
  suggested_challenges?: SuggestedChallenge[]
}

// ─── Campagne V1 §4.5 ──────────────────────────────────────────────────────

export type CampaignStatus = 'active' | 'paused' | 'completed' | 'abandoned'
export type AxisKind = 'radar' | 'lusr_component'

export interface ImprovementCampaign {
  id: string
  user_id: string
  title_slug: string
  axis: string
  axis_kind: AxisKind
  started_at: string
  ended_at?: string
  status: CampaignStatus
  playlist_group: string
  snapshot_value: number
  snapshot_sample: number
  current_value_raw?: number
  current_value_lowess?: number
  matches_since_start: number
  last_evaluated_at?: string
  mann_whitney_p?: number
  progression_confirmed: boolean
  auto_closure_suggested: boolean
  auto_closure_reason?: string
  linked_challenge_ids?: string[]
}

export interface StartCampaignBody {
  axis: string
  axis_kind: AxisKind
  playlist_group?: string
}

/**
 * Campagne close (historique). DTO dédié servi par GET /campaigns/history —
 * miroir de handlers.campaignHistoryItem. `delta` = final − snapshot (progression
 * sur l'axe), absent si la campagne n'a jamais été évaluée.
 */
export interface CampaignHistoryItem {
  id: string
  axis: string
  axis_kind: AxisKind
  playlist_group: string
  status: CampaignStatus
  started_at: string
  ended_at?: string
  snapshot_value: number
  final_value?: number
  delta?: number
}

// ─── API client ────────────────────────────────────────────────────────────

export const playerProfileApi = {
  getProfile: (playerSlug: string, windowDays = 30) =>
    api.get<PlayerProfile>(
      `/players/${encodeURIComponent(playerSlug)}/profile?window_days=${windowDays}`,
    ),
}

export const campaignApi = {
  start: (playerSlug: string, body: StartCampaignBody) =>
    api.post<ImprovementCampaign>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns`,
      body,
    ),

  getActive: (playerSlug: string) =>
    api.get<ImprovementCampaign | null>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns/active`,
    ),

  /** Campagnes closes (completed/abandoned), les plus récentes d'abord. */
  listEnded: (playerSlug: string) =>
    api.get<{ campaigns: CampaignHistoryItem[]; count: number }>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns/history`,
    ),

  getById: (playerSlug: string, id: string) =>
    api.get<ImprovementCampaign>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns/${encodeURIComponent(id)}`,
    ),

  pause: (playerSlug: string, id: string) =>
    api.post<void>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns/${encodeURIComponent(id)}/pause`,
    ),

  resume: (playerSlug: string, id: string) =>
    api.post<void>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns/${encodeURIComponent(id)}/resume`,
    ),

  close: (playerSlug: string, id: string) =>
    api.post<void>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns/${encodeURIComponent(id)}/close`,
    ),

  abandon: (playerSlug: string, id: string) =>
    api.post<void>(
      `/players/${encodeURIComponent(playerSlug)}/campaigns/${encodeURIComponent(id)}/abandon`,
    ),
}
