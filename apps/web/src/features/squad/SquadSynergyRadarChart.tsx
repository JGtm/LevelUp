/**
 * SquadSynergyRadarChart — wrapper teammates.06 (radar synergie sans aire centrale).
 *
 * Réutilise le `ChartCard` via le même cast que `RadarChart` (payload radar
 * a une structure spécifique non-`ChartSeries<T>`).
 */
import { useCallback, type ReactNode } from 'react'
import { ChartCard } from '@/components/charts/ChartCard'
import type { SquadSynergyRadarSeries } from '@/lib/api/types'
import {
  buildSquadSynergyRadarOption,
  type SquadSynergyRadarOpts,
} from './charts/squadSynergyRadarChart'

interface SquadSynergyRadarChartProps extends SquadSynergyRadarOpts {
  title?: ReactNode
  emptyMessage?: string
  rows: SquadSynergyRadarSeries[]
  height?: number
}

export function SquadSynergyRadarChart({
  rows,
  title,
  emptyMessage,
  colorByPlayer,
  axisLabels,
  rawLabel,
  height = 400,
}: SquadSynergyRadarChartProps) {
  const buildOption = useCallback(
    () => buildSquadSynergyRadarOption(rows, { colorByPlayer, axisLabels, rawLabel }),
    [rows, colorByPlayer, axisLabels, rawLabel],
  )

  return (
    <div data-testid="squad-synergy-radar">
      <ChartCard
        title={title}
        // Cast nécessaire : le payload radar n'est pas un ChartSeries<T> standard.
        series={rows as unknown as { key: string; datapoints: unknown[] }[]}
        buildOption={buildOption}
        height={height}
        emptyMessage={emptyMessage}
        reviewKey="squad.synergy_radar"
      />
    </div>
  )
}
