/**
 * BriefingTile — tuile KPI unitaire du socle du bandeau de briefing Explorer.
 *
 * Extraite du Strip (DEC-1, V3 compaction) pour être partagée avec les tuiles
 * Classement/Séries (ExplorerBriefingTiles) sans cycle d'import. Chrome = KpiCard
 * + barre d'accent optionnelle ; label COURT (uppercase) + icône (i) optionnelle
 * (tooltip de légende, V3 amendement) + valeur (bold, tabular) + sous-texte
 * optionnel. Tokens sémantiques uniquement.
 */
import type { ReactNode } from 'react'

import { KpiCard } from '@/components/cards/KpiCard'
import type { SemanticToken } from '@/lib/accessibility'

export interface TileProps {
  label: string
  value: ReactNode
  /** Icône (i) optionnelle (InfoTooltip) rendue à côté du label — reste hors de l'uppercase. */
  info?: ReactNode
  sub?: ReactNode
  accent?: SemanticToken
}

export function BriefingTile({ label, value, info, sub, accent }: TileProps) {
  return (
    <KpiCard accent={accent} className="h-full">
      <div className="px-3 py-2">
        <div className="flex items-center gap-1">
          <p className="text-3xs uppercase tracking-wide text-muted-foreground">{label}</p>
          {info}
        </div>
        <div className="mt-0.5 text-xl font-bold tabular-nums leading-tight text-foreground">
          {value}
        </div>
        {sub && <div className="mt-0.5 text-2xs text-muted-foreground">{sub}</div>}
      </div>
    </KpiCard>
  )
}
