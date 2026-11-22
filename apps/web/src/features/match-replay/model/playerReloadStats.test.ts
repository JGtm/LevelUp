/**
 * playerReloadStats.test.ts — LES RECHARGEMENTS D'UN JOUEUR DU DÉBUT DU MATCH À L'IMAGE LUE.
 *
 * Des documents écrits à la main donnent les valeurs exactes : aucun rechargement, un cumul qui ne recule jamais,
 * le rechargement daté exactement à l'image, deux joueurs, un bot, un index qui ne désigne personne, un document
 * sans couverture (grandeur absente, jamais un zéro), et le plancher que dit une trame arrêtée.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { vie } from '../test/equipmentUsageFixtures'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import { playerReloadsAt, readReloadSeries, reloadSeriesOf } from './playerReloadStats'

/** Une couverture de lecture entière : trames fermées, aucune arrêtée. */
const READ = {
  reloads: {
    frames: 100, closedFrames: 100, stoppedFrames: 0, read: 5, copies: 0, withoutPlayer: 0,
    beforeOrigin: 0, afterEnd: 0, published: 5,
  },
}
const STOPPED = { reloads: { ...READ.reloads, closedFrames: 98, stoppedFrames: 2 } }

const reload = (t: number, place: number) => ({ t, place })

function docWith(over: Record<string, unknown>): ReplayDocumentReady {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
      { filmIndex: 1, xuid: 'b1', name: 'Bravo', team: 1 },
    ],
    tracks: [vie(1, 'a1'), vie(2, 'b1')],
    coverage: READ,
    ...over,
  } as unknown as Partial<ReplayDocument>)
}

const SHEET = [scoreboardRow('a1', 'Alpha', 't0'), scoreboardRow('b1', 'Bravo', 't1')]

function lecteur(doc: ReplayDocumentReady) {
  const players = buildPlayers(doc, SHEET)
  const index = readReloadSeries(doc, players)
  return (xuid: string, frame: number) => {
    const player = players.find((p) => p.xuid === xuid) as ReplayPlayer
    return playerReloadsAt(reloadSeriesOf(index, player), frame)
  }
}

describe('playerReloadStats — le cumul par joueur', () => {
  const lire = lecteur(docWith({ reloads: [reload(30, 0), reload(10, 0), reload(20, 1), reload(50, 0)] }))

  it('un joueur sans rechargement a un zéro mesuré, pas une absence', () => {
    const l = lecteur(docWith({ reloads: [reload(20, 1)] }))
    expect(l('a1', 100)).toEqual({ count: 0, minimum: false })
  })

  it('compte les rechargements du joueur jusqu’à l’image, quel que soit l’ordre du document', () => {
    expect(lire('a1', 9).count).toBe(0)
    expect(lire('a1', 10).count).toBe(1)
    expect(lire('a1', 35).count).toBe(2)
    expect(lire('a1', 100).count).toBe(3)
  })

  it('le rechargement daté exactement à l’image compte', () => {
    expect(lire('a1', 30).count).toBe(2)
    expect(lire('a1', 29).count).toBe(1)
  })

  it('ne confond pas deux joueurs', () => {
    expect(lire('b1', 19).count).toBe(0)
    expect(lire('b1', 100).count).toBe(1)
  })

  it('ne recule jamais d’une image à l’autre', () => {
    let last = 0
    for (let f = 0; f <= 200; f++) {
      const n = lire('a1', f).count as number
      expect(n).toBeGreaterThanOrEqual(last)
      last = n
    }
  })
})

describe('playerReloadStats — ce que le document ne permet pas d’établir', () => {
  it('sans couverture des rechargements, la grandeur est absente, jamais un zéro', () => {
    const l = lecteur(docWith({ coverage: {}, reloads: [reload(10, 0)] }))
    expect(l('a1', 100)).toEqual({ count: null, minimum: false })
    expect(l('b1', 100)).toEqual({ count: null, minimum: false })
  })

  it('sans calque ni couverture (film ancien), rien n’est établi', () => {
    const l = lecteur(docWith({ coverage: undefined, reloads: undefined }))
    expect(l('a1', 100).count).toBeNull()
  })

  it('un index que le roster ne désigne pas ne revient à personne', () => {
    const l = lecteur(docWith({ reloads: [reload(10, 7)] }))
    expect(l('a1', 100).count).toBe(0)
    expect(l('b1', 100).count).toBe(0)
  })

  it('un joueur sans aucune vie n’a pas de série', () => {
    const doc = docWith({ tracks: [vie(1, 'a1')], reloads: [reload(10, 1)] })
    const players = buildPlayers(doc, SHEET)
    const index = readReloadSeries(doc, players)
    const bravo = players.find((p) => p.xuid === 'b1')
    if (bravo) expect(playerReloadsAt(reloadSeriesOf(index, bravo), 100).count).toBeNull()
  })
})

describe('playerReloadStats — un plancher quand le film perd des messages', () => {
  it('une trame arrêtée fait de chaque compte un minimum', () => {
    const l = lecteur(docWith({ coverage: STOPPED, reloads: [reload(10, 0)] }))
    expect(l('a1', 100)).toEqual({ count: 1, minimum: true })
    expect(l('b1', 100)).toEqual({ count: 0, minimum: true })
  })

  it('aucune trame arrêtée : le compte n’est pas un minimum', () => {
    const l = lecteur(docWith({ reloads: [reload(10, 0)] }))
    expect(l('a1', 100).minimum).toBe(false)
  })
})

describe('playerReloadStats — un bot', () => {
  const bot = { filmIndex: 2, xuid: '', name: 'Marc', team: 1, bot: true }
  const doc = docWith({
    roster: [{ filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 }, bot],
    tracks: [vie(1, 'a1'), { ...vie(3, ''), bot: 'Marc' }],
    reloads: [reload(10, 2), reload(20, 2), reload(15, 0)],
  })
  const players = buildPlayers(doc, [scoreboardRow('a1', 'Alpha', 't0')])
  const index = readReloadSeries(doc, players)

  it('se lit par sa clé de roster `bot:<nom>`, comme tout joueur', () => {
    const marc = players.find((p) => p.xuid === 'bot:Marc') as ReplayPlayer
    expect(marc).toBeDefined()
    expect(playerReloadsAt(reloadSeriesOf(index, marc), 100).count).toBe(2)
  })

  it('ne donne pas à l’humain les rechargements du bot', () => {
    const humain = players.find((p) => p.xuid === 'a1') as ReplayPlayer
    expect(playerReloadsAt(reloadSeriesOf(index, humain), 100).count).toBe(1)
  })
})

describe('playerReloadStats — un index porté par plusieurs identités', () => {
  const presence = (from: number, to: number) => ({ from, to })

  it('deux joueurs successifs sur un index : chacun a ses rechargements, aucun ne reçoit la somme', () => {
    const doc = docWith({
      roster: [
        { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0, presence: [presence(0, 49)] },
        { filmIndex: 0, xuid: 'c1', name: 'Charlie', team: 0, presence: [presence(50, 200)] },
      ],
      tracks: [vie(1, 'a1'), vie(1, 'c1')],
      reloads: [reload(10, 0), reload(20, 0), reload(60, 0)],
    })
    const players = buildPlayers(doc, [scoreboardRow('a1', 'Alpha', 't0'), scoreboardRow('c1', 'Charlie', 't0')])
    const index = readReloadSeries(doc, players)
    const at = (xuid: string) => playerReloadsAt(reloadSeriesOf(index, players.find((p) => p.xuid === xuid) as ReplayPlayer), 200)
    expect(at('a1').count).toBe(2)
    expect(at('c1').count).toBe(1)
  })

  it('un rechargement hors de toute présence ne revient à personne', () => {
    const doc = docWith({
      roster: [
        { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0, presence: [presence(0, 40)] },
        { filmIndex: 0, xuid: 'c1', name: 'Charlie', team: 0, presence: [presence(50, 200)] },
      ],
      tracks: [vie(1, 'a1'), vie(1, 'c1')],
      reloads: [reload(45, 0)],
    })
    const players = buildPlayers(doc, [scoreboardRow('a1', 'Alpha', 't0'), scoreboardRow('c1', 'Charlie', 't0')])
    const index = readReloadSeries(doc, players)
    for (const p of players) expect(playerReloadsAt(reloadSeriesOf(index, p), 200).count).toBe(0)
  })

  it('quatre bots sur un index, présences confondues : aucun ne reçoit de rechargement', () => {
    const noms = ['Ann', 'Bob', 'Cid', 'Dan']
    const doc = docWith({
      roster: noms.map((name) => ({ filmIndex: 5, xuid: '', name, team: 1, bot: true, presence: [presence(0, 200)] })),
      tracks: noms.map((name, i) => ({ ...vie(10 + i, ''), bot: name })),
      reloads: [reload(10, 5), reload(20, 5)],
    })
    const players = buildPlayers(doc, [])
    const index = readReloadSeries(doc, players)
    expect(players).toHaveLength(4)
    for (const p of players) expect(playerReloadsAt(reloadSeriesOf(index, p), 200).count).toBe(0)
  })

  it('une même identité inscrite deux fois sur l’index garde ses rechargements', () => {
    const doc = docWith({
      roster: [
        { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
        { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
      ],
      tracks: [vie(1, 'a1')],
      reloads: [reload(10, 0)],
    })
    const players = buildPlayers(doc, [scoreboardRow('a1', 'Alpha', 't0')])
    const index = readReloadSeries(doc, players)
    expect(playerReloadsAt(reloadSeriesOf(index, players[0]), 200).count).toBe(1)
  })
})

describe('playerReloadStats — la couverture dit qu’une part manque', () => {
  it('des rechargements sans joueur écartés : le compte est un minimum', () => {
    const coverage = { reloads: { ...READ.reloads, withoutPlayer: 2 } }
    const l = lecteur(docWith({ coverage, reloads: [reload(10, 0)] }))
    expect(l('a1', 100)).toEqual({ count: 1, minimum: true })
  })
})
