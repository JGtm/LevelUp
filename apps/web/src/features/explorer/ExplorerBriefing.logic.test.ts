import { describe, it, expect } from 'vitest'
import { formatSignedPoints, isFullHistoryScope, signOf } from './ExplorerBriefing.logic'

// formatSignedFixed a migré vers `@/lib/formatters` (number.ts) — testé dans
// `lib/formatters/formatters.test.ts`.

describe('formatSignedPoints', () => {
  it('convertit un ratio en points de pourcentage signés', () => {
    expect(formatSignedPoints(0.3)).toBe('+30 pts')
    expect(formatSignedPoints(-0.12)).toBe('−12 pts')
    expect(formatSignedPoints(0)).toBe('±0 pts')
  })
})

describe('isFullHistoryScope', () => {
  it('vrai quand scope == baseline (aucun filtre)', () => {
    expect(isFullHistoryScope(120, 120)).toBe(true)
  })
  it('faux quand le scope est un sous-ensemble filtré', () => {
    expect(isFullHistoryScope(30, 120)).toBe(false)
  })
  it('faux sans baseline (aucun delta à masquer de toute façon)', () => {
    expect(isFullHistoryScope(120, null)).toBe(false)
    expect(isFullHistoryScope(120, undefined)).toBe(false)
  })
  it('faux quand le scope est absent', () => {
    expect(isFullHistoryScope(null, 120)).toBe(false)
    expect(isFullHistoryScope(undefined, undefined)).toBe(false)
  })
})

describe('signOf', () => {
  it('retourne -1 / 0 / 1', () => {
    expect(signOf(2)).toBe(1)
    expect(signOf(-2)).toBe(-1)
    expect(signOf(0)).toBe(0)
    expect(signOf(null)).toBe(0)
  })
})
