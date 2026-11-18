/**
 * reliefLinesLayer.test.ts — le calque des traits du relief, au contexte enregistreur : chaque famille est UN
 * chemin et un seul `stroke`, à l'encre, l'opacité et l'épaisseur de la maquette ; les sommets passent par
 * pixel -> monde -> projection fournie par la vue ; le liseré est fermé ; une famille vide ne touche pas au contexte.
 */
import { describe, expect, it } from 'vitest'

import { recordingContext } from './canvasRecorder.testutil'
import { pixelToWorld } from './lineSet'
import { drawReliefLinesLayer, type ReliefLinesInput } from './reliefLinesLayer'
import type { LineSet, PointXY, ReliefCalibration } from './types'

/** Le calage d'une carte de 40 x 20 pixels de 0,5 unité, origine monde (-10, 5). */
const CAL: ReliefCalibration = { metersPerPixel: 0.5, originX: -10, originY: 5, widthPx: 40, heightPx: 20 }

/** Une projection quelconque (échelle, miroir et décalage) : le calque ne doit en connaître que la forme. */
const project = (p: PointXY): PointXY => ({ x: p.x * 10 + 100, y: -p.y * 10 + 50 })

const EMPTY: LineSet = { points: new Float32Array(0), offsets: Uint32Array.of(0) }

/** Une polyligne à deux sommets, en pixels du fond. */
function segment(x1: number, y1: number, x2: number, y2: number): LineSet {
  return { points: Float32Array.of(x1, y1, x2, y2), offsets: Uint32Array.of(0, 2) }
}

const INK = { edge: 'E', contour: 'C', wall: 'W', outline: 'O' }

function input(over: Partial<ReliefLinesInput> = {}): ReliefLinesInput {
  return { calibration: CAL, steps: EMPTY, contours: EMPTY, walls: EMPTY, outline: EMPTY, ink: INK, ...over }
}

describe('drawReliefLinesLayer', () => {
  it('ne trace rien sans aucun trait', () => {
    const { ops, ctx } = recordingContext()
    drawReliefLinesLayer(ctx, input(), project)
    expect(ops.filter((o) => ['beginPath', 'stroke', 'moveTo', 'lineTo'].includes(o.op))).toEqual([])
  })

  it('trace les sommets par pixel -> monde -> projection fournie', () => {
    const { ops, ctx } = recordingContext()
    drawReliefLinesLayer(ctx, input({ steps: segment(4, 6, 30, 12) }), project)
    const a = project(pixelToWorld(CAL, 4, 6))
    const b = project(pixelToWorld(CAL, 30, 12))
    expect(ops.find((o) => o.op === 'moveTo')).toEqual({ op: 'moveTo', args: [a.x, a.y] })
    expect(ops.find((o) => o.op === 'lineTo')).toEqual({ op: 'lineTo', args: [b.x, b.y] })
  })

  it('chaque famille est UN chemin et un seul stroke, à son encre, son opacité et son épaisseur', () => {
    const { ops, ctx } = recordingContext()
    drawReliefLinesLayer(
      ctx,
      input({ steps: segment(0, 0, 5, 5), contours: segment(1, 1, 6, 6), walls: segment(2, 2, 7, 7), outline: segment(3, 3, 8, 8) }),
      project,
    )
    expect(ops.filter((o) => o.op === 'beginPath')).toHaveLength(4)
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(4)
    // Du plus discret au plus net : courbes, murs de la coupe, ruptures, liseré.
    expect(ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).toEqual(['C', 'W', 'E', 'O'])
    expect(ops.filter((o) => o.op === 'set globalAlpha').map((o) => o.args[0])).toEqual([0.35, 0.85, 0.7, 1])
    expect(ops.filter((o) => o.op === 'set lineWidth').map((o) => o.args[0])).toEqual([0.75, 1.1, 1, 1.6])
  })

  it('une famille vide est sautée : ni chemin ni trait pour elle', () => {
    const { ops, ctx } = recordingContext()
    drawReliefLinesLayer(ctx, input({ steps: segment(0, 0, 5, 5) }), project)
    expect(ops.filter((o) => o.op === 'beginPath')).toHaveLength(1)
    expect(ops.filter((o) => o.op === 'set strokeStyle').map((o) => o.args[0])).toEqual(['E'])
  })

  it('le liseré est FERMÉ, les autres familles ne le sont pas', () => {
    const { ops, ctx } = recordingContext()
    drawReliefLinesLayer(ctx, input({ outline: segment(0, 0, 5, 0), steps: segment(0, 0, 5, 5) }), project)
    expect(ops.filter((o) => o.op === 'closePath')).toHaveLength(1)
  })

  it('rend le contexte dans l’état d’avant (save puis restore)', () => {
    const { ops, ctx } = recordingContext()
    drawReliefLinesLayer(ctx, input({ steps: segment(0, 0, 5, 5) }), project)
    expect(ops.filter((o) => o.op === 'save')).toHaveLength(1)
    expect(ops.filter((o) => o.op === 'restore')).toHaveLength(1)
  })
})
