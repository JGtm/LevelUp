/**
 * Queries TanStack Query — Explorer (Slice 4).
 */
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  ExplorerMatchesQueryRequest,
  ExplorerMatchesQueryResponse,
  ExplorerPlayerQueryRequest,
  ExplorerPlayerQueryResponse,
} from '@/lib/api/types'

/**
 * matchFiltersKeyOf — la part « filtres de match » de la clé de cache Explorer.
 *
 * TOUT FILTRE ENVOYÉ AU SERVEUR DOIT FIGURER ICI. Deux requêtes qui ne diffèrent que par
 * un filtre absent de cette clé se partagent une entrée de cache : le changement de filtre
 * ne déclenche aucun refetch, et le premier résultat empoisonne le second. C'est ce qui
 * est arrivé à `replay_scope`, ajouté au corps de requête (lot A) sans l'être à la clé.
 *
 * Les listes sont TRIÉES avant jointure : l'ordre de sélection de l'utilisateur ne doit pas
 * fabriquer deux clés pour un même filtre.
 *
 * Exportée pour être testée seule — c'est une fonction pure de la requête.
 */
export function matchFiltersKeyOf(request: ExplorerMatchesQueryRequest): string {
  return [
    request.match_start_date ?? '',
    request.match_end_date ?? '',
    [...(request.experience_types ?? [])].sort().join(','),
    [...(request.playlists ?? [])].sort().join(','),
    [...(request.map_names ?? [])].sort().join(','),
    [...(request.mode_names ?? [])].sort().join(','),
    request.squad_scope ?? '',
    request.replay_scope ?? '',
    request.match_id_search ?? '',
    [...(request.match_ids ?? [])].sort().join(','),
  ].join('|')
}

export function useExplorerMatches(
  playerSlug: string,
  request: ExplorerMatchesQueryRequest,
  filterHash: string,
  enabled: boolean = true,
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const perfTiers = request.perf_tiers ?? []
  const skillTiers = request.skill_tiers ?? []
  const rankedContext = request.ranked_context ?? ''
  const outcomeFilter = request.outcome_filter ?? []
  const matchFiltersKey = matchFiltersKeyOf(request)
  return useQuery({
    queryKey: queryKeys.explorer(playerSlug, titleSlug, filterHash, perfTiers, skillTiers, rankedContext, outcomeFilter, matchFiltersKey),
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
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.explorerPlayer(
      playerSlug,
      titleSlug,
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
