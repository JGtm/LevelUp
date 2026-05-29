import { useParams } from '@tanstack/react-router'

import { LeaderboardBlock } from '@/features/leaderboard/LeaderboardBlock'

export function PalmaresLeaderboardPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }

  return (
    <div className="flex flex-col gap-6 p-6">
      <LeaderboardBlock playerSlug={playerSlug} />
    </div>
  )
}
