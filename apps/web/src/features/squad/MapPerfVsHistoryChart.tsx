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
  const series = useMemo<ChartSeries<MapBreakdownRow>[]>(
    () => (rows.length > 0 ? [{ key: 'map-perf-vs-history', datapoints: rows }] : []),
    [rows],
  )

  const buildOption = useCallback(
    (s: ChartSeries<MapBreakdownRow>[]) =>
      buildMapPerfVsHistoryOption(s, { mapLabelOf, sessionLabel, historyLabel }),
    [mapLabelOf, sessionLabel, historyLabel],
  )

  const visibleCount = rows.filter(
    (r) => r.performance_avg !== undefined && r.historical_performance_avg !== undefined,
  ).length
  const cappedCount = Math.min(visibleCount, 20)
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
