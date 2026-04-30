/**
 * WinRateVsHistoryChart — wrapper squad pour le graphe win rate session vs historique.
 *
 * Barres horizontales groupées : yAxis=cartes, xAxis=win rate %.
 * Couleur par barre session : divergent-pos / neutral / neg selon l'écart à l'historique.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'
import {
  buildWinRateVsHistoryOption,
  type WinRateVsHistoryOpts,
} from './charts/winRateVsHistoryChart'

interface WinRateVsHistoryChartProps extends WinRateVsHistoryOpts {
  title?: string
  rows: MapBreakdownRow[]
}

export function WinRateVsHistoryChart({
  rows,
  title,
  mapLabelOf,
  sessionLabel,
  historyLabel,
}: WinRateVsHistoryChartProps) {
  const series = useMemo<ChartSeries<MapBreakdownRow>[]>(
    () => (rows.length > 0 ? [{ key: 'win-rate-vs-history', datapoints: rows }] : []),
    [rows],
  )

  const buildOption = useCallback(
    (s: ChartSeries<MapBreakdownRow>[]) =>
      buildWinRateVsHistoryOption(s, { mapLabelOf, sessionLabel, historyLabel }),
    [mapLabelOf, sessionLabel, historyLabel],
  )

  const height = Math.max(200, Math.min(600, rows.length * 32 + 60))

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={height}
    />
  )
}
