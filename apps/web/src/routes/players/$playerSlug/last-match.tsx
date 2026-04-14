/**
 * Route /players/$playerSlug/last-match — Dernier match.
 */
import { createFileRoute } from '@tanstack/react-router'
import { LastMatchPage } from '@/features/match-view/LastMatchPage'

export const Route = createFileRoute('/players/$playerSlug/last-match')({
  component: LastMatchPage,
})
