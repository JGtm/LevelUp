/**
 * Route /players/$playerSlug/explorer — page Explorer.
 */
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ExplorerPage } from '@/features/explorer/ExplorerPage'

export const Route = createFileRoute('/players/$playerSlug/explorer/')({
  validateSearch: z.object({
    mode: z.enum(['matches', 'player']).optional(),
    target: z.string().optional(),
  }),
  component: ExplorerPage,
})
