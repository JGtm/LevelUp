import { describe, it, expect } from 'vitest'
import { interpolate, nextPPTier, formatMultiplier, STREAK_PP_TIERS } from './format'

describe('interpolate', () => {
  it('remplace les placeholders simples', () => {
    expect(interpolate('Hello {name}!', { name: 'World' })).toBe('Hello World!')
  })

  it('plural: vide quand n=1', () => {
    expect(interpolate('{n} jour{plural}', { n: 1 })).toBe('1 jour')
  })

  it('plural: ajoute s quand n>1', () => {
    expect(interpolate('{n} jour{plural}', { n: 5 })).toBe('5 jours')
  })

  it('placeholder manquant: chaîne vide', () => {
    expect(interpolate('Hello {name}!', {})).toBe('Hello !')
  })
})

describe('nextPPTier', () => {
  it('retourne le palier 4 quand length=0', () => {
    expect(nextPPTier(0)).toEqual({ length: 4, multiplier: 1.1 })
  })

  it('retourne le palier 4 quand length=3', () => {
    expect(nextPPTier(3)).toEqual({ length: 4, multiplier: 1.1 })
  })

  it('retourne le palier 8 quand length=4', () => {
    expect(nextPPTier(4)).toEqual({ length: 8, multiplier: 1.25 })
  })

  it('retourne le palier 15 quand length=10', () => {
    expect(nextPPTier(10)).toEqual({ length: 15, multiplier: 1.5 })
  })

  it('retourne le palier 30 quand length=20', () => {
    expect(nextPPTier(20)).toEqual({ length: 30, multiplier: 1.75 })
  })

  it('retourne null quand length>=30 (max atteint)', () => {
    expect(nextPPTier(30)).toBeNull()
    expect(nextPPTier(100)).toBeNull()
  })
})

describe('formatMultiplier', () => {
  it('formatte 1.00 sans trailing zeros', () => {
    expect(formatMultiplier(1.0)).toBe('×1')
  })

  it('formatte 1.10 → ×1.1', () => {
    expect(formatMultiplier(1.1)).toBe('×1.1')
  })

  it('formatte 1.25 inchangé', () => {
    expect(formatMultiplier(1.25)).toBe('×1.25')
  })

  it('formatte 1.75 inchangé', () => {
    expect(formatMultiplier(1.75)).toBe('×1.75')
  })
})

describe('STREAK_PP_TIERS', () => {
  it('contient les 4 paliers du backend (4, 8, 15, 30)', () => {
    expect(STREAK_PP_TIERS.map((t) => t.length)).toEqual([4, 8, 15, 30])
    expect(STREAK_PP_TIERS.map((t) => t.multiplier)).toEqual([1.1, 1.25, 1.5, 1.75])
  })
})
