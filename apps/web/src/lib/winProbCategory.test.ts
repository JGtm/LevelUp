import { describe, it, expect } from 'vitest'

import { categorizeWinProb, formatWinProb, BALANCED_MARGIN } from './winProbCategory'

describe('categorizeWinProb', () => {
  it('favori net + victoire → expected', () => {
    expect(categorizeWinProb(0.8, true)).toBe('expected')
  })

  it('outsider net + victoire → upset', () => {
    expect(categorizeWinProb(0.2, true)).toBe('upset')
  })

  it('outsider net + défaite → strong-perf (défaite perdable)', () => {
    expect(categorizeWinProb(0.2, false)).toBe('strong-perf')
  })

  it('pronostic ~50/50 → balanced (quel que soit le résultat)', () => {
    expect(categorizeWinProb(0.5, true)).toBe('balanced')
    expect(categorizeWinProb(0.5, false)).toBe('balanced')
    expect(categorizeWinProb(0.5 + BALANCED_MARGIN, false)).toBe('balanced') // borne incluse
  })

  it('favori net + défaite → expected (choke, pas de 5e catégorie)', () => {
    expect(categorizeWinProb(0.85, false)).toBe('expected')
  })

  it('proba absente → null (rien affiché)', () => {
    expect(categorizeWinProb(null, true)).toBeNull()
    expect(categorizeWinProb(undefined, false)).toBeNull()
    expect(categorizeWinProb(Number.NaN, true)).toBeNull()
  })
})

describe('formatWinProb', () => {
  it('formate en pourcentage entier', () => {
    expect(formatWinProb(0.734)).toBe('73%')
    expect(formatWinProb(0)).toBe('0%')
    expect(formatWinProb(1)).toBe('100%')
  })

  it('absente → chaîne vide', () => {
    expect(formatWinProb(null)).toBe('')
    expect(formatWinProb(undefined)).toBe('')
  })
})
