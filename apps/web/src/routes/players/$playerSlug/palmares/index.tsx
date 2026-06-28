import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Classements/Communauté est passé de /palmares à /community.
export const Route = createFileRoute('/players/$playerSlug/palmares/')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/players/$playerSlug/community', params, replace: true })
  },
})
