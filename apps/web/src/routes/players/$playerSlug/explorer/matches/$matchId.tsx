/**
 * Route /players/$playerSlug/explorer/matches/$matchId — Match View.
 */
import { createFileRoute } from '@tanstack/react-router'
import { MatchViewPage } from '@/features/match-view/MatchViewPage'

export const Route = createFileRoute('/players/$playerSlug/explorer/matches/$matchId')({
  component: MatchViewPage,
})
