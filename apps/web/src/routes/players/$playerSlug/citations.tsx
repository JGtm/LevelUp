import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Citations est passé sous la section Carrière (/career/citations).
export const Route = createFileRoute('/players/$playerSlug/citations')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/players/$playerSlug/career/citations', params, replace: true })
  },
})
