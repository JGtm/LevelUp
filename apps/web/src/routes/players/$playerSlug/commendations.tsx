import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Commendations est passé sous la section Carrière (/career/commendations).
export const Route = createFileRoute('/players/$playerSlug/commendations')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/players/$playerSlug/career/commendations', params, replace: true })
  },
})
