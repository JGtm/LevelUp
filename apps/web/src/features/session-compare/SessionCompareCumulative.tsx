/**
 * SessionCompareCumulative — solde W/L cumulé par match (Session A vs B).
 * Chart 09 du mock session_compare.
 *
 * Chaque point = cumul des wins (+1) / losses (-1) depuis le premier match.
 * Le zéro = équilibre parfait.
 */
import { useMemo, useCallback } from 'react'

import { resolveToken } from '@/lib/accessibility'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareCumulativeProps {
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

export function SessionCompareCumulative({
  sessionA,
  sessionB,
  labels,
  height = 320,
}: SessionCompareCumulativeProps) {
  const series = useMemo<ChartSeries<ChartPoint2D>[]>(() => {
    const result: ChartSeries<ChartPoint2D>[] = []
    if (sessionA?.match_series?.length) {
      result.push({
        key: 'session-a-cumul',
        meta: { gamertag: labels.sessionA },
        datapoints: sessionA.match_series.map((p) => ({
          x: String(p.index),
          y: p.cumulative,
        })),
      })
    }
    if (sessionB?.match_series?.length) {
      result.push({
        key: 'session-b-cumul',
        meta: { gamertag: labels.sessionB },
        datapoints: sessionB.match_series.map((p) => ({
          x: String(p.index),
          y: p.cumulative,
        })),
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
