/**
 * Route /players/$playerSlug/stats — index.
 */
import { createFileRoute, Navigate } from '@tanstack/react-router'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/stats/')({
  component: RouteComponent,
})

function RouteComponent() {
  // Route.useParams() porte lang?/titleSlug/playerSlug (typés) — même forme que la
  // cible stats/timeseries, réutilisable tel quel (préserve titre ET langue).
  const params = Route.useParams()
  return (
    <Navigate
      to="/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries"
      params={params}
      replace
    />
  )
}
