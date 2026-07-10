/**
 * Queries TanStack Query — Timeseries (Slice 3B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  TimeseriesPageResponse,
  TimeseriesQueryRequest,
  MatchHistoryPageResponse,
  MatchHistoryQueryRequest,
} from '@/lib/api/types'

export function useTimeseriesPage(
  playerSlug: string,
  request: TimeseriesQueryRequest,
  filterHash: string,
) {
  return useQuery({
    queryKey: queryKeys.timeseries(playerSlug, filterHash),
    queryFn: () =>
      api.post<TimeseriesPageResponse>(`/players/${playerSlug}/pages/timeseries`, request),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

/** S56 — charge les 500 derniers matchs pour alimenter CombatYieldTimeseries. */
export function useCombatYieldHistory(playerSlug: string, filterHash: string, filters: MatchHistoryQueryRequest['filters']) {
  return useQuery({
    queryKey: queryKeys.combatYieldHistory(playerSlug, filterHash),
    queryFn: () =>
      api.post<MatchHistoryPageResponse>(
        `/players/${playerSlug}/pages/match-history/query`,
        {
          filters,
          pagination: { page: 1, page_size: 500 },
          columns: ['start_time', 'offensive_conversion', 'defensive_resistance'],
        } satisfies MatchHistoryQueryRequest,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
