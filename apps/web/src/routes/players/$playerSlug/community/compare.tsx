import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ComparePage } from '@/features/compare/ComparePage'

export const Route = createFileRoute('/players/$playerSlug/community/compare')({
  validateSearch: z.object({
    target: z.string().optional(),
    target2: z.string().optional(),
    from: z.enum(['explorer']).optional(),
  }),
  component: ComparePage,
})
