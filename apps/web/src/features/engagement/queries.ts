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
  EngagementProfileAPI,
  EngagementScoreResultAPI,
  EngagementTimeseriesRequest,
  EngagementTimeseriesResponse,
  FilterContextInput,
  SquadEngagementSessionAPI,
} from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

export function useMatchEngagement(playerSlug: string, matchId: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.engagementMatch(playerSlug, titleSlug, matchId),
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
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.engagementProfile(playerSlug, titleSlug),
    queryFn: () =>
      api.get<EngagementProfileAPI[]>(`/players/${playerSlug}/engagement_profile`),
    enabled: !!playerSlug,
    staleTime: 30 * 60 * 1000,
    retry: false,
  })
}

/**
 * useEngagementTimeseries — POST /players/{slug}/engagement/timeseries.
 *
 * Le scope (period / cascade / sessions / match_context) est passé via
 * `filters` pour aligner les paces avec les autres charts de la page
 * Timeseries. `filterHash` participe au queryKey pour invalider le cache
 * dès qu'un filtre bouge (cf. pattern useTimeseriesPage).
 *
 * La réponse est wrappée avec `granularity` adaptative — `points` peut
 * représenter des matchs individuels OU des agrégats session/week/month
 * selon la densité du scope filtré (cf. EngagementTimeseriesResponse).
 */
export function useEngagementTimeseries(
  playerSlug: string,
  filters: FilterContextInput,
  filterHash: string,
  limit: number = 50,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.engagementTimeseries(playerSlug, titleSlug, filterHash, limit),
    queryFn: () =>
      api.post<EngagementTimeseriesResponse>(
        `/players/${playerSlug}/engagement/timeseries`,
        { filters, limit } satisfies EngagementTimeseriesRequest,
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
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const xuids = teammates.map((t) => t.xuid)
  const gamertags = teammates.map((t) => t.gamertag)
  return useQuery({
    queryKey: queryKeys.engagementSquadSession(playerSlug, titleSlug, matchIds, xuids),
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
