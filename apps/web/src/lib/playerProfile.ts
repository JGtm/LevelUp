/**
 * Types + API client pour PlayerProfile V1 (Ascension).
 *
 * Miroir du JSON produit par `apps/go-api/internal/progression/profile/types.go`.
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4-§5.
 */
import { api } from './api/client'

import type { components } from '@/lib/api/generated'
// ─── Sections A1 ────────────────────────────────────────────────────────────

export type ParticipationAxisValue = components['schemas']['ParticipationAxisValue']

export type RadarAxisInsight = components['schemas']['RadarAxisInsight']

// ─── Section A2 ─────────────────────────────────────────────────────────────

export type StyleSignature = components['schemas']['StyleSignature']

export type EngagementSnapshot = components['schemas']['EngagementSnapshot']

// ─── Section B ──────────────────────────────────────────────────────────────

export type SkillRatingSnapshot = components['schemas']['SkillRatingSnapshot']

export type LUSRState = components['schemas']['LUSRState']

export type LUSRComponentBreakdown = components['schemas']['LUSRComponentBreakdown']

export type LOWESSTrend = components['schemas']['LOWESSTrend']

/**
 * Point de la sparkline de tendance LUSR 90 j (DEC-5/D2). `value` est le rating
 * LUSR déjà LISSÉ par LOWESS côté serveur (échelle points, identique à la valeur
 * « pts LUSR » du tier) — jamais le μ brut (DEC-6). `date` = jour UTC YYYY-MM-DD.
 */
export type SkillTrendPoint = components['schemas']['SkillTrendPoint']

// ─── Section C ──────────────────────────────────────────────────────────────

export type ProgressionLeverage = components['schemas']['ProgressionLeverage']

export type SuggestedChallenge = components['schemas']['SuggestedChallenge']

// ─── PlayerProfile complet ──────────────────────────────────────────────────

export type PlayerProfile = components['schemas']['PlayerProfile']

// ─── Campagne V1 §4.5 ──────────────────────────────────────────────────────

export type CampaignStatus = 'active' | 'paused' | 'completed' | 'abandoned'
export type AxisKind = 'radar' | 'lusr_component'

export type ImprovementCampaign = Omit<components['schemas']['ImprovementCampaign'], 'axis_kind' | 'status'> & {
  axis_kind: AxisKind
  status: CampaignStatus
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
export type CampaignHistoryItem = Omit<components['schemas']['CampaignHistoryItem'], 'axis_kind' | 'status'> & {
  axis_kind: AxisKind
  status: CampaignStatus
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
