/**
 * SquadWeaponKillsChart — wrapper teammates.09.
 *
 * Hauteur dynamique : `max(350, n_weapons * 38)` (cf. spec).
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadWeaponKills, SquadWeaponBar } from '@/lib/api/types'
import {
  buildSquadWeaponKillsOption,
  type SquadWeaponKillsOpts,
} from './charts/squadWeaponKillsChart'

interface SquadWeaponKillsChartProps extends SquadWeaponKillsOpts {
  title?: string
  emptyMessage?: string
  data: SquadWeaponKills | null | undefined
}

export function SquadWeaponKillsChart({ data, title, emptyMessage, ...opts }: SquadWeaponKillsChartProps) {
  const series = useMemo<ChartSeries<SquadWeaponBar>[]>(
    () => (data && data.bars.length > 0 ? [{ key: 'weapon-kills', datapoints: data.bars }] : []),
    [data],
  )
  const buildOption = useCallback(
    () => buildSquadWeaponKillsOption(data, opts),
    [data, opts],
  )
  const n = data?.bars.length ?? 0
  const height = Math.max(350, Math.min(800, n * 38))
  return (
    <ChartCard title={title} series={series} buildOption={buildOption} height={height} emptyMessage={emptyMessage} />
  )
}
