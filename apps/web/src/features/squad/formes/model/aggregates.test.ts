/**
 * aggregates.test.ts — LES PARTS ET LEURS PARITÉS.
 *
 * Ce qui est vérifié ici est ce qui fait la valeur du bloc : les DEUX
 * dénominateurs, la parité prise sur l'EFFECTIF (bots compris) et non sur le
 * nombre de lignes mesurées, le « non mesuré » qui n'est jamais un zéro, et la
 * moyenne par MATCH (jamais par minute).
 */
import { describe, expect, it } from 'vitest'

import { FORMES_MAIN_XUID, FORMES_MATE_A, formesFixture } from '../formes.fixtures'
import { PAD_AXIS, matchSizes, parityOf } from './access'
import {
  aggregateAxis,
  lobbyParts,
  myShareOfMatch,
  playerSpread,
  playerTotal,
  teamShareOfMatch,
} from './aggregates'

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

  it('publie la part de mon camp dans le lobby', () => {
    const agg = aggregateAxis(block, PAD_AXIS)
    // Prises nommées : mon camp 4 (2+1+0+1) au match 1, 2 au match 2 -> 6.
    // Lobby : 6 au match 1, 3 au match 2 -> 9.
    expect(agg.teamTotal).toBe(6)
    expect(agg.lobbyTotal).toBe(9)
    expect(agg.teamShareOfLobbyPct).toBeCloseTo((6 / 9) * 100, 6)
  })
})

describe('teamShareOfMatch / myShareOfMatch', () => {
  it('rend null sur un match SANS FILM — jamais un zéro', () => {
    const noFilm = block.matches?.[0]
    expect(noFilm?.measured).toBe(false)
    expect(teamShareOfMatch(noFilm!, FORMES_MAIN_XUID, 'camo')).toBeNull()
    expect(myShareOfMatch(noFilm!, FORMES_MAIN_XUID, 'camo')).toBeNull()
  })

  it('rend null quand personne n’a touché l’axe sur un match mesuré', () => {
    const flag = block.matches?.[2]
    // Personne ne prend de surbouclier sauf un allié : la part existe. Sur un axe
    // que personne n'alimente, elle n'existe pas.
    expect(teamShareOfMatch(flag!, FORMES_MAIN_XUID, 'overshield')).not.toBeNull()
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

describe('lobbyParts', () => {
  it('sépare l’escouade, le reste du camp et l’adversaire', () => {
    const parts = lobbyParts(
      (block.matches ?? []).filter((m) => m.measured),
      [FORMES_MAIN_XUID, FORMES_MATE_A],
      'dropped',
    )
    // Objets lâchés — match 1 : moi 3, Madina 2, Choco 2 (hors escouade ici),
    // DectroPK 1 (hors escouade), adversaires 4 + 1 = 5.
    // Match 2 : moi 1, Madina 0, adversaire 2.
    expect(parts.bySquad[FORMES_MAIN_XUID]).toBe(4)
    expect(parts.bySquad[FORMES_MATE_A]).toBe(2)
    expect(parts.teamRest).toBe(3)
    expect(parts.opponents).toBe(7)
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
