/**
 * wcagContrast.unit.test.ts — Tests unitaires des helpers WCAG 2.0.
 *
 * Cas connus de référence pour ne pas dériver de la formule officielle.
 */
import { describe, it, expect } from 'vitest'
import { relLuminance, contrastRatio, wcagGrade } from '../wcagContrast'

describe('relLuminance', () => {
  it('blanc = 1', () => {
    expect(relLuminance('#FFFFFF')).toBeCloseTo(1, 4)
  })
  it('noir = 0', () => {
    expect(relLuminance('#000000')).toBeCloseTo(0, 4)
  })
  it('gris moyen ≈ 0.184', () => {
    // #777777 — formule WCAG officielle donne ≈ 0.1845
    expect(relLuminance('#777777')).toBeCloseTo(0.184, 2)
  })
  it('accepte le format court #abc', () => {
    expect(relLuminance('#FFF')).toBeCloseTo(1, 4)
    expect(relLuminance('#000')).toBeCloseTo(0, 4)
  })
  it('rejette un hex invalide', () => {
    expect(() => relLuminance('#GGGGGG')).toThrow()
    expect(() => relLuminance('not-a-hex')).toThrow()
  })
})

describe('contrastRatio', () => {
  it('noir/blanc = 21 (max)', () => {
    expect(contrastRatio('#000000', '#FFFFFF')).toBeCloseTo(21, 1)
  })
  it('blanc/blanc = 1 (min)', () => {
    expect(contrastRatio('#FFFFFF', '#FFFFFF')).toBeCloseTo(1, 4)
  })
  it('symétrique', () => {
    expect(contrastRatio('#0072B2', '#FFFFFF')).toBeCloseTo(
      contrastRatio('#FFFFFF', '#0072B2'),
      4,
    )
  })
})

describe('wcagGrade', () => {
  it('21 → AAA', () => expect(wcagGrade(21)).toBe('AAA'))
  it('7 → AAA', () => expect(wcagGrade(7)).toBe('AAA'))
  it('4.5 → AA', () => expect(wcagGrade(4.5)).toBe('AA'))
  it('6.99 → AA', () => expect(wcagGrade(6.99)).toBe('AA'))
  it('3 → AA-large', () => expect(wcagGrade(3)).toBe('AA-large'))
  it('2.99 → fail', () => expect(wcagGrade(2.99)).toBe('fail'))
  it('1 → fail', () => expect(wcagGrade(1)).toBe('fail'))
})
