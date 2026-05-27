/**
 * SessionCompareSkillProgression — évolution du rating LUSR / CSR par match (A vs B).
 * Utilise skill_rating dans match_series. N'affiche rien si aucune session n'a de données.
 */
import { useMemo, useCallback } from 'react'

import { resolveToken } from '@/lib/accessibility'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { SessionCompareEntry } from '@/lib/api/types'

export interface SessionCompareSkillProgressionProps {
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

export function SessionCompareSkillProgression({
  sessionA,
  sessionB,
  labels,
  height = 280,
}: SessionCompareSkillProgressionProps) {
  const series = useMemo<ChartSeries<ChartPoint2D>[]>(() => {
    const result: ChartSeries<ChartPoint2D>[] = []
    const ptsA = sessionA?.match_series?.filter((p) => p.skill_rating != null)
    if (ptsA?.length) {
      result.push({
        key: 'session-a-skill',
        meta: { gamertag: labels.sessionA },
        datapoints: ptsA.map((p) => ({ x: String(p.index), y: p.skill_rating! })),
      })
    }
    const ptsB = sessionB?.match_series?.filter((p) => p.skill_rating != null)
    if (ptsB?.length) {
      result.push({
        key: 'session-b-skill',
        meta: { gamertag: labels.sessionB },
        datapoints: ptsB.map((p) => ({ x: String(p.index), y: p.skill_rating! })),
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
      smooth={false}
      showSymbol
      seriesColorResolver={seriesColorResolver}
      seriesNameResolver={seriesNameResolver}
    />
  )
}
