/**
 * hexComplement.test.ts — rotation HSL +180° pour les barres négatives.
 */
import { describe, it, expect } from 'vitest'
import { hexComplement } from './hexComplement'

describe('hexComplement', () => {
  it('indigo → résultat distinct (non-gris chromatique)', () => {
    const src = '#818cf8' // indigo ~234°
    const cmp = hexComplement(src)
    expect(cmp).not.toBe(src)
    expect(cmp).toMatch(/^#[0-9a-f]{6}$/)
  })

  it('rotation double = couleur d\'origine', () => {
    // complement(complement(x)) = x (à l'arrondi entier près)
    const src = '#818cf8'
    const back = hexComplement(hexComplement(src))
    // Tolérance de ±1 sur chaque canal (arrondi Math.round)
    const toRgb = (h: string) => [
      parseInt(h.slice(1, 3), 16),
      parseInt(h.slice(3, 5), 16),
      parseInt(h.slice(5, 7), 16),
    ]
    const [r1, g1, b1] = toRgb(src)
    const [r2, g2, b2] = toRgb(back)
    expect(Math.abs(r1 - r2)).toBeLessThanOrEqual(1)
    expect(Math.abs(g1 - g2)).toBeLessThanOrEqual(1)
    expect(Math.abs(b1 - b2)).toBeLessThanOrEqual(1)
  })

  it('rouge #e84747 → vert/cyan (~180°)', () => {
    const cmp = hexComplement('#e84747')
    // hue ~0° → complément ~180° = cyan. R doit baisser, B et G monter.
    const r = parseInt(cmp.slice(1, 3), 16)
    const g = parseInt(cmp.slice(3, 5), 16)
    const b = parseInt(cmp.slice(5, 7), 16)
    expect(g).toBeGreaterThan(r) // cyan : G > R
    expect(b).toBeGreaterThan(r)
  })

  it('format #RGB court accepté', () => {
    const cmp = hexComplement('#f00') // rouge court
    expect(cmp).toMatch(/^#[0-9a-f]{6}$/)
    expect(cmp).not.toBe('#ff0000')
  })

  it('hex invalide → fallback #888888', () => {
    expect(hexComplement('not-a-color')).toBe('#888888')
    expect(hexComplement('')).toBe('#888888')
  })

  it('gris #888888 (achromatique, s=0) → même gris (pas de teinte à inverser)', () => {
    // Un gris n'a pas de teinte — rotation de 0° sur s=0, retourne même luminosité.
    const cmp = hexComplement('#888888')
    expect(cmp).toBe('#888888')
  })

  it('résultat toujours en lowercase #rrggbb', () => {
    expect(hexComplement('#AABBCC')).toMatch(/^#[0-9a-f]{6}$/)
  })
})
