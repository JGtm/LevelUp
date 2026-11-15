/**
 * squadRankStrips.logic.test.ts — les décisions pures du sous-graphe « Rang » : bandeaux par
 * joueur, échelle commune (même amplitude, centrée sur chaque joueur), écart sur la session,
 * nature du rang, MMR d'équipe.
 */
import { describe, expect, it } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import {
  CSR_CHAIN_KEY,
  chainKeyOf,
  halfAmplitude,
  mmrBounds,
  RANK_MARGIN_MIN,
  rankChains,
  rankStrips,
  selectedChain,
  stripBounds,
  stripRange,
  teamMmrByMatch,
  UNKNOWN_LUSR_CHAIN_KEY,
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

const lusr = (order: number, rating: number, group: string, delta?: number) =>
  pt(order, { skill_rating: rating, skill_rating_type: 'lusr', skill_playlist_group: group, skill_delta: delta })

const csr = (order: number, rating: number, delta?: number, mmr?: number) =>
  pt(order, { skill_rating: rating, skill_rating_type: 'csr', skill_delta: delta, team_mmr: mmr })

describe('rankStrips', () => {
  it('un bandeau par joueur ayant au moins un rang, dans l’ordre donné ; les autres sont omis', () => {
    const rows = {
      B: [csr(0, 1200, 5), csr(1, 1210, 10)],
      A: [pt(0)],
      C: [csr(0, 1500, -3)],
    }
    expect(rankStrips(rows, ['B', 'A', 'C'], 'ranked').map((s) => s.gamertag)).toEqual(['B', 'C'])
    expect(rankStrips(rows, ['C', 'B'], 'ranked').map((s) => s.gamertag)).toEqual(['C', 'B'])
  })

  it('rang de fin et écart sur la session : fin − (début − variation du début)', () => {
    const [s] = rankStrips({ A: [csr(0, 1500, 12), csr(1, 1506, 6), csr(2, 1512, 6)] }, ['A'], 'ranked')
    expect(s.endRating).toBe(1512)
    // Rang avant la session : 1500 − 12 = 1488 ; écart : 1512 − 1488 = 24 = 12 + 6 + 6.
    expect(s.sessionDelta).toBe(24)
  })

  it('sans variation servie sur le premier match : l’écart part du premier rang', () => {
    const [s] = rankStrips({ A: [csr(0, 1500), csr(1, 1490, -10)] }, ['A'], 'ranked')
    expect(s.sessionDelta).toBe(-10)
  })

  it('les points sont triés par match et gardent la variation (null si absente)', () => {
    const [s] = rankStrips({ A: [csr(2, 1520), csr(0, 1500, 4)] }, ['A'], 'ranked')
    expect(s.points.map((p) => [p.order, p.rating, p.delta])).toEqual([[0, 1500, 4], [2, 1520, null]])
  })

  it('nature de la chaîne affichée ; centre : milieu de l’étendue, entier', () => {
    const [csrStrip] = rankStrips({ A: [csr(0, 1200)] }, ['A'], 'ranked')
    expect(csrStrip.kind).toBe('csr')
    const [lusrStrip] = rankStrips({ A: [lusr(0, 60, 'arena_slayer')] }, ['A'], 'arena_slayer')
    expect(lusrStrip.kind).toBe('lusr')
    const [c] = rankStrips({ A: [csr(0, 1500), csr(1, 1507)] }, ['A'], 'ranked')
    expect(c.center).toBe(1504)
  })

  it('un type inconnu compte comme LUSR (même règle que l’ancien graphe)', () => {
    const [s] = rankStrips({ A: [pt(0, { skill_rating: 70, skill_rating_type: 'autre' })] }, ['A'], UNKNOWN_LUSR_CHAIN_KEY)
    expect(s.kind).toBe('lusr')
  })
})

describe('échelle commune', () => {
  const rows = {
    A: [csr(0, 1500), csr(1, 1540)], // étendue 40
    B: [csr(0, 1200), csr(1, 1210)], // étendue 10
  }
  const strips = rankStrips(rows, ['A', 'B'], 'ranked')

  it('étendue d’un bandeau : max − min', () => {
    expect(strips.map(stripRange)).toEqual([40, 10])
  })

  it('demi-amplitude : la moitié du plus grand écart plus la marge (15 %), arrondie au-dessus', () => {
    // 40 / 2 + max(40 × 0,15 ; 4) = 20 + 6 = 26.
    expect(halfAmplitude(strips)).toBe(26)
  })

  it('marge plancher : des rangs plats gardent un bandeau lisible', () => {
    const flat = rankStrips({ A: [csr(0, 1500), csr(1, 1500)] }, ['A'], 'ranked')
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

describe('chaînes de rang (décision du 14/11 : une chaîne à la fois)', () => {
  // Deux chaînes LUSR pour un même joueur : deux échelles, jamais mélangées.
  const rows = {
    A: [
      lusr(0, 60, 'arena_slayer', 2),
      lusr(1, 62, 'arena_slayer', 2),
      lusr(2, 61, 'arena_slayer', -1),
      lusr(3, 140, 'arena_objectif', 5),
      lusr(4, 150, 'arena_objectif', 10),
    ],
    B: [lusr(0, 70, 'arena_slayer', 1), lusr(3, 130, 'arena_objectif', 3)],
  }

  it('la clé d’une chaîne : « ranked » pour le CSR, le groupe pour le LUSR, « lusr » sans groupe', () => {
    expect(chainKeyOf(csr(0, 1500))).toBe(CSR_CHAIN_KEY)
    expect(chainKeyOf(lusr(0, 60, 'btb'))).toBe('btb')
    expect(chainKeyOf(pt(0, { skill_rating: 60, skill_rating_type: 'lusr' }))).toBe(UNKNOWN_LUSR_CHAIN_KEY)
  })

  it('une chaîne par groupe présent, avec son nombre de matchs, la plus jouée d’abord', () => {
    expect(rankChains(rows, ['A', 'B']).map((c) => [c.key, c.matches, c.kind])).toEqual([
      ['arena_slayer', 3, 'lusr'],
      ['arena_objectif', 2, 'lusr'],
    ])
    const withCsr = { ...rows, B: [...rows.B, csr(5, 1500, 4), csr(6, 1504, 4), csr(7, 1508, 4), csr(8, 1510, 2)] }
    // Le CSR (4 matchs) devance l’arène slayer (3) puis l’arène objectif (2).
    expect(rankChains(withCsr, ['A', 'B']).map((c) => c.key)).toEqual(['ranked', 'arena_slayer', 'arena_objectif'])
  })

  it('à égalité de matchs : l’ordre alphabétique des clés, défaut stable', () => {
    const tie = { A: [lusr(0, 60, 'b_chain'), lusr(1, 61, 'a_chain')] }
    expect(rankChains(tie, ['A']).map((c) => c.key)).toEqual(['a_chain', 'b_chain'])
  })

  it('chaîne par défaut = la plus jouée ; le choix est gardé tant que la chaîne existe, sinon repli', () => {
    const chains = rankChains(rows, ['A', 'B'])
    expect(selectedChain(chains, null)).toBe('arena_slayer')
    expect(selectedChain(chains, 'arena_objectif')).toBe('arena_objectif')
    expect(selectedChain(chains, 'btb')).toBe('arena_slayer')
    expect(selectedChain([], null)).toBe('')
  })

  it('un même joueur : un bandeau par chaîne choisie, des points de cette seule chaîne, aucun lien entre les deux', () => {
    const slayer = rankStrips(rows, ['A', 'B'], 'arena_slayer')
    const objectif = rankStrips(rows, ['A', 'B'], 'arena_objectif')
    expect(slayer[0].points.map((p) => p.rating)).toEqual([60, 62, 61])
    expect(objectif[0].points.map((p) => p.rating)).toEqual([140, 150])
  })

  it('l’écart de session est calculé PAR CHAÎNE (jamais 150 − 58)', () => {
    const [aSlayer] = rankStrips(rows, ['A'], 'arena_slayer')
    // 61 − (60 − 2) = 3 = 2 + 2 − 1.
    expect(aSlayer.sessionDelta).toBe(3)
    expect(aSlayer.endRating).toBe(61)
    const [aObjectif] = rankStrips(rows, ['A'], 'arena_objectif')
    // 150 − (140 − 5) = 15 = 5 + 10.
    expect(aObjectif.sessionDelta).toBe(15)
  })

  it('l’amplitude commune ne dépend que de la chaîne choisie : l’autre chaîne ne gonfle pas l’étendue', () => {
    const slayer = rankStrips(rows, ['A', 'B'], 'arena_slayer')
    // Étendues de la chaîne slayer : A 62 − 60 = 2, B 0 -> demi-amplitude 1 + max(0,3 ; 4) = 5.
    expect(halfAmplitude(slayer)).toBe(5)
    // Mélangées (l’ancien défaut), l’étendue de A vaudrait 150 − 60 = 90.
    expect(halfAmplitude(slayer)).toBeLessThan(20)
  })

  it('un joueur sans match dans la chaîne choisie n’a pas de bandeau', () => {
    const only = { A: [lusr(0, 60, 'arena_slayer')], B: [lusr(1, 130, 'arena_objectif')] }
    expect(rankStrips(only, ['A', 'B'], 'arena_slayer').map((s) => s.gamertag)).toEqual(['A'])
    expect(rankStrips(only, ['A', 'B'], 'arena_objectif').map((s) => s.gamertag)).toEqual(['B'])
  })

  it('une seule chaîne dans la session : la liste n’a qu’une entrée (pas de sélecteur)', () => {
    expect(rankChains({ A: [csr(0, 1500), csr(1, 1510)] }, ['A'])).toHaveLength(1)
  })
})
