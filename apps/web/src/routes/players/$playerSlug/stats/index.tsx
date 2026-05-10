/**
 * Route /players/$playerSlug/stats — index.
 */
import { createFileRoute, Navigate, useParams } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/stats/')({
  component: RouteComponent,
})

function RouteComponent() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  return (
    <Navigate
      to="/players/$playerSlug/stats/timeseries"
      params={{ playerSlug }}
      replace
    />
  )
}
