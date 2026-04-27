import { createFileRoute } from '@tanstack/react-router'

import { SquadV2RouteHost } from '@/features/squad/v2/SquadV2RouteHost'

export const Route = createFileRoute('/players/$playerSlug/squad/v2')({
  component: SquadV2RouteHost,
})
