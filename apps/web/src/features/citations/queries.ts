/**
 * Queries TanStack Query — Citations (Slice 2B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { CitationsPageResponse, CitationsQueryRequest } from '@/lib/api/types'

export function useCitationsPage(
  playerSlug: string,
  request: CitationsQueryRequest,
  filterHash: string,
) {
  return useQuery({
    queryKey: queryKeys.citations(playerSlug, filterHash),
    queryFn: () =>
      api.post<CitationsPageResponse>(`/players/${playerSlug}/pages/citations`, request),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
