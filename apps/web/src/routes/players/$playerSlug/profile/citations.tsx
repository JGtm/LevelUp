import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/profile/citations')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/players/$playerSlug/profile/citations"!</div>
}
