/**
 * ReplayRegenBanner — le bandeau de la page de rejeu : « N rejeux à régénérer après la mise à jour », avec
 * « Régénérer » et « Plus tard ». Il n'existe que sous la capacité `can_regenerate_replays` (application de
 * bureau) et que s'il y a des rejeux en retard ou une file active ; « Plus tard » le masque pour la session.
 *
 * Un rejeu ouvert alors que son artefact est en retard s'affiche comme avant : le bandeau n'y change rien.
 */
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { bannerPhaseVisible, regenPhase } from './logic'
import { useReplayRegenStatus } from './queries'
import { ReplayRegenPanel } from './ReplayRegenPanel'
import { useRegenDismissed } from './dismissed'

export function ReplayRegenBanner() {
  const enabled = useDeploymentCapability('can_regenerate_replays')
  const { data: status } = useReplayRegenStatus(enabled)
  const { dismissed, dismiss } = useRegenDismissed()
  if (!enabled || !bannerPhaseVisible(regenPhase(status), dismissed)) return null
  return <ReplayRegenPanel status={status} variant="banner" onLater={dismiss} />
}
