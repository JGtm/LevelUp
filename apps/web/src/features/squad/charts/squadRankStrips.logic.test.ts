/**
 * squadRankStrips.logic.test.ts — les décisions pures du sous-graphe « Rang » : bandeaux par
 * joueur, échelle commune (même amplitude, centrée sur chaque joueur), écart sur la session,
 * nature du rang, MMR d'équipe.
 */
import { describe, expect, it } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import {
  halfAmplitude,
  mmrBounds,
  RANK_MARGIN_MIN,
  rankStrips,
  stripBounds,
  stripRange,
  teamMmrByMatch,
} from './squadRankStrips.logic'

const pt = (order: number, extra: Partial<SquadPerformanceSeriesPoint> = {}): SquadPerformanceSeriesPoint =>
  ({
    match_order: order,
    match_id: `m${order}`,
    start_time: '2026-09-10T20:00:00Z',
    kills: 10,
    deaths: 8,
    assists: 3,
    ...extra,
  }) as SquadPerformanceSeriesPoint

const csr = (order: number, rating: number, delta?: number, mmr?: number) =>
  pt(order, { skill_rating: rating, skill_rating_type: 'csr', skill_delta: delta, team_mmr: mmr })

describe('rankStrips', () => {
  it('un bandeau par joueur ayant au moins un rang, dans l’ordre donné ; les autres sont omis', () => {
    const rows = {
      B: [csr(0, 1200, 5), csr(1, 1210, 10)],
      A: [pt(0)],
      C: [csr(0, 1500, -3)],
    }
    expect(rankStrips(rows, ['B', 'A', 'C']).map((s) => s.gamertag)).toEqual(['B', 'C'])
    expect(rankStrips(rows, ['C', 'B']).map((s) => s.gamertag)).toEqual(['C', 'B'])
  })

  it('rang de fin et écart sur la session : fin − (début − variation du début)', () => {
    const [s] = rankStrips({ A: [csr(0, 1500, 12), csr(1, 1506, 6), csr(2, 1512, 6)] }, ['A'])
    expect(s.endRating).toBe(1512)
    // Rang avant la session : 1500 − 12 = 1488 ; écart : 1512 − 1488 = 24 = 12 + 6 + 6.
    expect(s.sessionDelta).toBe(24)
  })

  it('sans variation servie sur le premier match : l’écart part du premier rang', () => {
    const [s] = rankStrips({ A: [csr(0, 1500), csr(1, 1490, -10)] }, ['A'])
    expect(s.sessionDelta).toBe(-10)
  })

  it('les points sont triés par match et gardent la variation (null si absente)', () => {
    const [s] = rankStrips({ A: [csr(2, 1520), csr(0, 1500, 4)] }, ['A'])
    expect(s.points.map((p) => [p.order, p.rating, p.delta])).toEqual([[0, 1500, 4], [2, 1520, null]])
  })

  it('nature : la plus fréquente, à égalité la plus récente ; centre : milieu de l’étendue, entier', () => {
    const mixed = [
      csr(0, 1200),
      pt(1, { skill_rating: 60, skill_rating_type: 'lusr', skill_delta: 1 }),
      pt(2, { skill_rating: 62, skill_rating_type: 'lusr', skill_delta: 2 }),
    ]
    const [s] = rankStrips({ A: mixed }, ['A'])
    expect(s.kind).toBe('lusr')
    expect(s.endRating).toBe(62)
    const tie = rankStrips({ A: [csr(0, 1200), pt(1, { skill_rating: 60, skill_rating_type: 'lusr' })] }, ['A'])[0]
    expect(tie.kind).toBe('lusr')
    const [c] = rankStrips({ A: [csr(0, 1500), csr(1, 1507)] }, ['A'])
    expect(c.center).toBe(1504)
  })

  it('un type inconnu compte comme LUSR (même règle que l’ancien graphe)', () => {
    const [s] = rankStrips({ A: [pt(0, { skill_rating: 70, skill_rating_type: 'autre' })] }, ['A'])
    expect(s.kind).toBe('lusr')
  })
})

describe('échelle commune', () => {
  const rows = {
    A: [csr(0, 1500), csr(1, 1540)], // étendue 40
    B: [csr(0, 1200), csr(1, 1210)], // étendue 10
  }
  const strips = rankStrips(rows, ['A', 'B'])

  it('étendue d’un bandeau : max − min', () => {
    expect(strips.map(stripRange)).toEqual([40, 10])
  })

  it('demi-amplitude : la moitié du plus grand écart plus la marge (15 %), arrondie au-dessus', () => {
    // 40 / 2 + max(40 × 0,15 ; 4) = 20 + 6 = 26.
    expect(halfAmplitude(strips)).toBe(26)
  })

  it('marge plancher : des rangs plats gardent un bandeau lisible', () => {
    const flat = rankStrips({ A: [csr(0, 1500), csr(1, 1500)] }, ['A'])
    expect(halfAmplitude(flat)).toBe(RANK_MARGIN_MIN)
    expect(halfAmplitude([])).toBe(0)
  })

  it('TOUS les bandeaux ont la même amplitude, chacun centré sur son joueur', () => {
    const half = halfAmplitude(strips)
    const [a, b] = strips.map((s) => stripBounds(s, half))
    expect(a.max - a.min).toBe(b.max - b.min)
    expect(a).toEqual({ min: 1520 - 26, max: 1520 + 26 })
    expect(b).toEqual({ min: 1205 - 26, max: 1205 + 26 })
    // Le joueur à 40 points d’écart y tient avec marge ; celui à 10 points y reste petit : les pentes se comparent.
    expect(1540).toBeLessThan(a.max)
    expect(1500).toBeGreaterThan(a.min)
  })
})

describe('MMR d’équipe', () => {
  it('une valeur par match : la première servie', () => {
    const rows = { A: [csr(0, 1500, 1, 1400), csr(1, 1505, 1)], B: [csr(0, 1200, 1, 1999), csr(1, 1210, 1, 1410)] }
    expect(teamMmrByMatch(rows, ['A', 'B'], 3)).toEqual([1400, 1410, null])
  })

  it('bornes : étendue plus la marge ; null sans valeur', () => {
    expect(mmrBounds([1400, null, 1440])).toEqual({ min: 1400 - 6, max: 1440 + 6 })
    expect(mmrBounds([null, null])).toBeNull()
    expect(mmrBounds([1500])).toEqual({ min: 1500 - RANK_MARGIN_MIN, max: 1500 + RANK_MARGIN_MIN })
  })
})
