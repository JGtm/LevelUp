/**
 * Route /players/$playerSlug/stats/timeseries — Séries temporelles.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { TimeseriesPage } from '@/features/timeseries/TimeseriesPage'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries')({
  validateSearch: z.object({
    tab: z.enum(['summary', 'distributions', 'progression']).optional().catch('summary'),
  }),
  component: TimeseriesPage,
})
