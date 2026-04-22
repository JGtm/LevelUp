/**
 * Queries TanStack Query — Match View (Slice 4B).
 */
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { MatchViewResponse } from '@/lib/api/types'

export function useMatchView(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.matchView(playerSlug, matchId),
    queryFn: () =>
      api.get<MatchViewResponse>(`/players/${playerSlug}/matches/${matchId}`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
  })
}
