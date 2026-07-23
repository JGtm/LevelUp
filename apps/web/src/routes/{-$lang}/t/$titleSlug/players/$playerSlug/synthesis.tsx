import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : la Synthèse est passée sous la section Solo (/stats/synthesis).
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/synthesis')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/synthesis', params, replace: true })
  },
})
