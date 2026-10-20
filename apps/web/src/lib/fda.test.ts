import { describe, expect, it } from 'vitest'

import { FDA_ASSIST_WEIGHT, fdaTone, matchFda } from './fda'

describe('matchFda — le NET canonique sur UN match, jamais un quotient par les morts', () => {
  it('applique (frags + assistances/3 − morts) / 1', () => {
    // 15 frags, 6 assistances (= 2 frags effectifs), 6 morts -> 11.
    expect(matchFda({ kills: 15, assists: 6, deaths: 6 })).toBeCloseTo(11, 10)
  })

  it('passe SOUS ZÉRO quand les morts dépassent les frags effectifs', () => {
    expect(matchFda({ kills: 4, assists: 3, deaths: 9 })).toBeCloseTo(-4, 10)
  })

  it("n'est PAS le quotient (frags + assistances/3) / max(1, morts)", () => {
    // Le quotient rendrait (10 + 1) / 12 ≈ 0,92 — positif. Le net dit −1.
    expect(matchFda({ kills: 10, assists: 3, deaths: 12 })).toBeCloseTo(-1, 10)
  })

  it('rend null dès qu’un compteur manque — jamais un zéro de repli', () => {
    expect(matchFda({ kills: 5, assists: 2, deaths: null })).toBeNull()
    expect(matchFda({ kills: null, assists: 2, deaths: 1 })).toBeNull()
    expect(matchFda({ kills: 5, deaths: 1 })).toBeNull()
    expect(matchFda(null)).toBeNull()
  })

  it('trois assistances valent exactement un frag', () => {
    expect(FDA_ASSIST_WEIGHT * 3).toBeCloseTo(1, 10)
    expect(matchFda({ kills: 0, assists: 3, deaths: 0 })).toBeCloseTo(1, 10)
  })
})

describe('fdaTone — les trois paliers de lecture', () => {
  it('négatif -> destructive', () => {
    expect(fdaTone(-0.01)).toBe('destructive')
    expect(fdaTone(-7)).toBe('destructive')
  })

  it('de 0 à 1 INCLUS -> info (les bornes appartiennent au palier médian)', () => {
    expect(fdaTone(0)).toBe('info')
    expect(fdaTone(0.5)).toBe('info')
    expect(fdaTone(1)).toBe('info')
  })

  it('strictement au-delà de 1 -> success', () => {
    expect(fdaTone(1.001)).toBe('success')
    expect(fdaTone(12)).toBe('success')
  })
})
