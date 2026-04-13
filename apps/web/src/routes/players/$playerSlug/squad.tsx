/**
 * Route /players/$playerSlug/squad — page Escouade.
 */
import { createFileRoute } from '@tanstack/react-router'
import { SquadPage } from '@/features/squad/SquadPage'

export const Route = createFileRoute('/players/$playerSlug/squad')({
  component: SquadPage,
})
