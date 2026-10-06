/**
 * Tests — l'équipe d'un joueur du rejeu (`buildPlayers`, `groupByTeam` de rosterLogic).
 *
 * Séparés de `rosterLogic.test.ts` par RESPONSABILITÉ (seuil de taille du dépôt) : ici, la seule
 * question de l'appartenance — d'où vient l'équipe d'un joueur, et quels camps en sortent.
 */
import { describe, expect, it } from 'vitest'

import { scoreboardRow } from '../../features/match-replay/test/scoreboardRow'
import { testReplayDoc as doc } from '../../features/match-replay/test/testDoc'
import { buildPlayers, groupByTeam } from './rosterLogic'

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
