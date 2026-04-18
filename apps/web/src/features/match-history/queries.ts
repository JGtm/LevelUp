/**
 * Queries TanStack Query — Historique des parties (Slice 3).
 */
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  MatchHistoryPageResponse,
  MatchHistoryQueryRequest,
  FileTokenResponse,
} from '@/lib/api/types'

export function useMatchHistory(
  playerSlug: string,
  request: MatchHistoryQueryRequest,
  filterHash: string,
  page: number,
) {
  return useQuery({
    queryKey: queryKeys.matchHistory(playerSlug, filterHash, page),
    queryFn: () =>
      api.post<MatchHistoryPageResponse>(
        `/players/${playerSlug}/pages/match-history/query`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
  })
}

export function useMatchHistoryExport(playerSlug: string) {
  return useMutation({
    mutationFn: (request: MatchHistoryQueryRequest) =>
      api.post<FileTokenResponse>(
        `/players/${playerSlug}/pages/match-history/export`,
        request,
      ),
  })
}

export function useSetMatchExclusion(playerSlug: string) {
  return useMutation({
    mutationFn: ({ matchId, excluded }: { matchId: string; excluded: boolean }) =>
      api.patch(`/players/${playerSlug}/matches/${matchId}/exclusion`, { excluded }),
  })
}
