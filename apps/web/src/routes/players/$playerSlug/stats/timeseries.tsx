/**
 * Route /players/$playerSlug/stats/timeseries — Séries temporelles.
 */
import { createFileRoute } from '@tanstack/react-router'
import { TimeseriesPage } from '@/features/timeseries/TimeseriesPage'

export const Route = createFileRoute('/players/$playerSlug/stats/timeseries')({
  component: TimeseriesPage,
})
