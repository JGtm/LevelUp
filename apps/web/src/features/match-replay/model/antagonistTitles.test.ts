/**
 * antagonistTitles.test.ts : ce que chaque pion porte, à chaque image, des titres d'antagoniste du
 * point de vue (piste B du plan des antagonistes).
 *
 * Les instants sont ceux de l'axe du rejeu : ici, l'image vaut sa date en millisecondes
 * (`msOfFrame` identité), ce qui garde les cas lisibles.
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
const OWNERS: Record<number, string> = { 1: 'S', 2: 'e1', 3: 'e2', 4: 'a1' }
const ownerKeyOfSlot = (slot: number): string | null => OWNERS[slot] ?? null

/**
 * S meurt sous e1 (1000), puis sous e2 (2000 : égalité 1-1, e1 garde le titre), puis sous e2
 * (2500 : 2 contre 1, e2 prend la némésis), puis S élimine e1 (6000 : souffre-douleur e1).
 */
const KILLS = [k('e1', SUBJECT, 1000), k('e2', SUBJECT, 2000), k('e2', SUBJECT, 2500), k(SUBJECT, 'e1', 6000)]

function titlesAt(opts: { reducedMotion?: boolean; kills?: AntagonistKill[] } = {}) {
  const antagonists = buildAntagonists(opts.kills ?? KILLS, SUBJECT, isFoe)
  return createSlotTitles({
    antagonists,
    ownerKeyOfSlot,
    msOfFrame: (frame) => frame,
    holdMs: HOLD_MS,
    reducedMotion: opts.reducedMotion ?? false,
  })
}

const sansMouvement = (titles: readonly SlotTitle[]) => titles.map(({ role, count }) => ({ role, count }))

describe('createSlotTitles : qui porte quoi', () => {
  it('rien avant la première élimination comptée', () => {
    const titles = titlesAt()
    expect(titles(2, 999)).toEqual([])
    expect(titles(3, 999)).toEqual([])
  })

  it('le tenant porte son titre et son compte, les autres joueurs rien', () => {
    const titles = titlesAt()
    expect(sansMouvement(titles(2, 1500))).toEqual([{ role: 'nemesis', count: 1 }])
    expect(titles(3, 1500)).toEqual([])
    expect(titles(1, 1500)).toEqual([])
    expect(titles(4, 1500)).toEqual([])
  })

  it('un slot sans propriétaire à l’image ne porte rien', () => {
    expect(titlesAt()(99, 1500)).toEqual([])
  })

  it('l’égalité laisse le titre au tenant, sans pulsation', () => {
    const titles = titlesAt()
    expect(titles(3, 2450)).toEqual([])
    expect(titles(2, 2450)).toEqual([{ role: 'nemesis', count: 1 }])
  })

  it('un joueur qui tient les deux titres les porte tous les deux, némésis d’abord', () => {
    const titles = titlesAt({ kills: [k(SUBJECT, 'e1', 500), k('e1', SUBJECT, 1000)] })
    expect(sansMouvement(titles(2, 5000))).toEqual([
      { role: 'nemesis', count: 1 },
      { role: 'victim', count: 1 },
    ])
  })

  it('un allié n’est jamais marqué, même s’il élimine ou est éliminé par le point de vue', () => {
    const titles = titlesAt({ kills: [k('a1', SUBJECT, 1000), k(SUBJECT, 'a1', 2000)] })
    expect(titles(4, 5000)).toEqual([])
  })
})

describe('createSlotTitles : la pulsation dure 1,4 s de temps du rejeu', () => {
  it('le nouveau tenant pulse, avec la progression écoulée', () => {
    const titles = titlesAt()
    expect(titles(2, 1000)).toEqual([{ role: 'nemesis', count: 1, motion: { kind: 'gain', progress: 0 } }])
    const [t] = titles(2, 1700)
    expect(t.motion?.kind).toBe('gain')
    expect(t.motion?.progress).toBeCloseTo(0.5, 10)
  })

  it('l’ancien tenant s’efface pendant que le nouveau pulse', () => {
    const titles = titlesAt()
    const nouveau = titles(3, 2500 + 700)
    const ancien = titles(2, 2500 + 700)
    expect(nouveau[0]).toMatchObject({ role: 'nemesis', count: 2, motion: { kind: 'gain' } })
    expect(nouveau[0].motion?.progress).toBeCloseTo(0.5, 10)
    // Il garde le compte qu'il avait quand il a perdu le titre, jusqu'à disparaître.
    expect(ancien[0]).toMatchObject({ role: 'nemesis', count: 1, motion: { kind: 'loss' } })
    expect(ancien[0].motion?.progress).toBeCloseTo(0.5, 10)
  })

  it('au bout de 1,4 s : repos, le nouveau tenant sans mouvement et l’ancien sans icône', () => {
    const titles = titlesAt()
    expect(titles(3, 2500 + 1399)[0].motion).toBeDefined()
    expect(titles(3, 2500 + 1400)).toEqual([{ role: 'nemesis', count: 2 }])
    expect(titles(2, 2500 + 1400)).toEqual([])
  })

  it('avant la prise, l’ancien tenant garde son icône pleine', () => {
    expect(titlesAt()(2, 2499)).toEqual([{ role: 'nemesis', count: 1 }])
  })

  it('en mouvement réduit : l’état final tout de suite, aucun mouvement, aucune icône qui s’efface', () => {
    const titles = titlesAt({ reducedMotion: true })
    expect(titles(2, 1000)).toEqual([{ role: 'nemesis', count: 1 }])
    expect(titles(3, 2500)).toEqual([{ role: 'nemesis', count: 2 }])
    expect(titles(2, 2500)).toEqual([])
  })

  it('un titre repris par un joueur qui revient pulse de nouveau', () => {
    const kills = [k('e1', SUBJECT, 1000), k('e2', SUBJECT, 2000), k('e2', SUBJECT, 3000), k('e1', SUBJECT, 4000), k('e1', SUBJECT, 5000)]
    const titles = titlesAt({ kills })
    expect(titles(2, 5200)[0].motion).toMatchObject({ kind: 'gain' })
    expect(titles(3, 5200)[0].motion).toMatchObject({ kind: 'loss' })
  })
})

describe('createSlotTitles : un seul calcul par image', () => {
  it('lit l’index une fois par image, pour tous les slots', () => {
    const antagonists = buildAntagonists(KILLS, SUBJECT, isFoe)
    const at = vi.fn(antagonists.at)
    const titles = createSlotTitles({
      antagonists: { changes: antagonists.changes, at },
      ownerKeyOfSlot,
      msOfFrame: (frame) => frame,
      holdMs: HOLD_MS,
      reducedMotion: false,
    })
    for (const slot of [1, 2, 3, 4]) titles(slot, 3500)
    expect(at).toHaveBeenCalledTimes(1)
    titles(2, 3501)
    expect(at).toHaveBeenCalledTimes(2)
    titles(2, 3501)
    expect(at).toHaveBeenCalledTimes(2)
  })

  it('convertit l’image en millisecondes du rejeu', () => {
    const antagonists = buildAntagonists(KILLS, SUBJECT, isFoe)
    const titles = createSlotTitles({
      antagonists,
      ownerKeyOfSlot,
      msOfFrame: (frame) => frame * 100,
      holdMs: HOLD_MS,
      reducedMotion: true,
    })
    expect(titles(2, 9)).toEqual([])
    expect(titles(2, 10)).toEqual([{ role: 'nemesis', count: 1 }])
  })
})

describe('un bot n’est jamais marqué', () => {
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

  it('ni le bot que le point de vue élimine, ni un humain à côté', () => {
    // Le bot est la victime (sans xuid, nommée) : jamais un souffre-douleur ; l'humain l'est.
    const feed = [entry('S', '', 'Brute', 1000), entry('S', '', 'Brute', 1500), entry('S', 'e1', '', 2000)]
    const antagonists = createReplayAntagonists(feed, roster).forViewpoint('S')
    const owners: Record<number, string> = { 1: 'S', 2: 'e1', 3: 'bot:Brute' }
    const titles = createSlotTitles({
      antagonists,
      ownerKeyOfSlot: (slot) => owners[slot] ?? null,
      msOfFrame: (frame) => frame,
      holdMs: HOLD_MS,
      reducedMotion: true,
    })
    expect(titles(3, 5000)).toEqual([])
    expect(titles(2, 5000)).toEqual([{ role: 'victim', count: 1 }])
  })

  it('le point de vue bot n’a aucun antagoniste', () => {
    const feed = [entry('e1', 'bot:Brute', '', 1000)]
    const antagonists = createReplayAntagonists(feed, roster).forViewpoint('bot:Brute')
    const titles = createSlotTitles({
      antagonists,
      ownerKeyOfSlot: () => 'e1',
      msOfFrame: (frame) => frame,
      holdMs: HOLD_MS,
      reducedMotion: false,
    })
    expect(titles(2, 5000)).toEqual([])
  })
})
