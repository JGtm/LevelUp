import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute(
  '/players/$playerSlug/squad/contributions',
)({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/players/$playerSlug/squad/contributions"!</div>
}
