/**
 * Tests — l'équipe d'un joueur du rejeu (`buildPlayers`, `groupByTeam`, résolveurs d'encre et de
 * camp de rosterLogic).
 *
 * Séparés de `rosterLogic.test.ts` par RESPONSABILITÉ (seuil de taille du dépôt) : ici, la seule
 * question de l'appartenance — d'où vient l'équipe d'un joueur, quels camps en sortent, et quelle
 * encre et quel camp en reçoivent ses vies.
 */
import { describe, expect, it } from 'vitest'

import { scoreboardRow } from '../../features/match-replay/test/scoreboardRow'
import { testReplayDoc as doc } from '../../features/match-replay/test/testDoc'
import { buildFilmAllegiance } from './filmAllegiance'
import type { ReplayTrackReady } from './replayNormalize'
import {
  buildPlayers,
  buildSlotOwnership,
  campResolver,
  colorByXuidResolver,
  colorResolver,
  colorResolverOrLast,
  groupByTeam,
} from './rosterLogic'

/**
 * L'ÉQUIPE D'UN JOUEUR EST CELLE DU FILM (décision du 2026-10-06, ADR 0034 D-9) : `buildPlayers`
 * la pose depuis l'entrée de roster, et rien d'autre ne la donne — ni la feuille de match, ni
 * l'équipe d'une vie.
 */
describe('buildPlayers — l’équipe vient du roster du film', () => {
  it('pose le désignateur de l’entrée, bot compris (joint par `rosterEntryKey`)', () => {
    const d = doc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 1 },
        { xuid: '', bot: true, filmIndex: 8, name: 'Sandwolf [bot]', team: 0 },
      ],
    })
    const parCle = new Map(buildPlayers(d, []).map((p) => [p.xuid, p.team]))
    expect(parCle.get('A')).toBe(1)
    expect(parCle.get('bot:Sandwolf [bot]')).toBe(0)
  })

  it('une entrée dont le film TAIT l’équipe n’en reçoit aucune — la feuille ne la remplace pas', () => {
    const d = doc({ roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }] })
    const [p] = buildPlayers(d, [scoreboardRow('A', 'Alpha', 't0')])
    expect(p.board?.team_side).toBe('t0')
    expect(p.team).toBeUndefined()
  })

  it('un joueur que seules ses vies nomment n’a pas d’entrée, donc pas d’équipe — même si la vie en porte une', () => {
    const d = doc({ tracks: [{ slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 50, points: [{ t: 0, x: 0, y: 0 }] }] })
    expect(buildPlayers(d, [])[0].team).toBeUndefined()
  })
})

describe('groupByTeam — les camps du film', () => {
  it('range par désignateur ; la feuille ne fait que nommer, et n’ajoute aucun groupe', () => {
    const d = doc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 1 },
        { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 0 },
        { xuid: 'C', filmIndex: 2, name: 'Charlie', team: 1 },
        { xuid: 'Muet', filmIndex: 3, name: 'Muet' },
      ],
    })
    const board = [scoreboardRow('A', 'Alpha', 't1'), scoreboardRow('B', 'Bravo', 't0'), scoreboardRow('Muet', 'Muet', 't0')]
    const groups = groupByTeam(buildPlayers(d, board))
    expect(groups.map((g) => [g.team, g.side, g.players.map((p) => p.xuid)])).toEqual([
      [0, 't0', ['B']],
      [1, 't1', ['A', 'C']],
    ])
  })
})

/** Une vie immobile, de `start` à `end`, sur le slot donné ; un bot n'a pas de xuid de trace. */
function vie(slot: number, start: number, end: number, who: { xuid?: string; bot?: string }): ReplayTrackReady {
  return { slot, team: -1, ...who, startFrame: start, endFrame: end, points: [{ t: start, x: 0, y: 0 }] }
}

/**
 * L'ENCRE D'UNE VIE EST L'ALLÉGEANCE DE SON PROPRIÉTAIRE, LUE DANS LE FILM (2026-10-06). Le cas
 * qui a fait changer la source (D1) : un BOT du camp de la référence. La table d'identité de la
 * feuille est clée par xuid de BASE (`bid(44.0)`) ; le film désigne le bot par `bot:<nom>` — il
 * n'y était jamais trouvé et prenait l'encre ADVERSE.
 */
describe('résolveurs d’encre — l’allégeance du film (`filmAllegiance.ofPlayer`)', () => {
  const encre = (ally: boolean) => (ally ? 'allie' : 'adverse')
  const d = doc({
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
      { xuid: '', bot: true, filmIndex: 8, name: 'Sandwolf [bot]', team: 0 },
      { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 1 },
      { xuid: 'Muet', filmIndex: 2, name: 'Muet' },
    ],
    tracks: [
      vie(512, 0, 50, { xuid: 'A' }),
      vie(513, 0, 50, { bot: 'Sandwolf [bot]' }),
      vie(514, 0, 50, { xuid: 'B' }),
      vie(515, 0, 50, { xuid: 'Muet' }),
    ],
  })
  const board = [
    scoreboardRow('A', 'Alpha', 't0', { is_me: true }),
    scoreboardRow('bid(44.0)', 'Sandwolf', 't0', { is_bot: true }),
    scoreboardRow('B', 'Bravo', 't1'),
    scoreboardRow('Muet', 'Muet', 't0'),
  ]
  const players = buildPlayers(d, board)
  const own = buildSlotOwnership(players)

  it('un BOT du camp de la référence prend l’encre ALLIÉE ; l’autre camp l’encre adverse', () => {
    const color = colorResolver(own, encre, buildFilmAllegiance(players, 'A').ofPlayer, 'neutre')
    expect(color(512, 25)).toBe('allie')
    expect(color(513, 25)).toBe('allie')
    expect(color(514, 25)).toBe('adverse')
  })

  it('un joueur dont le film TAIT l’équipe prend l’encre NEUTRE, malgré sa ligne de feuille du camp allié', () => {
    const color = colorResolver(own, encre, buildFilmAllegiance(players, 'A').ofPlayer, 'neutre')
    expect(color(515, 25)).toBe('neutre')
  })

  it('vu d’une référence sans équipe du film, personne n’a d’encre de camp — elle comprise', () => {
    const color = colorResolver(own, encre, buildFilmAllegiance(players, 'Muet').ofPlayer, 'neutre')
    expect([512, 513, 514, 515].map((slot) => color(slot, 25))).toEqual(['neutre', 'neutre', 'neutre', 'neutre'])
  })

  it('la variante de frontière et la résolution par clé du film suivent la même règle', () => {
    const allyOf = buildFilmAllegiance(players, 'A').ofPlayer
    expect(colorResolverOrLast(own, encre, allyOf, 'neutre')(513, 51)).toBe('allie') // finVie + 1
    const parCle = colorByXuidResolver(players, encre, allyOf, 'neutre')
    expect(parCle('bot:Sandwolf [bot]')).toBe('allie')
    expect(parCle('Muet')).toBe('neutre')
    expect(parCle('inconnu')).toBeNull()
  })
})

describe('campResolver — le camp d’une vie pour l’opposition, lu dans le film', () => {
  it('l’équipe du film du propriétaire ; aucun camp quand le film la tait ou n’en donne aucune (-1)', () => {
    const d = doc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 1 },
        { xuid: 'M', filmIndex: 1, name: 'Muet' },
        { xuid: 'F', filmIndex: 2, name: 'Foxtrot', team: -1 },
      ],
      tracks: [vie(512, 0, 50, { xuid: 'A' }), vie(513, 0, 50, { xuid: 'M' }), vie(514, 0, 50, { xuid: 'F' })],
    })
    const camp = campResolver(buildSlotOwnership(buildPlayers(d, [scoreboardRow('M', 'Muet', 't0')])))
    expect(camp(512, 25)).toBe(1)
    expect(camp(513, 25)).toBeNull() // la ligne de feuille ne donne pas le camp
    expect(camp(514, 25)).toBeNull()
    expect(camp(512, 99)).toBeNull() // slot libre
  })
})
