import { createFileRoute } from '@tanstack/react-router'

import { PalmaresLeaderboardPage } from '@/features/palmares/PalmaresLeaderboardPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/players/$playerSlug/palmares/')({
  component: () => (
    <RouteCapabilityGate capability="world.leaderboard">
      <PalmaresLeaderboardPage />
    </RouteCapabilityGate>
  ),
})
