/**
 * CombatFdaChart (G1) — fin wrapper ChartCard consommant buildCombatFdaOption.
 * Reçoit les matchs PvP triés chronologiquement croissants.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { ExplorerTargetRecentMatch } from '@/lib/api/types'
import {
  buildCombatFdaOption,
  type CombatFdaLabels,
  type CombatFdaPoint,
} from './combatChartOptions'

export interface CombatFdaChartProps {
  /** Matchs triés chronologiquement croissants. */
  matches: ExplorerTargetRecentMatch[]
  labels: CombatFdaLabels
  height?: number
}

export function CombatFdaChart({ matches, labels, height = 300 }: CombatFdaChartProps) {
  const series = useMemo<ChartSeries<CombatFdaPoint>[]>(() => {
    if (matches.length === 0) return []
    return [
      {
        key: 'explorer.combat.fda',
        meta: { gamertag: 'combat_fda' },
        datapoints: matches.map((m) => ({
          x: m.start_time,
          kills: m.kills,
          deaths: m.deaths,
          assists: m.assists,
          kda: m.kda,
          outcome: m.outcome,
        })),
      },
    ]
  }, [matches])

  const buildOption = useCallback(
    (s: ChartSeries<CombatFdaPoint>[]) => buildCombatFdaOption(s, labels),
    [labels],
  )

  return <ChartCard series={series} buildOption={buildOption} height={height} />
}
