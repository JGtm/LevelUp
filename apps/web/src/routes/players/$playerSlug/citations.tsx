/**
 * Route /players/$playerSlug/citations — page Citations dédiée.
 */
import { createFileRoute } from '@tanstack/react-router'
import { CitationsPage } from '@/features/citations/CitationsPage'

export const Route = createFileRoute('/players/$playerSlug/citations')({
  component: CitationsPage,
})
