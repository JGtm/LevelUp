import { createFileRoute, redirect } from '@tanstack/react-router'
import { z } from 'zod'

// Redirection legacy : Face-à-face est passé sous /community/compare (section Communauté).
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/compare')({
  validateSearch: z.object({
    target: z.string().optional(),
    target2: z.string().optional(),
    from: z.enum(['explorer']).optional(),
  }),
  beforeLoad: ({ params, search }) => {
    throw redirect({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/compare', params, search, replace: true })
  },
})
