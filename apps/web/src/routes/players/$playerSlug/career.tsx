/**
 * Route /players/$playerSlug/career — hub Carrière.
 * Sprint 55 A3 : route canonique unique avec search param tab=progression|citations.
 */
import { createFileRoute } from '@tanstack/react-router'
import { CareerHubPage } from '@/features/career/CareerHubPage'
import { z } from 'zod'

export const Route = createFileRoute('/players/$playerSlug/career')({
  validateSearch: z.object({
    tab: z.enum(['progression', 'citations']).optional(),
  }),
  component: CareerHubPage,
})
