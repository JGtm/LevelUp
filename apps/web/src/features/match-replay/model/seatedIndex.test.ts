/**
 * seatedIndex.test.ts — « occupe une fiche à l'instant t » : le critère de la tuile devient celui du
 * pion (utilisateur, 2026-10-21). Le bot relevé de `3d58eb37` (présence [223, 319], vies jusqu'à
 * l'image 3777) en est le cas type.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { buildSeats, seatedIndexOf, seatOccupantAt } from './seatLogic'

function joueur(xuid: string, team: number | undefined, debuts: Array<[number, number]>): ReplayPlayer {
  return {
    xuid,
    filmName: xuid,
    team,
    lives: debuts.map(([a, b]) => ({ slot: 1, team: 0, startFrame: a, endFrame: b, points: [{ t: a, x: 0, y: 0 }, { t: b, x: 0, y: 0 }] })) as ReplayPlayer['lives'],
  }
}

function doc(roster: Array<Record<string, unknown>>): ReplayDocumentReady {
  return { frameIntervalMs: 100, frameCount: 4000, originMs: 0, roster } as unknown as ReplayDocumentReady
}

describe('seatedIndexOf', () => {
  const bot = joueur('bot:B', 1, [[223, 652], [752, 1069]])
  const humain = joueur('H', 1, [[0, 3999]])
  const sansEquipe = joueur('X', undefined, [[0, 100]])
  const document = doc([
    { xuid: 'H', filmIndex: 0, seat: 0, seatSource: 'lu', team: 1, presence: [{ from: 0, to: 3999 }] },
    { bot: true, name: 'B', filmIndex: 5, seat: 5, seatSource: 'apparie', team: 1, presence: [{ from: 223, to: 319 }] },
    { xuid: 'X', filmIndex: 7, seat: 7, seatSource: 'lu', presence: [{ from: 0, to: 100 }] },
  ])
  const seats = buildSeats([humain, bot, sansEquipe], document)
  const index = seatedIndexOf(seats)

  it('un bot relevé tient sa fiche jusqu’à la veille de la relève, plus après', () => {
    expect(index.isSeated(bot.xuid, 222)).toBe(false) // pas encore arrivé
    expect(index.isSeated(bot.xuid, 223)).toBe(true)
    expect(index.isSeated(bot.xuid, 319)).toBe(true)
    expect(index.isSeated(bot.xuid, 320)).toBe(false)
    expect(index.isSeated(bot.xuid, 800)).toBe(false) // sa 2e vie : aucun pion
  })

  it('un joueur sans équipe du film n’a aucune fiche, donc aucun pion', () => {
    expect(index.isSeated('X', 50)).toBe(false)
  })

  it('un inconnu n’a aucune fiche', () => {
    expect(index.isSeated('nul', 10)).toBe(false)
  })

  it('même critère que la tuile : occupe une fiche = la tuile le nomme', () => {
    for (const f of [0, 222, 223, 300, 319, 320, 800, 3999]) {
      const nommes = new Set(seats.map((s) => seatOccupantAt(s, f).player?.xuid).filter(Boolean))
      for (const p of [humain, bot, sansEquipe]) {
        expect(index.isSeated(p.xuid, f)).toBe(nommes.has(p.xuid))
      }
    }
  })
})
