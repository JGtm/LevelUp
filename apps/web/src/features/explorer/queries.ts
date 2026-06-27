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
  enabled: boolean = true,
) {
  const perfTiers = request.perf_tiers ?? []
  const skillTiers = request.skill_tiers ?? []
  const rankedContext = request.ranked_context ?? ''
  const outcomeFilter = request.outcome_filter ?? []
  const sortField = request.sort_field ?? ''
  const sortDir = request.sort_dir ?? ''
  const matchFiltersKey = [
    request.match_start_date ?? '',
    request.match_end_date ?? '',
    [...(request.experience_types ?? [])].sort().join(','),
    [...(request.playlists ?? [])].sort().join(','),
    [...(request.map_names ?? [])].sort().join(','),
    [...(request.mode_names ?? [])].sort().join(','),
    request.squad_scope ?? '',
    request.match_id_search ?? '',
    [...(request.match_ids ?? [])].sort().join(','),
  ].join('|')
  return useQuery({
    queryKey: queryKeys.explorer(playerSlug, filterHash, perfTiers, skillTiers, rankedContext, outcomeFilter, sortField, sortDir, matchFiltersKey),
    queryFn: () =>
      api.post<ExplorerMatchesQueryResponse>(
        `/players/${playerSlug}/pages/explorer/matches-query`,
        request,
      ),
    enabled: !!playerSlug && enabled,
    staleTime: 2 * 60 * 1000,
    placeholderData: keepPreviousData,
  })
}

export function useExplorerPlayer(playerSlug: string, request: ExplorerPlayerQueryRequest) {
  return useQuery({
    queryKey: queryKeys.explorerPlayer(
      playerSlug,
      request.target_gamertag,
      request.target_xuid ?? '',
      request.page ?? 1,
    ),
    queryFn: () =>
      api.post<ExplorerPlayerQueryResponse>(
        `/players/${playerSlug}/pages/explorer/player-query`,
        request,
      ),
    // Activée si on a un gamertag OU un xuid (le Classement transmet le xuid pour
    // un joueur non résoluble localement).
    enabled: !!playerSlug && (!!request.target_gamertag || !!request.target_xuid),
    staleTime: 5 * 60 * 1000,
    placeholderData: keepPreviousData,
  })
}
