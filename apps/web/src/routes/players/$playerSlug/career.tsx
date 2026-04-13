/**
 * Route /players/$playerSlug/career — page Carrière.
 */
import { createFileRoute } from '@tanstack/react-router'
import { CareerPage } from '@/features/career/CareerPage'

export const Route = createFileRoute('/players/$playerSlug/career')({
  component: CareerPage,
})
