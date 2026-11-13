/**
 * replayViewOrient.test.ts — le cadrage tourné : `projectTo` applique la rotation avant de
 * projeter, `projectFrameTo` non, et un calque cuit pour une orientation ne se réutilise pas pour
 * une autre.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'

import { worldToCanvas } from '../../../lib/replay/replayLogic'
import { orientBounds, orientWorld, QUARTER_TURNS, viewOrientation, type QuarterTurns } from './orientation'
import { cookIsReusable, projectFrameTo, projectTo, scaleOf, turnsOfView, type CanvasView } from './replayView'

const SCENE: ReplayBounds = { minX: -20, maxX: 60, minY: 0, maxY: 40 }

/** Le cadrage de la carte entière, tournée de `turns` quarts de tour. */
function viewAt(turns: QuarterTurns, dx = 0): CanvasView {
  const b = orientBounds(SCENE, turns)
  return {
    bounds: { ...b, minX: b.minX + dx, maxX: b.maxX + dx },
    width: 640,
    height: 480,
    pad: 16,
    orient: viewOrientation(turns, SCENE),
  }
}

describe('projectTo — carte tournée', () => {
  it.each(QUARTER_TURNS)('à %i quart(s) de tour : rotation puis worldToCanvas', (turns) => {
    const v = viewAt(turns)
    for (const p of [{ x: -20, y: 0 }, { x: 60, y: 40 }, { x: 5, y: 30 }]) {
      const attendu = worldToCanvas(orientWorld(p, turns, SCENE), v.bounds, v.width, v.height, v.pad)
      expect(projectTo(v, p)).toEqual(attendu)
    }
  })

  it('sans orientation, la projection est celle d’avant', () => {
    const v: CanvasView = { bounds: SCENE, width: 640, height: 480, pad: 16 }
    expect(projectTo(v, { x: 5, y: 30 })).toEqual(worldToCanvas({ x: 5, y: 30 }, SCENE, 640, 480, 16))
    expect(turnsOfView(v)).toBe(0)
  })

  it('l’échelle est celle du cadre tourné : un quart de tour change l’ajustement à la toile', () => {
    // 80 x 40 m dans 608 x 448 px utiles : la largeur limite (7,6 px/m) ; tourné, 40 x 80 : la hauteur (5,6).
    expect(scaleOf(viewAt(0))).toBeCloseTo(608 / 80, 9)
    expect(scaleOf(viewAt(1))).toBeCloseTo(448 / 80, 9)
    expect(scaleOf(viewAt(2))).toBeCloseTo(scaleOf(viewAt(0)), 9)
  })

  it('projectFrameTo ne tourne pas : elle lit un point déjà dans le repère tourné', () => {
    const v = viewAt(1)
    const p = { x: 5, y: 30 }
    expect(projectFrameTo(v, orientWorld(p, 1, SCENE))).toEqual(projectTo(v, p))
  })
})

describe('cookIsReusable — l’orientation', () => {
  it('un calque cuit pour la même orientation se réutilise décalé', () => {
    expect(cookIsReusable(viewAt(1), viewAt(1, 0.5))).toBe(true)
  })

  it('un calque cuit pour une autre orientation ne se réutilise jamais', () => {
    for (const a of QUARTER_TURNS) {
      for (const b of QUARTER_TURNS) {
        if (a === b) continue
        expect(cookIsReusable(viewAt(a), viewAt(b))).toBe(false)
      }
    }
  })
})
