/**
 * orientation.test.ts — la rotation de la carte par quarts de tour : aller-retour exact aux
 * quatre positions, sens antihoraire, bornes tournées, caps, grille de chaleur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'

import type { HeatGrid } from '../../../lib/replay/heatPaint'
import {
  normalizeTurns,
  orientBounds,
  orientHeatGrid,
  orientWorld,
  pivotOf,
  QUARTER_TURNS,
  shiftHeadingDeg,
  turnsToDegrees,
  unorientWorld,
  viewOrientation,
  type QuarterTurns,
} from './orientation'

// Des valeurs dyadiques : les opérations sont exactes en virgule flottante, l'égalité est stricte.
const BOUNDS: ReplayBounds = { minX: -8, maxX: 24, minY: -4, maxY: 12, minZ: -2, maxZ: 6 }
const POINTS = [
  { x: 0, y: 0 },
  { x: 24, y: 12 },
  { x: -8, y: -4 },
  { x: 3.5, y: -1.25 },
  { x: 16, y: 4 },
]

describe('orientWorld / unorientWorld', () => {
  it.each(QUARTER_TURNS)('aller-retour EXACT à %i quart(s) de tour', (turns) => {
    for (const p of POINTS) {
      const there = orientWorld(p, turns, BOUNDS)
      expect(unorientWorld(there, turns, BOUNDS)).toEqual(p)
      // Et l'autre sens : défaire puis refaire.
      expect(orientWorld(unorientWorld(p, turns, BOUNDS), turns, BOUNDS)).toEqual(p)
    }
  })

  it('à 0 quart de tour, la position ne bouge pas', () => {
    for (const p of POINTS) expect(orientWorld(p, 0, BOUNDS)).toEqual(p)
  })

  it('tourne dans le sens ANTIHORAIRE autour du centre : l’est passe au nord', () => {
    const c = pivotOf(BOUNDS)
    const est = { x: c.x + 10, y: c.y }
    expect(orientWorld(est, 1, BOUNDS)).toEqual({ x: c.x, y: c.y + 10 })
    expect(orientWorld(est, 2, BOUNDS)).toEqual({ x: c.x - 10, y: c.y })
    expect(orientWorld(est, 3, BOUNDS)).toEqual({ x: c.x, y: c.y - 10 })
  })

  it('le centre est le point fixe de toutes les rotations', () => {
    const c = pivotOf(BOUNDS)
    for (const t of QUARTER_TURNS) expect(orientWorld(c, t, BOUNDS)).toEqual(c)
  })

  it('quatre quarts de tour successifs ramènent le point', () => {
    let p = { x: 3.5, y: -1.25 }
    for (let i = 0; i < 4; i++) p = orientWorld(p, 1, BOUNDS)
    expect(p).toEqual({ x: 3.5, y: -1.25 })
  })
})

describe('orientBounds', () => {
  it('à 0 ou 2 quarts de tour, mêmes dimensions ; à 1 ou 3, largeur et hauteur échangées', () => {
    expect(orientBounds(BOUNDS, 0)).toEqual(BOUNDS)
    expect(orientBounds(BOUNDS, 2)).toEqual(BOUNDS)
    const t = orientBounds(BOUNDS, 1)
    expect(t.maxX - t.minX).toBe(BOUNDS.maxY - BOUNDS.minY)
    expect(t.maxY - t.minY).toBe(BOUNDS.maxX - BOUNDS.minX)
    expect(orientBounds(BOUNDS, 3)).toEqual(t)
  })

  it('garde le centre et l’amplitude verticale', () => {
    for (const turns of QUARTER_TURNS) {
      const b = orientBounds(BOUNDS, turns)
      expect(pivotOf(b)).toEqual(pivotOf(BOUNDS))
      expect([b.minZ, b.maxZ]).toEqual([-2, 6])
    }
  })

  it('est l’emprise des quatre coins tournés', () => {
    const corners = [
      { x: BOUNDS.minX, y: BOUNDS.minY },
      { x: BOUNDS.maxX, y: BOUNDS.minY },
      { x: BOUNDS.maxX, y: BOUNDS.maxY },
      { x: BOUNDS.minX, y: BOUNDS.maxY },
    ]
    for (const turns of QUARTER_TURNS) {
      const turned = corners.map((p) => orientWorld(p, turns, BOUNDS))
      const b = orientBounds(BOUNDS, turns)
      expect(Math.min(...turned.map((p) => p.x))).toBe(b.minX)
      expect(Math.max(...turned.map((p) => p.x))).toBe(b.maxX)
      expect(Math.min(...turned.map((p) => p.y))).toBe(b.minY)
      expect(Math.max(...turned.map((p) => p.y))).toBe(b.maxY)
    }
  })
})

describe('viewOrientation, normalizeTurns, degrés, caps', () => {
  it('n’existe pas à 0 quart de tour, porte le pivot sinon', () => {
    expect(viewOrientation(0, BOUNDS)).toBeUndefined()
    expect(viewOrientation(1, BOUNDS)).toEqual({ turns: 1, pivot: pivotOf(BOUNDS) })
  })

  it('normalise tout entier dans 0..3 et refuse le reste', () => {
    expect([0, 1, 2, 3, 4, 5, -1, -4, 7].map(normalizeTurns)).toEqual([0, 1, 2, 3, 0, 1, 3, 0, 3])
    expect(normalizeTurns(1.5)).toBe(0)
    expect(normalizeTurns(Number.NaN)).toBe(0)
  })

  it('affiche 0, 90, 180, 270 degrés', () => {
    expect(QUARTER_TURNS.map(turnsToDegrees)).toEqual([0, 90, 180, 270])
  })

  it('un cap gagne 90 degrés par quart de tour, comme la position tourne', () => {
    const c = pivotOf(BOUNDS)
    for (const turns of QUARTER_TURNS) {
      // Un point à l'est du centre : après rotation il est dans la direction du cap `turns x 90`.
      const au = orientWorld({ x: c.x + 4, y: c.y }, turns, BOUNDS)
      const rad = (shiftHeadingDeg(0, turns) * Math.PI) / 180
      expect(Math.atan2(au.y - c.y, au.x - c.x)).toBeCloseTo(Math.atan2(Math.sin(rad), Math.cos(rad)), 12)
    }
  })
})

describe('orientHeatGrid', () => {
  /** Une grille 3 x 2 de cellules de 4 m, coin (-8, -4), valeurs distinctes. */
  const grid: HeatGrid = {
    cell: 4,
    nx: 3,
    ny: 2,
    minX: -8,
    minY: -4,
    mode: 'presence',
    value: Float32Array.from([1, 2, 3, 4, 5, 6]),
    lo: 1,
    hi: 6,
    filled: 6,
  }
  const pivot = pivotOf(BOUNDS)

  /** La valeur de la cellule qui contient un point monde, ou null hors grille. */
  function valueAt(g: HeatGrid, p: { x: number; y: number }): number | null {
    const col = Math.floor((p.x - g.minX) / g.cell)
    const row = Math.floor((p.y - g.minY) / g.cell)
    if (col < 0 || row < 0 || col >= g.nx || row >= g.ny) return null
    return g.value[row * g.nx + col]
  }

  it('rend la grille elle-même à 0 quart de tour', () => {
    expect(orientHeatGrid(grid, 0, pivot)).toBe(grid)
  })

  it.each([1, 2, 3] as QuarterTurns[])('à %i quart(s) de tour, chaque cellule suit son centre tourné', (turns) => {
    const turned = orientHeatGrid(grid, turns, pivot)
    expect(turned.cell).toBe(grid.cell)
    expect(turned.nx * turned.ny).toBe(grid.nx * grid.ny)
    for (let row = 0; row < grid.ny; row++) {
      for (let col = 0; col < grid.nx; col++) {
        const centre = { x: grid.minX + (col + 0.5) * grid.cell, y: grid.minY + (row + 0.5) * grid.cell }
        expect(valueAt(turned, orientWorld(centre, turns, BOUNDS))).toBe(grid.value[row * grid.nx + col])
      }
    }
  })

  it('à 1 quart de tour, les dimensions s’échangent ; à 2, elles restent', () => {
    expect([orientHeatGrid(grid, 1, pivot).nx, orientHeatGrid(grid, 1, pivot).ny]).toEqual([2, 3])
    expect([orientHeatGrid(grid, 2, pivot).nx, orientHeatGrid(grid, 2, pivot).ny]).toEqual([3, 2])
  })
})
