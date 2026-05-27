/**
 * SessionComparePerfProgression — évolution du score de performance par match (A vs B).
 * Utilise perf_score dans match_series, même pattern que SessionCompareKDProgression.
 */
import { useMemo, useCallback } from 'react'

import { resolveToken } from '@/lib/accessibility'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionComparePerfProgressionProps {
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

export function SessionComparePerfProgression({
  sessionA,
  sessionB,
  labels,
  height = 280,
}: SessionComparePerfProgressionProps) {
  const series = useMemo<ChartSeries<ChartPoint2D>[]>(() => {
    const result: ChartSeries<ChartPoint2D>[] = []
    const ptsA = sessionA?.match_series?.filter((p) => p.perf_score != null)
    if (ptsA?.length) {
      result.push({
        key: 'session-a-perf',
        meta: { gamertag: labels.sessionA },
        datapoints: ptsA.map((p) => ({ x: String(p.index), y: p.perf_score! })),
      })
    }
    const ptsB = sessionB?.match_series?.filter((p) => p.perf_score != null)
    if (ptsB?.length) {
      result.push({
        key: 'session-b-perf',
        meta: { gamertag: labels.sessionB },
        datapoints: ptsB.map((p) => ({ x: String(p.index), y: p.perf_score! })),
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
