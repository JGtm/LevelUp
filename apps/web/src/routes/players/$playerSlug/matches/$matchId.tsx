import { createFileRoute } from '@tanstack/react-router'
import { MatchViewPage } from '@/features/match-view/MatchViewPage'

export const Route = createFileRoute('/players/$playerSlug/matches/$matchId')({
  component: MatchViewPage,
})
