/**
 * CombatScorePlacementChart (G3) — fin wrapper ChartCard consommant
 * buildCombatScoreOption (barres score + courbe placement sur axe inversé).
 * Reçoit les matchs PvP triés chronologiquement croissants.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { ExplorerTargetRecentMatch } from '@/lib/api/types'
import {
  buildCombatScoreOption,
  type CombatScoreLabels,
  type CombatScorePoint,
} from './combatChartOptions'

export interface CombatScorePlacementChartProps {
  /** Matchs triés chronologiquement croissants. */
  matches: ExplorerTargetRecentMatch[]
  labels: CombatScoreLabels
  height?: number
  /** Titre rendu dans la barre de titre ChartCard (style unifié). */
  title?: string
}

export function CombatScorePlacementChart({
  matches,
  labels,
  height = 300,
  title,
}: CombatScorePlacementChartProps) {
  const series = useMemo<ChartSeries<CombatScorePoint>[]>(() => {
    if (matches.length === 0) return []
    return [
      {
        key: 'explorer.combat.score',
        meta: { gamertag: 'combat_score' },
        datapoints: matches.map((m) => ({
          x: m.start_time,
          score: m.score,
          rank: m.rank ?? null,
        })),
      },
    ]
  }, [matches])

  const buildOption = useCallback(
    (s: ChartSeries<CombatScorePoint>[]) => buildCombatScoreOption(s, labels),
    [labels],
  )

  return <ChartCard title={title} series={series} buildOption={buildOption} height={height} />
}
