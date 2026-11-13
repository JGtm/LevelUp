/**
 * mapBackgroundDraw.test.ts — le fond de carte tourne avec la carte : l'image est posée dans le
 * rectangle de son emprise tournée, et ses coins tombent aux pixels que `projectTo` donne.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds, ReplayMapBackgroundCalibration } from '@/lib/api/types'

import { QUARTER_TURNS, viewOrientation, type QuarterTurns } from '../model/orientation'
import { projectTo, type CanvasView } from '../model/replayView'
import { recordingContext } from '../test/recordingContext'
import { worldExtent } from './mapBackground'
import { drawMapBackground, placeMapBackground } from './mapBackgroundDraw'

const CAL: ReplayMapBackgroundCalibration = {
  metersPerPixel: 0.5,
  originX: -10,
  originY: 30,
  widthPx: 80,
  heightPx: 40,
  convention: 'xMonde = originX + (px + 0.5) * metersPerPixel ; yMonde = originY - (py + 0.5) * metersPerPixel',
}
// Une scène plus petite que l'image, centrée sur elle : (-10..30) x (10..30) -> centre (10, 20).
const SCENE: ReplayBounds = { minX: -10, maxX: 30, minY: 10, maxY: 30 }
const IMAGE = { width: 80, height: 40 } as unknown as CanvasImageSource

function viewAt(turns: QuarterTurns): CanvasView {
  return { bounds: SCENE, width: 800, height: 800, pad: 20, orient: viewOrientation(turns, SCENE) }
}

describe('placeMapBackground', () => {
  it('sans fond, ne pose rien', () => {
    expect(placeMapBackground(null, viewAt(0))).toBeNull()
  })

  it('à 0 quart de tour, le rectangle est celui d’avant : coin haut-gauche et dimensions de l’image', () => {
    const v: CanvasView = { bounds: SCENE, width: 800, height: 800, pad: 20 }
    const r = placeMapBackground({ calibration: CAL }, v)
    const ext = worldExtent(CAL)
    const tl = projectTo(v, { x: ext.minX, y: ext.maxY })
    expect(r).toEqual({ x: tl.x, y: tl.y, width: expect.any(Number), height: expect.any(Number) })
    expect(r!.width / r!.height).toBeCloseTo(2, 9)
  })

  it('à 1 et 3 quarts de tour, largeur et hauteur du rectangle sont échangées', () => {
    const droit = placeMapBackground({ calibration: CAL }, viewAt(0))!
    for (const turns of [1, 3] as QuarterTurns[]) {
      const r = placeMapBackground({ calibration: CAL }, viewAt(turns))!
      expect(r.width / r.height).toBeCloseTo(0.5, 9)
      expect(r.width * r.height).toBeCloseTo(droit.width * droit.height, 6)
    }
    const demi = placeMapBackground({ calibration: CAL }, viewAt(2))!
    expect(demi.width).toBeCloseTo(droit.width, 9)
    expect(demi.height).toBeCloseTo(droit.height, 9)
  })
})

describe('drawMapBackground', () => {
  it('sans fond ou sans rectangle, ne peint rien', () => {
    const { ops, ctx } = recordingContext()
    drawMapBackground(ctx, null, { x: 0, y: 0, width: 1, height: 1 }, viewAt(0))
    drawMapBackground(ctx, { calibration: CAL, image: IMAGE }, null, viewAt(0))
    expect(ops).toEqual([])
  })

  it('à 0 quart de tour, un seul drawImage sur le rectangle, sans save ni rotation', () => {
    const { ops, ctx } = recordingContext()
    const r = placeMapBackground({ calibration: CAL }, viewAt(0))!
    drawMapBackground(ctx, { calibration: CAL, image: IMAGE }, r, viewAt(0))
    expect(ops).toEqual([{ op: 'drawImage', args: [IMAGE, r.x, r.y, r.width, r.height] }])
  })

  it.each(QUARTER_TURNS.filter((t) => t !== 0))(
    'à %i quart(s) de tour, les coins de l’image tombent aux pixels de projectTo',
    (turns) => {
      const view = viewAt(turns)
      const { ops, ctx } = recordingContext()
      const r = placeMapBackground({ calibration: CAL }, view)!
      drawMapBackground(ctx, { calibration: CAL, image: IMAGE }, r, view)
      expect(ops.map((o) => o.op)).toEqual(['save', 'translate', 'rotate', 'drawImage', 'restore'])
      const [tx, ty] = ops[1].args as [number, number]
      const angle = ops[2].args[0] as number
      const [, dx, dy, dw, dh] = ops[3].args as [unknown, number, number, number, number]
      // Le point local (lx, ly) de l'image, après `translate` puis `rotate`.
      const place = (lx: number, ly: number) => ({
        x: tx + lx * Math.cos(angle) - ly * Math.sin(angle),
        y: ty + lx * Math.sin(angle) + ly * Math.cos(angle),
      })
      const ext = worldExtent(CAL)
      const hautGauche = projectTo(view, { x: ext.minX, y: ext.maxY })
      const basDroite = projectTo(view, { x: ext.maxX, y: ext.minY })
      const a = place(dx, dy)
      const b = place(dx + dw, dy + dh)
      expect(a.x).toBeCloseTo(hautGauche.x, 6)
      expect(a.y).toBeCloseTo(hautGauche.y, 6)
      expect(b.x).toBeCloseTo(basDroite.x, 6)
      expect(b.y).toBeCloseTo(basDroite.y, 6)
    },
  )
})
