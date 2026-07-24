/**
 * SessionColumnBody — corps d'une colonne de session (TOUT ce qui est sous le L3) :
 * bande KPI + pile de graphes + tableau "Détail des matchs".
 *
 * Monté À L'IDENTIQUE par la colonne principale ET par le drawer compare → garantit que
 * "ce qui est sous le L3" est strictement le même rendu des deux côtés ; seules diffèrent
 * les DONNÉES (session active vs comparée) et le header L3 (navigation vs sélecteur).
 *
 * En mode `compact` (colonne divisée, drawer ouvert) : KPI abrégés + tableau compact —
 * exactement la "vue compacte" de la colonne principale.
 */
import type { IntensityMatchRow, SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'

import type { CompareScale } from './_compareScale'
import { SessionChartStack } from './SessionChartStack'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionSummaryCard } from './SessionSummaryCard'
import { useSessionT } from './_shared'

interface Props {
  entry: SessionCompareEntry | null
  matches: SessionDetailMatchRow[]
  playerSlug: string
  /** Colonne divisée (drawer ouvert) → KPI abrégés + tableau compact + donuts en % interne. */
  compact: boolean
  /** Côté de l'axe du profil de participation : 'right' (colonne principale) / 'left' (drawer). */
  participationSide?: 'left' | 'right'
  /** Bornes d'axe partagées A/B (mode comparaison) — fige les échelles pour comparabilité. */
  scale?: CompareScale
  /** Profil d'intensité (frags par phase) de la session — calculé côté Go (payload). */
  intensityRows?: IntensityMatchRow[]
}

export function SessionColumnBody({
  entry,
  matches,
  playerSlug,
  compact,
  participationSide = 'right',
  scale,
  intensityRows,
}: Props) {
  const t = useSessionT()

  return (
    <>
      <SessionSummaryCard entry={entry} compact={compact} />

      <SessionChartStack
        entry={entry}
        matches={matches}
        compact={compact}
        participationSide={participationSide}
        participationColor="compare-a"
        scale={scale}
        intensityRows={intensityRows}
      />

      {/* Tableau "Détail des matchs" — hors bloc/Card (juste un titre + le tableau). */}
      <div className="space-y-3">
        <h2 className="text-base font-semibold text-foreground">{t('session.detail.matches_card')}</h2>
        <SessionMatchesTable
          matches={matches}
          playerSlug={playerSlug}
          variant={compact ? 'compact' : 'full'}
          withFriends={entry?.with_friends ?? false}
        />
      </div>
    </>
  )
}
