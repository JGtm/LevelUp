/**
 * Route /players/$playerSlug/stats/sessions — Comparaison de sessions.
 */
import { createFileRoute } from '@tanstack/react-router'
import { SessionComparePage } from '@/features/session-compare/SessionComparePage'

export const Route = createFileRoute('/players/$playerSlug/stats/sessions')({
  component: SessionComparePage,
})
