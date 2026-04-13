/**
 * Queries TanStack Query — Session Compare (Slice 3C).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { SessionCompareResponse, SessionCompareRequest } from '@/lib/api/types'

export function useSessionComparePage(
  playerSlug: string,
  request: SessionCompareRequest,
  filterHash: string,
  sessionA: string,
  sessionB: string,
) {
  return useQuery({
    queryKey: queryKeys.sessionCompare(playerSlug, filterHash, sessionA, sessionB),
    queryFn: () =>
      api.post<SessionCompareResponse>(
        `/players/${playerSlug}/pages/session-compare`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
