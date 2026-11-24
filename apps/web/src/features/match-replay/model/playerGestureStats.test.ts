/**
 * playerGestureStats.test.ts — LES GESTES DU JOUEUR (coup de mêlée, lancer amorcé, accroupi, saut au bouton) DU DÉBUT
 * DU MATCH À L'IMAGE LUE.
 *
 * Des documents écrits à la main donnent les valeurs exactes : le cumul par joueur, un bot, un film qui ne porte pas
 * les gestes (grandeur absente, jamais un zéro), le plancher que dit une trame arrêtée, et les commandes tenues
 * (accroupi, saut) comptées par intervalle.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { vie } from '../test/equipmentUsageFixtures'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import { gestureSeriesOf, playerGesturesAt, readGestureSeries } from './playerGestureStats'

/** Une couverture entière : aucun geste retenu par la porte de cadence, aucune trame arrêtée. */
const CARRIED = { gated: 0, stoppedFrames: 0, withoutPlayer: 0 }

const gesture = (t: number, place: number, kind: 'melee' | 'throwStart') => ({ t, place, kind })

function docWith(over: Record<string, unknown>): ReplayDocumentReady {
  return testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
      { filmIndex: 1, xuid: 'b1', name: 'Bravo', team: 1 },
    ],
    tracks: [vie(1, 'a1'), vie(2, 'b1')],
    coverage: { gestures: CARRIED },
    ...over,
  } as unknown as Partial<ReplayDocument>)
}

const SHEET = [scoreboardRow('a1', 'Alpha', 't0'), scoreboardRow('b1', 'Bravo', 't1')]

function lecteur(doc: ReplayDocumentReady) {
  const players = buildPlayers(doc, SHEET)
  const index = readGestureSeries(doc, players)
  return (xuid: string, frame: number) => {
    const player = players.find((p) => p.xuid === xuid) as ReplayPlayer
    return playerGesturesAt(gestureSeriesOf(index, player), frame)
  }
}

describe('playerGestureStats — les gestes datés', () => {
  const lire = lecteur(
    docWith({
      gestures: [gesture(30, 0, 'melee'), gesture(10, 0, 'melee'), gesture(20, 0, 'throwStart'), gesture(25, 1, 'melee')],
    }),
  )

  it('compte les coups de mêlée et les lancers amorcés du joueur jusqu’à l’image, quel que soit l’ordre du document', () => {
    expect(lire('a1', 9)).toMatchObject({ melee: 0, throwStarts: 0 })
    expect(lire('a1', 10)).toMatchObject({ melee: 1, throwStarts: 0 })
    expect(lire('a1', 25)).toMatchObject({ melee: 1, throwStarts: 1 })
    expect(lire('a1', 300)).toMatchObject({ melee: 2, throwStarts: 1 })
  })

  it('ne confond pas deux joueurs', () => {
    expect(lire('b1', 24).melee).toBe(0)
    expect(lire('b1', 300).melee).toBe(1)
    expect(lire('b1', 300).throwStarts).toBe(0)
  })

  it('ne recule jamais d’une image à l’autre', () => {
    let last = 0
    for (let f = 0; f <= 300; f++) {
      const n = lire('a1', f).melee as number
      expect(n).toBeGreaterThanOrEqual(last)
      last = n
    }
  })
})

describe('playerGestureStats — ce que le document ne permet pas d’établir', () => {
  it('un film sous la porte de cadence (gestes retenus) ne porte pas les gestes : absents, jamais un zéro', () => {
    const l = lecteur(docWith({ coverage: { gestures: { ...CARRIED, gated: 12 } }, gestures: [] }))
    expect(l('a1', 300)).toMatchObject({ melee: null, throwStarts: null })
  })

  it('sans couverture des gestes (film ancien), rien n’est établi', () => {
    const l = lecteur(docWith({ coverage: {}, gestures: undefined }))
    expect(l('a1', 300)).toEqual({ melee: null, throwStarts: null, crouches: null, jumpPresses: null, minimum: false })
  })

  it('un film porteur sans geste d’un joueur donne un zéro mesuré', () => {
    const l = lecteur(docWith({ gestures: [gesture(10, 1, 'melee')] }))
    expect(l('a1', 300)).toMatchObject({ melee: 0, throwStarts: 0 })
  })

  it('une place que le roster ne désigne pas ne revient à personne', () => {
    const l = lecteur(docWith({ gestures: [gesture(10, 9, 'melee')] }))
    expect(l('a1', 300).melee).toBe(0)
    expect(l('b1', 300).melee).toBe(0)
  })

  it('un joueur sans aucune vie n’a pas de série', () => {
    const doc = docWith({ tracks: [vie(1, 'a1')], gestures: [gesture(10, 1, 'melee')] })
    const players = buildPlayers(doc, SHEET)
    const index = readGestureSeries(doc, players)
    const bravo = players.find((p) => p.xuid === 'b1')
    if (bravo) expect(playerGesturesAt(gestureSeriesOf(index, bravo), 300).melee).toBeNull()
  })
})

describe('playerGestureStats — un plancher quand le film perd des messages', () => {
  it('une trame arrêtée ou un geste sans joueur fait de chaque compte un minimum', () => {
    const arretee = lecteur(docWith({ coverage: { gestures: { ...CARRIED, stoppedFrames: 2 } }, gestures: [gesture(10, 0, 'melee')] }))
    expect(arretee('a1', 300)).toMatchObject({ melee: 1, minimum: true })
    const sansJoueur = lecteur(docWith({ coverage: { gestures: { ...CARRIED, withoutPlayer: 1 } }, gestures: [] }))
    expect(sansJoueur('a1', 300)).toMatchObject({ melee: 0, minimum: true })
  })

  it('aucune perte : le compte n’est pas un minimum', () => {
    expect(lecteur(docWith({ gestures: [gesture(10, 0, 'melee')] }))('a1', 300).minimum).toBe(false)
  })
})

describe('playerGestureStats — un bot', () => {
  const doc = docWith({
    roster: [{ filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 }, { filmIndex: 2, xuid: '', name: 'Marc', team: 1, bot: true }],
    tracks: [vie(1, 'a1'), { ...vie(3, ''), bot: 'Marc' }],
    gestures: [gesture(10, 2, 'melee'), gesture(20, 2, 'throwStart'), gesture(15, 0, 'melee')],
  })
  const players = buildPlayers(doc, [scoreboardRow('a1', 'Alpha', 't0')])
  const index = readGestureSeries(doc, players)

  it('se lit par sa clé de roster `bot:<nom>`, comme tout joueur', () => {
    const marc = players.find((p) => p.xuid === 'bot:Marc') as ReplayPlayer
    expect(playerGesturesAt(gestureSeriesOf(index, marc), 300)).toMatchObject({ melee: 1, throwStarts: 1 })
  })
})

describe('playerGestureStats — les commandes tenues, accroupi et saut', () => {
  const stance = (slot: number, kind: string, t0: number, t1: number) => ({ slot, kind, t0, t1 })
  const stances = [
    stance(1, 'crouchCommand', 10, 20),
    stance(1, 'crouchCommand', 40, 45),
    stance(1, 'jumpCommand', 15, 15),
    stance(2, 'crouchCommand', 12, 30),
  ]
  const lu = lecteur(docWith({ stances, coverage: { gestures: CARRIED, stances: { scanned: true, byKind: { crouchCommand: 3, jumpCommand: 1 } } } }))

  it('compte un accroupi ou un saut par intervalle de commande, dès son premier instant', () => {
    expect(lu('a1', 9)).toMatchObject({ crouches: 0, jumpPresses: 0 })
    expect(lu('a1', 10)).toMatchObject({ crouches: 1, jumpPresses: 0 })
    expect(lu('a1', 15)).toMatchObject({ crouches: 1, jumpPresses: 1 })
    expect(lu('a1', 300)).toMatchObject({ crouches: 2, jumpPresses: 1 })
    expect(lu('b1', 300)).toMatchObject({ crouches: 1, jumpPresses: 0 })
  })

  it('un genre que le film ne publie pas (aucun intervalle au décompte par genre) est absent, pas un zéro', () => {
    const l = lecteur(docWith({ stances: [], coverage: { gestures: CARRIED, stances: { scanned: true, byKind: { sprint: 4 } } } }))
    expect(l('a1', 300)).toMatchObject({ crouches: null, jumpPresses: null })
  })

  it('sans couverture des états de mouvement, rien n’est établi', () => {
    const l = lecteur(docWith({ stances, coverage: { gestures: CARRIED } }))
    expect(l('a1', 300)).toMatchObject({ crouches: null, jumpPresses: null })
  })
})
