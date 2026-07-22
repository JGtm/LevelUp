import { createFileRoute, redirect } from '@tanstack/react-router'

// Redirection legacy : Leaderboard PP est passé de /palmares à /community.
export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/palmares/prestige')({
  beforeLoad: ({ params }) => {
    throw redirect({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/prestige', params, replace: true })
  },
})
