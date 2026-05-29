/**
 * SessionCompareEngagement — engagement moyen (avg_residual_brut) A vs B
 * + progression par match (engagement_score depuis match_series).
 */
import { useMemo, useCallback } from 'react'

import { resolveToken } from '@/lib/accessibility'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareEngagementProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    progressionTitle: string
    sessionA: string
    sessionB: string
    empty: string
  }
  height?: number
}

function fmt(v: number | null | undefined): string {
  return v != null ? v.toFixed(1) : '—'
}

function winnerClass(
  a: number | null | undefined,
  b: number | null | undefined,
): { a: string; b: string } {
  if (a == null || b == null) return { a: 'text-foreground', b: 'text-foreground' }
  if (Math.abs(a - b) < 0.05) return { a: 'text-foreground', b: 'text-foreground' }
  return {
    a: a > b ? 'text-compare-a font-semibold' : 'text-foreground',
    b: a > b ? 'text-foreground' : 'text-compare-b font-semibold',
  }
}

export function SessionCompareEngagement({
  sessionA,
  sessionB,
  labels,
  height = 240,
}: SessionCompareEngagementProps) {
  const hasAvgData =
    sessionA?.avg_residual_brut != null || sessionB?.avg_residual_brut != null

  const avgColors = winnerClass(sessionA?.avg_residual_brut, sessionB?.avg_residual_brut)

  const series = useMemo<ChartSeries<ChartPoint2D>[]>(() => {
    const result: ChartSeries<ChartPoint2D>[] = []
    const ptsA = sessionA?.match_series?.filter((p) => p.engagement_score != null)
    if (ptsA?.length) {
      result.push({
        key: 'session-a-engagement',
        meta: { gamertag: labels.sessionA },
        datapoints: ptsA.map((p) => ({ x: String(p.index), y: p.engagement_score! })),
      })
    }
    const ptsB = sessionB?.match_series?.filter((p) => p.engagement_score != null)
    if (ptsB?.length) {
      result.push({
        key: 'session-b-engagement',
        meta: { gamertag: labels.sessionB },
        datapoints: ptsB.map((p) => ({ x: String(p.index), y: p.engagement_score! })),
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

  const hasProgression = series.length > 0

  // Mode single (sessionB absent) : la moyenne A-vs-B n'a pas de sens ; on n'affiche
  // que la progression d'engagement par match.
  if (sessionB == null) {
    if (!hasProgression) {
      return (
        <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
      )
    }
    return (
      <TimeseriesLineChart
        title=""
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

  if (!hasAvgData && !hasProgression) {
    return (
      <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
    )
  }

  return (
    <div className="space-y-4">
      {hasAvgData && (
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-xs text-muted-foreground">
              <th className="py-2 pr-4 text-left">{labels.title}</th>
              <th className="py-2 pr-4 text-right text-compare-a">A</th>
              <th className="py-2 text-right text-compare-b">B</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="py-2 pr-4 text-muted-foreground">{labels.title}</td>
              <td className={`py-2 pr-4 text-right tabular-nums ${avgColors.a}`}>
                {fmt(sessionA?.avg_residual_brut)}
              </td>
              <td className={`py-2 text-right tabular-nums ${avgColors.b}`}>
                {fmt(sessionB?.avg_residual_brut)}
              </td>
            </tr>
          </tbody>
        </table>
      )}

      {hasProgression && (
        <TimeseriesLineChart
          title={labels.progressionTitle}
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
      )}
    </div>
  )
}
