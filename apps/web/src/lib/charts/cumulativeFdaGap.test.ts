/**
 * Tests unitaires du helper canonique `cumulativeFdaGap` (CLAUDE.md n°6) :
 * cumul nominal, report D5 (réel OU attendu manquant, non-fini), vide,
 * tout-manquant ; et `meanFdaGap` (moyenne des gaps finis, null sinon).
 */
import { describe, it, expect } from 'vitest'

import { cumulativeFdaGap, meanFdaGap } from './cumulativeFdaGap'

describe('cumulativeFdaGap', () => {
  it('cumul nominal du différentiel réel − attendu (arrondi 2 décimales)', () => {
    const pts = cumulativeFdaGap([
      { real: 1.5, expected: 1.0 },
      { real: 0.8, expected: 1.2 },
      { real: 2.0, expected: 1.0 },
    ])
    expect(pts.map((p) => p.gap)).toEqual([0.5, -0.4, 1])
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 0.1, 1.1])
    expect(pts.map((p) => p.real)).toEqual([1.5, 0.8, 2])
    expect(pts.map((p) => p.expected)).toEqual([1, 1.2, 1])
  })

  it('report D5 : un match sans attendu ne fait pas avancer le cumul', () => {
    const pts = cumulativeFdaGap([
      { real: 1.5, expected: 1.0 },
      { real: 0.8, expected: null },
      { real: 2.0, expected: 1.0 },
    ])
    expect(pts[1].gap).toBeNull()
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 0.5, 1.5])
  })

  it('report D5 côté réel manquant également', () => {
    const pts = cumulativeFdaGap([
      { real: 1.5, expected: 1.0 },
      { real: null, expected: 1.0 },
    ])
    expect(pts[1].gap).toBeNull()
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 0.5])
  })

  it('valeur non-finie (Infinity) traitée comme absente (D5)', () => {
    const pts = cumulativeFdaGap([
      { real: 1.5, expected: 1.0 },
      { real: 0.8, expected: Infinity },
    ])
    expect(pts[1].gap).toBeNull()
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 0.5])
  })

  it('liste vide → []', () => {
    expect(cumulativeFdaGap([])).toEqual([])
  })

  it('tout-manquant → cumul plat à 0, tous les gaps null', () => {
    const pts = cumulativeFdaGap([
      { real: null, expected: null },
      { real: 1.0, expected: null },
    ])
    expect(pts.map((p) => p.gap)).toEqual([null, null])
    expect(pts.map((p) => p.cumulative)).toEqual([0, 0])
  })
})

describe('meanFdaGap', () => {
  it('moyenne des gaps finis', () => {
    expect(meanFdaGap([{ real: 1.6, expected: 1.0 }, { real: 1.4, expected: 1.0 }])).toBe(0.5)
  })

  it('ignore les paires sans attendu (D5)', () => {
    expect(meanFdaGap([{ real: 1.6, expected: 1.0 }, { real: 0.8, expected: null }])).toBe(0.6)
  })

  it('null si aucune paire exploitable', () => {
    expect(meanFdaGap([{ real: 1.0, expected: null }])).toBeNull()
    expect(meanFdaGap([])).toBeNull()
  })
})
