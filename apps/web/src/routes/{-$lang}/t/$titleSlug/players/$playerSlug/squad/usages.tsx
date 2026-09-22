import { createFileRoute } from '@tanstack/react-router'
import { SquadUsagesPage } from '@/features/squad/SquadUsagesPage'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/usages',
)({
  component: SquadUsagesPage,
})
