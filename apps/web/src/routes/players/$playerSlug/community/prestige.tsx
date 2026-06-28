/**
 * Route /players/$playerSlug/community/prestige — Leaderboard PP (Communauté).
 */
import { createFileRoute } from '@tanstack/react-router'
import { LeaderboardPPPage } from '@/features/prestige/LeaderboardPPPage'

export const Route = createFileRoute('/players/$playerSlug/community/prestige')({
  component: LeaderboardPPPage,
})
