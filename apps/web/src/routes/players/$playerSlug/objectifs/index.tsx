/**
 * Route /players/$playerSlug/objectifs — page Objectifs (Prestige).
 */
import { createFileRoute } from '@tanstack/react-router'
import { ObjectifsPage } from '@/features/prestige/ObjectifsPage'

export const Route = createFileRoute('/players/$playerSlug/objectifs/')({
  component: ObjectifsPage,
})
