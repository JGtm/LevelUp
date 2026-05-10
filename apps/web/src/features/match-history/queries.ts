/**
 * Queries TanStack Query — Match History mutations.
 *
 * Endpoints :
 *   PATCH /players/{slug}/matches/{matchId}/exclusion  → 204
 *   PATCH /players/{slug}/matches/{matchId}/favorite   → MatchFavoriteResponse
 */
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

interface MatchFavoriteResponse {
  player_slug: string
  match_id: string
  favorited: boolean
}

/**
 * useSetMatchExclusion — marque/démarque un match comme non pertinent.
 * Invalide matchHistory (préfixe joueur) pour retirer le match des listes.
 */
export function useSetMatchExclusion(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ matchId, excluded }: { matchId: string; excluded: boolean }) =>
      api.patch<void>(
        `/players/${playerSlug}/matches/${matchId}/exclusion`,
        { excluded },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['match-history', playerSlug] })
    },
  })
}

/**
 * useSetMatchFavorite — bascule l'état favori d'un match depuis la Home.
 * Invalide home + matchHistory pour refléter les listes "favoris".
 */
export function useSetMatchFavorite(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ matchId, favorite }: { matchId: string; favorite: boolean }) =>
      api.patch<MatchFavoriteResponse>(
        `/players/${playerSlug}/matches/${matchId}/favorite`,
        { favorited: favorite },
      ),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.home(playerSlug) })
      void queryClient.invalidateQueries({ queryKey: ['match-history', playerSlug] })
    },
  })
}
