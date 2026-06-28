import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : la section « Communauté » est passée de /palmares à /community.
export const Route = createFileRoute('/players/$playerSlug/palmares/relations')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/players/$playerSlug/community/relations', params, replace: true })
  },
})
