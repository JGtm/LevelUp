/**
 * Route /players/$playerSlug/palmares/prestige — Leaderboard PP (Communauté).
 */
import { createFileRoute } from '@tanstack/react-router'
import { LeaderboardPPPage } from '@/features/prestige/LeaderboardPPPage'

export const Route = createFileRoute('/players/$playerSlug/palmares/prestige')({
  component: LeaderboardPPPage,
})
