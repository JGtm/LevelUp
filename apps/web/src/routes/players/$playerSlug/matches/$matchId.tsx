import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/matches/$matchId')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/players/$playerSlug/matches/$matchId"!</div>
}
