import { createFileRoute, Navigate } from '@tanstack/react-router'

import { PalmaresLeaderboardPage } from '@/features/palmares/PalmaresLeaderboardPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/community/')({
  component: CommunityIndex,
})

/**
 * « Classements » : sans la capacité `can_compare_accounts` (application de bureau, un seul compte suivi),
 * l'onglet n'existe pas dans la navigation et son adresse renvoie vers « Relations ».
 */
function CommunityIndex() {
  const compareAccounts = useDeploymentCapability('can_compare_accounts')
  const { titleSlug, playerSlug } = Route.useParams()
  if (!compareAccounts) {
    return (
      <Navigate
        to="/{-$lang}/t/$titleSlug/players/$playerSlug/community/relations"
        params={{ titleSlug, playerSlug }}
        replace
      />
    )
  }
  return (
    <RouteCapabilityGate capability="world.leaderboard">
      <PalmaresLeaderboardPage />
    </RouteCapabilityGate>
  )
}
