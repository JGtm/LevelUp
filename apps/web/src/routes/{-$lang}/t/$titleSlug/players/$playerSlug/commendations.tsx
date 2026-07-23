import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Commendations est passé sous la section Carrière (/career/commendations).
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/commendations')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/commendations', params, replace: true })
  },
})
