import { describe, expect, it } from 'vitest'

import type { PostSyncStepTiming } from '@/lib/api/types'
import { buildTimelineSegments, dominantStep, slowestSteps, stepToken } from './timeline'

const t = (step: string, ms: number, items = 0): PostSyncStepTiming => ({
  step,
  duration_ms: ms,
  items,
})

describe('buildTimelineSegments', () => {
  it('écarte les étapes à 0 ms et calcule les proportions sur le reste', () => {
    const segments = buildTimelineSegments([
      t('scoring', 1000),
      t('media_scan', 0),
      t('weapon_kills', 3000),
    ])
    expect(segments).toHaveLength(2)
    expect(segments[0].pct).toBe(25)
    expect(segments[1].pct).toBe(75)
  })

  it('retourne [] pour des timings absents ou tous nuls', () => {
    expect(buildTimelineSegments(undefined)).toEqual([])
    expect(buildTimelineSegments([t('scoring', 0)])).toEqual([])
  })

  it('attribue une couleur stable par étape (ordre canonique)', () => {
    expect(stepToken('enrichment_rows')).toBe('chart-series-1')
    expect(stepToken('scoring')).toBe('chart-series-2')
    // Étape inconnue → token de cycle, jamais de crash.
    expect(stepToken('etape_future')).toBeTruthy()
  })
})

describe('slowestSteps', () => {
  it('classe par durée décroissante et borne à n', () => {
    const segments = buildTimelineSegments([
      t('scoring', 100),
      t('weapon_kills', 5000),
      t('citations', 800),
      t('aggregates', 1200),
    ])
    const top = slowestSteps(segments, 2)
    expect(top.map((s) => s.step)).toEqual(['weapon_kills', 'aggregates'])
  })
})

describe('dominantStep', () => {
  it('détecte un goulot au-delà du seuil ET de la durée minimale', () => {
    const dominant = dominantStep(
      [t('weapon_kills', 50_000), t('scoring', 10_000)],
      60,
      30_000,
    )
    expect(dominant?.step).toBe('weapon_kills')
    expect(dominant?.pct).toBeGreaterThanOrEqual(60)
  })

  it('ignore les pipelines courts même très déséquilibrés', () => {
    expect(dominantStep([t('weapon_kills', 900), t('scoring', 100)], 60, 30_000)).toBeUndefined()
  })

  it('ignore les pipelines équilibrés', () => {
    expect(
      dominantStep([t('weapon_kills', 40_000), t('scoring', 35_000)], 60, 30_000),
    ).toBeUndefined()
  })
})
