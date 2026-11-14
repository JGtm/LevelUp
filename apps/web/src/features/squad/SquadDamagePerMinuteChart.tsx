/**
 * SquadDamagePerMinuteChart — graphe frère de teammates.14 : dégâts infligés et subis par minute,
 * une barre par joueur. Les dégâts subis sont sous l'axe, comme les morts du graphe voisin.
 */
import { useCallback, useMemo, type ReactNode } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerMinuteEntry } from '@/lib/api/types'
import {
  buildSquadDamagePerMinuteOption,
  type SquadDamagePerMinuteOpts,
} from './charts/squadPerMinuteChart'

interface SquadDamagePerMinuteChartProps extends SquadDamagePerMinuteOpts {
  title?: ReactNode
  emptyMessage?: string
  rows: SquadPerMinuteEntry[]
}

export function SquadDamagePerMinuteChart({
  rows,
  title,
  emptyMessage,
  ...opts
}: SquadDamagePerMinuteChartProps) {
  const series = useMemo<ChartSeries<SquadPerMinuteEntry>[]>(
    () => (rows.length > 0 ? [{ key: 'squad-damage-per-minute', datapoints: rows }] : []),
    [rows],
  )
  const buildOption = useCallback(
    (s: ChartSeries<SquadPerMinuteEntry>[]) => buildSquadDamagePerMinuteOption(s, opts),
    [opts],
  )
  return <ChartCard title={title} series={series} buildOption={buildOption} height={350} emptyMessage={emptyMessage} />
}
