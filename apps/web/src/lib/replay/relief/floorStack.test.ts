import { describe, expect, it } from 'vitest'

import { STACK_LAYERS } from './constants'
import { buildFloorStack, stackHeight, stackTop, type FloorStack } from './floorStack'
import { reliefData, type FloorRect } from './reliefScene.testutil'

const W = 40
const H = 20
const build = (rects: FloorRect[], opts: { merge?: number } = {}) =>
  buildFloorStack(reliefData({ width: W, height: H, rects, ...opts }))
const at = (i: number, j: number): number => j * W + i
/** Les hauteurs d'un pixel, au centième : la pile les code au 1/256 d'unité. */
const stackColumn = (stack: FloorStack, p: number): number[] => {
  const out: number[] = []
  for (let k = 0; k < stack.count[p]; k++) out.push(Math.round(stackHeight(stack, p, k) * 100) / 100)
  return out
}

describe('buildFloorStack', () => {
  it('un sol plat couvre exactement les pixels qu’il recouvre, à sa hauteur', () => {
    const stack = build([{ i0: 5, j0: 4, i1: 14, j1: 9, z: 2.25 }])
    expect(stackColumn(stack, at(5, 4))).toEqual([2.25])
    expect(stackColumn(stack, at(14, 9))).toEqual([2.25])
    expect(stackColumn(stack, at(4, 4))).toEqual([])
    expect(stackColumn(stack, at(15, 9))).toEqual([])
    expect(stackColumn(stack, at(5, 10))).toEqual([])
    let covered = 0
    for (let p = 0; p < stack.count.length; p++) covered += stack.count[p]
    expect(covered).toBe(10 * 6)
  })

  it('empile deux sols l’un sur l’autre, du plus bas au plus haut, quel que soit l’ordre des faces', () => {
    const a = build([
      { i0: 0, j0: 0, i1: 19, j1: 19, z: 2.2 },
      { i0: 0, j0: 0, i1: 19, j1: 19, z: 0 },
    ])
    expect(stackColumn(a, at(3, 3))).toEqual([0, 2.2])
    expect(stackTop(a, at(3, 3))).toBeCloseTo(2.2, 2)
    expect(stackTop(a, at(30, 3))).toBeNull()
  })

  it('fusionne deux sols à moins de `merge` : le premier rencontré reste', () => {
    const merged = build([
      { i0: 0, j0: 0, i1: 9, j1: 9, z: 1 },
      { i0: 0, j0: 0, i1: 9, j1: 9, z: 1.3 },
    ])
    expect(stackColumn(merged, at(2, 2))).toEqual([1])
    const apart = build([
      { i0: 0, j0: 0, i1: 9, j1: 9, z: 1 },
      { i0: 0, j0: 0, i1: 9, j1: 9, z: 1.5 },
    ])
    expect(stackColumn(apart, at(2, 2))).toEqual([1, 1.5])
  })

  it('le seuil de fusion vient du JSON', () => {
    const wide = build(
      [
        { i0: 0, j0: 0, i1: 9, j1: 9, z: 1 },
        { i0: 0, j0: 0, i1: 9, j1: 9, z: 2 },
      ],
      { merge: 1.5 },
    )
    expect(stackColumn(wide, at(2, 2))).toEqual([1])
  })

  it('ne garde que quatre sols par pixel', () => {
    const four = build([0, 1, 2, 3].map((z) => ({ i0: 0, j0: 0, i1: 9, j1: 9, z })))
    expect(stackColumn(four, at(1, 1))).toEqual([0, 1, 2, 3])
    const five = build([0, 1, 2, 3, 4].map((z) => ({ i0: 0, j0: 0, i1: 9, j1: 9, z })))
    expect(five.count[at(1, 1)]).toBe(STACK_LAYERS)
    expect(stackColumn(five, at(1, 1))).toEqual([0, 1, 2, 3])
  })

  it('interpole la hauteur d’une rampe au centre du pixel', () => {
    // De 0 (bord gauche, x = 0) à 2 (bord droit, x = 1,0) sur dix pixels : z = 0,2 x.
    const stack = build([{ i0: 0, j0: 0, i1: 9, j1: 9, z: [0, 2, 2, 0] }])
    for (let i = 0; i < 10; i++) {
      const [z] = stackColumn(stack, at(i, 5))
      expect(z).toBeCloseTo((2 * (i + 0.5)) / 10, 2)
    }
  })

  it('code les sols sous le dessus le plus bas (z0) sans les écraser', () => {
    const stack = build([{ i0: 0, j0: 0, i1: 9, j1: 9, z: -20 }])
    expect(stackColumn(stack, at(2, 2))).toEqual([-20])
  })

  it('un sol qui déborde de l’image est rogné sans erreur', () => {
    const stack = build([{ i0: -5, j0: -5, i1: 60, j1: 40, z: 1 }])
    expect(stack.count[at(0, 0)]).toBe(1)
    expect(stack.count[at(W - 1, H - 1)]).toBe(1)
  })
})
