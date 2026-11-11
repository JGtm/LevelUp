/**
 * Tests — le menu de filtre du fil et ses lignes d'événement (lot 1 du plan « maquettes »).
 *
 * Ce qu'ils protègent :
 *  1. LE MENU : toutes les entrées proposées, sauf Objectifs sur un match sans objectif (Assassin) ;
 *     le filtre mémorisé qui n'existe pas pour ce match retombe sur Tout SANS écraser la préférence.
 *  2. LE FILTRE NE PORTE QUE SUR L'AFFICHAGE : le fil de la page (`entries`), dont la frise et les
 *     effets de mort tirent leurs kills, reste entier et intact.
 *  3. DEUX ÉTATS VIDES, DEUX TEXTES, dans chaque langue : « Rien à cet instant » (le filtre a des
 *     lignes plus tard) et « Aucune entrée pour ce match » (il n'en a aucune).
 *  4. LES RÈGLES DU FIL pour la ligne d'objectif : une seule ligne sans retour, pictogramme
 *     `shrink-0`, défilement interne de la liste.
 *  5. LE CLAVIER du menu : flèche bas ouvre, flèches déplacent, Échap ferme et rend le focus.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { buildFeedEntries, killsOfFeed, type ReplayFeedEntry } from '../model/killFeedLogic'
import { FEED_FILTER_KEY } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const FRAME_MS = 100
const T0 = 0

const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['foe', { gamertag: 'Cobra01' }],
])

const ROSTER = [
  { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
  { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
]

function killEvent(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 1_000, xuid: 'me', ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '',
    weaponTinted: false, assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null,
    assistDamagePct: null, victimXuid: 'foe', victimGamertag: 'Cobra01', ...over,
  }
}

const KILLS = [killEvent({ tMs: 2_000 }), killEvent({ tMs: 9_000, xuid: 'foe', victimXuid: 'me', victimGamertag: 'JGtm' })]

const CTF_DOC = testReplayDoc({
  frameIntervalMs: FRAME_MS,
  roster: ROSTER,
  objectives: [
    { stat: 'flag_grabs', t: 40, xuid: 'me', timeMs: 4_000 },
    { stat: 'flag_captures', t: 60, xuid: 'foe', timeMs: 6_000 },
    { stat: 'kills', t: 20, xuid: 'me', timeMs: 2_000 },
  ] as never,
  weaponPads: [{ x: 0, y: 0, weapon: 'powerup_overshield', spawns: [], presence: [] }] as never,
  padPickups: [{ pad: 0, t: 30, tLow: 20, tHigh: 40, xuid: 'me' }] as never,
  vehicles: [
    { slot: 1, gen: 0, family: 'Warthog', end: 'unknown', t0: 0, t1: 200, t1max: 200, samples: [],
      rides: [{ t0: 50, t1: 80, slot: 2, xuid: 'foe', src: 'event', aim: [] }] },
  ] as never,
})

const ASSASSIN_DOC = testReplayDoc({ frameIntervalMs: FRAME_MS, roster: ROSTER })

const ALLEGIANCE = filmAllegianceOf(CTF_DOC, [], 'me')

function renderFeed(opts: {
  doc: typeof CTF_DOC | null
  nowMs: number
  locale?: 'fr' | 'en'
  entries?: ReplayFeedEntry[]
}) {
  const entries = opts.entries ?? buildFeedEntries(KILLS, [], T0, null)
  return {
    entries,
    ...render(
      <ReplayKillFeed
        entries={entries}
        nowMs={opts.nowMs}
        playWindow={null}
        scoreboard={[]}
        xuidMeta={META}
        allegiance={ALLEGIANCE}
        locale={opts.locale ?? 'fr'}
        doc={opts.doc}
      />,
    ),
  }
}

function openMenu() {
  fireEvent.click(screen.getByRole('button', { name: /Filtrer le fil|Filter the feed/ }))
  return screen.getByRole('menu')
}

function choose(name: string | RegExp) {
  openMenu()
  fireEvent.click(screen.getByRole('menuitemradio', { name }))
}

beforeEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))
afterEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))

describe('le menu du fil — entrées proposées', () => {
  it('propose toutes les entrées sur un match à objectifs, dans l’ordre de la maquette', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    const items = within(openMenu()).getAllByRole('menuitemradio')
    expect(items.map((i) => i.textContent?.replace(/\d+$/, ''))).toEqual([
      'Tout', 'Éliminations', 'Objectifs', 'Armes et bonus', 'Véhicules', 'Médailles', 'Assistances',
    ])
  })

  it("Objectifs n'apparaît pas sur un match Assassin (aucun objectif) ; les six autres restent", () => {
    renderFeed({ doc: ASSASSIN_DOC, nowMs: 100_000 })
    const items = within(openMenu()).getAllByRole('menuitemradio')
    const labels = items.map((i) => i.textContent?.replace(/\d+$/, ''))
    expect(labels).not.toContain('Objectifs')
    expect(labels).toEqual(['Tout', 'Éliminations', 'Armes et bonus', 'Véhicules', 'Médailles', 'Assistances'])
  })

  it('le titre porte le filtre actif, et chaque entrée son nombre de lignes dans tout le match', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    expect(screen.getByRole('button', { name: /Filtrer le fil : Tout/ })).toBeTruthy()
    const objectifs = within(openMenu()).getByRole('menuitemradio', { name: /Objectifs/ })
    expect(objectifs.textContent).toContain('2')
  })

  it('un filtre mémorisé absent du match retombe sur Tout sans écraser la préférence', () => {
    window.localStorage.setItem(FEED_FILTER_KEY, 'objectives')
    renderFeed({ doc: ASSASSIN_DOC, nowMs: 100_000 })
    expect(screen.getByRole('button', { name: /Filtrer le fil : Tout/ })).toBeTruthy()
    expect(screen.getAllByText('Cobra01').length).toBeGreaterThan(0)
    expect(window.localStorage.getItem(FEED_FILTER_KEY)).toBe('objectives')
  })

  it('un filtre mémorisé proposé est appliqué tel quel', () => {
    window.localStorage.setItem(FEED_FILTER_KEY, 'objectives')
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    expect(screen.getByRole('button', { name: /Filtrer le fil : Objectifs/ })).toBeTruthy()
  })

  it('choisir un filtre le mémorise', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    choose(/Médailles/)
    expect(window.localStorage.getItem(FEED_FILTER_KEY)).toBe('medals')
  })
})

describe('le filtre ne porte que sur la liste affichée', () => {
  it("ne modifie pas le fil de la page : mêmes lignes, mêmes kills pour la frise et les effets de mort", () => {
    const entries = buildFeedEntries(KILLS, [], T0, null)
    Object.freeze(entries)
    const killsBefore = killsOfFeed(entries)
    renderFeed({ doc: CTF_DOC, nowMs: 100_000, entries })
    choose(/Objectifs/)
    choose(/Véhicules/)
    choose(/Éliminations/)
    expect(entries).toHaveLength(2)
    expect(killsOfFeed(entries)).toEqual(killsBefore)
    expect(entries.every((e) => e.event === undefined)).toBe(true)
  })

  it('Éliminations ne montre que les frags ; Objectifs que les objectifs, frags et assistances exclus', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    choose(/Objectifs/)
    const rows = screen.getAllByRole('listitem')
    expect(rows).toHaveLength(2)
    expect(rows.every((r) => r.getAttribute('data-feed-family') === 'objectives')).toBe(true)
    expect(screen.getByText('a capturé le drapeau')).toBeTruthy()
    expect(screen.getByText('a pris le drapeau')).toBeTruthy()
    choose(/Éliminations/)
    expect(screen.getAllByRole('listitem').every((r) => r.getAttribute('data-feed-family') === null)).toBe(true)
  })

  it('Armes et bonus et Véhicules montrent leurs lignes, nommées', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    choose(/Armes et bonus/)
    expect(screen.getByText(/a pris : Surbouclier/)).toBeTruthy()
    choose(/Véhicules/)
    expect(screen.getByText(/est monté à bord : Warthog/)).toBeTruthy()
  })

  it('Tout mêle frags et lignes d’événement, la plus récente en tête', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    const families = screen.getAllByRole('listitem').map((r) => r.getAttribute('data-feed-family'))
    expect(families).toContain('objectives')
    expect(families).toContain('pickups')
    expect(families).toContain('vehicles')
    expect(families).toContain(null)
  })
})

describe('les deux états vides', () => {
  it('« Aucune entrée pour ce match » quand le filtre n’a aucune ligne dans tout le match (FR)', () => {
    renderFeed({ doc: ASSASSIN_DOC, nowMs: 100_000 })
    choose(/Véhicules/)
    expect(screen.getByText('Aucune entrée pour ce match')).toBeTruthy()
    expect(screen.queryByText(/Rien à cet instant/)).toBeNull()
  })

  it('« No entry for this match » dans la langue anglaise', () => {
    renderFeed({ doc: ASSASSIN_DOC, nowMs: 100_000, locale: 'en' })
    choose(/Vehicles/)
    expect(screen.getByText('No entry for this match')).toBeTruthy()
    expect(screen.queryByText(/Nothing at this point/)).toBeNull()
  })

  it('« Rien à cet instant » quand le filtre a des lignes, mais plus tard (FR)', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 1_000 })
    choose(/Objectifs/)
    expect(screen.getByText(/Rien à cet instant/)).toBeTruthy()
    expect(screen.queryByText('Aucune entrée pour ce match')).toBeNull()
  })

  it('« Nothing at this point » dans la langue anglaise', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 1_000, locale: 'en' })
    choose(/Objectives/)
    expect(screen.getByText(/Nothing at this point/)).toBeTruthy()
    expect(screen.queryByText('No entry for this match')).toBeNull()
  })

  it('l’état vide est le bloc pointillé du projet, jamais une liste absente', () => {
    renderFeed({ doc: ASSASSIN_DOC, nowMs: 100_000 })
    choose(/Médailles/)
    const notice = screen.getByText('Aucune entrée pour ce match').closest('div')
    expect(notice?.className).toContain('border-dashed')
    expect(screen.getByRole('list')).toBeTruthy()
  })
})

describe("la ligne d'objectif tient les règles du fil", () => {
  it('une seule ligne sans retour, pictogramme shrink-0, libellé tronqué avant d’être renvoyé', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    choose(/Objectifs/)
    const row = screen.getAllByRole('listitem')[0]
    expect(row.className).toContain('flex-nowrap')
    expect(row.className).toContain('overflow-hidden')
    expect(row.querySelector('br')).toBeNull()
    expect(row.querySelector('svg')?.getAttribute('class')).toContain('shrink-0')
    expect(row.querySelector('.truncate')).not.toBeNull()
  })

  it('le défilement est celui de la liste : la rangée garde sa hauteur naturelle', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    choose(/Objectifs/)
    expect(screen.getAllByRole('listitem')[0].className).toContain('shrink-0')
    expect(screen.getByRole('list').className).toContain('overflow-y-auto')
  })

  it("l'instant brut décide du visible : une action n'apparaît pas avant son instant", () => {
    renderFeed({ doc: CTF_DOC, nowMs: 5_000 })
    choose(/Objectifs/)
    expect(screen.getByText('a pris le drapeau')).toBeTruthy()
    expect(screen.queryByText('a capturé le drapeau')).toBeNull()
  })

  it('sans horloge sûre, ni ligne ni entrée de menu', () => {
    const untrusted = testReplayDoc({
      frameIntervalMs: FRAME_MS,
      roster: ROSTER,
      coverage: { originResolved: false } as never,
      objectives: [{ stat: 'flag_grabs', t: 40, xuid: 'me', timeMs: 4_000 }] as never,
    })
    renderFeed({ doc: untrusted, nowMs: 100_000 })
    const labels = within(openMenu()).getAllByRole('menuitemradio').map((i) => i.textContent?.replace(/\d+$/, ''))
    expect(labels).not.toContain('Objectifs')
  })
})

describe('le menu au clavier', () => {
  it('Flèche bas ouvre depuis le déclencheur, le focus part du filtre actif', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    const trigger = screen.getByRole('button', { name: /Filtrer le fil/ })
    expect(trigger.getAttribute('aria-haspopup')).toBe('menu')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    fireEvent.keyDown(trigger, { key: 'ArrowDown' })
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(document.activeElement?.textContent).toContain('Tout')
  })

  it('les flèches déplacent le focus en boucle, Fin et Début vont aux bouts', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    const menu = openMenu()
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(document.activeElement?.textContent).toContain('Éliminations')
    fireEvent.keyDown(menu, { key: 'ArrowUp' })
    fireEvent.keyDown(menu, { key: 'ArrowUp' })
    expect(document.activeElement?.textContent).toContain('Assistances')
    fireEvent.keyDown(menu, { key: 'Home' })
    expect(document.activeElement?.textContent).toContain('Tout')
    fireEvent.keyDown(menu, { key: 'End' })
    expect(document.activeElement?.textContent).toContain('Assistances')
  })

  it('Échap ferme le menu et rend le focus au déclencheur', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    const menu = openMenu()
    fireEvent.keyDown(menu, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: /Filtrer le fil/ }))
  })

  it('l’entrée active est annoncée par aria-checked', () => {
    renderFeed({ doc: CTF_DOC, nowMs: 100_000 })
    const items = within(openMenu()).getAllByRole('menuitemradio')
    expect(items.filter((i) => i.getAttribute('aria-checked') === 'true')).toHaveLength(1)
    expect(items[0].getAttribute('aria-checked')).toBe('true')
  })
})
