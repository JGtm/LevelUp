import { describe, expect, it } from 'vitest'

import { bearingFloor, faceCount, floorHeightAt, meshWalls, triangleHeight, wallsInBand } from './floorMesh'
import { lineCount } from './lineSet'
import { reliefData, sceneCalibration, type FloorRect } from './reliefScene.testutil'

const cal = sceneCalibration(100, 100)
const meshOf = (rects: FloorRect[]) => reliefData({ width: 100, height: 100, rects }).floors.mesh

describe('triangleHeight', () => {
  const a: [number, number, number] = [0, 0, 0]
  const b: [number, number, number] = [10, 0, 10]
  const c: [number, number, number] = [0, 10, 0]

  it('interpole le plan du triangle et rend null hors du triangle', () => {
    expect(triangleHeight(a, b, c, 5, 2)).toBeCloseTo(5, 9)
    expect(triangleHeight(a, b, c, 9, 9)).toBeNull()
    expect(triangleHeight(a, b, c, -1, 0)).toBeNull()
  })

  it('compte un point sur l’arête partagée', () => {
    expect(triangleHeight(a, b, c, 5, 5)).toBeCloseTo(5, 9)
  })

  it('un triangle dégénéré ne contient rien', () => {
    expect(triangleHeight(a, a, b, 1, 1)).toBeNull()
  })
})

describe('bearingFloor : le sol porteur', () => {
  // Un sol bas partout, un sol haut sur la moitié droite (x de 5 à 10).
  const mesh = meshOf([
    { i0: 0, j0: 0, i1: 99, j1: 99, z: 0 },
    { i0: 50, j0: 0, i1: 99, j1: 99, z: 2.2 },
  ])

  it('prend le plus haut sol qui ne dépasse pas les pieds de plus de 0,35', () => {
    expect(bearingFloor(mesh, 7, 5, 2.2)).toBeCloseTo(2.2, 9)
    expect(bearingFloor(mesh, 7, 5, 2.0)).toBeCloseTo(2.2, 9)
    expect(bearingFloor(mesh, 7, 5, 1.8)).toBeCloseTo(0, 9)
    expect(bearingFloor(mesh, 7, 5, 0.1)).toBeCloseTo(0, 9)
  })

  it('un saut ne le change pas : en l’air, c’est le sol dessous', () => {
    for (const z of [0, 0.3, 0.9, 0.5]) expect(bearingFloor(mesh, 2, 5, z)).toBeCloseTo(0, 9)
  })

  it('rend null hors du maillage et quand aucun sol n’est à portée', () => {
    expect(bearingFloor(mesh, 50, 50, 0)).toBeNull()
    expect(bearingFloor(mesh, -3, 5, 0)).toBeNull()
    expect(bearingFloor(mesh, 2, 5, -1)).toBeNull()
  })

  it('suit la pente d’une rampe', () => {
    const ramp = meshOf([{ i0: 0, j0: 0, i1: 99, j1: 99, z: [0, 2, 2, 0] }])
    expect(bearingFloor(ramp, 5, 5, 1)).toBeCloseTo(1, 9)
    expect(bearingFloor(ramp, 2.5, 5, 0.5)).toBeCloseTo(0.5, 9)
  })

  it('floorHeightAt ne répond que dans la face', () => {
    expect(floorHeightAt(mesh, 1, 7, 5)).toBeCloseTo(2.2, 9)
    expect(floorHeightAt(mesh, 1, 2, 5)).toBeNull()
    expect(faceCount(mesh)).toBe(2)
  })
})

describe('murs du maillage', () => {
  it('un rectangle isolé a quatre murs, à la hauteur moyenne de leurs sommets', () => {
    const walls = meshWalls(meshOf([{ i0: 10, j0: 10, i1: 19, j1: 19, z: [1, 1, 3, 3] }]), cal)
    expect(walls.heights.length).toBe(4)
    expect(Array.from(walls.heights).sort()).toEqual([1, 2, 2, 3])
  })

  it('une arête commune à deux faces n’est pas un mur, même si leurs sommets sont des entrées distinctes de mêmes coordonnées', () => {
    const two = meshWalls(
      meshOf([
        { i0: 0, j0: 0, i1: 9, j1: 9, z: 1 },
        { i0: 10, j0: 0, i1: 19, j1: 9, z: 1 },
      ]),
      cal,
    )
    expect(two.heights.length).toBe(6)
  })

  it('porte les murs en pixels du fond (le bord d’un pixel est à demi-pixel de son centre)', () => {
    const walls = meshWalls(meshOf([{ i0: 10, j0: 20, i1: 19, j1: 29, z: 1 }]), cal)
    const xs: number[] = []
    const ys: number[] = []
    for (let k = 0; k < walls.segments.length; k += 2) {
      xs.push(walls.segments[k])
      ys.push(walls.segments[k + 1])
    }
    expect(Math.min(...xs)).toBeCloseTo(9.5, 4)
    expect(Math.max(...xs)).toBeCloseTo(19.5, 4)
    expect(Math.min(...ys)).toBeCloseTo(19.5, 4)
    expect(Math.max(...ys)).toBeCloseTo(29.5, 4)
  })

  it('wallsInBand garde les murs dont la hauteur est dans la bande de la coupe', () => {
    const walls = meshWalls(
      meshOf([
        { i0: 0, j0: 0, i1: 9, j1: 9, z: 0 },
        { i0: 20, j0: 0, i1: 29, j1: 9, z: 2.2 },
      ]),
      cal,
    )
    expect(lineCount(wallsInBand(walls, 0, 0.7))).toBe(4)
    expect(lineCount(wallsInBand(walls, 2.2, 0.7))).toBe(4)
    expect(lineCount(wallsInBand(walls, 1.1, 0.7))).toBe(0)
    expect(lineCount(wallsInBand(walls, 1.1, 1.2))).toBe(8)
    expect(lineCount(wallsInBand(walls, 10, 0.7))).toBe(0)
  })
})
