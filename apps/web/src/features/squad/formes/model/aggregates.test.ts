/**
 * aggregates.test.ts — LES PARTS ET LEURS PARITÉS.
 *
 * Ce qui est vérifié ici est ce qui fait la valeur du bloc : les DEUX
 * dénominateurs, la parité prise sur l'EFFECTIF (bots compris) et non sur le
 * nombre de lignes mesurées, le « non mesuré » qui n'est jamais un zéro, et la
 * moyenne par MATCH (jamais par minute).
 */
import { describe, expect, it } from 'vitest'

import { FORMES_MAIN_XUID, formesFixture } from '../formes.fixtures'
import { matchSizes, parityOf } from './access'
import { aggregateAxis, myShareOfMatch, playerSpread, playerTotal } from './aggregates'

const block = formesFixture()

describe('aggregateAxis', () => {
  it('rapporte la même mesure à DEUX dénominateurs', () => {
    // Camouflages : moi 2 (match 1) + 0 (match 2) = 2.
    // Mon camp : 2 + 5 + 0 + 1 = 8 au match 1, 0 + 3 = 3 au match 2 -> 11.
    // Lobby : 8 + 1 + 0 (match 1) = 9, 3 + 0 (match 2) = 3 -> 12.
    const agg = aggregateAxis(block, 'camo')
    expect(agg.team.value).toBe(2)
    expect(agg.team.total).toBe(11)
    expect(agg.lobby.total).toBe(12)
    expect(agg.team.pct).toBeCloseTo((2 / 11) * 100, 6)
    expect(agg.lobby.pct).toBeCloseTo((2 / 12) * 100, 6)
  })

  it('prend la parité sur l’EFFECTIF du match, pas sur les lignes mesurées', () => {
    // Les deux matchs mesurés déclarent 4 et 8 : la parité vaut 100/4 et 100/8,
    // même si le film ne nomme que 6 puis 3 joueurs.
    const agg = aggregateAxis(block, 'camo')
    expect(agg.team.parity).toBeCloseTo(25, 6)
    expect(agg.lobby.parity).toBeCloseTo(12.5, 6)
  })

  it('ne compte l’étendue que sur les matchs où l’axe est mesuré', () => {
    // Les murs : personne n'en pose au match 1 côté lobby ? Si — l'adversaire en
    // pose 2. Le joueur, lui, en pose 2 au match 2 seulement.
    const agg = aggregateAxis(block, 'wall')
    expect(agg.team.measured).toBeGreaterThan(0)
    expect(agg.team.min).not.toBeNull()
  })
})

describe('myShareOfMatch', () => {
  it('rend null sur un match SANS FILM — jamais un zéro', () => {
    const noFilm = block.matches?.[0]
    expect(noFilm?.measured).toBe(false)
    expect(myShareOfMatch(noFilm!, FORMES_MAIN_XUID, 'camo')).toBeNull()
  })
})

describe('playerSpread', () => {
  it('donne min, max et la moyenne PAR MATCH mesuré (jamais par minute)', () => {
    // Grappin du joueur : 1 au match mesuré 1, 3 au match mesuré 2.
    const spread = playerSpread(block, FORMES_MAIN_XUID, 'grapple')
    expect(spread).not.toBeNull()
    expect(spread?.min).toBe(1)
    expect(spread?.max).toBe(3)
    expect(spread?.matches).toBe(2)
    expect(spread?.mean).toBeCloseTo(2, 6)
  })
})

describe('playerTotal', () => {
  it('ne compte que les matchs mesurés', () => {
    expect(playerTotal(block, FORMES_MAIN_XUID, 'dropped')).toBe(4)
  })
})

describe('matchSizes / parityOf', () => {
  it('retombe sur les lignes mesurées quand le match ne porte pas ses effectifs', () => {
    const sizes = matchSizes({ match_id: 'x', measured: true, lobby: [], matches_total: 0 } as never)
    expect(sizes.lobby).toBe(0)
    expect(parityOf(sizes.lobby)).toBeNull()
  })
})
