import { describe, it, expect } from 'vitest'
import {
  formatSignedFixed,
  formatSignedPoints,
  signOf,
  outcomeCodeToValue,
} from './ExplorerBriefing.logic'

describe('formatSignedFixed', () => {
  it('préfixe + / − / ±', () => {
    expect(formatSignedFixed(0.3, 2)).toBe('+0.30')
    expect(formatSignedFixed(-1.5, 2)).toBe('−1.50')
    expect(formatSignedFixed(0, 2)).toBe('±0.00')
  })
  it('vide si absent', () => {
    expect(formatSignedFixed(null, 2)).toBe('')
    expect(formatSignedFixed(undefined, 0)).toBe('')
  })
})

describe('formatSignedPoints', () => {
  it('convertit un ratio en points de pourcentage signés', () => {
    expect(formatSignedPoints(0.3)).toBe('+30 pts')
    expect(formatSignedPoints(-0.12)).toBe('−12 pts')
    expect(formatSignedPoints(0)).toBe('±0 pts')
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

describe('outcomeCodeToValue', () => {
  it('mappe les codes backend', () => {
    expect(outcomeCodeToValue(1)).toBe('tie')
    expect(outcomeCodeToValue(2)).toBe('win')
    expect(outcomeCodeToValue(3)).toBe('loss')
    expect(outcomeCodeToValue(4)).toBe('dnf')
  })
})
