/**
 * livesMarks.test.ts — les règles des traces des vies qui ne se voient pas à l'œil : une apparition
 * par vie à sa première image visible, un cap jamais emprunté à une autre image, une marque
 * sans position COMPTÉE et jamais devinée, le filtre « qui », le fondu par âge et l'apparition
 * d'une vie en cours qui reste pleine.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { testReplayDoc } from '../test/testDoc'
import { buildCarrierPosAt } from './carrierPosition'
import type { ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import {
  buildLivesMarks,
  effectiveLivesWho,
  fadeByAge,
  LIVES_FADE_FLOOR,
  LIVES_FADE_MS,
  markAlpha,
  passesWho,
  type LivesResolvers,
} from './livesMarks'

function kill(over: Partial<ReplayKill> = {}): ReplayKill {
  return {
    replayMs: 2_000,
    medals: [],
    tMs: 1_000,
    xuid: 'K',
    teamID: 0,
    weaponKey: 'hinf_br75',
    weaponLabel: 'BR75',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: 'V',
    victimGamertag: 'Victime',
    ...over,
  }
}

function entry(over: Partial<ReplayFeedEntry>): ReplayFeedEntry {
  return { key: 'e', replayMs: 0, kill: null, medal: null, death: null, ...over }
}

/**
 * Un document 10 Hz : K vit de 0 à 100 (cap 90 degrés à la première image) ; V a deux vies, 0-20
 * (aucun cap à la première image, un cap plus tard) et 50-90 (cap 180 à la première image).
 */
function doc(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      {
        slot: 1, team: 0, xuid: 'K', startFrame: 0, endFrame: 100,
        points: [{ t: 0, x: 0, y: 0, h: 90 }, { t: 100, x: 10, y: 0 }],
      },
      {
        slot: 2, team: 1, xuid: 'V', startFrame: 0, endFrame: 20,
        points: [{ t: 0, x: 5, y: 5 }, { t: 10, x: 5, y: 5, h: 45 }, { t: 20, x: 5, y: 4 }],
      },
      {
        slot: 2, team: 1, xuid: 'V', startFrame: 50, endFrame: 90,
        points: [{ t: 50, x: 8, y: 8, h: 180 }, { t: 90, x: 8, y: 6 }],
      },
    ],
    ...over,
  })
}

/** Les lectures du canvas : K est de mon équipe et c'est le joueur suivi ; V est adverse. */
function resolvers(d: ReturnType<typeof doc>, over: Partial<LivesResolvers> = {}): LivesResolvers {
  return {
    posOf: buildCarrierPosAt(d),
    seated: () => true,
    colorOfXuid: (x) => (x === 'K' ? 'ink-ally' : 'ink-enemy'),
    colorOfSlot: (slot) => (slot === 1 ? 'ink-ally' : 'ink-enemy'),
    mineOfXuid: (x) => x === 'K',
    mineOfSlot: (slot) => slot === 1,
    followedOfXuid: (x) => x === 'K',
    followedOfSlot: (slot) => slot === 1,
    ...over,
  }
}

describe('buildLivesMarks — les apparitions', () => {
  it('une marque par vie, à sa première image, avec l’encre de son propriétaire', () => {
    const d = doc()
    const m = buildLivesMarks(d, [], resolvers(d))
    expect(m.spawns.map((s) => [s.frame, s.x, s.y, s.color])).toEqual([
      [0, 0, 0, 'ink-ally'],
      [0, 5, 5, 'ink-enemy'],
      [50, 8, 8, 'ink-enemy'],
    ])
    expect(m.spawns.map((s) => s.endFrame)).toEqual([100, 20, 90])
  })

  it('le cap est celui de la première image de la vie, ou rien : jamais celui d’une image plus tardive', () => {
    const d = doc()
    const m = buildLivesMarks(d, [], resolvers(d))
    // K : cap lu à l'image 0. V (1re vie) : le cap n'arrive qu'à l'image 10, donc anneau seul.
    expect(m.spawns.map((s) => s.headingDeg)).toEqual([90, null, 180])
  })

  it('une vie qui n’est dessinée qu’à son premier point apparaît à cette image, pas avant', () => {
    const d = doc({
      tracks: [
        { slot: 1, team: 0, xuid: 'K', startFrame: 0, endFrame: 100, points: [{ t: 30, x: 3, y: 4, h: 10 }, { t: 40, x: 4, y: 4 }] },
      ],
    })
    const m = buildLivesMarks(d, [], resolvers(d))
    expect(m.spawns).toHaveLength(1)
    expect(m.spawns[0]).toMatchObject({ frame: 30, x: 3, y: 4, headingDeg: 10 })
  })

  it('une vie sans propriétaire coloré n’a pas de marque, et elle est comptée', () => {
    const d = doc()
    const m = buildLivesMarks(d, [], resolvers(d, { colorOfSlot: (s) => (s === 1 ? 'ink-ally' : null) }))
    expect(m.spawns).toHaveLength(1)
    expect(m.skipped.spawns).toBe(2)
  })

  it('une vie sans aucun point est ignorée (rien à poser, rien à compter)', () => {
    const d = doc({ tracks: [{ slot: 1, team: 0, xuid: 'K', points: [] }] })
    const m = buildLivesMarks(d, [], resolvers(d))
    expect(m.spawns).toEqual([])
    expect(m.skipped.spawns).toBe(0)
  })
})

describe('buildLivesMarks — les morts et les tueurs', () => {
  it('un frag donne une croix à la victime et un triangle au tueur avec le trait vers la victime', () => {
    const d = doc()
    const m = buildLivesMarks(d, [entry({ kill: kill() })], resolvers(d))
    // Image 20 : K est interpolé en (2, 0) ; V est en (5, 4) à sa dernière position.
    expect(m.deaths).toEqual([
      { frame: 20, x: 5, y: 4, color: 'ink-enemy', mine: false, followed: false },
    ])
    expect(m.killers).toEqual([
      { frame: 20, x: 2, y: 0, victim: { x: 5, y: 4 }, color: 'ink-ally', mine: true, followed: true },
    ])
  })

  it('une mort sans tueur donne une croix, jamais un triangle', () => {
    const d = doc()
    const death = { replayMs: 9_000, xuid: 'V', kind: '', img: '', tinted: false }
    const m = buildLivesMarks(d, [entry({ death })], resolvers(d))
    expect(m.deaths).toHaveLength(1)
    expect(m.deaths[0]).toMatchObject({ frame: 90, x: 8, y: 6 })
    expect(m.killers).toEqual([])
  })

  it('un frag sans victime nommée donne le tueur, sans trait, et compte une croix manquante', () => {
    const d = doc()
    const m = buildLivesMarks(d, [entry({ kill: kill({ victimXuid: '' }) })], resolvers(d))
    expect(m.deaths).toEqual([])
    expect(m.skipped.deaths).toBe(1)
    expect(m.killers).toHaveLength(1)
    expect(m.killers[0].victim).toBeNull()
  })

  it('un joueur qui est sa propre victime n’est pas son tueur : la croix reste', () => {
    const d = doc()
    const m = buildLivesMarks(d, [entry({ kill: kill({ xuid: 'V', victimXuid: 'V' }) })], resolvers(d))
    expect(m.deaths).toHaveLength(1)
    expect(m.killers).toEqual([])
    expect(m.skipped.killers).toBe(0)
  })

  it('rien à une position inventée : un tueur sans fiche, ou sans position, est compté et absent', () => {
    const d = doc()
    const sansFiche = buildLivesMarks(d, [entry({ kill: kill() })], resolvers(d, { seated: (x) => x !== 'K' }))
    expect(sansFiche.killers).toEqual([])
    expect(sansFiche.skipped.killers).toBe(1)
    expect(sansFiche.deaths).toHaveLength(1)
    const sansPosition = buildLivesMarks(d, [entry({ kill: kill({ xuid: 'X' }) })], resolvers(d))
    expect(sansPosition.killers).toEqual([])
    expect(sansPosition.skipped.killers).toBe(1)
  })

  it('une victime introuvable est comptée sans croix ; le tueur reste, sans trait', () => {
    const d = doc()
    const m = buildLivesMarks(d, [entry({ kill: kill({ victimXuid: 'Z' }) })], resolvers(d))
    expect(m.deaths).toEqual([])
    expect(m.skipped.deaths).toBe(1)
    expect(m.killers[0].victim).toBeNull()
  })

  it('followedKnown dit si une marque porte le joueur suivi', () => {
    const d = doc()
    expect(buildLivesMarks(d, [], resolvers(d)).followedKnown).toBe(true)
    const nobody = resolvers(d, { followedOfXuid: () => false, followedOfSlot: () => false })
    expect(buildLivesMarks(d, [entry({ kill: kill() })], nobody).followedKnown).toBe(false)
  })
})

describe('le filtre « qui »', () => {
  const m = { color: 'c', mine: true, followed: false }
  const other = { color: 'c', mine: false, followed: true }

  it('tous garde tout, mon équipe garde les miens, joueur suivi garde le suivi', () => {
    expect([passesWho(m, 'all'), passesWho(other, 'all')]).toEqual([true, true])
    expect([passesWho(m, 'team'), passesWho(other, 'team')]).toEqual([true, false])
    expect([passesWho(m, 'followed'), passesWho(other, 'followed')]).toEqual([false, true])
  })

  it('« joueur suivi » sans joueur suivi retombe sur « tous », les autres choix sont inchangés', () => {
    expect(effectiveLivesWho('followed', false)).toBe('all')
    expect(effectiveLivesWho('followed', true)).toBe('followed')
    expect(effectiveLivesWho('team', false)).toBe('team')
  })
})

describe('le filtre « quand » et le fondu par âge', () => {
  const FRAME_MS = 100
  const framesFor = (ms: number) => ms / FRAME_MS

  it('toute la partie : pleine opacité, quelle que soit l’image', () => {
    expect(markAlpha(500, { when: 'all', frame: 0, frameMs: FRAME_MS })).toBe(1)
  })

  it('jusqu’ici : rien avant l’image de la marque, pleine à l’instant, plus pâle avec l’âge', () => {
    const t = (frame: number) => ({ when: 'sofar' as const, frame, frameMs: FRAME_MS })
    expect(markAlpha(100, t(99))).toBe(0)
    expect(markAlpha(100, t(100))).toBe(1)
    const mi = markAlpha(100, t(100 + framesFor(LIVES_FADE_MS / 2)))
    expect(mi).toBeGreaterThan(LIVES_FADE_FLOOR)
    expect(mi).toBeLessThan(1)
    expect(markAlpha(100, t(100 + framesFor(LIVES_FADE_MS / 2) + 1))).toBeLessThan(mi)
  })

  it('le plancher tient : une marque très ancienne reste visible à 15 %', () => {
    expect(fadeByAge(framesFor(LIVES_FADE_MS), FRAME_MS)).toBeCloseTo(LIVES_FADE_FLOOR, 10)
    expect(fadeByAge(framesFor(LIVES_FADE_MS * 10), FRAME_MS)).toBeCloseTo(LIVES_FADE_FLOOR, 10)
    expect(LIVES_FADE_FLOOR).toBe(0.15)
  })

  it('l’apparition d’une vie EN COURS reste pleine ; passé sa fin, elle vieillit depuis cette fin', () => {
    const t = (frame: number) => ({ when: 'sofar' as const, frame, frameMs: FRAME_MS })
    // Vie de l'image 0 à 5 000 images : aucune pâleur tant qu'elle dure, même 20 minutes après son début.
    expect(markAlpha(0, t(5_000), 5_000)).toBe(1)
    expect(markAlpha(0, t(5_001), 5_000)).toBeLessThan(1)
    expect(markAlpha(0, t(5_000 + framesFor(LIVES_FADE_MS)), 5_000)).toBeCloseTo(LIVES_FADE_FLOOR, 10)
    // Et jamais avant son apparition.
    expect(markAlpha(50, t(49), 100)).toBe(0)
  })
})
