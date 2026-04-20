import { createFileRoute } from '@tanstack/react-router'
import { SquadContributionsPage } from '@/features/squad/SquadContributionsPage'

export const Route = createFileRoute(
  '/players/$playerSlug/squad/contributions',
)({
  component: SquadContributionsPage,
})
