import { describe, it, expect } from 'vitest'
import { hexToRgb, recolorMaskData } from './recolor'

describe('hexToRgb', () => {
  it('parse un hex #RRGGBB', () => {
    expect(hexToRgb('#d62828')).toEqual([214, 40, 40])
    expect(hexToRgb('ff8800')).toEqual([255, 136, 0])
  })
})

describe('recolorMaskData (modèle additif)', () => {
  it('canal R -> couleur primaire', () => {
    const src = new Uint8ClampedArray([255, 0, 0, 255])
    const out = recolorMaskData(src, {
      primary: '#ff8800',
      secondary: '#000000',
      tertiary: '#000000',
    })
    expect([out[0], out[1], out[2]]).toEqual([255, 136, 0])
    expect(out[3]).toBe(255) // alpha = max(255,0,0)
  })

  it('canal G -> couleur secondaire', () => {
    const src = new Uint8ClampedArray([0, 255, 0, 255])
    const out = recolorMaskData(src, {
      primary: '#000000',
      secondary: '#1971c2',
      tertiary: '#000000',
    })
    expect([out[0], out[1], out[2]]).toEqual([25, 113, 194])
  })

  it('canal B -> couleur tertiaire', () => {
    const src = new Uint8ClampedArray([0, 0, 255, 255])
    const out = recolorMaskData(src, {
      primary: '#000000',
      secondary: '#000000',
      tertiary: '#f1f3f5',
    })
    expect([out[0], out[1], out[2]]).toEqual([241, 243, 245])
  })

  it('additif : R+G mélange primaire + secondaire (avec clamp)', () => {
    const src = new Uint8ClampedArray([255, 255, 0, 255])
    const out = recolorMaskData(src, {
      primary: '#800000',
      secondary: '#008000',
      tertiary: '#000000',
    })
    // 0x80 + 0 = 128 (R), 0 + 0x80 = 128 (G)
    expect([out[0], out[1], out[2]]).toEqual([128, 128, 0])
  })

  it('hors-zone (RGB=0) -> transparent (alpha 0)', () => {
    const src = new Uint8ClampedArray([0, 0, 0, 255])
    const out = recolorMaskData(src, {
      primary: '#ffffff',
      secondary: '#ffffff',
      tertiary: '#ffffff',
    })
    expect(out[3]).toBe(0)
  })
})
