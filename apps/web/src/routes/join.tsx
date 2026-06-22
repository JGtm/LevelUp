/**
 * Route /join — page de jonction à un groupe via lien d'invitation (?invite=CODE).
 */
import { createFileRoute } from '@tanstack/react-router'
import { JoinPage } from '@/features/auth/JoinPage'

export const Route = createFileRoute('/join')({
  component: JoinPage,
  validateSearch: (search: Record<string, unknown>): { invite?: string } => ({
    invite: typeof search.invite === 'string' ? search.invite : undefined,
  }),
})
