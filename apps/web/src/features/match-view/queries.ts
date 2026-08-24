/**
 * Queries TanStack Query — Match View (Slice 4B).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchObjectiveEvent, MatchPlayerPosition, MatchViewResponse } from '@/lib/api/types'

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

/**
 * useMatchObjectiveEvents — événements d'objectif filmés (CTF captures, etc.).
 *
 * Endpoint : GET /players/{slug}/matches/{matchId}/objective-events.
 * Dégradation propre : l'endpoint renvoie 503 (capability_not_supported) pour
 * les titres sans film. `api.get` transforme tout statut non-OK en `ApiError`
 * (throw), donc on désactive `retry` ; les consommateurs traitent l'absence de
 * données via `data ?? []` (la query reste en erreur silencieuse, `data`
 * undefined, sans crasher la page).
 *
 * `enabled` : la donnée n'alimente que les charts de l'onglet Chronologie — la
 * page ne la tire donc que quand cet onglet est actif (défaut `true` pour les
 * appelants qui l'affichent inconditionnellement).
 */
export function useMatchObjectiveEvents(playerSlug: string, matchId: string, enabled = true) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchObjectiveEvents(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<MatchObjectiveEvent[]>(
        `/players/${playerSlug}/matches/${matchId}/objective-events`,
      ),
    enabled: enabled && !!playerSlug && !!matchId,
    staleTime: 10 * 60 * 1000,
    retry: false,
  })
}

/**
 * useMatchPositions — positions joueurs keyframe v3 décodées du film (match-level).
 *
 * Endpoint : GET /players/{slug}/matches/{matchId}/positions.
 * Dégradation propre : 503 (capability_not_supported) pour les titres sans film
 * ou les matchs non backfillés. `api.get` transforme tout statut non-OK en throw,
 * donc `retry: false` ; les consommateurs traitent l'absence via `data ?? []`
 * (la query reste en erreur silencieuse, `data` undefined, sans crasher la page).
 *
 * `enabled` : la heatmap qui consomme ces positions vit dans l'onglet Chronologie —
 * la page ne tire la donnée que quand cet onglet est actif.
 */
export function useMatchPositions(playerSlug: string, matchId: string, enabled = true) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.matchPositions(playerSlug, titleSlug, matchId),
    queryFn: () =>
      api.get<MatchPlayerPosition[]>(
        `/players/${playerSlug}/matches/${matchId}/positions`,
      ),
    enabled: enabled && !!playerSlug && !!matchId,
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
