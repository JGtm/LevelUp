import { describe, expect, it } from 'vitest'

import type { ReplayBounds, ReplaySurface } from '@/lib/api/types'

import {
  altitudeTint,
  buildFloorGrid,
  floorRun,
  FLOOR_CELL_M,
  hasEdge,
  pointInPolygon,
} from './mapFloor'

/** Carré de 4 m de côté à l'altitude z, centré sur l'origine du repère de test. */
function slab(x0: number, y0: number, x1: number, y1: number, z: number): ReplaySurface {
  return { x0, y0, x1, y1, z, zb: z - 1 }
}

const bounds: ReplayBounds = { minX: 0, minY: 0, maxX: 10, maxY: 10, minZ: -2, maxZ: 4 }

describe('buildFloorGrid', () => {
  it('rasterise une emprise et garde son altitude', () => {
    const g = buildFloorGrid([slab(0, 0, 4, 4, 1)], bounds)
    expect(g.cell).toBe(FLOOR_CELL_M)
    expect(g.filled).toBeGreaterThan(0)
    expect(g.topZ[0]).toBe(1)
  })

  it('garde l’altitude LA PLUS HAUTE quand deux emprises se superposent', () => {
    // C'est la définition du sol reconstruit : la surface sur laquelle un joueur se tiendrait.
    const g = buildFloorGrid([slab(0, 0, 4, 4, 1), slab(0, 0, 4, 4, 2.5)], bounds)
    expect(g.topZ[0]).toBe(2.5)
  })

  it('écarte ce qui passe au-dessus de la tête des joueurs', () => {
    // maxZ = 4, marge de 1 m : une poutre à 6 m est un plafond, pas un sol.
    const g = buildFloorGrid([slab(0, 0, 4, 4, 6)], bounds)
    expect(g.filled).toBe(0)
  })

  it('écarte la dalle englobante et le mobilier', () => {
    const enorme = buildFloorGrid([slab(0, 0, 100, 100, 0)], bounds) // 10 000 m² > 4 000
    expect(enorme.filled).toBe(0)
    const minuscule = buildFloorGrid([slab(0, 0, 0.5, 0.5, 0)], bounds) // 0,25 m² < 1
    expect(minuscule.filled).toBe(0)
  })

  it('étalonne la teinte sur les altitudes RECONSTRUITES, pas sur les bornes de la carte', () => {
    // Les sols tiennent entre 0 et 1 alors que bounds couvre −2 à +4 : sans cet étalonnage la
    // carte serait un aplat uniforme.
    const g = buildFloorGrid([slab(0, 0, 4, 4, 0), slab(5, 5, 9, 9, 1)], bounds)
    expect(g.zLo).toBeGreaterThanOrEqual(0)
    expect(g.zHi).toBeLessThanOrEqual(1)
    expect(altitudeTint(g.zLo, g)).toBe(0)
    expect(altitudeTint(g.zHi, g)).toBe(1)
  })

  it('sans aucun sol, la teinte retombe sur les bornes de la carte', () => {
    const g = buildFloorGrid([], bounds)
    expect(g.filled).toBe(0)
    expect(g.zLo).toBe(-2)
    expect(g.zHi).toBe(4)
  })

  it('emploie l’emprise ORIENTÉE quand elle existe', () => {
    // Un losange inscrit dans la boîte 0..4 : le coin (0,0) est HORS du polygone, son centre
    // de cellule n'est donc pas rempli, alors que la boîte alignée l'aurait rempli.
    const s: ReplaySurface = {
      ...slab(0, 0, 4, 4, 1),
      poly: [
        [2, 0],
        [4, 2],
        [2, 4],
        [0, 2],
      ],
    }
    const g = buildFloorGrid([s], bounds)
    expect(Number.isNaN(g.topZ[0])).toBe(true) // coin bas-gauche : vide
    const mid = Math.floor(2 / FLOOR_CELL_M)
    expect(g.topZ[mid * g.nx + mid]).toBe(1) // centre : plein
  })
})

describe('floorRun', () => {
  it('rend 0 sur une cellule vide et la longueur de la plage sinon', () => {
    const g = buildFloorGrid([slab(0, 0, 4, 4, 1)], bounds)
    expect(floorRun(g, 0, 0)).toBe(Math.floor(4 / FLOOR_CELL_M) + 1)
    const dehors = Math.floor(8 / FLOOR_CELL_M)
    expect(floorRun(g, dehors, dehors)).toBe(0)
  })
})

describe('hasEdge', () => {
  const g = buildFloorGrid([slab(0, 0, 4, 4, 1)], bounds)

  it('trace une arête au bord du vide', () => {
    const last = Math.floor(4 / FLOOR_CELL_M) // dernière cellule remplie
    expect(hasEdge(g, last, 0, 'right')).toBe(true)
  })

  it('ne trace rien au milieu d’un sol continu', () => {
    expect(hasEdge(g, 2, 2, 'right')).toBe(false)
    expect(hasEdge(g, 2, 2, 'up')).toBe(false)
  })

  it('trace une arête à une marche franche, pas à une pente douce', () => {
    // 0,50 m > seuil de 0,45 : c'est une marche. 0,20 m : c'est une rampe.
    const marche = buildFloorGrid([slab(0, 0, 2, 4, 1), slab(2, 0, 4, 4, 1.5)], bounds)
    expect(hasEdge(marche, Math.floor(1.75 / FLOOR_CELL_M), 4, 'right')).toBe(true)
    const rampe = buildFloorGrid([slab(0, 0, 2, 4, 1), slab(2, 0, 4, 4, 1.2)], bounds)
    expect(hasEdge(rampe, Math.floor(1.75 / FLOOR_CELL_M), 4, 'right')).toBe(false)
  })

  it('ne trace rien depuis une cellule vide', () => {
    const dehors = Math.floor(8 / FLOOR_CELL_M)
    expect(hasEdge(g, dehors, dehors, 'right')).toBe(false)
  })
})

describe('pointInPolygon', () => {
  const carre: [number, number][] = [
    [0, 0],
    [4, 0],
    [4, 4],
    [0, 4],
  ]
  it('distingue dedans et dehors', () => {
    expect(pointInPolygon(2, 2, carre)).toBe(true)
    expect(pointInPolygon(5, 2, carre)).toBe(false)
    expect(pointInPolygon(2, -1, carre)).toBe(false)
  })
})
