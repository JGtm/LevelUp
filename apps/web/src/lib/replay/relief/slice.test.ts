import { describe, expect, it } from 'vitest'

import { CUT_ALPHA_ABOVE, CUT_ALPHA_BELOW, FLAG_ON_MAP } from './constants'
import { buildFloorStack } from './floorStack'
import { reliefData, reliefImage, type FloorRect, type SceneCell } from './reliefScene.testutil'
import { heightGreyByte } from './shading'
import { computeSlice, cutWeight, type SliceInput } from './slice'

const ABOVE = Math.round(CUT_ALPHA_ABOVE * 255)
const BELOW = Math.round(CUT_ALPHA_BELOW * 255)
const RAMP = { low: 0, high: 4 }
const BASE_GREY = 150

/** Une scène de quatre blocs de dix colonnes, une ligne de dix pixels de haut. */
function build(rects: FloorRect[], top: (i: number, j: number) => SceneCell, width: number): SliceInput {
  const data = reliefData({ width, height: 10, rects })
  return {
    image: reliefImage(width, 10, top),
    grey: new Uint8Array(width * 10).fill(BASE_GREY),
    stack: buildFloorStack(data),
    mpp: 0.1,
    ramp: RAMP,
    tolerance: 0.3,
  }
}

const block = (c0: number, c1: number, z: number): FloorRect => ({ i0: c0, j0: 0, i1: c1, j1: 9, z })

/**
 * Colonnes 0-9 : un sol à 0, dessus 0. 10-19 : un sol à 2,2, dessus 2,2. 20-29 : un sol à 0 sous un TOIT
 * (dessus 2,5). 30-39 : un sol à 2,2 que le fond a rendu PLUS BAS (dessus 1,0).
 */
function fourBlocks(): SliceInput {
  const tops = [0, 2.2, 2.5, 1.0]
  return build(
    [block(0, 9, 0), block(10, 19, 2.2), block(20, 29, 0), block(30, 39, 2.2)],
    (i) => ({ z: tops[Math.floor(i / 10)] }),
    40,
  )
}

const px = (col: number) => 5 * 40 + col
const at20 = (col: number) => 5 * 20 + col

describe('cutWeight', () => {
  it('vaut 1 dans la bande, décroît linéairement sur le fondu, puis 0', () => {
    expect(cutWeight(2.2, 2.2)).toBe(1)
    expect(cutWeight(2.9, 2.2)).toBe(1)
    expect(cutWeight(2.9 + 0.125, 2.2)).toBeCloseTo(0.5, 9)
    expect(cutWeight(3.2, 2.2)).toBe(0)
    expect(cutWeight(1.5 - 0.125, 2.2)).toBeCloseTo(0.5, 9)
    expect(cutWeight(0, 2.2)).toBe(0)
  })
})

describe('computeSlice à la hauteur de rez-de-chaussée', () => {
  const s = computeSlice(fourBlocks(), 0)

  it('le dessus de l’étage se montre en plein et garde ses traits', () => {
    expect(s.alpha[px(5)]).toBe(255)
    expect(s.keep[px(5)]).toBe(1)
    expect(s.grey[px(5)]).toBe(BASE_GREY)
  })

  it('ce qui est au-dessus de la coupe est presque effacé et perd ses traits', () => {
    expect(s.alpha[px(15)]).toBe(ABOVE)
    expect(s.keep[px(15)]).toBe(0)
  })

  it('un sol de la coupe sous un toit est révélé à plat, dans la teinte de sa hauteur, sans traits', () => {
    expect(s.alpha[px(25)]).toBe(255)
    expect(s.grey[px(25)]).toBe(heightGreyByte(0, RAMP))
    expect(s.keep[px(25)]).toBe(0)
  })

  it('un dessus qui tient sur un sol de la coupe y est ; un dessus lointain reste effacé', () => {
    // Colonnes 30-39 : le dessus (1,0) n'est pas un sol, il est à 1,0 > 0,7 + fondu : effacé loin du sol de la coupe,
    // mais plein dans les trois pixels qui jouxtent le sol à 0 de la colonne 29.
    expect(s.alpha[px(31)]).toBe(255)
    expect(s.keep[px(31)]).toBe(1)
    expect(s.alpha[px(35)]).toBe(ABOVE)
    expect(s.keep[px(35)]).toBe(0)
  })
})

describe('computeSlice à la hauteur de l’étage haut', () => {
  const s = computeSlice(fourBlocks(), 2.2)

  it('le dessus de l’étage se montre en plein', () => {
    expect(s.alpha[px(15)]).toBe(255)
    expect(s.keep[px(15)]).toBe(1)
  })

  it('le dessous reste en pâle, sans traits', () => {
    expect(s.alpha[px(5)]).toBe(BELOW)
    expect(s.keep[px(5)]).toBe(0)
  })

  it('un sol plus haut que le dessus est révélé : le fond l’avait rendu plus bas', () => {
    expect(s.alpha[px(35)]).toBe(255)
    expect(s.grey[px(35)]).toBe(heightGreyByte(2.2, RAMP))
    expect(s.keep[px(35)]).toBe(0)
  })

  it('un dessus de la bande qui coiffe un sol du dessous se montre aussi (il est dans la bande)', () => {
    expect(s.alpha[px(25)]).toBe(255)
    expect(s.keep[px(25)]).toBe(1)
  })
})

describe('computeSlice : proximité d’un sol de la coupe', () => {
  // Colonnes 0-9 : un sol à 0, dessus 0. Colonnes 10-19 : une caisse de 3,5 sans sol dans le maillage.
  const crate = (): SliceInput =>
    build([block(0, 9, 0)], (i) => ({ z: i < 10 ? 0 : 3.5 }), 20)
  const s = computeSlice(crate(), 0, { plan: true })

  it('ce qui tient sur un sol de la coupe et le jouxte y est ; plus loin, c’est effacé', () => {
    for (const c of [10, 11, 12]) {
      expect(s.alpha[at20(c)], `colonne ${c}`).toBe(255)
      expect(s.keep[at20(c)]).toBe(1)
    }
    for (const c of [13, 16, 19]) {
      expect(s.alpha[at20(c)], `colonne ${c}`).toBe(ABOVE)
      expect(s.keep[at20(c)]).toBe(0)
    }
  })

  it('le masque du fond Plan retombe sur l’alpha du relief là où il n’y a aucun sol', () => {
    expect(s.planAlpha).not.toBeNull()
    for (const c of [10, 13, 19]) expect(s.planAlpha![at20(c)]).toBe(s.alpha[at20(c)])
  })
})

describe('computeSlice : « Tout » et cas limites', () => {
  it('« Tout » ne masque rien : opacité pleine, gris de l’ombrage, garde = la carte', () => {
    const input = fourBlocks()
    input.image.flags[px(3)] &= ~FLAG_ON_MAP
    const s = computeSlice(input, null)
    expect(s.cut).toBeNull()
    expect(Array.from(new Set(s.alpha))).toEqual([255])
    expect(s.planAlpha).toBeNull()
    expect(s.keep[px(3)]).toBe(0)
    expect(s.keep[px(4)]).toBe(1)
    expect(s.grey[px(25)]).toBe(BASE_GREY)
  })

  it('une carte sans pile n’a pas de coupe : le résultat est celui de « Tout »', () => {
    const input = { ...fourBlocks(), stack: null }
    const s = computeSlice(input, 2.2)
    expect(s.cut).toBeNull()
    expect(Array.from(new Set(s.alpha))).toEqual([255])
  })

  it('un pixel de hauteur inconnue prend l’opacité du dessous dès qu’une coupe est active, et n’a pas de traits', () => {
    const input = fourBlocks()
    const known = input.image
    const unknown = reliefImage(40, 10, (i) => (i === 7 ? null : { z: 0 }))
    const s = computeSlice({ ...input, image: { ...known, heights: unknown.heights, flags: unknown.flags } }, 0)
    expect(s.alpha[px(7)]).toBe(BELOW)
    expect(s.keep[px(7)]).toBe(0)
  })

  it('un pixel connu hors de la zone a son opacité mais jamais de traits', () => {
    const input = fourBlocks()
    input.image.flags[px(3)] &= ~FLAG_ON_MAP
    const s = computeSlice(input, 0)
    expect(s.alpha[px(3)]).toBe(255)
    expect(s.keep[px(3)]).toBe(0)
  })

  it('ne modifie aucune entrée', () => {
    const input = fourBlocks()
    const grey = input.grey.slice()
    const flags = input.image.flags.slice()
    computeSlice(input, 0)
    computeSlice(input, 2.2)
    expect(Array.from(input.grey)).toEqual(Array.from(grey))
    expect(Array.from(input.image.flags)).toEqual(Array.from(flags))
  })
})

describe('computeSlice : masque du fond Plan', () => {
  it('se juge sur le sol le plus HAUT de la pile, pas sur le dessus du relief', () => {
    const at0 = computeSlice(fourBlocks(), 0, { plan: true })
    // Colonne 25 : un sol à 0 sous un toit à 2,5. Le Plan montre ce sol : plein à la coupe 0, pâle à la coupe 2,2.
    expect(at0.planAlpha![px(25)]).toBe(255)
    expect(at0.planAlpha![px(15)]).toBe(ABOVE)
    const at2 = computeSlice(fourBlocks(), 2.2, { plan: true })
    expect(at2.planAlpha![px(25)]).toBe(BELOW)
    expect(at2.planAlpha![px(15)]).toBe(255)
  })

  it('n’est calculé que sur demande', () => {
    expect(computeSlice(fourBlocks(), 0).planAlpha).toBeNull()
  })
})
