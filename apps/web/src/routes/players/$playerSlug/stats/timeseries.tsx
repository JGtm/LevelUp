import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/stats/timeseries')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/players/$playerSlug/stats/timeseries"!</div>
}
