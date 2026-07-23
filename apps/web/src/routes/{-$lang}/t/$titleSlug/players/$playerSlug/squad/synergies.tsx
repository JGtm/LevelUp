import { createFileRoute } from '@tanstack/react-router'
import { SquadSynergiesPage } from '@/features/squad/SquadSynergiesPage'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies')({
  component: SquadSynergiesPage,
})
