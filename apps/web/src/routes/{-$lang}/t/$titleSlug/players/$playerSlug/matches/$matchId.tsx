/**
 * Route /players/$playerSlug/matches/$matchId — vue détaillée d'un match.
 *
 * P8.6 (revue 2026-04-29) : loader prefetch la match view + neighbors au
 * navigation pour éliminer le flicker initial sur cette page lourde.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { MatchViewResponse, MatchNeighbors } from '@/lib/api/types'
import { MatchViewPage } from '@/features/match-view/MatchViewPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'
import { hasCapabilityIn } from '@/lib/capabilities/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId')({
  validateSearch: z.object({
    tab: z.enum(['summary', 'details']).optional().catch('summary'),
  }),
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
