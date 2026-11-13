/**
 * wallPlan.test.ts — la normalisation du plan des murs : pas de plan, arêtes bien formées,
 * arêtes refusées (comptées et signalées une seule fois).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { log } from '@/lib/accessibility/_logger'
import type { ReplayMapBackgroundPlan } from '@/lib/api/types'

import { normalizeWallPlan } from './wallPlan'

function plan(edges: ReplayMapBackgroundPlan['edges'], source = 'test'): ReplayMapBackgroundPlan {
  return { edges, faces: 2, source, vertices: 4 }
}

describe('normalizeWallPlan', () => {
  beforeEach(() => log._resetForTests())
  afterEach(() => vi.restoreAllMocks())

  it('rend null sans plan, sans liste ou avec une liste vide', () => {
    expect(normalizeWallPlan(undefined)).toBeNull()
    expect(normalizeWallPlan(null)).toBeNull()
    expect(normalizeWallPlan(plan(null))).toBeNull()
    expect(normalizeWallPlan(plan([]))).toBeNull()
  })

  it('garde les arêtes bien formées telles quelles', () => {
    const out = normalizeWallPlan(plan([[0, 1, 2, 3], [-4.5, 5, 6, -7.25]]))
    expect(out).toEqual({
      edges: [[0, 1, 2, 3], [-4.5, 5, 6, -7.25]],
      dropped: 0,
    })
  })

  it('écarte les arêtes nulles, courtes ou non finies, et les compte', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    const out = normalizeWallPlan(
      plan([[0, 0, 1, 1], null, [1, 2, 3], [0, 0, Number.NaN, 1], [0, 0, 1, Number.POSITIVE_INFINITY]]),
    )
    expect(out?.edges).toEqual([[0, 0, 1, 1]])
    expect(out?.dropped).toBe(4)
  })

  it('rend null quand aucune arête n est exploitable', () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    expect(normalizeWallPlan(plan([null, [1, 2]]))).toBeNull()
  })

  it('journalise une seule fois par plan, pas à chaque appel', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const p = plan([[0, 0, 1, 1], null], 'doc-a')
    normalizeWallPlan(p)
    normalizeWallPlan(p)
    normalizeWallPlan(p)
    expect(spy).toHaveBeenCalledTimes(1)
    normalizeWallPlan(plan([[0, 0, 1, 1], null], 'doc-b'))
    expect(spy).toHaveBeenCalledTimes(2)
  })

  it('ne journalise rien pour un plan sain', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    normalizeWallPlan(plan([[0, 0, 1, 1]]))
    expect(spy).not.toHaveBeenCalled()
  })
})
