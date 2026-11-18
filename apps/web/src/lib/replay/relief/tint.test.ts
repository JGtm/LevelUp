import { describe, expect, it } from 'vitest'

import { FLAG_KNOWN, FLAG_ON_MAP } from './constants'
import { buildTintTable, parseHexColor, reliefRamp, rgbToCss, tintRelief, toRgb } from './tint'

describe('parseHexColor', () => {
  it('lit #rrggbb et #rgb, casse et espaces tolérés', () => {
    expect(parseHexColor('#6b7078')).toEqual([107, 112, 120])
    expect(parseHexColor('  #FFAA00 ')).toEqual([255, 170, 0])
    expect(parseHexColor('#f80')).toEqual([255, 136, 0])
  })

  it('rend null pour tout ce qui n’est pas une couleur hexadécimale', () => {
    for (const bad of ['', 'red', '#12', '#12345', '#gggggg', 'rgb(1,2,3)']) expect(parseHexColor(bad)).toBeNull()
  })

  it('toRgb accepte aussi un triplet entier 0..255', () => {
    expect(toRgb([1, 2, 3])).toEqual([1, 2, 3])
    expect(toRgb([1, 2, 300])).toBeNull()
    expect(toRgb([1, 2.5, 3])).toBeNull()
    expect(toRgb('nope')).toBeNull()
  })
})

describe('reliefRamp : trois crans depuis UNE couleur de base', () => {
  const ramp = reliefRamp([100, 200, 50])

  it('bas = 30 % de la base, milieu = 78 %, haut = base éclaircie de 30 %', () => {
    expect(ramp?.low).toEqual([30, 60, 15])
    expect(ramp?.mid).toEqual([78, 156, 39])
    expect(ramp?.high).toEqual([147, 217, 112])
  })

  it('tire les encres des traits de la même base, plus claires que la rampe', () => {
    expect(ramp?.edge).toEqual([201, 236, 183])
    expect(ramp?.contour).toEqual([170, 225, 142])
    expect(ramp?.wall).toEqual([224, 244, 214])
  })

  it('une base illisible rend null', () => {
    expect(reliefRamp('pas une couleur')).toBeNull()
  })

  it('est identique pour la même base donnée en hexadécimal ou en triplet', () => {
    expect(reliefRamp('#64c832')).toEqual(ramp)
  })

  it('rgbToCss écrit la couleur pour un contexte 2D', () => {
    expect(rgbToCss([1, 2, 3])).toBe('rgb(1,2,3)')
  })
})

describe('buildTintTable', () => {
  const table = buildTintTable({ low: [10, 20, 30], mid: [110, 120, 130], high: [210, 220, 230] })
  const colour = (g: number) => [table[g * 3], table[g * 3 + 1], table[g * 3 + 2]]

  it('le gris 0 prend le bas, le gris 255 prend le haut, le milieu du gris prend le milieu', () => {
    expect(colour(0)).toEqual([10, 20, 30])
    expect(colour(255)).toEqual([210, 220, 230])
    const mid = colour(127)
    expect(mid[0]).toBeGreaterThanOrEqual(108)
    expect(mid[0]).toBeLessThanOrEqual(112)
  })

  it('interpole linéairement entre les crans', () => {
    // Gris 64 : f = 64/255, k = 2 f = 0,502 entre bas et milieu.
    expect(colour(64)[0]).toBeCloseTo(10 + 100 * (64 / 255) * 2, 0)
    for (let g = 1; g < 256; g++) expect(table[g * 3]).toBeGreaterThanOrEqual(table[(g - 1) * 3])
  })
})

describe('tintRelief', () => {
  const table = buildTintTable({ low: [10, 20, 30], mid: [110, 120, 130], high: [210, 220, 230] })

  it('teinte les pixels de la carte à leur gris, avec l’opacité de la coupe ; les autres restent transparents', () => {
    const grey = Uint8Array.from([0, 255, 255, 0])
    const alpha = Uint8Array.from([255, 128, 255, 255])
    const flags = Uint8Array.from([FLAG_KNOWN | FLAG_ON_MAP, FLAG_KNOWN | FLAG_ON_MAP, FLAG_KNOWN, 0])
    const out = tintRelief({ width: 4, height: 1, grey, alpha, flags }, table)
    expect(out.rgba.length).toBe(16)
    expect(Array.from(out.rgba.slice(0, 4))).toEqual([10, 20, 30, 255])
    expect(Array.from(out.rgba.slice(4, 8))).toEqual([210, 220, 230, 128])
    expect(Array.from(out.rgba.slice(8, 12))).toEqual([0, 0, 0, 0])
    expect(Array.from(out.rgba.slice(12, 16))).toEqual([0, 0, 0, 0])
  })

  it('une opacité nulle laisse le pixel transparent', () => {
    const out = tintRelief(
      { width: 1, height: 1, grey: Uint8Array.of(100), alpha: Uint8Array.of(0), flags: Uint8Array.of(FLAG_KNOWN | FLAG_ON_MAP) },
      table,
    )
    expect(Array.from(out.rgba)).toEqual([0, 0, 0, 0])
  })
})
