import { useParams } from '@tanstack/react-router'

import { LeaderboardBlock } from '@/features/leaderboard/LeaderboardBlock'

import { PalmaresShell } from './PalmaresShell'

export function PalmaresLeaderboardPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }

  return (
    <PalmaresShell playerSlug={playerSlug} activeTab="leaderboard">
      <LeaderboardBlock playerSlug={playerSlug} />
    </PalmaresShell>
  )
}
