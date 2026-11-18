/**
 * playerInstantStats.test.ts — CE QU'UN JOUEUR A FAIT DU DÉBUT DU MATCH À L'IMAGE LUE.
 *
 * Deux familles de preuves. Un match écrit à la main (pas de 100 ms) donne les valeurs exactes de chaque
 * grandeur : début de match, joueur mort, bot, coup fatal non mesuré, deux boucliers perdus dans une vie, fin
 * de match égale à la feuille. Les huit documents publiés par Go (`fixtures/go/`) prouvent, sur des films
 * réels, que les médailles, les vies et le bouclier se lisent sans lacune ni rupture de cohérence.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'
import { matchFda } from '@/lib/fda'
import { buildPlayers, vitalityPresence, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

import { buildFeedEntries, type MedalEvent, type ReplayFeedEntry, type ReplayKill } from './killFeedLogic'
import { readMatchMedals } from './matchMedals'
import {
  PERFECT_MEDAL_NAME,
  playerInstantStatsAt,
  readPlayerInstantSeries,
  type PlayerInstantInput,
} from './playerInstantStats'

const STEP_MS = 100

function killOf(replayMs: number, killer: string, headshot?: boolean, medals: MedalEvent[] = []): ReplayKill {
  return {
    tMs: replayMs, xuid: killer, ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '',
    weaponTinted: false, assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null,
    assistDamagePct: null, victimXuid: 'V', victimGamertag: 'V', headshot, replayMs, medals,
  } as ReplayKill
}

function medalOf(xuid: string, name: string, tMs: number): MedalEvent {
  return { tMs, xuid, gamertag: '', name, label: name, description: '', imageUrl: '' }
}

const killEntry = (k: ReplayKill): ReplayFeedEntry => ({ key: `k${k.replayMs}`, replayMs: k.replayMs, kill: k, medal: null, death: null })
const medalEntry = (m: MedalEvent): ReplayFeedEntry => ({ key: `m${m.tMs}`, replayMs: m.tMs, kill: null, medal: m, death: null })

const ticks = (...pairs: [number, number][]) => pairs.map(([t, v]) => ({ t, v }))
const series = (...pairs: [number, number][]) => ({ rounds: [], total: ticks(...pairs) })

/** Le joueur A : trois vies (0-100, 160-300, 360-500), kills 3 / 4 / 1 entre ses morts, deux morts. */
const A_SCORE = {
  xuid: 'A',
  score: series(),
  kills: series([20, 1], [50, 2], [80, 3], [200, 4], [250, 5], [260, 6], [270, 7], [400, 8]),
  deaths: series([100, 1], [300, 2]),
  assists: series([60, 1], [420, 3]),
}

/** Le calque des médailles est déclaré (la cuisson l'a lu) : `coverage.medals` est présent, même à zéro. */
const MEDALS_DECLARED = { medals: { decoded: 0, published: 0, noOrigin: 0, beforeOrigin: 0, afterEnd: 0, unknownName: 0 } } as never

const DOC = testReplayDoc({
  frameIntervalMs: STEP_MS,
  frameCount: 600,
  coverage: MEDALS_DECLARED,
  roster: [
    { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
    { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 1 },
    { xuid: '', bot: true, filmIndex: 2, name: 'Marc', team: 1 },
  ],
  tracks: [
    {
      slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 100,
      points: [
        { t: 0, x: 0, y: 0, sh: 1 }, { t: 30, x: 0, y: 0, sh: 0 }, { t: 40, x: 0, y: 0, sh: 0.5 },
        { t: 60, x: 0, y: 0, sh: 0 }, { t: 70, x: 0, y: 0, sh: 0 },
      ],
    },
    {
      slot: 513, team: 0, xuid: 'A', startFrame: 160, endFrame: 300,
      points: [{ t: 160, x: 0, y: 0, sh: 1 }, { t: 200, x: 0, y: 0, sh: 0 }],
    },
    // Première mesure déjà à zéro à la naissance : pas une chute observée.
    { slot: 514, team: 0, xuid: 'A', startFrame: 360, endFrame: 500, points: [{ t: 360, x: 0, y: 0, sh: 0 }] },
    { slot: 600, team: 1, xuid: 'B', startFrame: 0, endFrame: 500, points: [{ t: 0, x: 0, y: 0, sh: 1 }] },
    {
      slot: 700, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 200,
      points: [{ t: 0, x: 0, y: 0, sh: 1 }, { t: 50, x: 0, y: 0, sh: 0 }],
    },
  ],
})
const PRESENCE = vitalityPresence(DOC)
const [A, B, BOT_BY_ROSTER] = buildPlayers(DOC, [
  scoreboardRow('A', 'Alpha', 'Eagle'), scoreboardRow('B', 'Bravo', 'Cobra'), scoreboardRow('bid(7.0)', 'Marc', 'Cobra', { is_bot: true }),
])
const BOT: ReplayPlayer = BOT_BY_ROSTER

const TIMELINE = {
  players: [
    A_SCORE,
    { xuid: 'B', score: series(), kills: series([10, 1]), deaths: series(), assists: series() },
    // Le bot est publié sous le xuid de sa ligne de feuille.
    { xuid: 'bid(7.0)', score: series(), kills: series([30, 1], [40, 2]), deaths: series([150, 1]), assists: series() },
  ],
  teams: [],
  holdTicks: [],
} as unknown as ReplayScoreTimelineReady

// Éliminations de A : à la tête, non, non mesurée ; puis 20 s : à la tête ; et une médaille parfaite rattachée.
const PERFECT_ATTACHED = medalOf('A', PERFECT_MEDAL_NAME, 2_000)
const FEED: ReplayFeedEntry[] = [
  killEntry(killOf(2_000, 'A', true, [PERFECT_ATTACHED])),
  killEntry(killOf(5_000, 'A', false)),
  killEntry(killOf(8_000, 'A', undefined)),
  killEntry(killOf(20_000, 'A', true)),
  killEntry(killOf(3_000, 'B', true)),
  medalEntry(medalOf('A', PERFECT_MEDAL_NAME, 6_000)),
  medalEntry(medalOf('A', 'Double Kill', 7_000)),
  medalEntry(medalOf('B', PERFECT_MEDAL_NAME, 4_000)),
]

function input(player: ReplayPlayer, over: Partial<PlayerInstantInput> = {}): PlayerInstantInput {
  return { doc: DOC, player, presence: PRESENCE, scoreTimeline: TIMELINE, feed: FEED, ...over }
}
const lire = (player: ReplayPlayer, frame: number, over: Partial<PlayerInstantInput> = {}) =>
  playerInstantStatsAt(readPlayerInstantSeries(input(player, over)), frame)

describe('playerInstantStats — début de match', () => {
  const s = lire(A, 0)

  it('les compteurs sont à zéro et le FDA aussi, mesurés par le film', () => {
    expect(s.counters).toEqual({ kills: 0, deaths: 0, assists: 0 })
    expect(s.fda).toBe(0)
    expect([s.currentStreak, s.bestStreak]).toEqual([0, 0])
  })

  it('aucune élimination, médaille ni bouclier perdu ; vie en cours depuis la naissance ; aucune moyenne', () => {
    expect(s.headshots).toEqual({ count: 0, measured: 0 })
    expect([s.perfects, s.medals]).toEqual([0, []])
    expect([s.shieldsLost, s.respawns, s.currentLifeMs, s.averageLifeMs]).toEqual([0, 0, 0, null])
  })

  it('la première vie se mesure depuis le début du match quand il est donné', () => {
    expect(lire(A, 30, { matchStartFrame: 10 }).currentLifeMs).toBe(2_000)
    expect(lire(A, 30).currentLifeMs).toBe(3_000)
  })
})

describe('playerInstantStats — joueur vivant', () => {
  it('le FDA est le net du jeu, pas le quotient frags / morts', () => {
    const s = lire(A, 90)
    expect(s.counters).toEqual({ kills: 3, deaths: 0, assists: 1 })
    expect(s.fda).toBe(matchFda({ kills: 3, deaths: 0, assists: 1 }))
    expect(s.fda).toBeCloseTo(3 + 1 / 3, 10)
    const apres = lire(A, 350)
    expect(apres.fda).toBeCloseTo(7 + 1 / 3 - 2, 10)
  })

  it('série en cours et meilleure série : la mort remet la série à zéro, la meilleure reste', () => {
    expect([lire(A, 90).currentStreak, lire(A, 90).bestStreak]).toEqual([3, 3])
    expect([lire(A, 150).currentStreak, lire(A, 150).bestStreak]).toEqual([0, 3])
    expect([lire(A, 265).currentStreak, lire(A, 265).bestStreak]).toEqual([3, 3])
    expect([lire(A, 280).currentStreak, lire(A, 280).bestStreak]).toEqual([4, 4])
    expect([lire(A, 410).currentStreak, lire(A, 410).bestStreak]).toEqual([1, 4])
  })

  it('la vie en cours se mesure depuis sa naissance ; réapparitions comptées sans la première vie', () => {
    const s = lire(A, 410)
    expect(s.currentLifeMs).toBe(5_000)
    expect(s.respawns).toBe(2)
    expect(lire(A, 200).respawns).toBe(1)
  })

  it('une vie finie avant le début du match n’est pas une réapparition', () => {
    const [premiere] = A.lives
    const avant = { ...premiere, startFrame: 10, endFrame: 40 }
    const dedans = { ...premiere, startFrame: 41, endFrame: 200 }
    const joueur = { ...A, lives: [avant, dedans] }
    expect(lire(joueur, 100, { matchStartFrame: 50 }).respawns).toBe(0)
    expect(lire(joueur, 100).respawns).toBe(1)
  })

  it('moyenne des vies terminées : celles qui finissent avant l’image', () => {
    expect(lire(A, 100).averageLifeMs).toBeNull()
    expect(lire(A, 101).averageLifeMs).toBe(10_000)
    expect(lire(A, 310).averageLifeMs).toBe((10_000 + 14_000) / 2)
  })
})

describe('playerInstantStats — joueur mort', () => {
  it('la vie en cours est ABSENTE ; compteurs, moyenne et réapparitions restent lus', () => {
    const s = lire(A, 130)
    expect(s.currentLifeMs).toBeNull()
    expect(s.counters).toEqual({ kills: 3, deaths: 1, assists: 1 })
    expect(s.averageLifeMs).toBe(10_000)
    expect(s.respawns).toBe(0)
  })

  it('avant toute vie (pas encore apparu), pas de vie en cours non plus', () => {
    const tard = readPlayerInstantSeries(input({ ...A, lives: A.lives.slice(1) }))
    expect(playerInstantStatsAt(tard, 50).currentLifeMs).toBeNull()
    expect(playerInstantStatsAt(tard, 50).respawns).toBe(0)
  })

  it('un joueur sans aucune vie : ni réapparition, ni vie, ni bouclier affirmés', () => {
    const s = lire({ ...A, lives: [] }, 100)
    expect([s.respawns, s.currentLifeMs, s.averageLifeMs, s.shieldsLost]).toEqual([null, null, null, null])
  })
})

describe('playerInstantStats — boucliers perdus', () => {
  it('une recharge puis une nouvelle chute dans la même vie comptent deux', () => {
    expect(lire(A, 25).shieldsLost).toBe(0)
    expect(lire(A, 35).shieldsLost).toBe(1)
    expect(lire(A, 50).shieldsLost).toBe(1)
    expect(lire(A, 65).shieldsLost).toBe(2)
    // un second point à zéro d'affilée n'est pas une nouvelle chute
    expect(lire(A, 90).shieldsLost).toBe(2)
  })

  it('on réapparaît à plein : la vie suivante compte sa propre chute ; une première mesure à zéro à la naissance ne compte pas', () => {
    expect(lire(A, 250).shieldsLost).toBe(3)
    expect(lire(A, 450).shieldsLost).toBe(3)
  })

  it('un document qui ne publie jamais de bouclier n’affirme aucun compte', () => {
    expect(lire(A, 90, { presence: { shield: false, health: false } }).shieldsLost).toBeNull()
  })
})

describe('playerInstantStats — tirs à la tête', () => {
  it('une élimination dont le coup fatal n’est pas mesuré ne compte ni oui ni non', () => {
    // Trois éliminations à 8,5 s : tête, pas tête, non mesurée.
    expect(lire(A, 85).headshots).toEqual({ count: 1, measured: 2 })
    expect(lire(A, 85).counters?.kills).toBe(3)
    expect(lire(A, 250).headshots).toEqual({ count: 2, measured: 3 })
  })

  it('les éliminations d’un autre joueur ne comptent pas', () => {
    expect(lire(B, 100).headshots).toEqual({ count: 1, measured: 1 })
  })

  it('un fil sans aucune élimination (contexte absent) n’affirme rien', () => {
    expect(lire(A, 90, { feed: [] }).headshots).toBeNull()
    expect(lire(A, 90, { feed: null }).headshots).toBeNull()
    expect(lire(A, 90, { feed: [medalEntry(medalOf('A', 'x', 1))] }).headshots).toBeNull()
  })
})

describe('playerInstantStats — médailles et éliminations parfaites', () => {
  it('la médaille rattachée à une élimination prend l’instant de l’élimination ; les seules gardent le leur', () => {
    expect(lire(A, 19).perfects).toBe(0)
    expect(lire(A, 21).perfects).toBe(1)
    expect(lire(A, 61).perfects).toBe(2)
    expect(lire(A, 61).medals?.map((m) => m.name)).toEqual([PERFECT_MEDAL_NAME, PERFECT_MEDAL_NAME])
    expect(lire(A, 71).medals?.map((m) => m.name)).toEqual([PERFECT_MEDAL_NAME, PERFECT_MEDAL_NAME, 'Double Kill'])
  })

  it('sans feed, les médailles ne s’établissent pas', () => {
    expect([lire(A, 90, { feed: null }).perfects, lire(A, 90, { feed: null }).medals]).toEqual([null, null])
  })

  it('calque des médailles déclaré et vide : liste vide et zéro mesuré', () => {
    expect(lire(B, 10, { feed: [killEntry(killOf(1_000, 'B'))] })).toMatchObject({ perfects: 0, medals: [] })
  })

  it('calque des médailles non déclaré : absentes, jamais à zéro, même pour un humain', () => {
    const sansCalque = testReplayDoc({ frameIntervalMs: STEP_MS, frameCount: 600 })
    const feed = [killEntry(killOf(1_000, 'B'))]
    expect(sansCalque.coverage?.medals).toBeUndefined()
    const s = lire(B, 10, { doc: sansCalque, feed })
    expect([s.perfects, s.medals]).toEqual([null, null])
    // Un document sans couverture du tout (artefact ancien) ne déclare rien non plus.
    const s2 = lire(A, 90, { doc: sansCalque })
    expect([s2.perfects, s2.medals]).toEqual([null, null])
  })
})

describe('playerInstantStats — bot', () => {
  it('un bot se lit sous la clé du roster puis sous le xuid de sa ligne de feuille', () => {
    expect(BOT.xuid).toBe('bot:Marc')
    expect(BOT.board?.xuid).toBe('bid(7.0)')
    const s = lire(BOT, 160)
    expect(s.counters).toEqual({ kills: 2, deaths: 1, assists: 0 })
    expect(s.fda).toBe(1)
    expect([s.currentStreak, s.bestStreak]).toEqual([0, 2])
  })

  it('ses éliminations viennent du xuid de la feuille ; ses vies et son bouclier comptent comme ceux de tout joueur', () => {
    const feed = [killEntry(killOf(3_500, 'bid(7.0)', true)), killEntry(killOf(3_600, 'A', true))]
    const s = lire(BOT, 60, { feed })
    expect(s.headshots).toEqual({ count: 1, measured: 1 })
    expect(s.shieldsLost).toBe(1)
    expect(s.currentLifeMs).toBe(6_000)
  })

  it('le document ne date aucune médaille du bot : absentes, pas à zéro', () => {
    const s = lire(BOT, 160)
    expect([s.perfects, s.medals]).toEqual([null, null])
  })

  it('un bot nommé par une médaille se lit comme un joueur', () => {
    const feed = [medalEntry(medalOf('bid(7.0)', PERFECT_MEDAL_NAME, 1_000))]
    expect(lire(BOT, 160, { feed }).perfects).toBe(1)
  })

  it('un joueur que le calque de score ne publie pas n’a ni compteur, ni FDA, ni série', () => {
    const s = lire(BOT, 160, { scoreTimeline: { ...TIMELINE, players: [] } as ReplayScoreTimelineReady })
    expect([s.counters, s.fda, s.currentStreak, s.bestStreak]).toEqual([null, null, null, null])
    expect(s.shieldsLost).toBe(1)
  })
})

describe('playerInstantStats — fin de match égale à la feuille', () => {
  // Fixation sur un match écrit à la main : aucune fixture du dépôt ne porte à la fois un calque de score
  // et la feuille d'un même match (les documents Go de `fixtures/go/` n'ont pas de `scoreTimeline`).
  it('à la dernière image, frags, morts et assistances de chaque joueur égalent la ligne de la feuille', () => {
    const sheet = [
      scoreboardRow('A', 'Alpha', 'Eagle', { kills: 8, deaths: 2, assists: 3 }),
      scoreboardRow('B', 'Bravo', 'Cobra', { kills: 1, deaths: 0, assists: 0 }),
      scoreboardRow('bid(7.0)', 'Marc', 'Cobra', { is_bot: true, kills: 2, deaths: 1, assists: 0 }),
    ]
    const players = buildPlayers(DOC, sheet)
    const last = DOC.frameCount
    for (const row of sheet) {
      const p = players.find((x) => (x.board?.xuid ?? x.xuid) === row.xuid)!
      const s = lire(p, last)
      expect(s.counters, row.gamertag).toEqual({ kills: row.kills, deaths: row.deaths, assists: row.assists })
      expect(s.fda, row.gamertag).toBe(matchFda({ kills: row.kills, deaths: row.deaths, assists: row.assists }))
    }
  })

  it('la meilleure série de fin de match n’est jamais inférieure à la série en cours ni aux frags d’une vie', () => {
    const s = lire(A, DOC.frameCount)
    expect(s.bestStreak).toBe(4)
    expect(s.bestStreak!).toBeGreaterThanOrEqual(s.currentStreak!)
    expect(s.shieldsLost).toBe(3)
    expect(B.lives).toHaveLength(1)
  })
})

describe('playerInstantStats — documents réels publiés par Go', () => {
  for (const entry of goFixtureEntries()) {
    it(`${entry.film} (${entry.build}) : médailles, vies et bouclier cohérents sur tout le film`, () => {
      const doc = testReplayDoc(loadGoFixture(entry).doc)
      const players = buildPlayers(doc, [])
      const presence = vitalityPresence(doc)
      const feed = buildFeedEntries([], readMatchMedals(doc, 'fr'), 0, doc)
      const last = doc.frameCount + 1_000
      for (const p of players) {
        const read = readPlayerInstantSeries({ doc, player: p, presence, feed })
        const end = playerInstantStatsAt(read, last)
        // Médailles : toutes celles que le document date au joueur, nommées, rien d'autre.
        if (!p.bot) {
          const own = doc.medals.filter((m) => m.xuid === p.xuid && m.name)
          expect(end.medals?.length, p.xuid).toBe(own.length)
          expect(end.perfects, p.xuid).toBe(own.filter((m) => m.name === PERFECT_MEDAL_NAME).length)
        }
        // Vies : la naissance n'est pas une réapparition.
        if (p.lives.length > 0) {
          expect(end.respawns, p.xuid).toBe(p.lives.length - 1)
          expect(end.shieldsLost).not.toBeNull()
        }
        // Le temps ne fait que cumuler : jamais un compte qui recule d'une image à l'autre.
        let previous = { shields: 0, respawns: 0, medals: 0 }
        for (let f = 0; f <= doc.frameCount; f += 97) {
          const s = playerInstantStatsAt(read, f)
          const now = { shields: s.shieldsLost ?? 0, respawns: s.respawns ?? 0, medals: s.medals?.length ?? 0 }
          expect(now.shields).toBeGreaterThanOrEqual(previous.shields)
          expect(now.respawns).toBeGreaterThanOrEqual(previous.respawns)
          expect(now.medals).toBeGreaterThanOrEqual(previous.medals)
          previous = now
        }
      }
    })
  }
})
