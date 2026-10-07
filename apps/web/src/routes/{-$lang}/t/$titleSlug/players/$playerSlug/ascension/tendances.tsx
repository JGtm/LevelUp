import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Tendances est passée sous la section Solo (/stats/tendances).
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tendances')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/tendances', params, replace: true })
  },
})
