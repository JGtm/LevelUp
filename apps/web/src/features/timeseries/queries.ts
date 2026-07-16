/**
 * Queries TanStack Query — Timeseries (Slice 3B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  TimeseriesPageResponse,
  TimeseriesQueryRequest,
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
