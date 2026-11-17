/**
 * orientation.test.ts — la rotation de la carte par quarts de tour : aller-retour exact aux
 * quatre positions, sens antihoraire, bornes tournées, caps, grille de chaleur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds } from '@/lib/api/types'

import type { HeatGrid } from './heatPaint'
import {
  autoTurns,
  normalizeTurns,
  orientBounds,
  orientAbout,
  orientHeatGrid,
  pivotOf,
  cssRotation,
  QUARTER_TURNS,
  rotateCanvasAbout,
  shiftHeadingDeg,
  turnsToDegrees,
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

describe('orientAbout — le sens et le pivot', () => {
  const c = pivotOf(BOUNDS)

  it('à 0 quart de tour, la position ne bouge pas', () => {
    for (const p of POINTS) expect(orientAbout(p, 0, c)).toEqual(p)
  })

  it('tourne dans le sens HORAIRE autour du centre : le nord passe à droite', () => {
    const nord = { x: c.x, y: c.y + 10 }
    expect(orientAbout(nord, 1, c)).toEqual({ x: c.x + 10, y: c.y })
    expect(orientAbout(nord, 2, c)).toEqual({ x: c.x, y: c.y - 10 })
    expect(orientAbout(nord, 3, c)).toEqual({ x: c.x - 10, y: c.y })
  })

  it('le centre est le point fixe de toutes les rotations', () => {
    for (const t of QUARTER_TURNS) expect(orientAbout(c, t, c)).toEqual(c)
  })

  it('quatre quarts de tour successifs ramènent le point', () => {
    let p = { x: 3.5, y: -1.25 }
    for (let i = 0; i < 4; i++) p = orientAbout(p, 1, c)
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
      const turned = corners.map((p) => orientAbout(p, turns, pivotOf(BOUNDS)))
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

  it('un cap perd 90 degrés par quart de tour, comme la position tourne', () => {
    const c = pivotOf(BOUNDS)
    for (const turns of QUARTER_TURNS) {
      // Un point à l'est du centre : après rotation il est dans la direction du cap `turns x 90`.
      const au = orientAbout({ x: c.x + 4, y: c.y }, turns, c)
      const rad = (shiftHeadingDeg(0, turns) * Math.PI) / 180
      expect(au.x - c.x).toBeCloseTo(4 * Math.cos(rad), 12)
      expect(au.y - c.y).toBeCloseTo(4 * Math.sin(rad), 12)
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
        expect(valueAt(turned, orientAbout(centre, turns, pivot))).toBe(grid.value[row * grid.nx + col])
      }
    }
  })

  it('à 1 quart de tour, les dimensions s’échangent ; à 2, elles restent', () => {
    expect([orientHeatGrid(grid, 1, pivot).nx, orientHeatGrid(grid, 1, pivot).ny]).toEqual([2, 3])
    expect([orientHeatGrid(grid, 2, pivot).nx, orientHeatGrid(grid, 2, pivot).ny]).toEqual([3, 2])
  })
})

describe('l’application de la rotation à un dessin', () => {
  it('la toile se translate au pivot puis tourne de turns x 90 degrés horaires', () => {
    const ops: Array<[string, ...number[]]> = []
    const ctx = {
      translate: (x: number, y: number) => ops.push(['translate', x, y]),
      rotate: (a: number) => ops.push(['rotate', a]),
    }
    rotateCanvasAbout(ctx, 3, 40, 25)
    expect(ops).toEqual([
      ['translate', 40, 25],
      ['rotate', (3 * Math.PI) / 2],
    ])
  })

  it.each([
    [1, 'rotate(90deg)'],
    [2, 'rotate(180deg)'],
    [3, 'rotate(270deg)'],
  ] as Array<[QuarterTurns, string]>)('le CSS à %i quart(s) de tour : %s', (turns, css) => {
    expect(cssRotation(turns)).toBe(css)
  })
})

describe('autoTurns — la règle Auto, par surface ajustée à la boîte (D-7, D-14)', () => {
  // Boîte du Standard et du Cinéma à 1440 x 900, moins la barre : 1390 x 700.
  const BOITE = { largeur: 1390, hauteur: 700 }
  const RONIN = { largeur: 659, hauteur: 1288 }
  const NADAIR = { largeur: 2133, hauteur: 947 }
  const DETACHMENT = { largeur: 1526, hauteur: 1666 }
  // Gain de 22,1 % : la boîte 1105 x 1000 et la carte 100 x 200 donnent 1,221 fois la surface droite.
  const CARTE_22 = { largeur: 100, hauteur: 200 }
  const BOITE_22 = { largeur: 1105, hauteur: 1000 }

  it('Ronin (659 x 1288, gain de plus de 25 %) tourne quand il est droit', () => {
    expect(autoTurns(RONIN, BOITE, false)).toBe(1)
  })

  it('Nadair (2133 x 947) ne tourne pas : posé tourné, il tient moins bien', () => {
    expect(autoTurns(NADAIR, BOITE, false)).toBe(0)
  })

  it('Detachment (1526 x 1666, gain de 19 %) ne tourne pas quand il est droit', () => {
    expect(autoTurns(DETACHMENT, BOITE, false)).toBe(0)
  })

  it('Detachment déjà tourné (gain de 19 %, sous 20 %) revient droit', () => {
    expect(autoTurns(DETACHMENT, BOITE, true)).toBe(0)
  })

  it('gain de 22 % : ne tourne pas quand il est droit (seuil 25 %)', () => {
    expect(autoTurns(CARTE_22, BOITE_22, false)).toBe(0)
  })

  it('gain de 22 % : reste tourné quand il l’est déjà (retour seulement sous 20 %)', () => {
    expect(autoTurns(CARTE_22, BOITE_22, true)).toBe(1)
  })

  it('un plan déjà tourné et très gagnant (Ronin) reste tourné', () => {
    expect(autoTurns(RONIN, BOITE, true)).toBe(1)
  })

  it('carte de taille nulle : 0, sans division par zéro, tourné ou non', () => {
    expect(autoTurns({ largeur: 0, hauteur: 0 }, BOITE, false)).toBe(0)
    expect(autoTurns({ largeur: 0, hauteur: 1288 }, BOITE, true)).toBe(0)
  })

  it('boîte de taille nulle : 0, sans division par zéro, tourné ou non', () => {
    expect(autoTurns(RONIN, { largeur: 0, hauteur: 0 }, false)).toBe(0)
    expect(autoTurns(RONIN, { largeur: 1390, hauteur: 0 }, true)).toBe(0)
  })
})
