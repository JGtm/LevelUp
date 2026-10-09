import { createFileRoute } from '@tanstack/react-router'
import { SquadEmprisePage } from '@/features/squad/SquadEmprisePage'

export const Route = createFileRoute(
  '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/emprise',
)({
  component: SquadEmprisePage,
})
