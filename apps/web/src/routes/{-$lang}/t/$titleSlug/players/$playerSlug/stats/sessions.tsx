/**
 * Route /players/$playerSlug/stats/sessions — détail de session avec comparaison intégrée.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SessionDetailPage } from '@/features/session-detail/SessionDetailPage'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/stats/sessions')({
  validateSearch: z.object({
    session: z.string().optional(),
  }),
  component: SessionDetailPage,
})
