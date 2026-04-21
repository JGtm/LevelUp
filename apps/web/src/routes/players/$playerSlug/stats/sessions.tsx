/**
 * Route /players/$playerSlug/stats/sessions — détail de session avec comparaison intégrée.
 */
import { createFileRoute } from '@tanstack/react-router'
import { SessionDetailPage } from '@/features/session-detail/SessionDetailPage'

export const Route = createFileRoute('/players/$playerSlug/stats/sessions')({
  component: SessionDetailPage,
})
