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
import { useAppShellStore } from '@/stores/appShellStore'
import { patchFavoriteFlag } from '@/lib/home-tiles/homeTiles.logic'

interface SetExclusionVars {
  matchId: string
  excluded: boolean
}

interface MatchFavoriteResponse {
  player_slug: string
  match_id: string
  favorited: boolean
}

/**
 * useSetMatchExclusion — marque/démarque un match comme non pertinent.
 *
 * Côté backend : l'endpoint déclenche un recompute synchrone perf_score + LUSR
 * (cf. sync.MatchRecomputer) → la réponse 204 arrive après plusieurs secondes
 * sur un gros joueur. L'appelant doit s'appuyer sur `isPending` pour bloquer
 * l'UI le temps du recompute.
 *
 * Erreurs typées remontées au caller via `onError` :
 *   - 422 `ranked_not_excludable` → match classé non excluable
 *   - 404 `match_not_found`        → match absent du registre
 *
 * Invalide matchHistory (préfixe joueur) + matchView (match courant) pour
 * refléter immédiatement les nouveaux scores et l'état du flag.
 */
export function useSetMatchExclusion(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ matchId, excluded }: SetExclusionVars) =>
      api.patch<void>(
        `/players/${playerSlug}/matches/${matchId}/exclusion`,
        { excluded },
      ),
    onSuccess: (_, { matchId }) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.matchHistoryAll(playerSlug) })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.matchView(playerSlug, useAppShellStore.getState().currentTitleSlug, matchId),
      })
    },
  })
}

/**
 * useSetMatchFavorite — bascule l'état favori d'un match depuis la Home.
 * La tuile change sur place dans les réponses de l'accueil déjà en cache (tête et pages : aucun
 * rechargement), seule la liste des favoris est rechargée (son contenu change), et l'historique
 * Explorer est invalidé.
 */
export function useSetMatchFavorite(playerSlug: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ matchId, favorite }: { matchId: string; favorite: boolean }) =>
      api.patch<MatchFavoriteResponse>(
        `/players/${playerSlug}/matches/${matchId}/favorite`,
        { favorited: favorite },
      ),
    onSuccess: (_data, { matchId, favorite }) => {
      const { currentTitleSlug, locale } = useAppShellStore.getState()
      queryClient.setQueriesData({ queryKey: queryKeys.homeAll(playerSlug) }, (old) =>
        patchFavoriteFlag(old, matchId, favorite),
      )
      void queryClient.invalidateQueries({
        queryKey: queryKeys.homeFavorites(playerSlug, currentTitleSlug, locale),
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.matchHistoryAll(playerSlug) })
    },
  })
}
