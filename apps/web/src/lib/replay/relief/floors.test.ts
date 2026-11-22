import { describe, expect, it } from 'vitest'

import { floorRanks, meshShareOf, nearestRank, quantizeCut, rankHeight, snapCut } from './floors'
import { reliefData } from './reliefScene.testutil'

const LEVELS = [0, 2.21, 4.4]

describe('étages en rangs (D-14)', () => {
  it('numérote les étages du plus bas au plus haut : le rang 0 est « Bas »', () => {
    expect(floorRanks([4.4, 0, 2.21])).toEqual([
      { rank: 0, z: 0 },
      { rank: 1, z: 2.21 },
      { rank: 2, z: 4.4 },
    ])
    expect(floorRanks([])).toEqual([])
  })

  it('rankHeight rend la hauteur de coupe d’un rang, null s’il n’existe pas', () => {
    expect(rankHeight(LEVELS, 1)).toBe(2.21)
    expect(rankHeight(LEVELS, 3)).toBeNull()
    expect(rankHeight([], 0)).toBeNull()
  })

  it('nearestRank accroche à l’étage le plus proche dans 0,7 u, sinon rien', () => {
    expect(nearestRank(LEVELS, 2.6)).toBe(1)
    expect(nearestRank(LEVELS, 2.21 + 0.69)).toBe(1)
    expect(nearestRank(LEVELS, 2.21 + 0.71)).toBeNull()
    expect(nearestRank(LEVELS, 1.3)).toBeNull()
    expect(nearestRank([], 1)).toBeNull()
  })
})

describe('accroche de la coupe', () => {
  it('prend l’étage détecté le plus proche à moins de 0,7 u', () => {
    expect(snapCut(LEVELS, 2.6)).toBe(2.21)
    expect(snapCut(LEVELS, 0.69)).toBe(0)
    expect(snapCut(LEVELS, 4.0)).toBe(4.4)
  })

  it('sinon arrondit au quart d’unité', () => {
    expect(snapCut(LEVELS, 1.3)).toBe(1.25)
    expect(snapCut(LEVELS, 1.1)).toBe(1)
    expect(snapCut([], 3.9)).toBe(4)
  })

  it('quantizeCut ne rend jamais −0', () => {
    expect(Object.is(quantizeCut(-0.01), 0)).toBe(true)
    expect(quantizeCut(2.21)).toBe(2.25)
  })
})

describe('part tirée du maillage (D-16)', () => {
  it('lit floors.meshShare du JSON et l’arrondit en pourcentage entier', () => {
    const data = reliefData({ width: 10, height: 10, meshShare: 0.4649, rects: [{ i0: 0, j0: 0, i1: 9, j1: 9, z: 1 }] })
    expect(meshShareOf(data)).toEqual({ share: 0.4649, percent: 46 })
  })

  it('rend null sans maillage, ou quand le JSON ne dit pas la part', () => {
    const none = reliefData({ width: 10, height: 10, degradations: ['sans-maillage'], meshShare: 0.5 })
    expect(meshShareOf(none)).toBeNull()
    const data = reliefData({ width: 10, height: 10, rects: [{ i0: 0, j0: 0, i1: 9, j1: 9, z: 1 }] })
    expect(meshShareOf(data)).toBeNull()
  })

  it('une part de zéro est une part, pas une absence', () => {
    const data = reliefData({ width: 10, height: 10, meshShare: 0, rects: [{ i0: 0, j0: 0, i1: 9, j1: 9, z: 1 }] })
    expect(meshShareOf(data)).toEqual({ share: 0, percent: 0 })
  })
})
