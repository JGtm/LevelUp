/**
 * Queries TanStack Query — Match View (Slice 4B).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { MatchEventTimeline, MatchViewResponse } from '@/lib/api/types'

export function useMatchView(playerSlug: string, matchId: string) {
  return useQuery({
    queryKey: queryKeys.matchView(playerSlug, matchId),
    queryFn: () =>
      api.get<MatchViewResponse>(`/players/${playerSlug}/matches/${matchId}`),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
  })
}

/**
 * useMatchEvents — timeline canonique d'events d'un match (kill-feed / timeline),
 * chargée on-demand. `types` filtre par type d'event côté serveur (ex. ['kill']
 * pour un kill-feed) ; vide = tous.
 *
 * Dégradation : un titre sans timeline d'events répond 503 → la query passe en
 * erreur ; le consommateur masque la section (pas de throw remontant à l'UI).
 * Lazy par construction : le hook n'est monté que quand la section est rendue
 * (onglet Détails actif) ; `enabled` couvre le cas slug/matchId manquant.
 */
export function useMatchEvents(
  playerSlug: string,
  matchId: string,
  types?: string[],
) {
  const typesKey = types && types.length > 0 ? types.join(',') : ''
  const qs = typesKey ? `?types=${encodeURIComponent(typesKey)}` : ''
  return useQuery({
    queryKey: queryKeys.matchEvents(playerSlug, matchId, typesKey),
    queryFn: () =>
      api.get<MatchEventTimeline>(
        `/players/${playerSlug}/matches/${matchId}/events${qs}`,
      ),
    enabled: !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
    retry: false,
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
