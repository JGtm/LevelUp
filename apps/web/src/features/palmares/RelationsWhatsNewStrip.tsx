/**
 * RelationsWhatsNewStrip — bande compacte « Quoi de neuf » du hub Relations.
 *
 * Rendue entre les cartes hero et les chips de filtre, UNIQUEMENT si au moins un
 * groupe est non vide. Deux groupes : « Nouvelles têtes » (client, first_seen ≤
 * 30 j) et « Retrouvailles » (flag serveur is_revived). Aucun appel réseau : tout
 * vient de `relations[]`. Gamertags cliquables → Explorer (réutilise goToExplorer).
 */
import { useMemo } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import type { RelationInsight } from '@/lib/api/types'

import type { PalmaresText } from './i18n'
import { computeWhatsNew, hasWhatsNew, type WhatsNewGroup } from './whatsNew'

type RelationsText = PalmaresText['relations']

/** WhatsNewGroupView — un groupe (label + gamertags cliquables + compteur +N). */
function WhatsNewGroupView({
  group,
  label,
  tooltip,
  onPlayerClick,
}: {
  group: WhatsNewGroup
  label: string
  tooltip: string
  onPlayerClick: (gamertag: string) => void
}) {
  if (group.total === 0) return null
  return (
    <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
      <Tooltip content={tooltip}>
        <span className="text-xs font-medium text-foreground">{label}</span>
      </Tooltip>
      <span className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
        {group.players.map((r) => (
          <button
            key={r.xuid}
            type="button"
            className="text-sm font-semibold text-info hover:underline"
            onClick={() => onPlayerClick(r.gamertag)}
          >
            {r.gamertag}
          </button>
        ))}
        {group.overflow > 0 && <span className="text-xs text-muted-foreground">+{group.overflow}</span>}
      </span>
    </div>
  )
}

export function RelationsWhatsNewStrip({
  relations,
  labels,
  onPlayerClick,
}: {
  relations: RelationInsight[]
  labels: RelationsText
  onPlayerClick: (gamertag: string) => void
}) {
  const whatsNew = useMemo(() => computeWhatsNew(relations), [relations])
  if (!hasWhatsNew(whatsNew)) return null
  return (
    <div
      className="flex flex-wrap items-center gap-x-6 gap-y-2 rounded-lg border border-border bg-card px-4 py-2"
      data-testid="palmares-relations-whats-new"
    >
      <span className="text-xs font-semibold uppercase tracking-label-md text-muted-foreground">
        {labels.whatsNew.sectionTitle}
      </span>
      <WhatsNewGroupView
        group={whatsNew.newFaces}
        label={labels.whatsNew.newFaces}
        tooltip={labels.whatsNew.newFacesTooltip}
        onPlayerClick={onPlayerClick}
      />
      <WhatsNewGroupView
        group={whatsNew.reunions}
        label={labels.whatsNew.reunions}
        tooltip={labels.whatsNew.reunionsTooltip}
        onPlayerClick={onPlayerClick}
      />
    </div>
  )
}
