/**
 * SquadMapHeatmapChart — wrapper teammates.03.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadMapHeatmap } from '@/lib/api/types'
import {
  buildSquadMapHeatmapOption,
  type SquadMapHeatmapOpts,
} from './charts/squadMapHeatmapChart'

interface SquadMapHeatmapChartProps extends SquadMapHeatmapOpts {
  title?: string
  emptyMessage?: string
  data: SquadMapHeatmap | null | undefined
}

export function SquadMapHeatmapChart({ data, title, emptyMessage, ...opts }: SquadMapHeatmapChartProps) {
  const series = useMemo<ChartSeries<SquadMapHeatmap>[]>(
    () => (data ? [{ key: 'squad-map-heatmap', datapoints: [data] }] : []),
    [data],
  )
  const buildOption = useCallback(
    (s: ChartSeries<SquadMapHeatmap>[]) => buildSquadMapHeatmapOption(s, opts),
    [opts],
  )
  const playerCount = data?.players.length ?? 0
  const height = Math.max(220, Math.min(640, playerCount * 60 + 160))
  return <ChartCard title={title} series={series} buildOption={buildOption} height={height} emptyMessage={emptyMessage} />
}
