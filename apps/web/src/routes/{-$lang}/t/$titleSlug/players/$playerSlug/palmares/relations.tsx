import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : la section « Communauté » est passée de /palmares à /community.
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/palmares/relations')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/relations', params, replace: true })
  },
})
