/**
 * playerMovementStats.test.ts — LE SPRINT ET LES SAUTS D'UN JOUEUR DU DÉBUT DU MATCH À L'IMAGE LUE.
 *
 * Deux familles de preuves. Des documents écrits à la main (pas de 100 ms) donnent les valeurs exactes : aucun
 * sprint, sprint en cours, sprint terminé, deux sprints, joueur mort, sauts, grandeur non publiée, bot, slot
 * recyclé, intervalles qui se recouvrent. Un cliquet vérifie que les deux comptes ne reculent jamais. Enfin, sur
 * les documents réels publiés par Go (`fixtures/go/`), le sprint à la dernière image égale la somme des
 * intervalles du joueur dans le document, et les sauts leur nombre.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import { buildPlayers, buildSlotOwnership, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { vie } from '../test/equipmentUsageFixtures'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import {
  movementSeriesOf,
  playerMovementStatsAt,
  readMovementSeries,
  STANCE_KIND_JUMP_DERIVED,
  STANCE_KIND_MOVE_COMMAND,
  STANCE_KIND_SPRINT,
} from './playerMovementStats'

/** Une couverture qui établit les deux grandeurs : le balayage a tourné et la dérivation du saut a examiné des montées. */
const COVERED = { stances: { scanned: true, jumpEpisodes: 10, jumpsDerived: 3 } }

const sprint = (slot: number, t0: number, t1: number) => ({ slot, kind: STANCE_KIND_SPRINT, t0, t1 })
const jump = (slot: number, t0: number, t1: number) => ({ slot, kind: STANCE_KIND_JUMP_DERIVED, t0, t1 })

/** Un document de deux joueurs (slots 1 et 2, une vie de 0 à 100), pas de 100 ms, que le test surcharge. */
function docWith(over: Record<string, unknown>): ReplayDocumentReady {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
      { filmIndex: 1, xuid: 'b1', name: 'Bravo', team: 1 },
    ],
    tracks: [vie(1, 'a1'), vie(2, 'b1')],
    coverage: COVERED,
    ...over,
  } as unknown as Partial<ReplayDocument>)
}

const SHEET = [scoreboardRow('a1', 'Alpha', 't0'), scoreboardRow('b1', 'Bravo', 't1')]

/** Le lecteur d'un document : `lire(xuid, frame)` répond pour le joueur de ce xuid de roster. */
function lecteur(doc: ReplayDocumentReady) {
  const players = buildPlayers(doc, SHEET)
  const index = readMovementSeries(doc, players)
  return (xuid: string, frame: number) => {
    const player = players.find((p) => p.xuid === xuid) as ReplayPlayer
    return playerMovementStatsAt(movementSeriesOf(index, player), frame)
  }
}

describe('playerMovementStats — aucun sprint', () => {
  const lire = lecteur(docWith({ stances: [sprint(1, 10, 30)] }))

  it('un joueur sans intervalle a un zéro mesuré, pas une absence', () => {
    expect(lire('b1', 0)).toEqual({ sprintMs: 0, jumps: 0, jumpsEstimated: true, moveMs: null })
    expect(lire('b1', 100)).toEqual({ sprintMs: 0, jumps: 0, jumpsEstimated: true, moveMs: null })
  })

  it('avant le premier sprint, le cumul est nul pour le joueur qui sprintera', () => {
    expect(lire('a1', -1).sprintMs).toBe(0)
    expect(lire('a1', 10).sprintMs).toBe(0)
  })

  it('un intervalle de durée nulle ou négative ne compte pas', () => {
    const l = lecteur(docWith({ stances: [sprint(1, 20, 20), sprint(1, 40, 35)] }))
    expect(l('a1', 100).sprintMs).toBe(0)
  })
})

describe('playerMovementStats — sprint en cours, puis terminé', () => {
  const lire = lecteur(docWith({ stances: [sprint(1, 10, 30)] }))

  it('un sprint en cours à l’image compte jusqu’à elle', () => {
    expect(lire('a1', 15).sprintMs).toBe(500)
    expect(lire('a1', 20).sprintMs).toBe(1000)
  })

  it('un sprint terminé compte sa durée entière, et ne bouge plus ensuite', () => {
    expect(lire('a1', 30).sprintMs).toBe(2000)
    expect(lire('a1', 31).sprintMs).toBe(2000)
    expect(lire('a1', 100).sprintMs).toBe(2000)
  })

  it('le sprint de l’un n’est pas celui de l’autre', () => {
    expect(lire('b1', 20).sprintMs).toBe(0)
  })
})

describe('playerMovementStats — deux sprints', () => {
  const lire = lecteur(docWith({ stances: [sprint(1, 40, 45), sprint(1, 10, 30)] }))

  it('le cumul additionne les sprints terminés et celui en cours, quel que soit l’ordre du document', () => {
    expect(lire('a1', 35).sprintMs).toBe(2000)
    expect(lire('a1', 42).sprintMs).toBe(2200)
    expect(lire('a1', 99).sprintMs).toBe(2500)
  })

  it('deux intervalles qui se recouvrent ou se touchent comptent une seule fois', () => {
    const l = lecteur(docWith({ stances: [sprint(1, 10, 30), sprint(1, 20, 40), sprint(1, 40, 50), sprint(1, 10, 30)] }))
    expect(l('a1', 25).sprintMs).toBe(1500)
    expect(l('a1', 100).sprintMs).toBe(4000)
  })
})

describe('playerMovementStats — joueur mort', () => {
  // Une vie de 0 à 50, puis une autre de 60 à 100 sur le même slot ; le sprint s'arrête avec la première vie.
  const doc = docWith({
    tracks: [vie(1, 'a1', 0, 50), vie(1, 'a1', 60, 100), vie(2, 'b1')],
    stances: [sprint(1, 40, 50), sprint(1, 70, 80)],
  })
  const lire = lecteur(doc)

  it('le cumul se fige pendant la mort', () => {
    expect(lire('a1', 50).sprintMs).toBe(1000)
    expect(lire('a1', 55).sprintMs).toBe(1000)
    expect(lire('a1', 59).sprintMs).toBe(1000)
  })

  it('la vie suivante reprend le cumul où il s’était arrêté', () => {
    expect(lire('a1', 75).sprintMs).toBe(1500)
    expect(lire('a1', 100).sprintMs).toBe(2000)
  })
})

describe('playerMovementStats — sauts', () => {
  const lire = lecteur(docWith({ stances: [jump(1, 35, 40), jump(1, 15, 20), jump(1, 60, 65), jump(2, 5, 9)] }))

  it('un saut compte dès son premier instant, dans l’ordre du temps', () => {
    expect(lire('a1', 14).jumps).toBe(0)
    expect(lire('a1', 15).jumps).toBe(1)
    expect(lire('a1', 36).jumps).toBe(2)
    expect(lire('a1', 100).jumps).toBe(3)
  })

  it('le compte des sauts est une estimation : le document ne publie que des sauts dérivés', () => {
    expect(lire('a1', 100).jumpsEstimated).toBe(true)
    expect(lire('b1', 100)).toEqual({ sprintMs: 0, jumps: 1, jumpsEstimated: true, moveMs: null })
  })

  it('les sauts ne se mêlent pas au sprint', () => {
    const l = lecteur(docWith({ stances: [jump(1, 15, 20), sprint(1, 10, 30)] }))
    expect(l('a1', 100)).toEqual({ sprintMs: 2000, jumps: 1, jumpsEstimated: true, moveMs: null })
  })

  it('les autres états de mouvement ne sont ni un sprint ni un saut', () => {
    const l = lecteur(docWith({ stances: [{ slot: 1, kind: 'crouch', t0: 10, t1: 30 }, { slot: 1, kind: 'slide', t0: 5, t1: 8 }] }))
    expect(l('a1', 100)).toEqual({ sprintMs: 0, jumps: 0, jumpsEstimated: true, moveMs: null })
  })
})

describe('playerMovementStats — grandeur non publiée', () => {
  const stances = [sprint(1, 10, 30), jump(1, 15, 20)]
  const ABSENT = { sprintMs: null, jumps: null, jumpsEstimated: false, moveMs: null }

  it('sans couverture des états de mouvement (artefact antérieur), rien n’est établi', () => {
    expect(lecteur(docWith({ stances, coverage: {} }))('a1', 100)).toEqual(ABSENT)
    expect(lecteur(docWith({ stances, coverage: undefined }))('a1', 100)).toEqual(ABSENT)
  })

  it('un balayage refusé ou un film sans les composants n’établit rien, même avec des intervalles', () => {
    const refuse = { stances: { scanned: false, jumpEpisodes: 10 } }
    const absent = { stances: { scanned: true, absent: true, jumpEpisodes: 10 } }
    expect(lecteur(docWith({ stances, coverage: refuse }))('a1', 100)).toEqual(ABSENT)
    expect(lecteur(docWith({ stances, coverage: absent }))('a1', 100)).toEqual(ABSENT)
  })

  it('une dérivation du saut qui n’a examiné aucune montée n’établit pas les sauts, mais le sprint reste établi', () => {
    for (const coverage of [{ stances: { scanned: true } }, { stances: { scanned: true, jumpEpisodes: 0 } }]) {
      expect(lecteur(docWith({ stances, coverage }))('a1', 100)).toEqual({
        sprintMs: 2000,
        jumps: null,
        jumpsEstimated: false,
        moveMs: null,
      })
    }
  })

  it('un document sans aucun intervalle mais balayé donne des zéros mesurés, pas des absences', () => {
    expect(lecteur(docWith({ stances: [] }))('a1', 100)).toEqual({ sprintMs: 0, jumps: 0, jumpsEstimated: true, moveMs: null })
  })

  it('un joueur sans aucune vie n’a pas de série : rien n’est établi pour lui', () => {
    const doc = docWith({
      roster: [
        { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
        { filmIndex: 1, xuid: 'b1', name: 'Bravo', team: 1 },
        { filmIndex: 2, xuid: 'c1', name: 'Charlie', team: 1 },
      ],
      stances,
    })
    const players = buildPlayers(doc, SHEET)
    const sans = players.find((p) => p.xuid === 'c1') as ReplayPlayer
    expect(sans.lives).toHaveLength(0)
    expect(movementSeriesOf(readMovementSeries(doc, players), sans)).toBeNull()
    expect(playerMovementStatsAt(null, 50)).toEqual(ABSENT)
  })

  it('un intervalle dont le slot n’a pas de propriétaire à son premier instant ne revient à personne', () => {
    const l = lecteur(docWith({ stances: [sprint(9, 10, 30), sprint(1, 150, 190)], tracks: [vie(1, 'a1'), vie(2, 'b1')] }))
    expect(l('a1', 100).sprintMs).toBe(0)
    expect(l('b1', 100).sprintMs).toBe(0)
  })
})

describe('playerMovementStats — bot', () => {
  const doc = docWith({
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
      { filmIndex: 8, xuid: '', bot: true, name: 'Marc', team: 1 },
    ],
    tracks: [vie(1, 'a1'), { ...vie(8, ''), bot: 'Marc' }],
    stances: [sprint(8, 10, 30), jump(8, 12, 15), jump(1, 20, 25)],
  })
  const sheet = [scoreboardRow('a1', 'Alpha', 't0'), scoreboardRow('bid(7.0)', 'Marc', 't1', { is_bot: true })]
  const players = buildPlayers(doc, sheet)
  const index = readMovementSeries(doc, players)
  const bot = players.find((p) => p.bot) as ReplayPlayer

  it('se lit par sa clé de roster `bot:<nom>`, comme tout joueur', () => {
    expect(bot.xuid).toBe('bot:Marc')
    expect(playerMovementStatsAt(movementSeriesOf(index, bot), 100)).toEqual({
      sprintMs: 2000,
      jumps: 1,
      jumpsEstimated: true,
      moveMs: null,
    })
  })

  it('se lit aussi par le xuid de sa ligne de feuille', () => {
    expect(bot.board?.xuid).toBe('bid(7.0)')
    expect(index.get('bid(7.0)')).toBe(index.get('bot:Marc'))
  })

  it('les intervalles du bot ne vont pas à l’humain, et inversement', () => {
    const humain = players.find((p) => p.xuid === 'a1') as ReplayPlayer
    expect(playerMovementStatsAt(movementSeriesOf(index, humain), 100)).toEqual({
      sprintMs: 0,
      jumps: 1,
      jumpsEstimated: true,
      moveMs: null,
    })
  })
})

describe('playerMovementStats — attribution', () => {
  it('rend l’intervalle à la VIE qui occupe le slot à son premier instant, pas au dernier occupant', () => {
    const doc = docWith({
      tracks: [vie(1, 'a1', 0, 50), vie(1, 'b1', 60, 100)],
      stances: [sprint(1, 10, 30), sprint(1, 70, 80)],
    })
    const l = lecteur(doc)
    expect(l('a1', 100).sprintMs).toBe(2000)
    expect(l('b1', 100).sprintMs).toBe(1000)
  })
})

describe('playerMovementStats — les comptes ne reculent jamais', () => {
  it('image par image, pour chaque joueur, sprint et sauts ne font que croître', () => {
    const doc = docWith({
      tracks: [vie(1, 'a1', 0, 50), vie(1, 'a1', 60, 100), vie(2, 'b1')],
      stances: [sprint(1, 10, 30), sprint(1, 20, 40), sprint(1, 70, 90), sprint(2, 5, 95), jump(1, 12, 15), jump(1, 80, 85)],
    })
    const lire = lecteur(doc)
    for (const xuid of ['a1', 'b1']) {
      let sprintMs = 0
      let jumps = 0
      for (let f = -2; f <= 130; f++) {
        const s = lire(xuid, f)
        expect(s.sprintMs as number).toBeGreaterThanOrEqual(sprintMs)
        expect(s.jumps as number).toBeGreaterThanOrEqual(jumps)
        sprintMs = s.sprintMs as number
        jumps = s.jumps as number
      }
    }
  })
})

/** La somme des sprints et le nombre de sauts de chaque joueur d'un document, lus sans passer par le module. */
function expectedByPlayer(doc: ReplayDocumentReady, players: readonly ReplayPlayer[]) {
  const ownerAt = buildSlotOwnership(players).ownerAtFrame
  const out = new Map<string, { sprintFrames: number; jumps: number }>()
  for (const s of doc.stances) {
    const owner = ownerAt(s.slot, s.t0)
    if (!owner) continue
    const cur = out.get(owner.xuid) ?? { sprintFrames: 0, jumps: 0 }
    if (s.kind === STANCE_KIND_SPRINT) cur.sprintFrames += s.t1 - s.t0
    if (s.kind === STANCE_KIND_JUMP_DERIVED) cur.jumps += 1
    out.set(owner.xuid, cur)
  }
  return out
}

describe('playerMovementStats — fin de match égale la somme des intervalles (documents réels publiés par Go)', () => {
  it('chaque document, joueur par joueur', () => {
    let compared = 0
    let withSprint = 0
    for (const entry of goFixtureEntries()) {
      const doc = testReplayDoc(loadGoFixture(entry).doc)
      const players = buildPlayers(doc, [])
      const index = readMovementSeries(doc, players)
      const expected = expectedByPlayer(doc, players)
      const interval = doc.frameIntervalMs as number
      for (const player of players) {
        if (player.lives.length === 0) continue
        const want = expected.get(player.xuid) ?? { sprintFrames: 0, jumps: 0 }
        const got = playerMovementStatsAt(movementSeriesOf(index, player), doc.frameCount)
        expect(got.sprintMs, `${entry.film} ${player.xuid} sprint`).toBeCloseTo(want.sprintFrames * interval, 6)
        expect(got.jumps, `${entry.film} ${player.xuid} sauts`).toBe(want.jumps)
        expect(got.jumpsEstimated).toBe(true)
        compared++
        if (want.sprintFrames > 0) withSprint++
      }
    }
    // Sans matière dans les documents réels, l'égalité ne prouverait rien.
    expect(compared).toBeGreaterThan(0)
    expect(withSprint).toBeGreaterThan(0)
  })

  it('les deux comptes ne reculent jamais, image par image, sur un document réel', () => {
    const entry = goFixtureEntries()[0]
    const doc = testReplayDoc(loadGoFixture(entry).doc)
    const players = buildPlayers(doc, [])
    const index = readMovementSeries(doc, players)
    for (const player of players) {
      const series = movementSeriesOf(index, player)
      let sprintMs = 0
      let jumps = 0
      for (let f = 0; f <= doc.frameCount; f += 7) {
        const s = playerMovementStatsAt(series, f)
        if (series === null) continue
        expect(s.sprintMs as number).toBeGreaterThanOrEqual(sprintMs)
        expect(s.jumps as number).toBeGreaterThanOrEqual(jumps)
        sprintMs = s.sprintMs as number
        jumps = s.jumps as number
      }
    }
  })
})

describe('playerMovementStats — commande de déplacement tenue', () => {
  const control = { holds: 4, withHole: 0, innerHoles: 0, entries: 9, published: 4, intervals: 4, noTrack: 0, noPlayer: 0 }
  const COVERED_WITH_CONTROL = { stances: { scanned: true, jumpEpisodes: 10, jumpsDerived: 3, control } }
  const move = (slot: number, t0: number, t1: number) => ({ slot, kind: STANCE_KIND_MOVE_COMMAND, t0, t1 })

  it('somme le temps tenu jusqu’à l’image, une commande en cours comptant jusqu’à elle', () => {
    const lire = lecteur(docWith({ coverage: COVERED_WITH_CONTROL, stances: [move(1, 10, 30), move(1, 50, 60)] }))
    expect(lire('a1', 5).moveMs).toBe(0)
    expect(lire('a1', 20).moveMs).toBe(1000)
    expect(lire('a1', 40).moveMs).toBe(2000)
    expect(lire('a1', 55).moveMs).toBe(2500)
    expect(lire('a1', 100).moveMs).toBe(3000)
  })

  it('le temps du sprint et celui de la commande ne se mélangent pas', () => {
    const lire = lecteur(docWith({ coverage: COVERED_WITH_CONTROL, stances: [move(1, 10, 30), sprint(1, 10, 20)] }))
    expect(lire('a1', 100).moveMs).toBe(2000)
    expect(lire('a1', 100).sprintMs).toBe(1000)
  })

  it('un joueur sans commande a un zéro mesuré quand la vue de contrôle est lue', () => {
    const lire = lecteur(docWith({ coverage: COVERED_WITH_CONTROL, stances: [move(1, 10, 30)] }))
    expect(lire('b1', 100).moveMs).toBe(0)
  })

  it('sans couverture de la vue de contrôle, la grandeur est absente, jamais un zéro', () => {
    const lire = lecteur(docWith({ stances: [move(1, 10, 30)] }))
    expect(lire('a1', 100).moveMs).toBeNull()
    expect(lire('b1', 100).moveMs).toBeNull()
  })

  it('un balayage refusé n’établit pas la commande non plus', () => {
    const refused = { stances: { scanned: false, jumpEpisodes: 0, jumpsDerived: 0, control } }
    const lire = lecteur(docWith({ coverage: refused, stances: [move(1, 10, 30)] }))
    expect(lire('a1', 100).moveMs).toBeNull()
  })
})
