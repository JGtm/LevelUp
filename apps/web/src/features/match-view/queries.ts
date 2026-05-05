/**
 * Queries TanStack Query — Match View (Slice 4B).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { MatchViewResponse, MatchNeighbors } from '@/lib/api/types'

export function useMatchView(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.matchView(playerSlug, matchId),
    queryFn: () =>
      api.get<MatchViewResponse>(`/players/${playerSlug}/matches/${matchId}`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
  })
}

export function useMatchNeighbors(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.matchNeighbors(playerSlug, matchId),
    queryFn: () =>
      api.get<MatchNeighbors>(`/players/${playerSlug}/matches/${matchId}/neighbors`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 5 * 60 * 1000,
  })
}

interface MatchFavoriteResponse {
  player_slug: string
  match_id: string
  favorited: boolean
}

/**
 * useToggleMatchFavorite — bascule l'état favori d'un match.
 *
 * Endpoint : PATCH /players/{slug}/matches/{matchId}/favorite, body { favorited }.
 * Invalide la query matchView pour refléter le nouvel état dans le header.
 */
export function useToggleMatchFavorite(playerSlug: string, matchId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (favorited: boolean) =>
      api.patch<MatchFavoriteResponse>(
        `/players/${playerSlug}/matches/${matchId}/favorite`,
        { favorited },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.matchView(playerSlug, matchId),
      })
    },
  })
}
