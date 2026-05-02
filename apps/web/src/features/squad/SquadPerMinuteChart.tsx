/**
 * SquadPerMinuteChart — wrapper teammates.14.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerMinuteEntry } from '@/lib/api/types'
import {
  buildSquadPerMinuteOption,
  type SquadPerMinuteOpts,
} from './charts/squadPerMinuteChart'

interface SquadPerMinuteChartProps extends SquadPerMinuteOpts {
  title?: string
  rows: SquadPerMinuteEntry[]
}

export function SquadPerMinuteChart({ rows, title, ...opts }: SquadPerMinuteChartProps) {
  const series = useMemo<ChartSeries<SquadPerMinuteEntry>[]>(
    () => (rows.length > 0 ? [{ key: 'squad-per-minute', datapoints: rows }] : []),
    [rows],
  )
  const buildOption = useCallback(
    (s: ChartSeries<SquadPerMinuteEntry>[]) => buildSquadPerMinuteOption(s, opts),
    [opts],
  )
  return <ChartCard title={title} series={series} buildOption={buildOption} height={350} />
}
