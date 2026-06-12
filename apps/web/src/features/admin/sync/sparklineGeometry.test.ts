import { describe, expect, it } from 'vitest'

import { lastPoint, sparklinePoints } from './sparklineGeometry'

describe('sparklinePoints', () => {
  it('série vide → chaîne vide', () => {
    expect(sparklinePoints([], 80, 24)).toBe('')
  })

  it('point unique → ligne médiane horizontale', () => {
    expect(sparklinePoints([5], 80, 24)).toBe('0,12 80,12')
  })

  it('série constante → ligne médiane (range = 0)', () => {
    const pts = sparklinePoints([3, 3, 3], 80, 24)
    // 3 points, tous à y médian (12), x à 0 / 40 / 80.
    expect(pts).toBe('0,12 40,12 80,12')
  })

  it('croissante : premier point en bas, dernier en haut (origine SVG inversée)', () => {
    const pts = sparklinePoints([0, 10], 100, 20, 0).split(' ')
    const [x0, y0] = pts[0].split(',').map(Number)
    const [x1, y1] = pts[1].split(',').map(Number)
    expect(x0).toBe(0)
    expect(x1).toBe(100)
    expect(y0).toBeGreaterThan(y1) // min en bas (grand y), max en haut (petit y)
    expect(y1).toBe(0) // max → haut, pad=0
  })

  it('répartit x uniformément sur la largeur', () => {
    const pts = sparklinePoints([1, 2, 3, 4, 5], 80, 24).split(' ')
    const xs = pts.map((p) => Number(p.split(',')[0]))
    expect(xs).toEqual([0, 20, 40, 60, 80])
  })
})

describe('lastPoint', () => {
  it('série vide → null', () => {
    expect(lastPoint([], 80, 24)).toBeNull()
  })

  it('retourne le dernier point (valeur courante)', () => {
    const lp = lastPoint([0, 10], 100, 20, 0)
    expect(lp).toEqual({ x: 100, y: 0 })
  })
})
