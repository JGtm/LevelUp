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

export const Route = createFileRoute('/players/$playerSlug/matches/$matchId')({
  validateSearch: z.object({
    tab: z.enum(['summary', 'details']).optional().catch('summary'),
  }),
  loader: ({ params, context }) => {
    // Prefetch parallèle : la vue + les voisins (prev/next).
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.matchView(params.playerSlug, params.matchId),
      queryFn: () =>
        api.get<MatchViewResponse>(
          `/players/${params.playerSlug}/matches/${params.matchId}`,
        ),
    })
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.matchNeighbors(params.playerSlug, params.matchId),
      queryFn: () =>
        api.get<MatchNeighbors>(
          `/players/${params.playerSlug}/matches/${params.matchId}/neighbors`,
        ),
    })
  },
  component: MatchViewPage,
})
