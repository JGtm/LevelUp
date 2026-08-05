/**
 * Route /{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/ — vue détaillée
 * d'un match (index).
 *
 * P8.6 (revue 2026-04-29) : le loader préfetche la vue + les voisins à la navigation
 * pour éliminer le flicker initial sur cette page lourde. Déplacé depuis `$matchId.tsx`
 * lorsque celui-ci est devenu un layout pour héberger le rejeu 2D dans son Outlet.
 */
import { createFileRoute } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { MatchViewResponse, MatchNeighbors } from '@/lib/api/types'
import { MatchViewPage } from '@/features/match-view/MatchViewPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'
import { hasCapabilityIn } from '@/lib/capabilities/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/',
)({
  loader: ({ params, context }) => {
    // Gate multi-titre : un titre sans capability `matchmaking` n'a pas de détail
    // match → on n'amorce PAS les prefetch (évite des 404 inutiles). Fail-open :
    // halo_infinite (matchmaking toujours présent) garde le prefetch inchangé.
    const { availableTitles, currentTitleSlug } = useAppShellStore.getState()
    const caps =
      availableTitles.find((t) => t.slug === currentTitleSlug)?.capabilities ?? null
    if (!hasCapabilityIn(caps, 'matchmaking')) return

    // Prefetch parallèle : la vue + les voisins (prev/next).
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.matchView(params.playerSlug, params.titleSlug, params.matchId),
      queryFn: () =>
        api.get<MatchViewResponse>(
          `/players/${params.playerSlug}/matches/${params.matchId}`,
        ),
    })
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.matchNeighbors(params.playerSlug, params.titleSlug, params.matchId),
      queryFn: () =>
        api.get<MatchNeighbors>(
          `/players/${params.playerSlug}/matches/${params.matchId}/neighbors`,
        ),
    })
  },
  component: () => (
    <RouteCapabilityGate capability="matchmaking">
      <MatchViewPage />
    </RouteCapabilityGate>
  ),
})
