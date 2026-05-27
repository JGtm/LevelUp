/**
 * SessionCompareRadar — radar 3 axes (K/D, Win%, Précision) Session A vs B.
 * Chart 07 du mock session_compare.
 *
 * K/D est normalisé sur 0-100 (kda / 3.0 × 100, plafonné à 100).
 * Win% et Accuracy% sont déjà en 0-100 via les metrics.
 */
import { useMemo } from 'react'

import { RadarChart, type RadarSeriesPayload } from '@/components/charts/RadarChart'
import type { SessionCompareEntry, SessionCompareMetricRow } from '@/lib/api/types'

export interface SessionCompareRadarProps {
  sessionA: SessionCompareEntry | null
  sessionB: SessionCompareEntry | null
  metrics: SessionCompareMetricRow[]
  labels: {
    title: string
    axisKD: string
    axisWinRate: string
    axisAccuracy: string
    sessionA: string
    sessionB: string
    empty: string
  }
  height?: number
}

function parseMetricFloat(metrics: SessionCompareMetricRow[], key: string, side: 'a' | 'b'): number {
  const row = metrics.find((m) => m.key === key)
  if (!row) return 0
  const raw = side === 'a' ? row.value_a : row.value_b
  return parseFloat(raw.replace('%', '').trim()) || 0
}

export function SessionCompareRadar({
  sessionA,
  sessionB,
  metrics,
  labels,
  height = 360,
}: SessionCompareRadarProps) {
  const series = useMemo<RadarSeriesPayload[]>(() => {
    if (!sessionA && !sessionB) return []

    const buildAxes = (side: 'a' | 'b') => {
      const kd = parseMetricFloat(metrics, 'kd_ratio', side)
      const winRate = parseMetricFloat(metrics, 'win_rate', side)
      const accuracy = parseMetricFloat(metrics, 'accuracy', side)
      return [
        { axis: 'kd', value: Math.min(Math.round(kd / 3.0 * 100), 100), raw: kd },
        { axis: 'winrate', value: Math.round(winRate), raw: winRate },
        { axis: 'accuracy', value: Math.round(accuracy), raw: accuracy },
      ]
    }

    const result: RadarSeriesPayload[] = []
    if (sessionA) result.push({ key: 'session-a', meta: { gamertag: labels.sessionA }, axes: buildAxes('a') })
    if (sessionB) result.push({ key: 'session-b', meta: { gamertag: labels.sessionB }, axes: buildAxes('b') })
    return result
  }, [sessionA, sessionB, metrics, labels])

  return (
    <RadarChart
      title={labels.title}
      series={series}
      emptyMessage={labels.empty}
      height={height}
      seriesNameResolver={(s) => s.meta?.gamertag ?? s.key}
      axisLabels={{ kd: labels.axisKD, winrate: labels.axisWinRate, accuracy: labels.axisAccuracy }}
    />
  )
}
