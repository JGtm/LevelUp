import { describe, it, expect } from 'vitest'

import { buildCareerXpSeries } from './careerXpSeries'
import type { TimeseriesMatchRow } from '@/lib/api/types'

/** Ligne minimale : seul career_xp_estimated importe pour ce builder. */
function row(xp: number | null): TimeseriesMatchRow {
  return { career_xp_estimated: xp } as unknown as TimeseriesMatchRow
}

describe('buildCareerXpSeries', () => {
  it('retourne hasData=false et des séries vides sans matchs', () => {
    const s = buildCareerXpSeries([])
    expect(s.hasData).toBe(false)
    expect(s.perMatch).toEqual([])
    expect(s.cumulative).toEqual([])
  })

  it('masque le chart (hasData=false) quand aucune estimation', () => {
    const s = buildCareerXpSeries([row(null), row(null)])
    expect(s.hasData).toBe(false)
    expect(s.perMatch).toEqual([null, null])
    expect(s.cumulative).toEqual([null, null])
  })

  it('cumule en reportant sur les nuls et démarre au premier match connu', () => {
    // Ordre chronologique : trou en tête, un trou interne (Firefight/score absent).
    const s = buildCareerXpSeries([row(null), row(100), row(200), row(null), row(50)])
    expect(s.hasData).toBe(true)
    // XP par match : les nuls restent nuls (barre omise).
    expect(s.perMatch).toEqual([null, 100, 200, null, 50])
    // Cumul : null tant qu'aucune XP, puis reporté sur le trou interne (pas de reset).
    expect(s.cumulative).toEqual([null, 100, 300, 300, 350])
  })
})
