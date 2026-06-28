import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Leaderboard PP est passé de /palmares à /community.
export const Route = createFileRoute('/players/$playerSlug/palmares/prestige')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/players/$playerSlug/community/prestige', params, replace: true })
  },
})
