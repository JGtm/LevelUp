/**
 * Queries TanStack Query — Historique des parties (Slice 3).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type {
  MatchHistoryPageResponse,
  MatchHistoryQueryRequest,
  FileTokenResponse,
  HomePageResponse,
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

export function useSetMatchFavorite(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ matchId, favorite }: { matchId: string; favorite: boolean }) =>
      api.patch(`/players/${playerSlug}/matches/${matchId}/favorite`, { favorited: favorite }),
    onMutate: async ({ matchId, favorite }) => {
      // Annuler les refetches en cours pour éviter d'écraser l'optimistic update
      await queryClient.cancelQueries({ queryKey: queryKeys.home(playerSlug) })

      const previous = queryClient.getQueryData<HomePageResponse>(queryKeys.home(playerSlug))

      // Mise à jour optimiste du cache home
      queryClient.setQueryData<HomePageResponse>(queryKeys.home(playerSlug), (old) => {
        if (!old) return old
        const updateMatch = (m: HomePageResponse['recent_matches'][number]) =>
          m.match_id === matchId ? { ...m, is_favorite: favorite } : m
        const updatedRecent = old.recent_matches.map(updateMatch)
        const updatedFavorites = favorite
          ? updatedRecent.filter((m) => m.is_favorite)
          : old.favorite_matches.filter((m) => m.match_id !== matchId)
        return {
          ...old,
          recent_matches: updatedRecent,
          favorite_matches: updatedFavorites,
        }
      })

      return { previous }
    },
    onError: (_err, _vars, context) => {
      // Rollback en cas d'erreur
      if (context?.previous) {
        queryClient.setQueryData(queryKeys.home(playerSlug), context.previous)
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.home(playerSlug) })
    },
  })
}
