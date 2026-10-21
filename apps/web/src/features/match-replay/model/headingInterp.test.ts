import { describe, expect, it } from 'vitest'

import { HEADING_INTERP_MAX_SPAN_FRAMES, headingTowardNext, lerpHeadingDeg } from './headingInterp'

describe('lerpHeadingDeg — le plus court chemin sur le cercle', () => {
  it('interpole linéairement quand le chemin ne traverse pas 0/360', () => {
    expect(lerpHeadingDeg(10, 50, 0.5)).toBeCloseTo(30, 9)
    expect(lerpHeadingDeg(10, 50, 0)).toBeCloseTo(10, 9)
    expect(lerpHeadingDeg(10, 50, 1)).toBeCloseTo(50, 9)
  })

  it('traverse 0/360 quand c est le plus court, dans les deux sens', () => {
    expect(lerpHeadingDeg(350, 10, 0.5)).toBeCloseTo(0, 9)
    expect(lerpHeadingDeg(10, 350, 0.5)).toBeCloseTo(0, 9)
    expect(lerpHeadingDeg(350, 30, 0.5)).toBeCloseTo(10, 9)
    expect(lerpHeadingDeg(30, 350, 0.5)).toBeCloseTo(10, 9)
  })

  it('rend un cap dans [0, 360)', () => {
    for (const [a, b, f] of [[350, 10, 0.2], [10, 350, 0.9], [0, 0, 0.5], [359, 1, 0.5]] as const) {
      const h = lerpHeadingDeg(a, b, f)
      expect(h).toBeGreaterThanOrEqual(0)
      expect(h).toBeLessThan(360)
    }
  })
})

describe('headingTowardNext', () => {
  const pts = [
    { t: 0, h: 0 },
    { t: 2, h: 20 },
  ]

  it('tourne vers la lecture suivante', () => {
    expect(headingTowardNext(pts, 0, 1)).toBeCloseTo(10, 9)
  })

  it('tient le cap sans lecture suivante', () => {
    expect(headingTowardNext(pts, 1, 5)).toBe(20)
  })

  it('tient le cap au-delà de la portée', () => {
    const loin = [{ t: 0, h: 0 }, { t: HEADING_INTERP_MAX_SPAN_FRAMES + 1, h: 90 }]
    expect(headingTowardNext(loin, 0, 3)).toBe(0)
    const limite = [{ t: 0, h: 0 }, { t: HEADING_INTERP_MAX_SPAN_FRAMES, h: 90 }]
    expect(headingTowardNext(limite, 0, 5)).toBeCloseTo(45, 9)
  })
})
