/**
 * Tests unitaires — logique pure de highlight best/worst du classement.
 * Pas de rendu (fonctions stateless), donc pas de jsdom.
 */
import { describe, it, expect } from 'vitest'
import type { LeaderboardEntry } from '@/lib/api/types'
import { computeColumnExtremes, columnHighlightStyle } from './LeaderboardBlock.highlight'

const WIN = 'var(--ac-outcome-win)'
const LOSS = 'var(--ac-outcome-loss)'

/** Entrée enrichie minimale (match_count présent → prise en compte). */
function entry(over: Partial<LeaderboardEntry>): LeaderboardEntry {
  return { rank: 1, gamertag: 'GT', xuid: 'x', csr_value: 1500, match_count: 10, ...over } as LeaderboardEntry
}

const A = entry({ csr_value: 1900, kills: 200, deaths: 50, assists: 60, win_rate: 0.7, kda: 25, accuracy: 550 })
const B = entry({ csr_value: 1700, kills: 100, deaths: 80, assists: 40, win_rate: 0.5, kda: 15, accuracy: 500 })
const C = entry({ csr_value: 1800, kills: 150, deaths: 65, assists: 50, win_rate: 0.6, kda: 20, accuracy: 525 })

describe('computeColumnExtremes', () => {
  it('calcule min/max par colonne sur les entrées enrichies', () => {
    const ex = computeColumnExtremes([A, B, C])
    expect(ex.kills).toEqual({ min: 100, max: 200 })
    expect(ex.deaths).toEqual({ min: 50, max: 80 })
    expect(ex.csr).toEqual({ min: 1700, max: 1900 })
    // FDA = kda / match_count : 2.5 / 1.5 / 2.0
    expect(ex.fda).toEqual({ min: 1.5, max: 2.5 })
  })

  it('renvoie {null,null} (neutre) en dessous de 2 valeurs', () => {
    const ex = computeColumnExtremes([A])
    expect(ex.kills).toEqual({ min: null, max: null })
  })

  it('ignore les entrées non enrichies (match_count absent)', () => {
    const stranger = { rank: 9, gamertag: 'World', xuid: 'w', csr_value: 2200 } as LeaderboardEntry
    const ex = computeColumnExtremes([A, B, stranger])
    // stranger n'entre pas dans le calcul → max csr reste 1900 (A), pas 2200.
    expect(ex.csr).toEqual({ min: 1700, max: 1900 })
  })
})

describe('columnHighlightStyle', () => {
  const ex = computeColumnExtremes([A, B, C])

  it('colonne normale : max=best (vert), min=worst (rouge), milieu=neutre', () => {
    expect(columnHighlightStyle('kills', 200, ex).color).toBe(WIN)
    expect(columnHighlightStyle('kills', 100, ex).color).toBe(LOSS)
    expect(Object.keys(columnHighlightStyle('kills', 150, ex))).toHaveLength(0)
  })

  it('colonne inversée (Morts) : min=best (vert), max=worst (rouge)', () => {
    expect(columnHighlightStyle('deaths', 50, ex).color).toBe(WIN)
    expect(columnHighlightStyle('deaths', 80, ex).color).toBe(LOSS)
  })

  it('colonne inversée (Dégâts/frag) : moins de dégâts par frag = meilleur', () => {
    const dpkA = columnHighlightStyle('dmgKill', 1, computeColumnExtremes([entry({ damage_dealt: 100, kills: 100, assists: 0 }), entry({ damage_dealt: 400, kills: 100, assists: 0 })]))
    expect(dpkA.color).toBe(WIN) // 100/100 = 1 (le plus bas) → best
  })

  it('Taux de victoire = uniforme (best/worst seulement, pas de seuil ≥55/≤45)', () => {
    // 0.6 n'est ni le max (0.7) ni le min (0.5) → neutre, malgré ≥55%.
    expect(Object.keys(columnHighlightStyle('winRate', 0.6, ex))).toHaveLength(0)
    expect(columnHighlightStyle('winRate', 0.7, ex).color).toBe(WIN)
    expect(columnHighlightStyle('winRate', 0.5, ex).color).toBe(LOSS)
  })

  it('best en gras (fontWeight 600), worst un peu moins (500)', () => {
    expect(columnHighlightStyle('kills', 200, ex).fontWeight).toBe(600)
    expect(columnHighlightStyle('kills', 100, ex).fontWeight).toBe(500)
  })
})
