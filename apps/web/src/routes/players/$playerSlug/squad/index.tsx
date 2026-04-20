import { createFileRoute, Navigate, useParams } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/squad/')({
  component: RouteComponent,
})

function RouteComponent() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  return (
    <Navigate
      to="/players/$playerSlug/squad/synergies"
      params={{ playerSlug }}
      replace
    />
  )
}
