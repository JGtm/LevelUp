/**
 * Queries TanStack Query — Explorer (Slice 4).
 */
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  ExplorerMatchesQueryRequest,
  ExplorerMatchesQueryResponse,
  ExplorerPlayerQueryRequest,
  ExplorerPlayerQueryResponse,
} from '@/lib/api/types'

export function useExplorerMatches(
  playerSlug: string,
  request: ExplorerMatchesQueryRequest,
  filterHash: string,
) {
  return useQuery({
    queryKey: queryKeys.explorer(playerSlug, filterHash),
    queryFn: () =>
      api.post<ExplorerMatchesQueryResponse>(
        `/players/${playerSlug}/pages/explorer/matches-query`,
        request,
      ),
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
    placeholderData: keepPreviousData,
  })
}

export function useExplorerPlayer(playerSlug: string, request: ExplorerPlayerQueryRequest) {
  return useQuery({
    queryKey: queryKeys.explorerPlayer(playerSlug, request.target_gamertag, request.page ?? 1),
    queryFn: () =>
      api.post<ExplorerPlayerQueryResponse>(
        `/players/${playerSlug}/pages/explorer/player-query`,
        request,
      ),
    enabled: !!playerSlug && !!request.target_gamertag,
    staleTime: 5 * 60 * 1000,
    placeholderData: keepPreviousData,
  })
}
