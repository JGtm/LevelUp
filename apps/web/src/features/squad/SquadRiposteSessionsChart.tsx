/**
 * SquadRiposteSessionsChart — wrapper de la frise « soirée par soirée » de la riposte.
 *
 * Même forme que `SquadSessionTimelineChart` (la grammaire de référence) : l'option vit
 * dans un module PUR (`charts/squadRiposteSessionsChart.ts`), le composant ne fait que la
 * brancher sur `ChartCard`. `frameless` : la frise est déjà DANS la carte « Riposte », un
 * second cadre ferait un cadre dans un cadre.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'

import {
  buildSquadRiposteSessionsOption,
  type SquadRiposteSessionsOpts,
} from './charts/squadRiposteSessionsChart'
import type { FriseRiposte } from './squadRiposte.logic'

export interface SquadRiposteSessionsChartProps extends SquadRiposteSessionsOpts {
  frise: FriseRiposte
  title: string
  emptyMessage: string
}

export function SquadRiposteSessionsChart({
  frise,
  title,
  emptyMessage,
  ...opts
}: SquadRiposteSessionsChartProps) {
  const series = useMemo<ChartSeries<FriseRiposte>[]>(
    () => (frise.soirees.length > 0 ? [{ key: 'riposte-sessions', datapoints: [frise] }] : []),
    [frise],
  )
  const buildOption = useCallback(
    (s: ChartSeries<FriseRiposte>[]) => buildSquadRiposteSessionsOption(s, opts),
    [opts],
  )
  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={320}
      emptyMessage={emptyMessage}
      frameless
    />
  )
}
