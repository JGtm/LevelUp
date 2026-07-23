/**
 * Tests P4 — « Écart d'engagement cumulé » (session).
 *
 * - `computeCumulativeEngagementGap` (pur) : zip match_series ↔ matches, cumul de
 *   engagement_score × durée/60 (événements), report D5.
 * - `buildSessionEngagementGapOption` (pur) : aire signée divergente ancrée à 0 + markLine 0.
 */
import { describe, it, expect } from 'vitest'

import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'

import {
  buildSessionEngagementGapOption,
  computeCumulativeEngagementGap,
  type EngagementGapPoint,
} from './SessionEngagementCumulative'

function match(startTime: string, durationSeconds: number, map = 'Bazaar'): SessionDetailMatchRow {
  return {
    assists: 0,
    deaths: 0,
    is_ranked: false,
    kills: 0,
    match_id: `m-${startTime}`,
    pair_name: 'Slayer',
    playlist_name: 'pl',
    start_time: startTime,
    map_name: map,
    duration_seconds: durationSeconds,
  } as SessionDetailMatchRow
}

function entry(scores: Array<number | undefined>): SessionCompareEntry {
  return {
    match_series: scores.map((s, i) => ({ index: i, engagement_score: s })),
  } as unknown as SessionCompareEntry
}

interface OptShape {
  backgroundColor: string
  series?: Array<{
    type: string
    data: number[]
    areaStyle?: { origin?: number; color?: { type?: string } }
    markLine?: { data: Array<{ yAxis: number }> }
  }>
  xAxis?: { boundaryGap?: boolean }
}

describe('computeCumulativeEngagementGap', () => {
  it('cumul de engagement_score × durée/60 (événements)', () => {
    // scores [3, -1] évén./min ; durée 600 s (10 min) → contrib [30, -10] ; cumul [30, 20].
    const pts = computeCumulativeEngagementGap(
      [match('2025-01-01T10:00:00Z', 600), match('2025-01-01T11:00:00Z', 600)],
      entry([3, -1]),
    )
    expect(pts.map((p) => p.value)).toEqual([30, -10])
    expect(pts.map((p) => p.cumulative)).toEqual([30, 20])
  })

  it('report D5 : un match sans score ne fait pas avancer le cumul', () => {
    const pts = computeCumulativeEngagementGap(
      [
        match('2025-01-01T10:00:00Z', 600),
        match('2025-01-01T11:00:00Z', 600),
        match('2025-01-01T12:00:00Z', 600),
      ],
      entry([3, undefined, -1]),
    )
    expect(pts[1].value).toBeNull()
    expect(pts.map((p) => p.cumulative)).toEqual([30, 30, 20])
  })

  it('match_series vide → aucun point', () => {
    expect(computeCumulativeEngagementGap([match('2025-01-01T10:00:00Z', 600)], entry([]))).toEqual([])
  })
})

describe('buildSessionEngagementGapOption', () => {
  const opts = { seriesLabel: 'Écart cumulé', matchLabel: 'Écart du match', axisLabel: 'événements' }

  it('série vide → option de fond minimale (pas de série)', () => {
    const opt = buildSessionEngagementGapOption([], opts) as unknown as OptShape
    expect(opt.backgroundColor).toBeTruthy()
    expect(opt.series).toBeUndefined()
  })

  it('une série ligne : data = cumul, aire ancrée à 0, markLine 0', () => {
    const series: ChartSeries<EngagementGapPoint>[] = [
      {
        key: 'g',
        datapoints: computeCumulativeEngagementGap(
          [match('2025-01-01T10:00:00Z', 600), match('2025-01-01T11:00:00Z', 600)],
          entry([3, -1]),
        ),
      },
    ]
    const opt = buildSessionEngagementGapOption(series, opts) as unknown as OptShape
    expect(opt.series).toHaveLength(1)
    expect(opt.series![0].type).toBe('line')
    expect(opt.series![0].data).toEqual([30, 20])
    expect(opt.series![0].areaStyle?.origin).toBe(0)
    expect(opt.series![0].markLine?.data[0].yAxis).toBe(0)
    expect((opt.series![0].areaStyle?.color as { type?: string })?.type).toBe('linear')
    expect(opt.xAxis?.boundaryGap).toBe(false)
  })
})
