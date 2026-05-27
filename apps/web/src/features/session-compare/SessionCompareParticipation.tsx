/**
 * SessionCompareParticipation — radar 6 axes (Combat/Survie/Support/Score/Objectif/Impact) A vs B.
 * Chart 13 du mock session_compare.
 *
 * Utilise RadarChart avec 2 séries (session A et session B).
 * Valeurs normalisées 0..100 par le backend (analysis/narrative).
 */
import { useMemo, useCallback } from 'react'

import { resolveToken } from '@/lib/accessibility'
import { RadarChart } from '@/components/charts/RadarChart'
import type { RadarSeriesPayload } from '@/components/charts/RadarChart'
import type { SessionCompareEntry, SessionParticipationAxis } from '@/lib/api/types'

export interface SessionCompareParticipationProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  labels: {
    title: string
    sessionA: string
    sessionB: string
    empty: string
    combat: string
    survival: string
    support: string
    score: string
    objective: string
    impact: string
  }
  height?: number
}

function toRadarAxes(axes: SessionParticipationAxis[] | undefined) {
  return (axes ?? []).map((a) => ({ axis: a.name, value: a.value, raw: a.value }))
}

export function SessionCompareParticipation({
  sessionA,
  sessionB,
  labels,
  height = 320,
}: SessionCompareParticipationProps) {
  const series = useMemo<RadarSeriesPayload[]>(() => {
    const result: RadarSeriesPayload[] = []
    if (sessionA?.participation?.length) {
      result.push({
        key: 'session-a-participation',
        meta: { gamertag: labels.sessionA },
        axes: toRadarAxes(sessionA.participation),
      })
    }
    if (sessionB?.participation?.length) {
      result.push({
        key: 'session-b-participation',
        meta: { gamertag: labels.sessionB },
        axes: toRadarAxes(sessionB.participation),
      })
    }
    return result
  }, [sessionA, sessionB, labels])

  const axisLabels = useMemo(
    () => ({
      combat: labels.combat,
      survival: labels.survival,
      support: labels.support,
      score: labels.score,
      objective: labels.objective,
      impact: labels.impact,
    }),
    [labels],
  )

  const seriesNameResolver = useCallback(
    (s: RadarSeriesPayload) => (s.meta?.gamertag as string | undefined) ?? s.key,
    [],
  )

  const seriesColorOverride = useMemo(
    () => [resolveToken('compare-a'), resolveToken('compare-b')],
    [],
  )

  // Inject color into each series via meta.color so ChartCard picks it up.
  const seriesWithColors = useMemo(
    () =>
      series.map((s, i) => ({
        ...s,
        meta: { ...s.meta, color: seriesColorOverride[i] ?? resolveToken('compare-a') },
      })),
    [series, seriesColorOverride],
  )

  return (
    <RadarChart
      title={labels.title}
      series={seriesWithColors}
      emptyMessage={labels.empty}
      height={height}
      seriesNameResolver={seriesNameResolver}
      axisLabels={axisLabels}
    />
  )
}
