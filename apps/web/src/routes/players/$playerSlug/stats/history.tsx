/**
 * Route /players/$playerSlug/stats/history — Historique des parties.
 */
import { createFileRoute } from '@tanstack/react-router'
import { MatchHistoryPage } from '@/features/match-history/MatchHistoryPage'

export const Route = createFileRoute('/players/$playerSlug/stats/history')({
  component: MatchHistoryPage,
})
