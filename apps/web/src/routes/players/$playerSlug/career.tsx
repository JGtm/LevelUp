/**
 * Route /players/$playerSlug/career — hub Carrière.
 * Sprint 55 A3 : route canonique unique avec search param tab=progression|citations.
 *
 * P8.6 (revue 2026-04-29) : loader prefetch la page carrière au navigation.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { CareerPageResponse } from '@/lib/api/types'
import { CareerHubPage } from '@/features/career/CareerHubPage'

export const Route = createFileRoute('/players/$playerSlug/career')({
  validateSearch: z.object({
    tab: z.enum(['progression', 'citations']).optional(),
  }),
  loader: ({ params, context }) => {
    void context.queryClient.prefetchQuery({
      queryKey: queryKeys.career(params.playerSlug),
      queryFn: () =>
        api.get<CareerPageResponse>(`/players/${params.playerSlug}/pages/career`),
    })
  },
  component: CareerHubPage,
})
