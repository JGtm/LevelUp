import { describe, it, expect } from 'vitest'
import { outcomeCodeToValue, outcomeCodeToTapeValue } from './outcome'

describe('outcomeCodeToValue', () => {
  it('mappe les codes du contrat ADR 0006', () => {
    expect(outcomeCodeToValue(2)).toBe('win')
    expect(outcomeCodeToValue(3)).toBe('loss')
    expect(outcomeCodeToValue(1)).toBe('tie')
    expect(outcomeCodeToValue(4)).toBe('dnf')
  })
  it('retourne null hors contrat (défaut le plus sûr : ne jamais fabriquer)', () => {
    expect(outcomeCodeToValue(0)).toBeNull()
    expect(outcomeCodeToValue(99)).toBeNull()
    expect(outcomeCodeToValue(null)).toBeNull()
    expect(outcomeCodeToValue(undefined)).toBeNull()
  })
})

describe('outcomeCodeToTapeValue', () => {
  it('mappe les codes du contrat ADR 0006', () => {
    expect(outcomeCodeToTapeValue(2)).toBe('win')
    expect(outcomeCodeToTapeValue(3)).toBe('loss')
    expect(outcomeCodeToTapeValue(1)).toBe('tie')
    expect(outcomeCodeToTapeValue(4)).toBe('dnf')
  })
  it("défaut de frise 'dnf' pour un code hors contrat (typage non-null)", () => {
    expect(outcomeCodeToTapeValue(0)).toBe('dnf')
    expect(outcomeCodeToTapeValue(null)).toBe('dnf')
    expect(outcomeCodeToTapeValue(undefined)).toBe('dnf')
  })
})
