/**
 * SquadEngagementGapChart — « Écart d'engagement cumulé » par joueur (onglet
 * Dynamique). Une courbe cumulée par joueur : `(pace_observed − team_expected) ×
 * durée`, en événements, couleur par joueur, ligne repère 0.
 *
 * Réutilise la MÊME query `useSquadEngagementSession` que SquadEngagementSection
 * (dédup par le cache TanStack Query — pas de second fetch). Monté sous le
 * FeatureGate `engagement` du parent (pas de self-gate).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useSquadEngagementSession, type SquadTeammateEntry } from '@/features/engagement/queries'

import type { SquadText } from './i18n'
import { buildSquadEngagementGapOption } from './charts/squadEngagementGapChart'

const SUBCHART_HEIGHT = 280

interface Props {
  playerSlug: string
  matchIds: string[]
  teammates: SquadTeammateEntry[]
  /** gamertag → couleur hex (getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
  t: SquadText
  emptyMessage?: string
  height?: number
}

export function SquadEngagementGapChart({
  playerSlug,
  matchIds,
  teammates,
  colorByPlayer,
  t,
  emptyMessage,
  height = SUBCHART_HEIGHT,
}: Props) {
  const query = useSquadEngagementSession(playerSlug, matchIds, teammates)
  const session = query.data

  // Série sentinelle : non vide dès que la session porte au moins un match ; le
  // builder relit directement la session (comme SquadFdaGapCumulativeCard).
  const series = useMemo<ChartSeries<number>[]>(
    () => ((session?.labels.length ?? 0) > 0 ? [{ key: 'engagement-gap-flat', datapoints: [1] }] : []),
    [session],
  )

  // `buildOption` fait partie des dépendances du useMemo de ChartCard : une
  // lambda écrite dans le JSX est neuve à chaque rendu, donc l'option ECharts
  // est rebâtie et l'animation d'entrée REJOUÉE même à donnée inchangée.
  const buildOption = useCallback(
    () => (session ? buildSquadEngagementGapOption(session, { colorByPlayer }) : {}),
    [session, colorByPlayer],
  )

  return (
    <ChartCard
      title={
        <span className="flex items-center gap-1.5">
          {t.engagementGap.title}
          <InfoTooltip content={t.engagementGap.tooltip} />
        </span>
      }
      series={series}
      height={height}
      emptyMessage={emptyMessage}
      buildOption={buildOption}
    />
  )
}
