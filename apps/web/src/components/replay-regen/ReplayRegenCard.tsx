/**
 * ReplayRegenCard — la carte des Paramètres (onglet Application) : la ligne d'état de la régénération des
 * rejeux, avec le même composant que le bandeau de la page de rejeu (`ReplayRegenPanel`) et, en plus,
 * « tous les rejeux sont à jour ». Sous la capacité `can_regenerate_replays` seulement.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ReplayRegenPanel } from './ReplayRegenPanel'
import { useReplayRegenStatus } from './queries'
import { REPLAY_REGEN_TEXT } from './text'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'

export function ReplayRegenCard() {
  const locale = useAppShellStore((s) => s.locale)
  const t = REPLAY_REGEN_TEXT[locale]
  const enabled = useDeploymentCapability('can_regenerate_replays')
  const { data: status } = useReplayRegenStatus(enabled)
  if (!enabled) return null
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.cardTitle}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{t.cardIntro}</p>
        <ReplayRegenPanel status={status} variant="row" />
      </CardContent>
    </Card>
  )
}
