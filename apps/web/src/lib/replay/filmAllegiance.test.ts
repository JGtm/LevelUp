/**
 * Tests — l'allégeance lue dans le film (`filmAllegiance.ts`) : la règle « allié = même équipe du
 * film que la référence » et ses trois bords (joueur sans équipe, référence sans équipe, mode sans
 * camps).
 */
import { describe, expect, it } from 'vitest'

import { scoreboardRow } from '../../features/match-replay/test/scoreboardRow'
import { testReplayDoc as doc } from '../../features/match-replay/test/testDoc'
import { buildFilmAllegiance, filmAllegianceOf } from './filmAllegiance'
import { buildPlayers } from './rosterLogic'

/**
 * Un 2 contre 2 avec un BOT dans chaque camp : la référence `A` (camp 0) a pour coéquipier le bot
 * « Sandwolf », que le film désigne par `bot:<nom>` et la feuille par `bid(44.0)` ; `Muet` est un
 * humain dont le film tait l'équipe, alors que sa ligne de feuille le met du côté de `A`.
 */
function match() {
  const d = doc({
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
      { xuid: '', bot: true, filmIndex: 8, name: 'Sandwolf [bot]', team: 0 },
      { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 1 },
      { xuid: '', bot: true, filmIndex: 9, name: 'Ritzy [bot]', team: 1 },
      { xuid: 'Muet', filmIndex: 2, name: 'Muet' },
    ],
  })
  const board = [
    scoreboardRow('A', 'Alpha', 't0', { is_me: true }),
    scoreboardRow('bid(44.0)', 'Sandwolf', 't0', { is_bot: true }),
    scoreboardRow('B', 'Bravo', 't1'),
    scoreboardRow('bid(0.0)', 'Ritzy', 't1', { is_bot: true }),
    scoreboardRow('Muet', 'Muet', 't0'),
  ]
  return { d, board, players: buildPlayers(d, board) }
}

describe('buildFilmAllegiance — allié = même équipe du film que la référence', () => {
  it('humains : même camp allié, l’autre adverse ; la référence est alliée d’elle-même', () => {
    const a = buildFilmAllegiance(match().players, 'A')
    expect(a.allyTeam).toBe(0)
    expect(a.ofXuid('A')).toBe(true)
    expect(a.ofXuid('B')).toBe(false)
    expect(a.ofTeam(0)).toBe(true)
    expect(a.ofTeam(1)).toBe(false)
  })

  it('un BOT du camp de la référence est ALLIÉ, par sa clé du film comme par son xuid de base (`board.xuid`)', () => {
    const { players } = match()
    const a = buildFilmAllegiance(players, 'A')
    const sandwolf = players.find((p) => p.xuid === 'bot:Sandwolf [bot]')
    expect(a.ofPlayer(sandwolf)).toBe(true)
    expect(a.ofXuid('bot:Sandwolf [bot]')).toBe(true)
    expect(a.ofXuid('bid(44.0)')).toBe(true)
    expect(a.ofXuid('bid(0.0)')).toBe(false)
  })

  it('un joueur dont le film TAIT l’équipe n’a pas d’encre de camp — sa ligne de feuille ne la donne pas', () => {
    const { players } = match()
    const a = buildFilmAllegiance(players, 'A')
    expect(a.ofXuid('Muet')).toBeNull()
    expect(a.ofPlayer(players.find((p) => p.xuid === 'Muet'))).toBeNull()
    expect(a.teamOfXuid('Muet')).toBeNull()
  })

  it('la référence se retrouve par sa ligne de feuille : un bot regardé situe son camp', () => {
    const a = buildFilmAllegiance(match().players, 'bid(0.0)')
    expect(a.allyTeam).toBe(1)
    expect(a.ofXuid('B')).toBe(true)
    expect(a.ofXuid('A')).toBe(false)
    expect(a.ofXuid('bot:Ritzy [bot]')).toBe(true)
  })

  it('identifiant inconnu, vide ou absent : aucune allégeance', () => {
    const a = buildFilmAllegiance(match().players, 'A')
    expect(a.ofXuid('inconnu')).toBeNull()
    expect(a.ofXuid('')).toBeNull()
    expect(a.ofXuid(null)).toBeNull()
    expect(a.ofPlayer(null)).toBeNull()
    expect(a.ofTeam(undefined)).toBeNull()
  })
})

describe('buildFilmAllegiance — référence sans équipe du film : personne n’a d’encre de camp', () => {
  it('référence dont le film tait l’équipe : null pour tous, elle comprise', () => {
    const a = buildFilmAllegiance(match().players, 'Muet')
    expect(a.allyTeam).toBeNull()
    for (const x of ['Muet', 'A', 'B', 'bid(44.0)']) expect(a.ofXuid(x)).toBeNull()
    expect(a.ofTeam(0)).toBeNull()
    expect(a.ofTeam(1)).toBeNull()
  })

  it('référence absente du film (ou nulle) : idem — aucun camp n’est deviné', () => {
    for (const ref of ['hors-film', null, undefined]) {
      const a = buildFilmAllegiance(match().players, ref)
      expect(a.allyTeam).toBeNull()
      expect(a.ofXuid('A')).toBeNull()
      expect(a.ofTeam(0)).toBeNull()
    }
  })
})

describe('buildFilmAllegiance — mode sans camps (désignateur -1)', () => {
  const ffa = doc({
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha', team: -1 },
      { xuid: 'B', filmIndex: 1, name: 'Bravo', team: -1 },
      { xuid: 'C', filmIndex: 2, name: 'Charlie' },
    ],
  })

  it('la référence est alliée d’elle-même, tout autre joueur est adverse ; un sans-équipe reste neutre', () => {
    const a = buildFilmAllegiance(buildPlayers(ffa, []), 'A')
    expect(a.allyTeam).toBeNull() // « aucune équipe » n'est pas un camp allié
    expect(a.ofXuid('A')).toBe(true)
    expect(a.ofXuid('B')).toBe(false)
    expect(a.ofXuid('C')).toBeNull()
  })

  it('« aucune équipe » n’est pas un camp : ni encre de camp, ni membres, ni camp listé', () => {
    const a = buildFilmAllegiance(buildPlayers(ffa, []), 'A')
    expect(a.ofTeam(-1)).toBeNull()
    expect(a.membersOf(-1)).toEqual([])
    expect(a.camps).toEqual([])
    expect(a.teamOfXuid('B')).toBe(-1)
  })

  it('dans un mode à camps, un joueur déclaré « aucune équipe » n’a pas d’encre de camp', () => {
    const d = doc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
        { xuid: 'X', filmIndex: 1, name: 'Xray', team: -1 },
      ],
    })
    expect(buildFilmAllegiance(buildPlayers(d, []), 'A').ofXuid('X')).toBeNull()
  })
})

describe('buildFilmAllegiance — camps et membres', () => {
  it('les camps du film, croissants, nommés par la feuille de leurs membres ; un sans-équipe n’en crée pas', () => {
    expect(buildFilmAllegiance(match().players, 'A').camps).toEqual([
      { team: 0, side: 't0' },
      { team: 1, side: 't1' },
    ])
  })

  it('les membres d’un camp sont désignés par leur clé du FILM, bots compris', () => {
    const a = buildFilmAllegiance(match().players, 'A')
    expect(a.membersOf(0)).toEqual(['A', 'bot:Sandwolf [bot]'])
    expect(a.membersOf(1)).toEqual(['B', 'bot:Ritzy [bot]'])
    expect(a.membersOf(7)).toEqual([])
  })

  it('l’équipe d’un joueur se lit par sa clé du film comme par son xuid de base', () => {
    const a = buildFilmAllegiance(match().players, 'A')
    expect(a.teamOfXuid('bot:Ritzy [bot]')).toBe(1)
    expect(a.teamOfXuid('bid(0.0)')).toBe(1)
    expect(a.teamOfXuid('A')).toBe(0)
  })
})

describe('filmAllegianceOf — depuis le document et la feuille', () => {
  it('rend la même allégeance que la jointure de `buildPlayers`', () => {
    const { d, board } = match()
    const a = filmAllegianceOf(d, board, 'A')
    expect(a.ofXuid('bid(44.0)')).toBe(true)
    expect(a.ofXuid('B')).toBe(false)
    expect(a.camps.map((c) => c.team)).toEqual([0, 1])
  })
})
