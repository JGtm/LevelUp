/**
 * playerThrowsPickups.test.ts — LANCERS DE GRENADE ET RAMASSAGES D'ARMES OU DE GRENADES JUSQU'À L'IMAGE LUE.
 *
 * Un match écrit à la main donne les valeurs exactes : début de match, lancer daté à l'image, joueur mort,
 * bot, grandeur non publiée par l'artefact, comptes qui ne reculent jamais. Les documents publiés par Go
 * (`fixtures/go/`) prouvent, sur des films réels, que le total à la dernière image égale le nombre de lancers
 * du joueur dans le document.
 */
import { describe, expect, it } from 'vitest'

import { buildPlayers, rosterEntryKey, type ReplayPlayer } from '@/lib/replay/rosterLogic'
import { weaponLabelKeyOf } from '../layers/useReplayWeaponPads'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { testReplayDoc } from '../test/testDoc'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayDocument } from '@/lib/api/types'
import {
  GRENADE_PICKUP_KIND_SCHEMA,
  PICKUPS_SCHEMA,
  playerThrowsPickupsAt,
  readPlayerThrowsPickupsSeries,
  type PlayerThrowsPickups,
} from './playerThrowsPickups'

const WEAPON_W = '000012ab'
const WEAPON_KEY = weaponLabelKeyOf(WEAPON_W)

type Grenade = NonNullable<ReplayDocument['grenades']>[number]
type Pickup = NonNullable<ReplayDocument['pickups']>[number]

function grenade(t: number, filmIndex: number, rank: number, slot = 0): Grenade {
  return { t, slot, i: filmIndex, x: 0, y: 0, rank, s: 'throw' }
}

function pickup(t: number, slot: number, kind: string, extra: Partial<Pickup> = {}): Pickup {
  return { t, slot, w: WEAPON_W, kind, class: 0, ...extra } as Pickup
}

/**
 * Le match : A (filmIndex 0) vit deux fois (slot 512 de 0 à 100, slot 513 de 160 à 300) ; B (1) ; un bot
 * « Marc » (2, slot 700, de 0 à 200). Les lancers et les ramassages sont datés dans ces vies.
 */
function matchDoc(over: Partial<ReplayDocument> = {}): ReplayDocumentReady {
  return testReplayDoc({
    frameIntervalMs: 100,
    frameCount: 400,
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
      { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 1 },
      { xuid: '', bot: true, filmIndex: 2, name: 'Marc', team: 1 },
    ],
    tracks: [
      { slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 0, y: 0 }] },
      { slot: 513, team: 0, xuid: 'A', startFrame: 160, endFrame: 300, points: [{ t: 160, x: 0, y: 0 }, { t: 300, x: 0, y: 0 }] },
      { slot: 600, team: 1, xuid: 'B', startFrame: 0, endFrame: 400, points: [{ t: 0, x: 0, y: 0 }, { t: 400, x: 0, y: 0 }] },
      { slot: 700, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }, { t: 200, x: 0, y: 0 }] },
    ],
    grenades: [
      grenade(10, 0, 0, 512),
      grenade(50, 0, 1, 512),
      grenade(50, 0, 0, 512),
      grenade(90, 0, 0, 512),
      grenade(170, 0, 2, 513),
      grenade(60, 1, 0, 600),
      grenade(30, 2, 1, 700),
    ],
    pickups: [
      pickup(20, 512, 'weapon', { xuid: 'A' }),
      pickup(40, 512, 'weapon'),
      pickup(25, 700, 'weapon'),
      pickup(60, 512, 'grenade', { xuid: 'A', family: 'grenade_frag', class: 2 }),
      pickup(70, 512, 'grenade', { class: 2 }),
      pickup(75, 999, 'weapon'),
    ],
    ...over,
  } as Partial<ReplayDocument>)
}

function playersOf(doc: ReplayDocumentReady) {
  return buildPlayers(doc, [])
}

function byXuid(players: ReplayPlayer[], xuid: string): ReplayPlayer {
  const p = players.find((x) => x.xuid === xuid)
  if (!p) throw new Error(`joueur absent : ${xuid}`)
  return p
}

/** Les lectures d'un joueur à l'image, sur le document donné. */
function lire(doc: ReplayDocumentReady, xuid: string, frame: number): PlayerThrowsPickups {
  const players = playersOf(doc)
  const series = readPlayerThrowsPickupsSeries({ doc, players, player: byXuid(players, xuid) })
  return playerThrowsPickupsAt(series, frame)
}

describe('playerThrowsPickups — début de match', () => {
  const doc = matchDoc()

  it('aucun lancer ni ramassage avant le premier : sections vides, pas de zéro publié', () => {
    expect(lire(doc, 'A', 0)).toEqual({ grenadesThrown: [], weaponPickups: [], grenadePickups: [] })
  })

  it('un joueur avec un seul lancer et sans ramassage : listes de ramassage vides, pas absentes', () => {
    const s = lire(doc, 'B', 400)
    expect(s.grenadesThrown).toEqual([{ rank: 0, count: 1 }])
    expect(s.weaponPickups).toEqual([])
    expect(s.grenadePickups).toEqual([])
  })
})

describe('playerThrowsPickups — lancers par type', () => {
  const doc = matchDoc()

  it('les types sont rangés et comptés séparément, un lancer daté à l\'image compte', () => {
    expect(lire(doc, 'A', 49).grenadesThrown).toEqual([{ rank: 0, count: 1 }])
    expect(lire(doc, 'A', 50).grenadesThrown).toEqual([{ rank: 0, count: 2 }, { rank: 1, count: 1 }])
  })

  it('une image avant le lancer ne le compte pas', () => {
    expect(lire(doc, 'A', 9).grenadesThrown).toEqual([])
    expect(lire(doc, 'A', 10).grenadesThrown).toEqual([{ rank: 0, count: 1 }])
  })

  it('le lancer est attribué par l\'index de film, jamais par le slot', () => {
    expect(lire(doc, 'B', 400).grenadesThrown).toEqual([{ rank: 0, count: 1 }])
  })
})

describe('playerThrowsPickups — joueur mort', () => {
  const doc = matchDoc()

  it('après sa mort, le joueur garde ce qu\'il a lancé et ramassé', () => {
    const avant = lire(doc, 'A', 100)
    const mort = lire(doc, 'A', 150)
    expect(mort).toEqual(avant)
    expect(mort.grenadesThrown).toEqual([{ rank: 0, count: 3 }, { rank: 1, count: 1 }])
  })

  it('une vie suivante ajoute ses lancers aux précédents', () => {
    expect(lire(doc, 'A', 170).grenadesThrown).toEqual([
      { rank: 0, count: 3 },
      { rank: 1, count: 1 },
      { rank: 2, count: 1 },
    ])
  })
})

describe('playerThrowsPickups — bot', () => {
  const doc = matchDoc()

  it('un bot est lu par sa clé de roster, lancers et ramassages compris', () => {
    const players = playersOf(doc)
    const bot = byXuid(players, 'bot:Marc')
    const s = playerThrowsPickupsAt(readPlayerThrowsPickupsSeries({ doc, players, player: bot }), 30)
    expect(s.grenadesThrown).toEqual([{ rank: 1, count: 1 }])
    expect(s.weaponPickups).toEqual([{ key: WEAPON_KEY, count: 1 }])
  })
})

describe('playerThrowsPickups — ramassages', () => {
  const doc = matchDoc()

  it('un ramassage nommé par son xuid et un ramassage anonyme dans le slot d\'un joueur comptent pour lui', () => {
    expect(lire(doc, 'A', 19).weaponPickups).toEqual([])
    expect(lire(doc, 'A', 20).weaponPickups).toEqual([{ key: WEAPON_KEY, count: 1 }])
    expect(lire(doc, 'A', 40).weaponPickups).toEqual([{ key: WEAPON_KEY, count: 2 }])
  })

  it("un ramassage sans propriétaire à l'instant n'est compté pour personne", () => {
    const players = playersOf(doc)
    const total = players.reduce((n, p) => {
      const weapon = lire(doc, p.xuid, 400).weaponPickups ?? []
      return n + weapon.reduce((m, w) => m + w.count, 0)
    }, 0)
    // Trois ramassages d'arme attribués (A deux, le bot un) ; celui du slot 999 ne l'est à personne.
    expect(total).toBe(3)
  })

  it('un ramassage de grenade porte son type, et un type non nommé reste à l\'identifiant absent', () => {
    expect(lire(doc, 'A', 60).grenadePickups).toEqual([{ family: 'grenade_frag', count: 1 }])
    expect(lire(doc, 'A', 70).grenadePickups).toEqual([
      { family: 'grenade_frag', count: 1 },
      { family: null, count: 1 },
    ])
  })
})

describe('playerThrowsPickups — grandeur non publiée par l\'artefact', () => {
  it('avant le schéma 30 : aucun ramassage publié, les deux sections sont absentes (null), pas zéro', () => {
    const doc = matchDoc({ schemaVersion: PICKUPS_SCHEMA - 1 })
    const s = lire(doc, 'A', 400)
    expect(s.weaponPickups).toBeNull()
    expect(s.grenadePickups).toBeNull()
    expect(s.grenadesThrown).toEqual([
      { rank: 0, count: 3 },
      { rank: 1, count: 1 },
      { rank: 2, count: 1 },
    ])
  })

  it('au schéma 30 : les armes se comptent, la grenade reste absente (classe 2 lue « item »)', () => {
    const doc = matchDoc({ schemaVersion: PICKUPS_SCHEMA })
    const s = lire(doc, 'A', 400)
    expect(s.weaponPickups).toEqual([{ key: WEAPON_KEY, count: 2 }])
    expect(s.grenadePickups).toBeNull()
  })

  it('au schéma 31 : les grenades se comptent', () => {
    const doc = matchDoc({ schemaVersion: GRENADE_PICKUP_KIND_SCHEMA })
    expect(lire(doc, 'A', 400).grenadePickups).toEqual([
      { family: 'grenade_frag', count: 1 },
      { family: null, count: 1 },
    ])
  })
})

describe('playerThrowsPickups — les comptes ne reculent jamais', () => {
  it('chaque compte est croissant d\'une image à l\'autre', () => {
    const doc = matchDoc()
    const players = playersOf(doc)
    for (const p of players) {
      const series = readPlayerThrowsPickupsSeries({ doc, players, player: p })
      let previous = new Map<string, number>()
      for (let frame = 0; frame <= doc.frameCount; frame++) {
        const now = new Map<string, number>()
        const s = playerThrowsPickupsAt(series, frame)
        for (const g of s.grenadesThrown) now.set(`t${g.rank}`, g.count)
        for (const w of s.weaponPickups ?? []) now.set(`w${w.key}`, w.count)
        for (const g of s.grenadePickups ?? []) now.set(`g${g.family}`, g.count)
        for (const [k, v] of previous) expect(now.get(k) ?? 0, `${p.xuid} image ${frame} ${k}`).toBeGreaterThanOrEqual(v)
        previous = now
      }
    }
  })
})

describe('playerThrowsPickups — documents publiés par Go', () => {
  it('le total de lancers à la dernière image égale le nombre de lancers du joueur dans le document', () => {
    let checked = 0
    for (const entry of goFixtureEntries()) {
      const doc = testReplayDoc(loadGoFixture(entry).doc)
      const players = playersOf(doc)
      for (const p of players) {
        const films = new Set(doc.roster.filter((r) => rosterEntryKey(r) === p.xuid).map((r) => r.filmIndex))
        const expected = doc.grenades.filter((g) => films.has(g.i)).length
        const total = lire(doc, p.xuid, doc.frameCount).grenadesThrown.reduce((n, g) => n + g.count, 0)
        expect(total, `${entry.file} ${p.xuid}`).toBe(expected)
        checked++
      }
    }
    expect(checked).toBeGreaterThan(0)
  })
})
