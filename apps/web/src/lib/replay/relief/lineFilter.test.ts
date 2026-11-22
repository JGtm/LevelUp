import { describe, expect, it } from 'vitest'

import { filterLinesByKeep, type KeepGrid } from './lineFilter'
import { LineSetBuilder, emptyLineSet, lineCount, lineSetBuffers, traceLineSet, type PathTarget } from './lineSet'
import type { LineSet } from './types'

/** 40 × 10 pixels, gardés aux colonnes 10 à 19. */
const GRID: KeepGrid = (() => {
  const keep = new Uint8Array(400)
  for (let j = 0; j < 10; j++) for (let i = 10; i <= 19; i++) keep[j * 40 + i] = 1
  return { width: 40, height: 10, keep }
})()
const MPP = 0.1 // un tronçon fait au plus 5 pixels

function lines(...polylines: Array<Array<[number, number]>>): LineSet {
  const b = new LineSetBuilder()
  for (const poly of polylines) {
    for (const [x, y] of poly) b.add(x, y)
    b.end()
  }
  return b.build()
}

/** Les polylignes d'un ensemble, en tableaux de couples arrondis à 0,01. */
function toArrays(set: LineSet): number[][][] {
  const out: number[][][] = []
  for (let k = 0; k < lineCount(set); k++) {
    const poly: number[][] = []
    for (let v = set.offsets[k]; v < set.offsets[k + 1]; v++) {
      poly.push([Math.round(set.points[2 * v] * 100) / 100, Math.round(set.points[2 * v + 1] * 100) / 100])
    }
    out.push(poly)
  }
  return out
}

describe('filterLinesByKeep', () => {
  it('ne garde d’un long segment que les tronçons dont le milieu est gardé', () => {
    const out = filterLinesByKeep(lines([[0, 5], [39, 5]]), GRID, MPP, false)
    // 8 tronçons de 4,875 px ; les milieux 12,2 et 17,1 sont gardés (colonnes 12 et 17), pas 7,3 ni 21,9.
    expect(toArrays(out)).toEqual([[[9.75, 5], [19.5, 5]]])
  })

  it('un segment entièrement hors de la garde disparaît', () => {
    expect(lineCount(filterLinesByKeep(lines([[0, 5], [8, 5]]), GRID, MPP, false))).toBe(0)
  })

  it('un segment entièrement dans la garde reste entier', () => {
    const out = filterLinesByKeep(lines([[11, 2], [18, 2]]), GRID, MPP, false)
    expect(toArrays(out)).toEqual([[[11, 2], [18, 2]]])
  })

  it('une rupture qui tient SUR le sol de l’étage y reste : le masque est aussi lu à 1,5 pixel de part et d’autre', () => {
    // Un mur vertical à la colonne 8 : son milieu n'est pas gardé, le pixel à 1,5 px de lui (colonne 10) l'est.
    const wall = lines([[8, 0], [8, 9]])
    expect(lineCount(filterLinesByKeep(wall, GRID, MPP, false))).toBe(0)
    expect(toArrays(filterLinesByKeep(wall, GRID, MPP, true))).toEqual([[[8, 0], [8, 9]]])
  })

  it('une courbe de niveau ne profite pas de ce voisinage', () => {
    const contour = lines([[8, 0], [8, 9]])
    expect(lineCount(filterLinesByKeep(contour, GRID, MPP, false))).toBe(0)
  })

  it('joint en une seule polyligne les tronçons gardés contigus, y compris d’un segment au suivant', () => {
    // Segment 1 de 0 à 12 : seul le dernier tronçon (8 à 12) est gardé. Segment 2 de 12 à 30 : les deux premiers (12 à 21).
    const out = filterLinesByKeep(lines([[0, 5], [12, 5], [30, 5]]), GRID, MPP, false)
    expect(toArrays(out)).toEqual([[[8, 5], [12, 5], [21, 5]]])
  })

  it('sépare en deux polylignes deux parties gardées que la garde interrompt', () => {
    const keep = GRID.keep.slice()
    for (let j = 0; j < 10; j++) for (let i = 14; i <= 15; i++) keep[j * 40 + i] = 0
    // mpp 0,5 : un tronçon fait un pixel, donc neuf tronçons de 10 à 19.
    const out = filterLinesByKeep(lines([[10, 5], [19, 5]]), { ...GRID, keep }, 0.5, false)
    expect(lineCount(out)).toBe(2)
  })

  it('un point hors de la grille n’est jamais gardé', () => {
    expect(lineCount(filterLinesByKeep(lines([[-20, 5], [-5, 5]]), GRID, MPP, true))).toBe(0)
    expect(lineCount(filterLinesByKeep(lines([[50, 5], [60, 5]]), GRID, MPP, true))).toBe(0)
  })

  it('l’échelle du fond fixe la longueur des tronçons : à 0,5 u, un fond plus fin les rallonge en pixels', () => {
    // mpp 0,05 : un tronçon fait 10 px, donc 4 tronçons de 9,75 px sur 39 px ; milieux 4,9 / 14,6 / 24,4 / 34,1.
    const out = filterLinesByKeep(lines([[0, 5], [39, 5]]), GRID, 0.05, false)
    expect(toArrays(out)).toEqual([[[9.75, 5], [19.5, 5]]])
  })

  it('ne modifie pas l’ensemble d’entrée', () => {
    const input = lines([[0, 5], [39, 5]])
    const before = Array.from(input.points)
    filterLinesByKeep(input, GRID, MPP, true)
    expect(Array.from(input.points)).toEqual(before)
  })
})

describe('LineSetBuilder et traceLineSet', () => {
  it('écarte une polyligne d’un seul sommet et finit la dernière polyligne ouverte', () => {
    const b = new LineSetBuilder()
    b.add(1, 1)
    b.end()
    b.add(2, 2)
    b.add(3, 3)
    const set = b.build()
    expect(lineCount(set)).toBe(1)
    expect(Array.from(set.points)).toEqual([2, 2, 3, 3])
    expect(lineCount(emptyLineSet())).toBe(0)
  })

  it('trace chaque polyligne par la projection de la vue, fermée pour un liseré', () => {
    const calls: string[] = []
    const target: PathTarget = {
      moveTo: (x, y) => calls.push(`M${x},${y}`),
      lineTo: (x, y) => calls.push(`L${x},${y}`),
      closePath: () => calls.push('Z'),
    }
    const set = lines([[0, 0], [2, 0], [2, 2]], [[5, 5], [6, 5]])
    traceLineSet(target, set, (x, y) => ({ x: x * 10, y: y * 10 }))
    expect(calls).toEqual(['M0,0', 'L20,0', 'L20,20', 'M50,50', 'L60,50'])
    calls.length = 0
    traceLineSet(target, set, (x, y) => ({ x, y }), true)
    expect(calls.filter((c) => c === 'Z').length).toBe(2)
  })

  it('expose les buffers à transférer', () => {
    const set = lines([[0, 0], [1, 1]])
    expect(lineSetBuffers(set)).toHaveLength(2)
  })
})
