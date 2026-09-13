/**
 * _positionsHeat.test.ts — la grille de chaleur des positions, calée sur le fond de carte.
 *
 * Ce que ces tests verrouillent : le cadre monde déduit du calage, le pas (plancher 0,5 m),
 * le dépôt avec la ligne 0 EN HAUT, le rejet (jamais le rabattement) des positions hors
 * cadre, l'échelle quantile et son repli dégénéré, la couverture publiée en pied.
 */
import { describe, expect, it } from 'vitest'

import type { MatchPlayerPosition, ReplayMapBackgroundCalibration } from '@/lib/api/types'

import {
  buildPositionsGrid,
  coveredShare,
  hasTeamSplit,
  mapFrame,
  positionsCellSize,
} from './_positionsHeat'

/** Calage synthétique : 1 px = 1 m, image 10×10, coin monde (0, 10). */
const CAL: ReplayMapBackgroundCalibration = {
  metersPerPixel: 1,
  originX: 0,
  originY: 10,
  widthPx: 10,
  heightPx: 10,
  convention: 'test',
} as ReplayMapBackgroundCalibration

const FRAME = mapFrame(CAL)

function at(x: number, y: number, team = -1): MatchPlayerPosition {
  return { timeMs: 0, x, y, z: 0, team }
}

describe('mapFrame / positionsCellSize', () => {
  it('déduit le cadre monde du calage', () => {
    expect(FRAME).toEqual({ originX: 0, originY: 10, widthM: 10, heightM: 10 })
  })

  it('tient le plancher (rayon d’engagement, 2 m) sur une carte de taille normale', () => {
    expect(positionsCellSize(FRAME)).toBe(2)
  })

  it('agrandit le pas sur une carte qui dépasserait le plafond de cellules', () => {
    expect(positionsCellSize({ ...FRAME, widthM: 40000, heightM: 40000 })).toBeGreaterThan(2)
  })
})

describe('buildPositionsGrid', () => {
  it('rend null quand rien ne tombe sur le plan', () => {
    expect(buildPositionsGrid([], FRAME)).toBeNull()
    expect(buildPositionsGrid([at(-100, -100)], FRAME)).toBeNull()
  })

  it('dépose dans la cellule du plan, ligne 0 EN HAUT', () => {
    // (0, 10) = coin haut-gauche du fond -> (col 0, row 0). (9,5 ; 0,5) -> coin bas-droit.
    const grid = buildPositionsGrid([at(0.1, 9.9), at(9.9, 0.1)], FRAME)
    expect(grid?.nx).toBe(5)
    expect(grid?.ny).toBe(5)
    expect(grid?.cells).toEqual(
      expect.arrayContaining([
        { col: 0, row: 0, value: 1 },
        { col: 4, row: 4, value: 1 },
      ]),
    )
  })

  it('cumule les positions d’une même cellule', () => {
    const grid = buildPositionsGrid([at(1, 5), at(1.2, 4.9)], FRAME)
    expect(grid?.cells).toEqual([{ col: 0, row: 2, value: 2 }])
    expect(grid?.filled).toBe(1)
  })

  it('IGNORE une position hors cadre plutôt que la rabattre sur un bord', () => {
    const grid = buildPositionsGrid([at(1, 5), at(50, 50)], FRAME)
    expect(grid?.cells).toHaveLength(1)
  })

  it('échelle dégénérée (toutes les cellules égales) : repli sur [0, max]', () => {
    const grid = buildPositionsGrid([at(1, 5), at(9, 5)], FRAME)
    expect(grid?.lo).toBe(0)
    expect(grid?.hi).toBe(1)
  })

  it('échelle quantile quand les cellules diffèrent', () => {
    const dense = Array.from({ length: 20 }, () => at(1, 5))
    const grid = buildPositionsGrid([...dense, at(5, 5), at(9, 5)], FRAME)
    expect(grid?.lo).toBe(1)
    expect(grid?.hi).toBe(20)
  })
})

describe('hasTeamSplit / coveredShare', () => {
  it('détecte un camp attribué', () => {
    expect(hasTeamSplit([at(1, 1), at(2, 2)])).toBe(false)
    expect(hasTeamSplit([at(1, 1), at(2, 2, 1)])).toBe(true)
  })

  it('publie la part des positions qui tombent sur le plan', () => {
    expect(coveredShare([], FRAME)).toBe(0)
    expect(coveredShare([at(1, 5), at(50, 50)], FRAME)).toBe(0.5)
  })
})
