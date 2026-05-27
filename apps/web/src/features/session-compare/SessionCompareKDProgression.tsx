/**
 * SessionCompareKDProgression — évolution du ratio K/D par match (Session A vs B).
 * Chart 10 du mock session_compare.
 */
import { useMemo, useCallback } from 'react'

import { resolveToken } from '@/lib/accessibility'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareKDProgressionProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    sessionA: string
    sessionB: string
    empty: string
  }
  height?: number
}

export function SessionCompareKDProgression({
  sessionA,
  sessionB,
  labels,
  height = 280,
}: SessionCompareKDProgressionProps) {
  const series = useMemo<ChartSeries<ChartPoint2D>[]>(() => {
    const result: ChartSeries<ChartPoint2D>[] = []
    if (sessionA?.match_series?.length) {
      result.push({
        key: 'session-a-kd',
        meta: { gamertag: labels.sessionA },
        datapoints: sessionA.match_series.map((p) => ({ x: String(p.index), y: p.kd })),
      })
    }
    if (sessionB?.match_series?.length) {
      result.push({
        key: 'session-b-kd',
        meta: { gamertag: labels.sessionB },
        datapoints: sessionB.match_series.map((p) => ({ x: String(p.index), y: p.kd })),
      })
    }
    return result
  }, [sessionA, sessionB, labels])

  const seriesColorResolver = useCallback(
    (_s: ChartSeries<ChartPoint2D>, idx: number) =>
      idx === 0 ? resolveToken('compare-a') : resolveToken('compare-b'),
    [],
  )

  const seriesNameResolver = useCallback(
    (s: ChartSeries<ChartPoint2D>) => (s.meta?.gamertag as string | undefined) ?? s.key,
    [],
  )

  return (
    <TimeseriesLineChart
      title={labels.title}
      series={series}
      emptyMessage={labels.empty}
      height={height}
      xAxisType="category"
      outcomeMarkers={false}
      smooth
      showSymbol
      seriesColorResolver={seriesColorResolver}
      seriesNameResolver={seriesNameResolver}
    />
  )
}
