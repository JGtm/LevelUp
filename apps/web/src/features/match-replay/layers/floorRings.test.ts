/**
 * Tests — floorRings (le langage d'étage partagé par les pions et les objectifs).
 *
 * CE QU'ILS PROTÈGENT : le NOMBRE d'anneaux est l'étage, le premier rayon est un paramètre (les
 * objectifs le posent au-delà de leur anneau de livraison), le pas ne l'est pas, et chaque anneau
 * pâlit d'un cran. Le facteur d'échelle multiplie rayons et trait, jamais le compte.
 */
import { describe, expect, it } from 'vitest'

import {
  drawFloorRings,
  FLOOR_RING_ALPHA,
  FLOOR_RING_ALPHA_DECAY,
  FLOOR_RING_GAP,
  FLOOR_RING_RADIUS,
  floorRingRadius,
  floorInRange,
  FLAT_SPAN,
} from './floorRings'

function ctxEnregistreur() {
  const arcs: { r: number; alpha: number; width: number; ink: string }[] = []
  const ctx = {
    globalAlpha: 1,
    strokeStyle: '',
    lineWidth: 1,
    beginPath: () => {},
    arc(this: { globalAlpha: number; lineWidth: number; strokeStyle: string }, _x: number, _y: number, r: number) {
      arcs.push({ r, alpha: this.globalAlpha, width: this.lineWidth, ink: this.strokeStyle })
    },
    stroke: () => {},
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, arcs }
}

describe('floorRingRadius', () => {
  it('part du premier rayon et avance d’un pas constant', () => {
    expect(floorRingRadius(1)).toBe(FLOOR_RING_RADIUS)
    expect(floorRingRadius(3)).toBeCloseTo(FLOOR_RING_RADIUS + 2 * FLOOR_RING_GAP, 10)
    expect(floorRingRadius(2, 10)).toBeCloseTo(10 + FLOOR_RING_GAP, 10)
  })
})

describe('drawFloorRings', () => {
  it('trace exactement `fl` anneaux, et aucun au sol', () => {
    const { ctx, arcs } = ctxEnregistreur()
    drawFloorRings(ctx, { x: 0, y: 0 }, 2, '#123456')
    expect(arcs.map((a) => a.r)).toEqual([floorRingRadius(1), floorRingRadius(2)])
    expect(arcs.every((a) => a.ink === '#123456')).toBe(true)

    const sol = ctxEnregistreur()
    drawFloorRings(sol.ctx, { x: 0, y: 0 }, 0, '#123456')
    expect(sol.arcs).toHaveLength(0)
  })

  it('pâlit d’un cran par anneau, et rend l’opacité à 1 en sortant', () => {
    const { ctx, arcs } = ctxEnregistreur()
    drawFloorRings(ctx, { x: 0, y: 0 }, 2, '#123456')
    expect(arcs[0].alpha).toBeCloseTo(FLOOR_RING_ALPHA, 10)
    expect(arcs[1].alpha).toBeCloseTo(FLOOR_RING_ALPHA - FLOOR_RING_ALPHA_DECAY, 10)
    expect(ctx.globalAlpha).toBe(1)
  })

  it('le premier rayon est un paramètre, l’échelle multiplie rayons et trait', () => {
    const { ctx, arcs } = ctxEnregistreur()
    drawFloorRings(ctx, { x: 0, y: 0 }, 2, '#123456', { first: 10.8, k: 2 })
    expect(arcs[0].r).toBeCloseTo(10.8 * 2, 10)
    expect(arcs[1].r).toBeCloseTo((10.8 + FLOOR_RING_GAP) * 2, 10)
    expect(arcs[0].width).toBe(2)
  })
})

describe('floorInRange', () => {
  it('range un z dans un étage sur une amplitude connue (0 = sol)', () => {
    const range = { min: 0, max: 30 }
    expect(floorInRange(0, range)).toBe(0)
    expect(floorInRange(15, range)).toBe(1)
    expect(floorInRange(30, range)).toBe(2)
  })

  it('rend le sol sur un terrain plat, quel que soit le z porté', () => {
    // Sans amplitude, `floorOf` seul rendrait l’étage 1 (altitudeRatio = 0,5 par convention).
    expect(floorInRange(7, { min: 0, max: 0 })).toBe(0)
    expect(floorInRange(7, { min: 5, max: 5 + FLAT_SPAN / 2 })).toBe(0)
    expect(floorInRange(7, { min: 12, max: 3 })).toBe(0)
  })
})
