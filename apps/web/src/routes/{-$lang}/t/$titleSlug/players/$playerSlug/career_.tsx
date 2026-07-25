/**
 * Route /players/$playerSlug/career — hub Carrière.
 * Sprint 55 A3 : route canonique unique avec search param tab=progression|citations.
 *
 * P8.6 (revue 2026-04-29) : loader prefetch la page carrière au navigation.
 *
 * career_.tsx (trailing underscore) : non-layout route — n'agit PAS comme parent
 * pour les routes dans le dossier career/ (ex: career/season-pass).
 */
import { createFileRoute } from '@tanstack/react-router'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { CareerPageResponse } from '@/lib/api/types'
import { CareerHubPage } from '@/features/career/CareerHubPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/career_')({
  loader: ({ params, context }) => {
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.career(params.playerSlug, params.titleSlug),
      queryFn: () =>
        api.get<CareerPageResponse>(`/players/${params.playerSlug}/pages/career`),
    })
  },
  component: () => (
    <RouteCapabilityGate capability="career">
      <CareerHubPage />
    </RouteCapabilityGate>
  ),
})
