import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/stats/sessions')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/players/$playerSlug/stats/sessions"!</div>
}
