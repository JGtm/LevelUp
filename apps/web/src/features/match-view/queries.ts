/**
 * Queries TanStack Query — Match View (Slice 4B).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchViewResponse } from '@/lib/api/types'

export function useMatchView(playerSlug: string, matchId: string) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchView(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<MatchViewResponse>(`/players/${playerSlug}/matches/${matchId}`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
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
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useMutation({
    mutationFn: (favorited: boolean) =>
      api.patch<MatchFavoriteResponse>(
        `/players/${playerSlug}/matches/${matchId}/favorite`,
        { favorited },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.matchView(playerSlug, titleSlug, matchId),
      })
    },
  })
}
