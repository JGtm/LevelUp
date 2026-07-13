import { describe, it, expect } from 'vitest'
import type { KPIStats } from '@/lib/api/types'
import {
  aggregateKda,
  scopeWinRate,
  formatSignedFixed,
  formatSignedPoints,
  signOf,
  outcomeCodeToValue,
  perfTierLabelKey,
} from './ExplorerBriefing.logic'

function kpis(partial: Partial<KPIStats>): KPIStats {
  return {
    matches_count: 0,
    total_play_seconds: 0,
    avg_match_seconds: 0,
    kills_per_game: 0,
    kills_per_minute: 0,
    deaths_per_game: 0,
    deaths_per_minute: 0,
    assists_per_game: 0,
    assists_per_minute: 0,
    avg_accuracy: 0,
    avg_life_seconds: 0,
    outcomes: { wins: 0, losses: 0, ties: 0, dnf: 0 },
    ...partial,
  } as KPIStats
}

describe('aggregateKda', () => {
  it('applique la formule ADR 0006 (frags + assists/3 − morts) par match', () => {
    const k = kpis({ kills_per_game: 20, assists_per_game: 3, deaths_per_game: 10 })
    expect(aggregateKda(k)).toBeCloseTo(20 + 1 - 10, 5) // 11
  })
  it('peut être négatif', () => {
    const k = kpis({ kills_per_game: 5, assists_per_game: 0, deaths_per_game: 20 })
    expect(aggregateKda(k)).toBe(-15)
  })
})

describe('scopeWinRate', () => {
  it('wins / matchs', () => {
    expect(scopeWinRate(kpis({ matches_count: 10, outcomes: { wins: 7, losses: 2, ties: 1, dnf: 0 } }))).toBe(0.7)
  })
  it('null si aucun match', () => {
    expect(scopeWinRate(kpis({ matches_count: 0 }))).toBeNull()
  })
})

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

describe('perfTierLabelKey', () => {
  it('mappe 1..5 vers les clés de palier', () => {
    expect(perfTierLabelKey(1)).toBe('excellent')
    expect(perfTierLabelKey(3)).toBe('correct')
    expect(perfTierLabelKey(5)).toBe('mauvais')
  })
})
