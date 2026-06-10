/**
 * MapPerfVsHistoryChart — wrapper squad pour le grouped-bar teammates.13.
 *
 * Spec : .ai/charts_specs/teammates/13_map_perf_vs_history.yaml
 * Builder : ./charts/mapPerfVsHistoryChart.ts
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'
import {
  buildMapPerfVsHistoryOption,
  type MapPerfVsHistoryOpts,
} from './charts/mapPerfVsHistoryChart'

interface MapPerfVsHistoryChartProps extends MapPerfVsHistoryOpts {
  title?: string
  emptyMessage?: string
  rows: MapBreakdownRow[]
}

export function MapPerfVsHistoryChart({
  rows,
  title,
  emptyMessage,
  mapLabelOf,
  sessionLabel,
  historyLabel,
}: MapPerfVsHistoryChartProps) {
  // Seules les lignes portant les DEUX champs de performance sont joignables
  // par le builder : filtrer ICI pour que ChartCard rende son état vide quand
  // aucune ligne n'est exploitable (sinon option sans séries → canvas blanc
  // titré — le « bloc vide » que la refonte états vides élimine ; cas couvert
  // avant par les gates `.some(...)` supprimées).
  const joinable = useMemo(
    () =>
      rows.filter(
        (r) => r.performance_avg !== undefined && r.historical_performance_avg !== undefined,
      ),
    [rows],
  )
  const series = useMemo<ChartSeries<MapBreakdownRow>[]>(
    () => (joinable.length > 0 ? [{ key: 'map-perf-vs-history', datapoints: joinable }] : []),
    [joinable],
  )

  const buildOption = useCallback(
    (s: ChartSeries<MapBreakdownRow>[]) =>
      buildMapPerfVsHistoryOption(s, { mapLabelOf, sessionLabel, historyLabel }),
    [mapLabelOf, sessionLabel, historyLabel],
  )

  const cappedCount = Math.min(joinable.length, 20)
  const height = Math.max(200, Math.min(600, cappedCount * 32 + 60))

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={height}
      emptyMessage={emptyMessage}
    />
  )
}
