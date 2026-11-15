/**
 * wallPlanPaint.test.ts — le trait du plan des murs : arêtes projetées par la fonction donnée, un
 * seul chemin, 1 px, et rien sans plan.
 */
import { describe, expect, it } from 'vitest'

import { recordingContext } from '@/features/match-replay/test/recordingContext'

import { strokeWallPlan, WALL_PLAN_ALPHA, WALL_PLAN_LINE_WIDTH } from './wallPlanPaint'

const PLAN = {
  edges: [
    [0, 0, 10, 0],
    [5, 5, 5, 9],
  ],
  dropped: 0,
} as const

describe('strokeWallPlan', () => {
  it('trace chaque arête entre les points que la projection donne', () => {
    const { ops, ctx } = recordingContext()
    strokeWallPlan(ctx, PLAN, (p) => ({ x: p.x * 2 + 1, y: 100 - p.y }), 'rgb(1 2 3)')
    const segments = ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo')
    expect(segments.map((s) => s.args)).toEqual([
      [1, 100],
      [21, 100],
      [11, 95],
      [11, 91],
    ])
  })

  it('un seul chemin, 1 px, à l’encre et à l’opacité du calque', () => {
    const { ops, ctx } = recordingContext()
    strokeWallPlan(ctx, PLAN, (p) => p, 'rgb(1 2 3)')
    expect(ops.filter((o) => o.op === 'beginPath')).toHaveLength(1)
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(1)
    expect(ops).toContainEqual({ op: 'set lineWidth', args: [WALL_PLAN_LINE_WIDTH] })
    expect(ops).toContainEqual({ op: 'set strokeStyle', args: ['rgb(1 2 3)'] })
    expect(ops).toContainEqual({ op: 'set globalAlpha', args: [WALL_PLAN_ALPHA] })
    expect(WALL_PLAN_LINE_WIDTH).toBe(1)
  })

  it('sans plan, ou avec un plan vide, ne touche pas au contexte', () => {
    for (const plan of [null, { edges: [], dropped: 0 }]) {
      const { ops, ctx } = recordingContext()
      strokeWallPlan(ctx, plan, (p) => p, 'rgb(1 2 3)')
      expect(ops).toEqual([])
    }
  })
})
