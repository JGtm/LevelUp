/**
 * Queries TanStack Query — Match View (Slice 4B) + Last Match (Slice 4C).
 */
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  MatchViewResponse,
  LastMatchResolveRequest,
  LastMatchResolveResponse,
} from '@/lib/api/types'

export function useMatchView(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.matchView(playerSlug, matchId),
    queryFn: () =>
      api.get<MatchViewResponse>(`/players/${playerSlug}/matches/${matchId}`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
  })
}

export function useLastMatchResolve(playerSlug: string) {
  return useMutation({
    mutationFn: (body: LastMatchResolveRequest) =>
      api.post<LastMatchResolveResponse>(
        `/players/${playerSlug}/pages/last-match/resolve`,
        body,
      ),
  })
}
