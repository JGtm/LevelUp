/**
 * ExplorerLiveStatusBadge — badge discret expliquant pourquoi une section live
 * de l'encart "Profil joueur cible" (Explorer mode Joueur) est vide ou
 * partielle, au lieu de ne rien dire (Lot A3, fin de la dégradation muette —
 * .ai/V7.1/PLAN_EXPLORER_LIVE_REPAIR_2026-07.md).
 *
 * Statuts (ExplorerLiveSectionStatus, miroir domain.ExplorerLiveSectionStatus
 * côté Go) :
 *  - "ok"            : rien à signaler, le badge ne rend rien.
 *  - "no_auth"       : aucun token Halo exploitable, le fetch live n'a jamais
 *    été tenté.
 *  - "failed"        : fetch live tenté et en échec (err déjà loggé côté API).
 *  - "local_partial" : repli sur un calcul/bucketing local structurellement
 *    incomplet.
 * Statut absent (undefined/null) → pas de badge (dégradation gracieuse pour
 * les fixtures/tests antérieurs à Lot A3, qui ne posent pas encore le champ).
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { Badge } from '@/components/ui/badge'
import type { ExplorerLiveSectionStatus } from '@/lib/api/types'

const STATUS_KEYS: Partial<Record<ExplorerLiveSectionStatus, ExplorerManifestKey>> = {
  no_auth: 'explorer.target_profile.live_status.no_auth',
  failed: 'explorer.target_profile.live_status.failed',
  local_partial: 'explorer.target_profile.live_status.local_partial',
}

export interface ExplorerLiveStatusBadgeProps {
  status?: ExplorerLiveSectionStatus | null
  className?: string
}

export function ExplorerLiveStatusBadge({ status, className }: ExplorerLiveStatusBadgeProps) {
  const locale = useAppShellStore((s) => s.locale)
  if (!status || status === 'ok') return null
  const key = STATUS_KEYS[status]
  if (!key) return null
  return (
    <Badge
      variant="outline"
      className={`text-xs ${className ?? ''}`}
      data-testid={`explorer-live-status-badge-${status}`}
    >
      {formatMessage(explorerManifest, key, locale)}
    </Badge>
  )
}
