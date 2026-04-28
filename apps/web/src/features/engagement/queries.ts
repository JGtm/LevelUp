/**
 * Queries TanStack Query — Engagement (Phase 4 + wiring Phase 5).
 *
 * Endpoints :
 *   GET /api/v1/players/{slug}/matches/{match_id}/engagement
 *   GET /api/v1/players/{slug}/engagement_profile
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { EngagementCoefficientAPI, EngagementScoreResultAPI } from '@/lib/api/types'
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
