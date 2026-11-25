/**
 * antagonistTitles.test.ts : ce que chaque pion porte, à chaque image, des titres d'antagoniste du
 * point de vue (piste B du plan des antagonistes), et la pulsation d'une prise de titre.
 *
 * Les instants sont ceux de l'axe du rejeu : ici, l'image vaut sa date en millisecondes
 * (`msOfFrame` identité), ce qui garde les cas lisibles. L'horloge murale est simulée : `wall`.
 */
import { describe, expect, it, vi } from 'vitest'

import { buildAntagonists, type AntagonistKill } from '@/lib/antagonists/antagonists'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import { createReplayAntagonists } from './replayAntagonists'
import { createSlotTitles, type SlotTitle } from './antagonistTitles'

const HOLD_MS = 1400
const SUBJECT = 'S'

const k = (killer: string, victim: string, t: number): AntagonistKill => ({ killer, victim, t })
const isFoe = (key: string): boolean => key.startsWith('e')

/** Les slots du test : un slot par joueur, la clé du film est la lettre du slot. */
const OWNERS: Record<number, string> = { 1: 'S', 2: 'e1', 3: 'e2', 4: 'a1', 5: 'e3' }
const ownerKeyOfSlot = (slot: number): string | null => OWNERS[slot] ?? null

/**
 * S meurt sous e1 (1000), puis sous e2 (2000 : égalité 1-1, e1 garde le titre), puis sous e2
 * (2500 : 2 contre 1, e2 prend la némésis), puis S élimine e1 (6000 : souffre-douleur e1).
 */
const KILLS = [k('e1', SUBJECT, 1000), k('e2', SUBJECT, 2000), k('e2', SUBJECT, 2500), k(SUBJECT, 'e1', 6000)]

/** Une série d'une seule vague : e1 puis e2 puis e3 prennent la némésis en 400 ms. */
const RAFALE = [
  k('e1', SUBJECT, 1000),
  k('e2', SUBJECT, 1100),
  k('e2', SUBJECT, 1200),
  k('e3', SUBJECT, 1300),
  k('e3', SUBJECT, 1301),
  k('e3', SUBJECT, 1302),
]

/** Un lecteur de titres et son horloge murale : `wall.now` avance à la main. */
function lecteur(opts: { reducedMotion?: boolean; kills?: AntagonistKill[] } = {}) {
  const wall = { now: 0 }
  const reader = createSlotTitles({
    antagonists: buildAntagonists(opts.kills ?? KILLS, SUBJECT, isFoe),
    ownerKeyOfSlot,
    msOfFrame: (frame) => frame,
    holdMs: HOLD_MS,
    reducedMotion: opts.reducedMotion ?? false,
    now: () => wall.now,
  })
  return { ...reader, wall }
}

const sansMouvement = (titles: readonly SlotTitle[]) => titles.map(({ role, count }) => ({ role, count }))

describe('createSlotTitles : qui porte quoi, au repos', () => {
  it('rien avant la première élimination comptée', () => {
    const { titlesOfSlot } = lecteur()
    expect(titlesOfSlot(2, 999)).toEqual([])
    expect(titlesOfSlot(3, 999)).toEqual([])
  })

  it('le tenant porte son titre et son compte, les autres joueurs rien, sans aucun mouvement', () => {
    const { titlesOfSlot } = lecteur()
    expect(titlesOfSlot(2, 1000)).toEqual([{ role: 'nemesis', count: 1 }])
    expect(titlesOfSlot(2, 1500)).toEqual([{ role: 'nemesis', count: 1 }])
    expect(titlesOfSlot(3, 1500)).toEqual([])
    expect(titlesOfSlot(1, 1500)).toEqual([])
    expect(titlesOfSlot(4, 1500)).toEqual([])
  })

  it('un slot sans propriétaire à l’image ne porte rien', () => {
    expect(lecteur().titlesOfSlot(99, 1500)).toEqual([])
  })

  it('l’égalité laisse le titre au tenant', () => {
    const { titlesOfSlot } = lecteur()
    expect(titlesOfSlot(3, 2450)).toEqual([])
    expect(titlesOfSlot(2, 2450)).toEqual([{ role: 'nemesis', count: 1 }])
  })

  it('un joueur qui tient les deux titres les porte tous les deux, némésis d’abord', () => {
    const { titlesOfSlot } = lecteur({ kills: [k(SUBJECT, 'e1', 500), k('e1', SUBJECT, 1000)] })
    expect(sansMouvement(titlesOfSlot(2, 5000))).toEqual([
      { role: 'nemesis', count: 1 },
      { role: 'victim', count: 1 },
    ])
  })

  it('un allié n’est jamais marqué, même s’il élimine ou est éliminé par le point de vue', () => {
    const { titlesOfSlot } = lecteur({ kills: [k('a1', SUBJECT, 1000), k(SUBJECT, 'a1', 2000)] })
    expect(titlesOfSlot(4, 5000)).toEqual([])
  })
})

describe('la pulsation démarre quand la lecture franchit un changement de titre, et dure 1,4 s d’horloge murale', () => {
  it('le nouveau tenant pulse, l’ancien s’efface, au rythme de l’horloge murale', () => {
    const { titlesOfSlot, advance, wall } = lecteur()
    advance(2490, 2510)
    expect(titlesOfSlot(3, 2510)).toEqual([{ role: 'nemesis', count: 2, motion: { kind: 'gain', progress: 0 } }])
    expect(titlesOfSlot(2, 2510)).toEqual([{ role: 'nemesis', count: 1, motion: { kind: 'loss', progress: 0 } }])
    wall.now = 700
    expect(titlesOfSlot(3, 2510)[0].motion).toEqual({ kind: 'gain', progress: 0.5 })
    // L'ancien tenant garde le compte qu'il avait en perdant le titre, jusqu'à disparaître.
    expect(titlesOfSlot(2, 2510)[0]).toMatchObject({ count: 1, motion: { kind: 'loss', progress: 0.5 } })
  })

  it('la durée ne dépend pas de la vitesse de lecture : ×1 et ×8 donnent la même courbe', () => {
    // ×1 : un pas de 20 ms du rejeu ; ×8 : un pas de 160 ms. La pulsation part du même franchissement.
    for (const [de, vers] of [[990, 1010], [930, 1090]] as const) {
      const { titlesOfSlot, advance, wall } = lecteur()
      advance(de, vers)
      const progressions = [0, 350, 700, 1050, 1399].map((ms) => {
        wall.now = ms
        return titlesOfSlot(2, vers)[0].motion?.progress
      })
      expect(progressions).toEqual([0, 0.25, 0.5, 0.75, 1399 / 1400])
      wall.now = 1400
      expect(titlesOfSlot(2, vers)).toEqual([{ role: 'nemesis', count: 1 }])
    }
  })

  it('la pulsation va à son terme si la lecture se met en pause, puis l’état de repos', () => {
    const { titlesOfSlot, advance, pulsing, wall } = lecteur()
    advance(990, 1010)
    expect(pulsing()).toBe(true)
    // Plus aucun pas de lecture : l'horloge murale seule fait avancer la pulsation, l'image reste la même.
    wall.now = 1000
    expect(titlesOfSlot(2, 1010)[0].motion?.progress).toBeCloseTo(1000 / 1400, 10)
    expect(pulsing()).toBe(true)
    wall.now = 1400
    expect(pulsing()).toBe(false)
    expect(titlesOfSlot(2, 1010)).toEqual([{ role: 'nemesis', count: 1 }])
  })

  it('un saut du curseur ne déclenche aucune pulsation : l’état de repos juste', () => {
    const { titlesOfSlot, pulsing } = lecteur()
    // Aucun pas de lecture : le curseur est posé juste après le changement, puis loin devant.
    expect(titlesOfSlot(2, 1010)).toEqual([{ role: 'nemesis', count: 1 }])
    expect(titlesOfSlot(3, 2600)).toEqual([{ role: 'nemesis', count: 2 }])
    expect(titlesOfSlot(2, 2600)).toEqual([])
    expect(pulsing()).toBe(false)
  })

  it('un retour en arrière ne déclenche aucune pulsation', () => {
    const { titlesOfSlot, advance, pulsing } = lecteur()
    advance(2600, 2400)
    advance(1010, 990)
    expect(pulsing()).toBe(false)
    expect(titlesOfSlot(2, 990)).toEqual([])
    expect(titlesOfSlot(2, 2400)).toEqual([{ role: 'nemesis', count: 1 }])
  })

  it('un saut pendant la pulsation montre l’état de repos de l’instant atteint, pas un titre périmé', () => {
    const { titlesOfSlot, advance, wall } = lecteur()
    advance(2490, 2510)
    wall.now = 300
    // Le curseur revient avant le changement : le nouveau tenant n'y a pas le titre, l'ancien le garde, sans mouvement.
    expect(titlesOfSlot(3, 2000)).toEqual([])
    expect(titlesOfSlot(2, 2000)).toEqual([{ role: 'nemesis', count: 1 }])
  })

  it('un pas qui ne franchit aucun changement ne déclenche rien', () => {
    const { advance, pulsing } = lecteur()
    advance(1010, 1500)
    expect(pulsing()).toBe(false)
  })

  it('une égalité ne pulse pas : le titre ne change pas de main', () => {
    const { advance, pulsing } = lecteur()
    advance(1990, 2010)
    expect(pulsing()).toBe(false)
  })

  it('plusieurs changements franchis dans le même pas : une pulsation, au tenant final, sans cumul', () => {
    const { titlesOfSlot, advance, wall } = lecteur({ kills: RAFALE })
    advance(1050, 1400)
    wall.now = 700
    // e3 est le nouveau tenant ; e1, qui tenait le titre avant le pas, s'efface ; e2 ne l'a tenu que dans le pas.
    expect(titlesOfSlot(5, 1400)).toEqual([{ role: 'nemesis', count: 3, motion: { kind: 'gain', progress: 0.5 } }])
    expect(titlesOfSlot(2, 1400)).toEqual([{ role: 'nemesis', count: 1, motion: { kind: 'loss', progress: 0.5 } }])
    expect(titlesOfSlot(3, 1400)).toEqual([])
  })

  it('un nouveau changement pendant une pulsation la relance, sans cumul', () => {
    const { titlesOfSlot, advance, wall } = lecteur()
    advance(990, 1010)
    wall.now = 500
    advance(2490, 2510)
    expect(titlesOfSlot(3, 2510)).toEqual([{ role: 'nemesis', count: 2, motion: { kind: 'gain', progress: 0 } }])
    expect(titlesOfSlot(2, 2510)).toEqual([{ role: 'nemesis', count: 1, motion: { kind: 'loss', progress: 0 } }])
    wall.now = 1899
    expect(titlesOfSlot(3, 2510)[0].motion?.progress).toBeCloseTo(1399 / 1400, 10)
    wall.now = 1900
    expect(titlesOfSlot(3, 2510)).toEqual([{ role: 'nemesis', count: 2 }])
  })

  it('en mouvement réduit : aucune pulsation, aucune icône qui s’efface, l’état de repos', () => {
    const { titlesOfSlot, advance, pulsing } = lecteur({ reducedMotion: true })
    advance(2490, 2510)
    expect(pulsing()).toBe(false)
    expect(titlesOfSlot(3, 2510)).toEqual([{ role: 'nemesis', count: 2 }])
    expect(titlesOfSlot(2, 2510)).toEqual([])
  })

  it('lit aussi par la clé du film, sans passer par un slot (occupant d’un véhicule)', () => {
    const { titlesOfKey, advance } = lecteur()
    expect(titlesOfKey('e1', 1500)).toEqual([{ role: 'nemesis', count: 1 }])
    expect(titlesOfKey('e2', 1500)).toEqual([])
    advance(990, 1010)
    expect(titlesOfKey('e1', 1010)[0].motion).toEqual({ kind: 'gain', progress: 0 })
  })
})

describe('createSlotTitles : un seul calcul par image', () => {
  it('lit l’index une fois par image, pour tous les slots', () => {
    const antagonists = buildAntagonists(KILLS, SUBJECT, isFoe)
    const at = vi.fn(antagonists.at)
    const { titlesOfSlot } = createSlotTitles({
      antagonists: { changes: antagonists.changes, at },
      ownerKeyOfSlot,
      msOfFrame: (frame) => frame,
      holdMs: HOLD_MS,
      reducedMotion: false,
      now: () => 0,
    })
    for (const slot of [1, 2, 3, 4]) titlesOfSlot(slot, 3500)
    expect(at).toHaveBeenCalledTimes(1)
    titlesOfSlot(2, 3501)
    expect(at).toHaveBeenCalledTimes(2)
    titlesOfSlot(2, 3501)
    expect(at).toHaveBeenCalledTimes(2)
  })

  it('convertit l’image en millisecondes du rejeu', () => {
    const { titlesOfSlot } = createSlotTitles({
      antagonists: buildAntagonists(KILLS, SUBJECT, isFoe),
      ownerKeyOfSlot,
      msOfFrame: (frame) => frame * 100,
      holdMs: HOLD_MS,
      reducedMotion: true,
      now: () => 0,
    })
    expect(titlesOfSlot(2, 9)).toEqual([])
    expect(titlesOfSlot(2, 10)).toEqual([{ role: 'nemesis', count: 1 }])
  })
})

describe('un bot est marqué comme tout joueur', () => {
  const roster = [
    { xuid: 'S', team: 0, bot: false, lives: [] },
    { xuid: 'e1', team: 1, bot: false, lives: [] },
    { xuid: 'bot:Brute', team: 1, bot: true, lives: [] },
  ] as unknown as ReplayPlayer[]
  const entry = (killer: string, victim: string, victimGamertag: string, replayMs: number): ReplayFeedEntry => ({
    key: `k-${replayMs}`,
    replayMs,
    kill: { xuid: killer, victimXuid: victim, victimGamertag, replayMs, medals: [] } as unknown as ReplayKill,
    medal: null,
    death: null,
  })
  const lire = (antagonists: ReturnType<ReturnType<typeof createReplayAntagonists>['forViewpoint']>, owners: Record<number, string>) =>
    createSlotTitles({
      antagonists,
      ownerKeyOfSlot: (slot) => owners[slot] ?? null,
      msOfFrame: (frame) => frame,
      holdMs: HOLD_MS,
      reducedMotion: false,
      now: () => 0,
    })

  it('le bot que le point de vue élimine compte comme l’humain : chacun a son titre', () => {
    const feed = [entry('S', '', 'Brute', 1000), entry('S', '', 'Brute', 1500), entry('S', 'e1', '', 2000)]
    const reader = lire(createReplayAntagonists(feed, roster).forViewpoint('S'), { 1: 'S', 2: 'e1', 3: 'bot:Brute' })
    reader.advance(900, 3000)
    expect(reader.titlesOfSlot(3, 5000)).toEqual([{ role: 'victim', count: 2, motion: { kind: 'gain', progress: 0 } }])
    expect(reader.titlesOfSlot(2, 5000)).toEqual([])
  })

  it('le point de vue bot a ses antagonistes : l’adversaire qui l’élimine est sa némésis', () => {
    const reader = lire(createReplayAntagonists([entry('S', '', 'Brute', 1000)], roster).forViewpoint('bot:Brute'), { 1: 'S' })
    reader.advance(900, 3000)
    expect(reader.titlesOfSlot(1, 5000)).toEqual([{ role: 'nemesis', count: 1, motion: { kind: 'gain', progress: 0 } }])
  })
})
