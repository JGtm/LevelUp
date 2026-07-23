import { createFileRoute } from '@tanstack/react-router'
import { SquadDynamiquePage } from '@/features/squad/SquadDynamiquePage'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/dynamique',
)({
  component: SquadDynamiquePage,
})
