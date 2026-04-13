/**
 * Queries TanStack Query — Escouade / Coéquipiers (Slice 6).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { TeammatesQueryRequest, TeammatesPageResponse } from '@/lib/api/types'

export function useTeammates(
  playerSlug: string,
  request: TeammatesQueryRequest,
  filterContextHash: string,
) {
  return useQuery({
    queryKey: queryKeys.teammates(playerSlug, filterContextHash),
    queryFn: () =>
      api.post<TeammatesPageResponse>(
        `/players/${playerSlug}/pages/teammates`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
