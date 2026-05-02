/**
 * SquadSessionTimelineChart — wrapper teammates.04.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadSessionPoint } from '@/lib/api/types'
import {
  buildSquadSessionTimelineOption,
  type SquadSessionTimelineOpts,
} from './charts/squadSessionTimelineChart'

interface SquadSessionTimelineChartProps extends SquadSessionTimelineOpts {
  title?: string
  rows: SquadSessionPoint[]
}

export function SquadSessionTimelineChart({ rows, title, ...opts }: SquadSessionTimelineChartProps) {
  const series = useMemo<ChartSeries<SquadSessionPoint>[]>(
    () => (rows.length > 0 ? [{ key: 'squad-session-timeline', datapoints: rows }] : []),
    [rows],
  )
  const buildOption = useCallback(
    (s: ChartSeries<SquadSessionPoint>[]) => buildSquadSessionTimelineOption(s, opts),
    [opts],
  )
  return <ChartCard title={title} series={series} buildOption={buildOption} height={360} />
}
