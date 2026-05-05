/**
 * SquadSynergyRadarChart — wrapper teammates.06 (radar synergie sans aire centrale).
 *
 * Réutilise le `ChartCard` via le même cast que `RadarChart` (payload radar
 * a une structure spécifique non-`ChartSeries<T>`).
 */
import { useCallback } from 'react'
import { ChartCard } from '@/components/charts/ChartCard'
import type { SquadSynergyRadarSeries } from '@/lib/api/types'
import {
  buildSquadSynergyRadarOption,
  type SquadSynergyRadarOpts,
} from './charts/squadSynergyRadarChart'

interface SquadSynergyRadarChartProps extends SquadSynergyRadarOpts {
  title?: string
  rows: SquadSynergyRadarSeries[]
  height?: number
}

export function SquadSynergyRadarChart({
  rows,
  title,
  colorByPlayer,
  axisLabels,
  height = 400,
}: SquadSynergyRadarChartProps) {
  const buildOption = useCallback(
    () => buildSquadSynergyRadarOption(rows, { colorByPlayer, axisLabels }),
    [rows, colorByPlayer, axisLabels],
  )

  return (
    <div data-testid="squad-synergy-radar">
      <ChartCard
        title={title}
        // Cast nécessaire : le payload radar n'est pas un ChartSeries<T> standard.
        series={rows as unknown as { key: string; datapoints: unknown[] }[]}
        buildOption={buildOption}
        height={height}
      />
    </div>
  )
}
