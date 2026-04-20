import { createFileRoute } from '@tanstack/react-router'

import { PalmaresLeaderboardPage } from '@/features/palmares/PalmaresLeaderboardPage'

export const Route = createFileRoute('/players/$playerSlug/palmares/')({
  component: PalmaresLeaderboardPage,
})
