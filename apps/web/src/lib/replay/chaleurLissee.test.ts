/**
 * chaleurLissee — la chaleur lissée garde l'UNITÉ de la lecture (moyenne pondérée, pas densité) et
 * ne s'étend qu'à quelques dixièmes de pas au-delà des cellules servies.
 */
import { describe, expect, it } from 'vitest'

import { buildTacticalGrid, type TacticalCell, type TacticalGrid } from './heatPaint'

import { flouDeLaChaleur, lisserLaGrille, peindreLaChaleurLissee, sousDivision } from './chaleurLissee'

function grille(cells: TacticalCell[], opts: { signee?: boolean } = {}): TacticalGrid {
  return buildTacticalGrid(
    cells,
    { cell: 2, nx: 10, ny: 10, minX: 0, minY: 0 },
    { lo: 1, hi: 5, signee: opts.signee, borne: 5 },
    cells.length,
  )
}

/** La valeur lissée de la sous-cellule qui contient le point (x, y) exprimé en cellules servies. */
function valeurEn(g: TacticalGrid, x: number, y: number): number | undefined {
  const sous = Math.round(2 / g.cell)
  const col = Math.floor(x * sous)
  const row = Math.floor(y * sous)
  return g.cells.find((c) => c.col === col && c.row === row)?.value
}

describe('lisserLaGrille', () => {
  it('une cellule isolée : sa valeur en son centre, rien à plus d’un pas', () => {
    const lissee = lisserLaGrille(grille([{ col: 4, row: 4, value: 3 }]))!
    expect(lissee.cell).toBeLessThan(2)
    expect(valeurEn(lissee, 4.5, 4.5)).toBeCloseTo(3, 6)
    expect(valeurEn(lissee, 6.5, 4.5)).toBeUndefined()
    expect(valeurEn(lissee, 4.5, 2.5)).toBeUndefined()
  })

  it('la tache d’une cellule isolée reste de l’ordre de son aire (disque de 0,7 pas de rayon)', () => {
    const sous = sousDivision(10, 10)
    const lissee = lisserLaGrille(grille([{ col: 4, row: 4, value: 3 }]))!
    const aire = lissee.cells.length / (sous * sous)
    expect(aire).toBeGreaterThan(0.8)
    expect(aire).toBeLessThan(1.8)
  })

  it('entre deux cellules voisines : une valeur ENTRE les deux, jamais au-delà (unité gardée)', () => {
    const lissee = lisserLaGrille(grille([{ col: 4, row: 4, value: 1 }, { col: 5, row: 4, value: 5 }]))!
    const milieu = valeurEn(lissee, 5.0, 4.5)!
    expect(milieu).toBeGreaterThan(1)
    expect(milieu).toBeLessThan(5)
    for (const c of lissee.cells) {
      expect(c.value).toBeGreaterThanOrEqual(1 - 1e-9)
      expect(c.value).toBeLessThanOrEqual(5 + 1e-9)
    }
  })

  it('une lecture signée garde ses deux signes et son échelle', () => {
    const lissee = lisserLaGrille(grille([{ col: 2, row: 2, value: -4 }, { col: 7, row: 7, value: 4 }], { signee: true }))!
    expect(valeurEn(lissee, 2.5, 2.5)).toBeCloseTo(-4, 6)
    expect(valeurEn(lissee, 7.5, 7.5)).toBeCloseTo(4, 6)
    expect(lissee.signee).toBe(true)
    expect(lissee.borne).toBe(5)
  })

  it('même cadre : emprise inchangée, pas divisé', () => {
    const g = grille([{ col: 1, row: 1, value: 2 }])
    const lissee = lisserLaGrille(g)!
    const sous = sousDivision(g.nx, g.ny)
    expect(lissee.nx).toBe(g.nx * sous)
    expect(lissee.ny).toBe(g.ny * sous)
    expect(lissee.cell * sous).toBeCloseTo(g.cell, 9)
    expect(lissee.filled).toBe(lissee.cells.length)
  })

  it('rien à peindre : null', () => {
    expect(lisserLaGrille(grille([]))).toBeNull()
  })
})

describe('sousDivision', () => {
  it('jusqu’à 6 sous-cellules par côté, moins sur une grande grille', () => {
    expect(sousDivision(10, 10)).toBe(6)
    expect(sousDivision(400, 400)).toBe(1)
    expect(sousDivision(150, 150)).toBe(3)
  })
})

describe('peindreLaChaleurLissee', () => {
  it('sans filtre de canvas : peint directement sur le contexte, bords nets', () => {
    const appels: string[] = []
    const ctx = {
      canvas: { width: 100, height: 100 },
      fillRect: () => appels.push('fillRect'),
      fillStyle: '',
      globalAlpha: 1,
      save: () => {},
      restore: () => {},
      beginPath: () => {},
      rect: () => {},
      clip: () => {},
    } as unknown as CanvasRenderingContext2D
    const lissee = lisserLaGrille(grille([{ col: 4, row: 4, value: 3 }]))!
    expect(() =>
      peindreLaChaleurLissee(ctx, lissee, { topLeftWorld: { x: 0, y: 0 }, scale: 5 }, ['#000', '#fff'], 0),
    ).not.toThrow()
  })

  it('le flou est proportionnel au pas servi et à l’échelle', () => {
    expect(flouDeLaChaleur(2, 10)).toBeCloseTo(3)
    expect(flouDeLaChaleur(0, 10)).toBe(0)
  })
})
