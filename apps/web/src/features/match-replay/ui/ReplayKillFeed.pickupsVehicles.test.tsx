/**
 * Tests — le fil : armes ramassées au sol et destructions de véhicule (lot 1 bis du plan
 * « maquettes »).
 *
 * Ce qu'ils protègent :
 *  1. LES ARMES AU SOL entrent dans « Armes et bonus » quand elles ne sont pas de base : la ligne
 *     dit « a ramassé : » (et non « a pris : » d'un socle), l'arme de base ramassée n'apparaît pas,
 *     le compte du menu suit.
 *  2. LES DESTRUCTIONS DE VÉHICULE, fabriquées : avec auteur et un joueur à bord, avec auteur
 *     seul, avec un destructeur bot, avec plusieurs joueurs à bord (leur nombre), sans auteur
 *     (« Ghost détruit »), FR et EN ; elles sont de la famille Véhicules (filtre et compte) et
 *     tiennent les règles du fil (une seule ligne sans retour, pictogramme `shrink-0`).
 *  3. LE FIL DE LA PAGE reste intact : le filtre ne porte que sur la liste affichée.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { eventFeedEntry } from '../model/feedEventLines'
import { buildFeedEntries, type ReplayFeedEntry } from '../model/killFeedLogic'
import { FEED_FILTER_KEY } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const FRAME_MS = 100
const START = '0x0000AAAA'
const HAMMER = '0x0000CCCC'

const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['foe', { gamertag: 'Cobra01' }],
])

const ROSTER = [
  { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
  { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
]

function life(slot: number, xuid: string) {
  return {
    slot, xuid, team: -1, startFrame: 0, endFrame: 900,
    points: [{ t: 0, x: 0, y: 0 }, { t: 900, x: 1, y: 1 }],
  }
}

function ground(over: Record<string, unknown>) {
  return { t0: 100, t1: 300, t1max: 300, x: 0, y: 0, origin: 'dropped', dropper: 2, end: 'pickup', picker: 1, ...over }
}

const GROUND_DOC = testReplayDoc({
  frameIntervalMs: FRAME_MS,
  roster: ROSTER,
  tracks: [life(1, 'me'), life(2, 'foe')] as never,
  loadouts: [{ slot: 1, src: 'birth', k: [0], w: [START] }] as never,
  weaponLabels: {
    [START]: { en: 'Assault Rifle', fr: 'Fusil d’assaut' },
    [HAMMER]: { en: 'Hammer', fr: 'Marteau' },
  } as never,
  groundWeapons: [
    ground({ w: '0000cccc', t1: 300 }),
    ground({ w: '0000aaaa', t1: 500, t1max: 500, picker: 2 }),
  ] as never,
})

const PLAIN_DOC = testReplayDoc({ frameIntervalMs: FRAME_MS, roster: ROSTER })
const ALLEGIANCE = filmAllegianceOf(GROUND_DOC, [], 'me')

function destroyed(
  over: Partial<{ xuid: string | null; botName: string | null; aboardXuids: string[]; name: string }>,
  ms: number,
) {
  return eventFeedEntry(`vd-${ms}`, ms, {
    family: 'vehicles', destroyed: true, xuid: 'me', botName: null, aboardXuids: [], name: 'Ghost', ...over,
  })
}

function renderFeed(opts: { doc: typeof GROUND_DOC; entries?: ReplayFeedEntry[]; locale?: 'fr' | 'en'; nowMs?: number }) {
  const entries = opts.entries ?? buildFeedEntries([], [], 0, null)
  return render(
    <ReplayKillFeed
      entries={entries}
      nowMs={opts.nowMs ?? 100_000}
      playWindow={null}
      scoreboard={[]}
      xuidMeta={META}
      allegiance={ALLEGIANCE}
      locale={opts.locale ?? 'fr'}
      doc={opts.doc}
    />,
  )
}

function choose(name: string | RegExp) {
  fireEvent.click(screen.getByRole('button', { name: /Filtrer le fil|Filter the feed/ }))
  fireEvent.click(screen.getByRole('menuitemradio', { name }))
}

function menuCount(name: RegExp) {
  fireEvent.click(screen.getByRole('button', { name: /Filtrer le fil|Filter the feed/ }))
  const item = within(screen.getByRole('menu')).getByRole('menuitemradio', { name })
  const count = Number(item.textContent?.match(/\d+$/)?.[0])
  fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })
  return count
}

beforeEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))
afterEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))

describe('armes ramassées au sol dans « Armes et bonus »', () => {
  it('une arme spéciale ramassée au sol : « a ramassé : » au nom du ramasseur, FR', () => {
    renderFeed({ doc: GROUND_DOC })
    choose(/Armes et bonus/)
    const rows = screen.getAllByRole('listitem')
    expect(rows).toHaveLength(1)
    expect(rows[0].getAttribute('data-feed-family')).toBe('pickups')
    expect(rows[0].textContent).toContain('JGtm')
    expect(rows[0].textContent).toContain('a ramassé : Marteau')
  })

  it('l’arme de base ramassée n’apparaît pas, et le compte du menu ne la compte pas', () => {
    renderFeed({ doc: GROUND_DOC })
    expect(menuCount(/Armes et bonus/)).toBe(1)
    choose(/Armes et bonus/)
    expect(screen.queryByText(/Fusil d’assaut/)).toBeNull()
  })

  it('EN : « picked up off the ground »', () => {
    renderFeed({ doc: GROUND_DOC, locale: 'en' })
    choose(/Weapons and bonuses|Weapons/)
    expect(screen.getByText(/picked up off the ground: Hammer/)).toBeTruthy()
  })

  it('une ligne du sol tient les règles du fil : une seule ligne, pictogramme shrink-0', () => {
    renderFeed({ doc: GROUND_DOC })
    choose(/Armes et bonus/)
    const row = screen.getAllByRole('listitem')[0]
    expect(row.className).toContain('flex-nowrap')
    expect(row.querySelector('br')).toBeNull()
    expect(row.querySelector('svg')?.getAttribute('class')).toContain('shrink-0')
  })

  it("l'instant brut décide du visible : pas de ligne avant le ramassage", () => {
    renderFeed({ doc: GROUND_DOC, nowMs: 20_000 })
    choose(/Armes et bonus/)
    expect(screen.queryByText(/a ramassé/)).toBeNull()
  })
})

describe('destructions de véhicule (événements fabriqués)', () => {
  const entries = [
    destroyed({ aboardXuids: ['foe'] }, 10_000),
    destroyed({ xuid: 'foe', name: 'Warthog' }, 20_000),
    destroyed({ xuid: null, name: 'Ghost' }, 30_000),
  ]

  it('Véhicules les montre, avec auteur et joueur à bord, avec auteur seul, sans auteur (FR)', () => {
    renderFeed({ doc: PLAIN_DOC, entries })
    choose(/Véhicules/)
    const texts = screen.getAllByRole('listitem').map((r) => r.textContent)
    expect(texts).toHaveLength(3)
    expect(texts.some((x) => x?.includes('JGtm') && x.includes('a détruit : Ghost (Cobra01 à bord)'))).toBe(true)
    expect(texts.some((x) => x?.includes('Cobra01') && x.includes('a détruit : Warthog'))).toBe(true)
    const neutral = screen.getAllByRole('listitem').find((r) => r.textContent?.includes('Ghost détruit'))
    expect(neutral).toBeTruthy()
    expect(neutral?.textContent).not.toContain('JGtm')
    expect(neutral?.textContent).not.toContain('Cobra01')
  })

  it('EN : « destroyed: » et « Ghost destroyed »', () => {
    renderFeed({ doc: PLAIN_DOC, entries, locale: 'en' })
    choose(/Vehicles/)
    expect(screen.getByText(/destroyed: Ghost \(Cobra01 aboard\)/)).toBeTruthy()
    expect(screen.getByText('Ghost destroyed')).toBeTruthy()
  })

  it('comptent dans Véhicules et rien d’autre ; le menu propose Véhicules même sans autre ligne', () => {
    renderFeed({ doc: PLAIN_DOC, entries })
    expect(menuCount(/Véhicules/)).toBe(3)
    expect(menuCount(/Armes et bonus/)).toBe(0)
    expect(menuCount(/Éliminations/)).toBe(0)
  })

  it('Armes et bonus ne montre aucune destruction', () => {
    renderFeed({ doc: PLAIN_DOC, entries })
    choose(/Armes et bonus/)
    expect(screen.queryByText(/détruit/)).toBeNull()
    expect(screen.getByText('Aucune entrée pour ce match')).toBeTruthy()
  })

  it('une ligne de destruction tient les règles du fil : une seule ligne, pictogramme shrink-0, libellé tronqué', () => {
    renderFeed({ doc: PLAIN_DOC, entries })
    choose(/Véhicules/)
    for (const row of screen.getAllByRole('listitem')) {
      expect(row.className).toContain('flex-nowrap')
      expect(row.className).toContain('overflow-hidden')
      expect(row.className).toContain('shrink-0')
      expect(row.querySelector('br')).toBeNull()
      expect(row.querySelector('svg')?.getAttribute('class')).toContain('shrink-0')
      expect(row.querySelector('.truncate')).not.toBeNull()
    }
  })

  it("l'instant brut décide du visible, et le fil de la page n'est ni modifié ni filtré", () => {
    Object.freeze(entries)
    renderFeed({ doc: PLAIN_DOC, entries, nowMs: 15_000 })
    choose(/Véhicules/)
    expect(screen.getAllByRole('listitem')).toHaveLength(1)
    choose(/Éliminations/)
    expect(entries).toHaveLength(3)
  })

  it('un destructeur bot se nomme par le roster, sans encre de joueur ; plusieurs joueurs à bord : leur nombre', () => {
    renderFeed({
      doc: PLAIN_DOC,
      entries: [destroyed({ xuid: null, botName: 'Cobra07 [bot]', aboardXuids: ['me', 'foe', 'x9'] }, 10_000)],
    })
    choose(/Véhicules/)
    const text = screen.getByRole('listitem').textContent
    expect(text).toContain('Cobra07')
    expect(text).not.toContain('[bot]')
    expect(text).toContain('a détruit : Ghost (3 à bord)')
  })

  it('EN : le nombre à bord', () => {
    renderFeed({
      doc: PLAIN_DOC,
      entries: [destroyed({ xuid: null, aboardXuids: ['me', 'foe'] }, 10_000)],
      locale: 'en',
    })
    choose(/Vehicles/)
    expect(screen.getByText('Ghost destroyed (2 aboard)')).toBeTruthy()
  })

  it('un nom de joueur à bord inconnu de la feuille ne laisse jamais un xuid brut', () => {
    renderFeed({ doc: PLAIN_DOC, entries: [destroyed({ aboardXuids: ['2533274812345678'] }, 10_000)] })
    choose(/Véhicules/)
    expect(screen.getByRole('listitem').textContent).not.toContain('2533274812345678')
  })
})
