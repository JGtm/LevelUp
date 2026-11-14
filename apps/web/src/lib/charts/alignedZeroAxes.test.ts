import { describe, expect, it } from 'vitest'
import { alignedZeroAxes, niceCeil } from './alignedZeroAxes'

describe('niceCeil', () => {
  it('arrondit au palier joli supérieur', () => {
    expect(niceCeil(0.72)).toBe(0.8)
    expect(niceCeil(1)).toBe(1)
    expect(niceCeil(1.01)).toBe(1.5)
    expect(niceCeil(260)).toBe(300)
    expect(niceCeil(9.1)).toBe(10)
  })
  it('zéro, négatif ou non fini : 0', () => {
    expect(niceCeil(0)).toBe(0)
    expect(niceCeil(-3)).toBe(0)
    expect(niceCeil(Number.NaN)).toBe(0)
  })
})

function zeroShare(b: [number, number]): number {
  return -b[0] / (b[1] - b[0])
}

describe('alignedZeroAxes', () => {
  it('les deux axes placent le zéro à la même hauteur', () => {
    const { left, right } = alignedZeroAxes({ neg: 0.7, pos: 1.2 }, { neg: 250, pos: 900 })
    expect(zeroShare(left)).toBeCloseTo(zeroShare(right), 10)
  })
  it('contient toutes les valeurs de chaque axe', () => {
    const { left, right } = alignedZeroAxes({ neg: 0.7, pos: 1.2 }, { neg: 250, pos: 900 })
    expect(left[0]).toBeLessThanOrEqual(-0.7)
    expect(left[1]).toBeGreaterThanOrEqual(1.2)
    expect(right[0]).toBeLessThanOrEqual(-250)
    expect(right[1]).toBeGreaterThanOrEqual(900)
  })
  it("un axe de droite sans valeur négative suit tout de même le zéro de l'axe de gauche", () => {
    const { left, right } = alignedZeroAxes({ neg: 1, pos: 1 }, { neg: 0, pos: 600 })
    expect(zeroShare(left)).toBeCloseTo(zeroShare(right), 10)
    expect(right[0]).toBeLessThan(0)
    expect(right[1]).toBeGreaterThanOrEqual(600)
  })
  it('rien sous le zéro : axes qui partent de zéro', () => {
    const { left, right } = alignedZeroAxes({ neg: 0, pos: 1.2 }, { neg: 0, pos: 640 })
    expect(left[0]).toBeCloseTo(0, 10)
    expect(left[1]).toBe(1.5)
    expect(right[0]).toBeCloseTo(0, 10)
    expect(right[1]).toBe(800)
  })
  it('étendues nulles : bornes de repli', () => {
    const { left, right } = alignedZeroAxes({ neg: 0, pos: 0 }, { neg: 0, pos: 0 })
    expect(left[1]).toBeGreaterThan(0)
    expect(right[1]).toBeGreaterThan(0)
  })
})
