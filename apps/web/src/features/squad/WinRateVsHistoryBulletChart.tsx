/**
 * WinRateVsHistoryBulletChart — wrapper squad pour le bullet chart teammates.02.
 *
 * Spec : .ai/charts_specs/teammates/02_map_winrate_bullet.yaml
 * Builder : ./charts/winRateVsHistoryBulletChart.ts
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'
import {
  buildWinRateVsHistoryBulletOption,
  type WinRateVsHistoryBulletOpts,
} from './charts/winRateVsHistoryBulletChart'

interface WinRateVsHistoryBulletChartProps extends WinRateVsHistoryBulletOpts {
  title?: string
  emptyMessage?: string
  rows: MapBreakdownRow[]
}

export function WinRateVsHistoryBulletChart({
  rows,
  title,
  emptyMessage,
  mapLabelOf,
  sessionLabel,
  historyLabel,
  parityLabel,
  zeroWinrateLabel,
  countsLabel,
}: WinRateVsHistoryBulletChartProps) {
  const series = useMemo<ChartSeries<MapBreakdownRow>[]>(
    () => (rows.length > 0 ? [{ key: 'win-rate-vs-history-bullet', datapoints: rows }] : []),
    [rows],
  )

  const buildOption = useCallback(
    (s: ChartSeries<MapBreakdownRow>[]) =>
      buildWinRateVsHistoryBulletOption(s, {
        mapLabelOf,
        sessionLabel,
        historyLabel,
        parityLabel,
        zeroWinrateLabel,
        countsLabel,
      }),
    [mapLabelOf, sessionLabel, historyLabel, parityLabel, zeroWinrateLabel, countsLabel],
  )

  const height = Math.max(200, Math.min(600, rows.length * 32 + 60))

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
