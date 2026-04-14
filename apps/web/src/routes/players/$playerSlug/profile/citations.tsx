/**
 * Route /players/$playerSlug/profile/citations — Citations (commendations + médailles).
 */
import { createFileRoute } from '@tanstack/react-router'
import { CitationsPage } from '@/features/citations/CitationsPage'

export const Route = createFileRoute('/players/$playerSlug/profile/citations')({
  component: CitationsPage,
})
