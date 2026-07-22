import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Citations est passé sous la section Carrière (/career/citations).
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/citations')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations', params, replace: true })
  },
})
