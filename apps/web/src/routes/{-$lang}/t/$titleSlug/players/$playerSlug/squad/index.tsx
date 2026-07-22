import { createFileRoute, Navigate } from '@tanstack/react-router'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/squad/')({
  component: RouteComponent,
})

function RouteComponent() {
  // Route.useParams() porte lang?/titleSlug/playerSlug (typés) — même forme que la
  // cible squad/synergies, donc réutilisable tel quel (préserve le segment titre
  // ET la langue courante).
  const params = Route.useParams()
  return (
    <Navigate
      to="/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies"
      params={params}
      replace
    />
  )
}
