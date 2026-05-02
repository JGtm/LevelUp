/**
 * SquadFirstEventsChart — wrapper teammates.17.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadFirstEvents, SquadFirstEventsRow } from '@/lib/api/types'
import {
  buildSquadFirstEventsOption,
  type SquadFirstEventsOpts,
} from './charts/squadFirstEventsChart'

interface SquadFirstEventsChartProps extends SquadFirstEventsOpts {
  title?: string
  data: SquadFirstEvents | null | undefined
  height?: number
}

export function SquadFirstEventsChart({
  data,
  title,
  height = 420,
  ...opts
}: SquadFirstEventsChartProps) {
  const series = useMemo<ChartSeries<SquadFirstEventsRow>[]>(
    () => (data && data.rows.length > 0 ? [{ key: 'first-events', datapoints: data.rows }] : []),
    [data],
  )
  const buildOption = useCallback(
    () => buildSquadFirstEventsOption(data, opts),
    [data, opts],
  )
  return <ChartCard title={title} series={series} buildOption={buildOption} height={height} />
}
