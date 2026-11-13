/**
 * wallPlanLayer.test.ts — le calque du plan des murs, au contexte enregistreur : les arêtes
 * tombent aux pixels que `worldToCanvas` donne, un plan absent ne touche pas au contexte, et
 * l'interrupteur éteint retire le calque de la scène.
 */
import { describe, expect, it } from 'vitest'

import { worldToCanvas } from '../../../lib/replay/replayLogic'
import type { CanvasView } from '../model/replayView'
import type { WallPlanReady } from '../model/wallPlan'
import { composeScene, LAYER_ORDER, sceneLayers, type LayerPaint, type ReplayScene } from './replayCompose'
import { recordingContext } from '../test/recordingContext'
import { drawWallPlanLayer, WALL_PLAN_ALPHA, WALL_PLAN_LINE_WIDTH } from './wallPlanLayer'

/** Un cadrage connu : 200 x 100 m, toile 1000 x 600, marge 20. */
const VIEW: CanvasView = {
  bounds: { minX: -100, maxX: 100, minY: -50, maxY: 50 },
  width: 1000,
  height: 600,
  pad: 20,
}

const PLAN: WallPlanReady = {
  edges: [
    [-100, -50, 100, 50],
    [0, 0, 10, 0],
    [-20, 30, -20, 40],
  ],
  dropped: 0,
}

/** La projection attendue d'un point monde — l'oracle est `worldToCanvas`, pas le calque. */
const px = (x: number, y: number) => worldToCanvas({ x, y }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)

describe('drawWallPlanLayer', () => {
  it('trace chaque arête entre les pixels que worldToCanvas donne', () => {
    const { ops, ctx } = recordingContext()
    drawWallPlanLayer(ctx, PLAN, VIEW, { ink: 'rgb(1 2 3)' })
    const segments = ops.filter((o) => o.op === 'moveTo' || o.op === 'lineTo')
    expect(segments).toHaveLength(PLAN.edges.length * 2)
    PLAN.edges.forEach(([x1, y1, x2, y2], i) => {
      const a = px(x1, y1)
      const b = px(x2, y2)
      expect(segments[2 * i]).toEqual({ op: 'moveTo', args: [a.x, a.y] })
      expect(segments[2 * i + 1]).toEqual({ op: 'lineTo', args: [b.x, b.y] })
    })
  })

  it('trace un trait de 1 px CSS, à l’encre demandée, en UN SEUL chemin', () => {
    const { ops, ctx } = recordingContext()
    drawWallPlanLayer(ctx, PLAN, VIEW, { ink: 'rgb(1 2 3)' })
    expect(ops).toContainEqual({ op: 'set strokeStyle', args: ['rgb(1 2 3)'] })
    expect(ops).toContainEqual({ op: 'set lineWidth', args: [WALL_PLAN_LINE_WIDTH] })
    expect(WALL_PLAN_LINE_WIDTH).toBe(1)
    expect(ops).toContainEqual({ op: 'set globalAlpha', args: [WALL_PLAN_ALPHA] })
    expect(ops.filter((o) => o.op === 'beginPath')).toHaveLength(1)
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(1)
  })

  it('ne touche pas au contexte sans plan, ni avec un plan vide', () => {
    for (const plan of [null, { edges: [], dropped: 0 }]) {
      const { ops, ctx } = recordingContext()
      drawWallPlanLayer(ctx, plan, VIEW, { ink: 'rgb(1 2 3)' })
      expect(ops).toEqual([])
    }
  })

  it('suit le cadrage : un zoom déplace et agrandit les arêtes comme worldToCanvas', () => {
    const zoomed: CanvasView = { ...VIEW, bounds: { minX: -50, maxX: 50, minY: -25, maxY: 25 } }
    const { ops, ctx } = recordingContext()
    drawWallPlanLayer(ctx, PLAN, zoomed, { ink: 'rgb(1 2 3)' })
    const first = ops.find((o) => o.op === 'moveTo')
    const expected = worldToCanvas({ x: -100, y: -50 }, zoomed.bounds, zoomed.width, zoomed.height, zoomed.pad)
    expect(first).toEqual({ op: 'moveTo', args: [expected.x, expected.y] })
  })
})

describe('plan-murs dans la scène', () => {
  const temoin = (id: string): LayerPaint => (ctx, frame, dpr) => ctx.fillText(id, frame, dpr)
  const scene = (over: { wallPlan?: boolean; has?: boolean }): ReplayScene => ({
    toggles: {
      zones: true, shotFx: true, placements: true, killFx: true, modeObjectives: true,
      wallPlan: over.wallPlan ?? true,
    },
    has: {
      background: true, floor: false, heat: false, zoneNames: false, objectivesCooked: false,
      projectiles: false, placements: false, rifts: false, fireMarks: false, shotFx: false,
      zoneStates: false, objectivePulses: false, killFx: false, wallPlan: over.has ?? true,
    },
    paint: Object.fromEntries(LAYER_ORDER.map((id) => [id, temoin(id)])) as ReplayScene['paint'],
  })
  const peints = (s: ReplayScene): string[] => {
    const { ctx } = recordingContext()
    return composeScene(ctx, sceneLayers(s), 0, 1)
  }

  it('se peint quand l’interrupteur est allumé et que la carte a un plan', () => {
    expect(peints(scene({}))).toContain('plan-murs')
  })

  it('ne se peint pas quand l’interrupteur est éteint', () => {
    expect(peints(scene({ wallPlan: false }))).not.toContain('plan-murs')
  })

  it('ne se peint pas quand la carte n’a pas de plan', () => {
    expect(peints(scene({ has: false }))).not.toContain('plan-murs')
  })

  it('se pose sur le fond, sous les pions', () => {
    const ordre = peints(scene({}))
    expect(ordre.indexOf('fond-carte')).toBeLessThan(ordre.indexOf('plan-murs'))
    expect(ordre.indexOf('plan-murs')).toBeLessThan(ordre.indexOf('trajectoires'))
  })
})
