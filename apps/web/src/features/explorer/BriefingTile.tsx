/**
 * BriefingTile — tuile KPI unitaire du socle du bandeau de briefing Explorer.
 *
 * Extraite du Strip (DEC-1, V3 compaction) pour être partagée avec les tuiles
 * Classement/Séries (ExplorerBriefingTiles) sans cycle d'import. Chrome = KpiCard
 * + barre d'accent optionnelle ; label COURT (uppercase) + valeur (bold, tabular)
 * + slot graphique optionnel `chart` (micro-sparkline) rendu entre la valeur et le
 * sous-texte + sous-texte optionnel. Tokens sémantiques uniquement.
 */
import type { ReactNode } from 'react'

import { KpiCard } from '@/components/cards/KpiCard'
import type { SemanticToken } from '@/lib/accessibility'

export interface TileProps {
  label: string
  value: ReactNode
  /** Slot graphique optionnel (ex. micro-sparkline), rendu entre la valeur et le sous-texte. */
  chart?: ReactNode
  sub?: ReactNode
  accent?: SemanticToken
}

export function BriefingTile({ label, value, chart, sub, accent }: TileProps) {
  return (
    <KpiCard accent={accent} className="h-full">
      <div className="px-3 py-2">
        <p className="text-3xs uppercase tracking-wide text-muted-foreground">{label}</p>
        <div className="mt-0.5 text-xl font-bold tabular-nums leading-tight text-foreground">
          {value}
        </div>
        {chart && <div className="mt-1">{chart}</div>}
        {sub && <div className="mt-0.5 text-2xs text-muted-foreground">{sub}</div>}
      </div>
    </KpiCard>
  )
}
