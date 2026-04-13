import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/last-match')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/players/$playerSlug/last-match"!</div>
}
