/**
 * Queries TanStack Query — Engagement (Phase 4 + wiring Phase 5).
 *
 * Endpoints :
 *   GET /api/v1/players/{slug}/matches/{match_id}/engagement
 *   GET /api/v1/players/{slug}/engagement_profile
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type {
  EngagementCoefficientAPI,
  EngagementMatchSummaryAPI,
  EngagementScoreResultAPI,
  SquadEngagementSessionAPI,
} from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

export function useMatchEngagement(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.engagementMatch(playerSlug, matchId),
    queryFn: () =>
      api.get<EngagementScoreResultAPI>(
        `/players/${playerSlug}/matches/${matchId}/engagement`,
      ),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
    // 503 (engagement_unavailable) ou 422 (pve_not_supported) sont des etats
    // attendus — on ne retry pas ces cas pour eviter le bruit reseau.
    retry: false,
  })
}

export function useEngagementProfile(playerSlug: string) {
  return useQuery({
    queryKey: queryKeys.engagementProfile(playerSlug),
    queryFn: () =>
      api.get<EngagementCoefficientAPI[]>(`/players/${playerSlug}/engagement_profile`),
    enabled: !!playerSlug,
    staleTime: 30 * 60 * 1000,
    retry: false,
  })
}

export function useEngagementTimeseries(playerSlug: string, limit: number = 50) {
  return useQuery({
    queryKey: queryKeys.engagementTimeseries(playerSlug, limit),
    queryFn: () =>
      api.get<EngagementMatchSummaryAPI[]>(
        `/players/${playerSlug}/engagement/timeseries?limit=${limit}`,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

export interface SquadTeammateEntry {
  xuid: string
  gamertag: string
}

export function useSquadEngagementSession(
  playerSlug: string,
  matchIds: string[],
  teammates: SquadTeammateEntry[],
) {
  const xuids = teammates.map((t) => t.xuid)
  const gamertags = teammates.map((t) => t.gamertag)
  return useQuery({
    queryKey: queryKeys.engagementSquadSession(playerSlug, matchIds, xuids),
    queryFn: () => {
      const params = new URLSearchParams()
      if (matchIds.length > 0) params.set('match_ids', matchIds.join(','))
      if (xuids.length > 0) params.set('teammates', xuids.join(','))
      if (gamertags.length > 0) params.set('teammate_gamertags', gamertags.join(','))
      const qs = params.toString()
      return api.get<SquadEngagementSessionAPI>(
        `/players/${playerSlug}/pages/squad/v2/engagement${qs ? '?' + qs : ''}`,
      )
    },
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}
