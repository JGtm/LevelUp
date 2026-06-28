import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : la Synthèse est passée sous la section Solo (/stats/synthesis).
export const Route = createFileRoute('/players/$playerSlug/synthesis')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/players/$playerSlug/stats/synthesis', params, replace: true })
  },
})
