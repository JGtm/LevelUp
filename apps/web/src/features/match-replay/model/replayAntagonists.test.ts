import { describe, expect, it } from 'vitest'

import { botKey, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import {
  antagonistKillsOf,
  antagonistSubjectKey,
  botKillsOf,
  createReplayAntagonists,
} from './replayAntagonists'

type Board = ReplayPlayer['board']

/** Un joueur du film : clé, équipe, ligne de feuille éventuelle, bot ou non. */
function player(xuid: string, team: number | undefined, extra: Partial<ReplayPlayer> = {}): ReplayPlayer {
  return { xuid, team, lives: [], ...extra } as ReplayPlayer
}

/** Une ligne du fil portant une élimination. */
function killEntry(
  xuid: string,
  victimXuid: string,
  replayMs: number,
  victimGamertag = '',
): ReplayFeedEntry {
  return {
    key: `${xuid}>${victimXuid}${victimGamertag}@${replayMs}`,
    replayMs,
    kill: { xuid, victimXuid, victimGamertag, replayMs, medals: [] } as unknown as ReplayKill,
    medal: null,
    death: null,
  }
}

const ME = 'xuid(me)'
const ALLY = 'xuid(ally)'
const FOE1 = 'xuid(foe1)'
const FOE2 = 'xuid(foe2)'
const BOT_FOE = botKey('Spartan-117')

/** 4 contre 4 réduit : deux alliés (camp 0), deux adversaires humains et un bot adverse (camp 1). */
const campagne: ReplayPlayer[] = [
  player(ME, 0),
  player(ALLY, 0),
  player(FOE1, 1),
  player(FOE2, 1),
  player(BOT_FOE, 1, { bot: true, board: { xuid: 'bid(3.0)' } as Board }),
]

describe('antagonistKillsOf : clés du film', () => {
  it('prend le tueur dans kill.xuid et la victime dans kill.victimXuid', () => {
    expect(antagonistKillsOf([killEntry(FOE1, ME, 1000)], campagne)).toEqual([
      { killer: FOE1, victim: ME, t: 1000 },
    ])
  })

  it("retombe sur botKey(nom) quand la victime n'a pas de xuid", () => {
    expect(antagonistKillsOf([killEntry(ME, '', 2000, 'Spartan-117')], campagne)).toEqual([
      { killer: ME, victim: BOT_FOE, t: 2000 },
    ])
  })

  it('ignore une élimination sans victime nommée', () => {
    expect(antagonistKillsOf([killEntry(FOE1, '', 1000, '')], campagne)).toEqual([])
  })

  it('ignore les lignes du fil qui ne sont pas des éliminations', () => {
    const medal: ReplayFeedEntry = { key: 'm', replayMs: 5, kill: null, medal: null, death: null }
    expect(antagonistKillsOf([medal], campagne)).toEqual([])
  })

  it('ramène un xuid de feuille à la clé du film du joueur', () => {
    const roster = [player('film-key', 1, { board: { xuid: 'xuid(feuille)' } as Board }), player(ME, 0)]
    expect(antagonistKillsOf([killEntry('xuid(feuille)', ME, 10)], roster)).toEqual([
      { killer: 'film-key', victim: ME, t: 10 },
    ])
  })
})

describe('antagonistSubjectKey : le point de vue ramené à la clé du film', () => {
  it('retrouve la clé du film par la ligne de feuille', () => {
    const roster = [player('film-key', 0, { board: { xuid: 'xuid(feuille)' } as Board })]
    expect(antagonistSubjectKey('xuid(feuille)', roster)).toBe('film-key')
  })

  it('accepte la clé du film elle-même', () => {
    expect(antagonistSubjectKey(ME, campagne)).toBe(ME)
  })

  it('rend null sans point de vue ou pour un joueur absent du roster', () => {
    expect(antagonistSubjectKey(null, campagne)).toBeNull()
    expect(antagonistSubjectKey('xuid(inconnu)', campagne)).toBeNull()
  })
})

describe("createReplayAntagonists : l'index d'un sujet", () => {
  it('compte les adversaires humains dans les deux rôles', () => {
    const feed = [killEntry(FOE1, ME, 1000), killEntry(ME, FOE2, 2000), killEntry(FOE1, ME, 3000)]
    const at = createReplayAntagonists(feed, campagne).forSubject(ME).at(3000)
    expect(at.nemesis).toEqual({ holder: FOE1, count: 2, tiedWith: [] })
    expect(at.victim).toEqual({ holder: FOE2, count: 1, tiedWith: [] })
  })

  it('un bot est un antagoniste comme un autre : la victime bot du sujet est son souffre-douleur', () => {
    const feed = [killEntry(ME, '', 1000, 'Spartan-117'), killEntry(ME, '', 1500, 'Spartan-117')]
    const at = createReplayAntagonists(feed, campagne).forSubject(ME).at(1500)
    expect(at.victim).toEqual({ holder: BOT_FOE, count: 2, tiedWith: [] })
  })

  it('ignore un tueur sans victime nommée', () => {
    const feed = [killEntry(FOE1, '', 1000, '')]
    expect(createReplayAntagonists(feed, campagne).forSubject(ME).changes).toEqual([])
  })

  it('un sujet bot a ses antagonistes : l’adversaire qui l’élimine est sa némésis, celui qu’il élimine sa victime', () => {
    const feed = [killEntry(ME, '', 1000, 'Spartan-117'), killEntry(ME, '', 1200, 'Spartan-117')]
    const botKills = [{ killer: BOT_FOE, victim: ALLY, t: 2000 }, { killer: BOT_FOE, victim: ALLY, t: 2500 }]
    const at = createReplayAntagonists(feed, campagne, botKills).forSubject(BOT_FOE).at(2500)
    expect(at.nemesis).toEqual({ holder: ME, count: 2, tiedWith: [] })
    expect(at.victim).toEqual({ holder: ALLY, count: 2, tiedWith: [] })
  })

  it('une élimination dont un bot est l’auteur (lue dans le film) fait de lui la némésis de sa victime', () => {
    const botKills = [{ killer: BOT_FOE, victim: ME, t: 3000 }]
    const at = createReplayAntagonists([], campagne, botKills).forSubject(ME).at(3000)
    expect(at.nemesis).toEqual({ holder: BOT_FOE, count: 1, tiedWith: [] })
  })

  it('compte tous les autres joueurs en mêlée générale (équipe -1)', () => {
    const melee = [player(ME, -1), player(ALLY, -1), player(FOE1, -1)]
    const feed = [killEntry(ALLY, ME, 1000), killEntry(ME, FOE1, 2000)]
    const at = createReplayAntagonists(feed, melee).forSubject(ME).at(2000)
    expect(at.nemesis).toEqual({ holder: ALLY, count: 1, tiedWith: [] })
    expect(at.victim).toEqual({ holder: FOE1, count: 1, tiedWith: [] })
  })

  it('ignore une trahison (tueur du même camp) et un coup sur un allié', () => {
    const feed = [killEntry(ALLY, ME, 1000), killEntry(ME, ALLY, 2000)]
    expect(createReplayAntagonists(feed, campagne).forSubject(ME).changes).toEqual([])
  })

  it('écarte les doublons du fil (même tueur, même victime, même instant)', () => {
    const feed = [killEntry(FOE1, ME, 1000), killEntry(FOE1, ME, 1000), killEntry(FOE2, ME, 2000)]
    const at = createReplayAntagonists(feed, campagne).forSubject(ME).at(2000)
    expect(at.nemesis).toEqual({ holder: FOE1, count: 1, tiedWith: [FOE2] })
  })

  it("lit l'adversaire du point de vue du sujet, pas celui d'un autre joueur", () => {
    // FOE1 sujet : ME est son adversaire ; FOE2 (même camp que lui) ne l'est pas.
    const feed = [killEntry(ME, FOE1, 1000), killEntry(FOE2, FOE1, 2000)]
    const at = createReplayAntagonists(feed, campagne).forSubject(FOE1).at(2000)
    expect(at.nemesis).toEqual({ holder: ME, count: 1, tiedWith: [] })
  })
})

describe('createReplayAntagonists : cache par (document, sujet)', () => {
  const feed = [killEntry(FOE1, ME, 1000), killEntry(FOE1, ALLY, 1500), killEntry(ME, FOE1, 2000)]

  it('sert le même index deux fois pour le même sujet', () => {
    const source = createReplayAntagonists(feed, campagne)
    expect(source.forSubject(ME)).toBe(source.forSubject(ME))
  })

  it('rend un autre index pour un autre sujet', () => {
    const source = createReplayAntagonists(feed, campagne)
    expect(source.forSubject(ALLY)).not.toBe(source.forSubject(ME))
    expect(source.forSubject(ALLY).at(9999).nemesis).toEqual({ holder: FOE1, count: 1, tiedWith: [] })
  })

  it('rend un autre index pour un autre document', () => {
    expect(createReplayAntagonists(feed, campagne).forSubject(ME)).not.toBe(
      createReplayAntagonists(feed, campagne).forSubject(ME),
    )
  })

  it('rend le même index vide, sans antagoniste, pour un sujet absent', () => {
    const source = createReplayAntagonists(feed, campagne)
    expect(source.forSubject(null)).toBe(source.forSubject(null))
    expect(source.forSubject(null).at(9999)).toEqual({ nemesis: null, victim: null })
  })

  it('forViewpoint ramène le point de vue à la clé du film puis sert le même index', () => {
    const roster = [player('film-me', 0, { board: { xuid: 'xuid(feuille)' } as Board }), player(FOE1, 1)]
    const source = createReplayAntagonists([killEntry(FOE1, 'film-me', 1000)], roster)
    expect(source.forViewpoint('xuid(feuille)')).toBe(source.forSubject('film-me'))
    expect(source.forViewpoint('xuid(feuille)').at(1000).nemesis?.holder).toBe(FOE1)
    expect(source.forViewpoint(null)).toBe(source.forSubject(null))
  })
})

describe('botKillsOf : les éliminations d’un bot lues dans le film', () => {
  const doc = (rows: Array<{ t: number; xuid: string; victim?: string }>) => ({
    frameIntervalMs: 100,
    eliminations: rows,
  }) as unknown as Parameters<typeof botKillsOf>[0]

  it('ne garde que les éliminations dont le tueur est un bot, à leur instant sur l’axe du rejeu', () => {
    const rows = [
      { t: 10, xuid: BOT_FOE, victim: ME },
      { t: 20, xuid: FOE1, victim: ME },
    ]
    expect(botKillsOf(doc(rows), campagne)).toEqual([{ killer: BOT_FOE, victim: ME, t: 1000 }])
  })

  it('ignore une élimination de bot sans victime : aucune victime n’est devinée', () => {
    expect(botKillsOf(doc([{ t: 10, xuid: BOT_FOE }]), campagne)).toEqual([])
  })

  it('une victime bot est rendue par sa clé du film', () => {
    expect(botKillsOf(doc([{ t: 10, xuid: BOT_FOE, victim: botKey('Autre') }]), campagne)).toEqual([
      { killer: BOT_FOE, victim: botKey('Autre'), t: 1000 },
    ])
  })
})
