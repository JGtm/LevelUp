/**
 * Tests — la ligne de frag du fil est un BOUTON (lot 2 du plan « maquettes du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. SEULES LES LIGNES DE FRAG sont cliquables : médaille seule, mort neutre, objectif, prise et
 *     montée en véhicule restent des lignes. Sans magasin de lecture ou sans document, aucune
 *     ligne ne l'est (la page sans rejeu garde le fil en lecture seule).
 *  2. LE CLIC fait deux demandes au magasin : poser la lecture sur l'image du frag (le canvas
 *     exécute), et mettre en valeur le tueur et la victime. L'image est celle qui laisse la ligne
 *     VISIBLE au fil.
 *  3. LA LIGNE CHOISIE se voit (`aria-pressed`), un autre clic la remplace, et recliquer la même
 *     REJOUE (jeton neuf, nouvelle demande de saut).
 *  4. LE CLAVIER : Entrée et Espace activent la ligne, et Espace ne déclenche PAS le raccourci
 *     global de lecture ; la souris ne vole pas le focus (Espace reste « Lecture » après un clic).
 *  5. LE FILTRE du lot 1 est intact : une ligne filtrée se clique pareil.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { buildFeedEntries } from '../model/killFeedLogic'
import { createPlaybackStore, type PlaybackStore } from '../model/playbackStore'
import { FEED_FILTER_KEY } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const FRAME_MS = 100

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
    tMs: 1_000, xuid: 'me', ally: true, teamID: 0, weaponKey: 'br75', weaponLabel: 'BR75',
    weaponImageUrl: '/br75.png', weaponTinted: true, assistState: '', assistGamertag: '',
    assistTeamID: null, killerDamagePct: null, assistDamagePct: null, victimXuid: 'foe',
    victimGamertag: 'Cobra01', ...over,
  }
}

// 2 050 ms tombe ENTRE deux images (pas de 100 ms) : l'image posée doit être la 21e, pas la 20e.
const KILLS = [
  killEvent({ tMs: 2_050 }),
  killEvent({ tMs: 9_000, xuid: 'foe', victimXuid: 'me', victimGamertag: 'JGtm' }),
]

const DOC = testReplayDoc({
  frameIntervalMs: FRAME_MS,
  roster: ROSTER,
  objectives: [{ stat: 'flag_grabs', t: 40, xuid: 'me', timeMs: 4_000 }] as never,
})

const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')

function renderFeed(over: { store?: PlaybackStore | null; doc?: typeof DOC | null; locale?: 'fr' | 'en' } = {}) {
  const store = over.store === undefined ? createPlaybackStore() : over.store
  const entries = buildFeedEntries(KILLS, [], 0, null)
  const view = render(
    <ReplayKillFeed
      entries={entries}
      nowMs={100_000}
      playWindow={null}
      scoreboard={[]}
      xuidMeta={META}
      allegiance={ALLEGIANCE}
      locale={over.locale ?? 'fr'}
      doc={over.doc === undefined ? DOC : over.doc}
      playbackStore={store ?? undefined}
    />,
  )
  return { store, ...view }
}

beforeEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))
afterEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))

describe('quelles lignes sont cliquables', () => {
  it('chaque ligne de frag est un bouton à état (aria-pressed), les autres lignes non', () => {
    renderFeed()
    expect(screen.getAllByRole('button', { pressed: false })).toHaveLength(2)
    // La ligne d'objectif est présente dans la liste, mais n'est pas un bouton.
    const lignes = screen.getAllByRole('listitem')
    expect(lignes).toHaveLength(3)
    expect(lignes.filter((l) => l.querySelector('button'))).toHaveLength(2)
  })

  it('sans magasin de lecture, aucune ligne n’est cliquable', () => {
    renderFeed({ store: null })
    expect(screen.queryAllByRole('button', { pressed: false })).toHaveLength(0)
  })

  it('sans document (pas d’image à viser), aucune ligne n’est cliquable', () => {
    renderFeed({ doc: null })
    expect(screen.queryAllByRole('button', { pressed: false })).toHaveLength(0)
  })

  it('porte une infobulle dans la langue de la page', () => {
    renderFeed({ locale: 'en' })
    expect(screen.getAllByRole('button', { pressed: false })[0].getAttribute('title')).toContain('highlight')
  })
})

describe('le clic', () => {
  it('demande le saut, sur l’image qui laisse la ligne visible, et pose la mise en valeur', () => {
    const { store } = renderFeed()
    const saut = vi.fn()
    store!.onSeekRequest(saut)
    // Le plus récent en tête : le 2e bouton est le frag de 2 050 ms.
    fireEvent.click(screen.getAllByRole('button', { pressed: false })[1])
    expect(saut).toHaveBeenCalledWith(21)
    expect(21 * FRAME_MS).toBeGreaterThanOrEqual(2_050)
    expect(store!.focus()).toMatchObject({
      frame: 21,
      killerXuid: 'me',
      victimXuid: 'foe',
      weapon: { imageUrl: '/br75.png', tinted: true },
    })
  })

  it('marque la ligne choisie, et un autre clic la remplace', () => {
    renderFeed()
    const [recent, ancien] = screen.getAllByRole('button', { pressed: false })
    fireEvent.click(ancien)
    expect(screen.getAllByRole('button', { pressed: true })).toHaveLength(1)
    expect(ancien.getAttribute('aria-pressed')).toBe('true')
    fireEvent.click(recent)
    expect(recent.getAttribute('aria-pressed')).toBe('true')
    expect(ancien.getAttribute('aria-pressed')).toBe('false')
  })

  it('recliquer la même ligne la REJOUE : jeton neuf et nouvelle demande de saut', () => {
    const { store } = renderFeed()
    const saut = vi.fn()
    store!.onSeekRequest(saut)
    const [ligne] = screen.getAllByRole('button', { pressed: false })
    fireEvent.click(ligne)
    const premier = store!.focus()?.token ?? 0
    fireEvent.click(ligne)
    expect(store!.focus()?.token).toBe(premier + 1)
    expect(saut).toHaveBeenCalledTimes(2)
    expect(ligne.getAttribute('aria-pressed')).toBe('true')
  })

  it('la ligne se désélectionne quand la mise en valeur est effacée (Lecture, saut manuel)', () => {
    const { store } = renderFeed()
    const [ligne] = screen.getAllByRole('button', { pressed: false })
    fireEvent.click(ligne)
    expect(ligne.getAttribute('aria-pressed')).toBe('true')
    act(() => store!.clearFocus())
    expect(ligne.getAttribute('aria-pressed')).toBe('false')
  })

  it('la ligne « sélectionnée » se voit par ses deux filets de camp (tokens), pas par une teinte seule', () => {
    renderFeed()
    const [ligne] = screen.getAllByRole('button', { pressed: false })
    fireEvent.click(ligne)
    const li = ligne.closest('li') as HTMLElement
    expect(li.getAttribute('data-selected')).toBe('true')
    expect(li.style.boxShadow).toContain('var(--ac-')
  })
})

describe('le clavier et la souris', () => {
  it('Entrée et Espace activent la ligne', async () => {
    const { store } = renderFeed()
    const user = userEvent.setup()
    const [ligne] = screen.getAllByRole('button', { pressed: false })
    ligne.focus()
    await user.keyboard('{Enter}')
    expect(store!.focus()?.token).toBe(1)
    await user.keyboard(' ')
    expect(store!.focus()?.token).toBe(2)
  })

  it('Espace sur la ligne ne déclenche PAS le raccourci global de lecture', async () => {
    renderFeed()
    const global = vi.fn()
    window.addEventListener('keydown', global)
    const user = userEvent.setup()
    const [ligne] = screen.getAllByRole('button', { pressed: false })
    ligne.focus()
    await user.keyboard(' ')
    window.removeEventListener('keydown', global)
    expect(global).not.toHaveBeenCalled()
  })

  it('un clic de souris ne laisse pas le focus sur la ligne : Espace reste « Lecture »', () => {
    renderFeed()
    const [ligne] = screen.getAllByRole('button', { pressed: false })
    // Le focus d'un clic naît du mousedown : l'annuler ne bloque pas le clic.
    const accepte = fireEvent.mouseDown(ligne)
    expect(accepte).toBe(false)
  })
})

describe('le filtre du lot 1 reste intact', () => {
  it('une ligne affichée par un filtre se clique pareil', () => {
    window.localStorage.setItem(FEED_FILTER_KEY, 'kills')
    const { store } = renderFeed()
    expect(screen.getAllByRole('listitem')).toHaveLength(2)
    fireEvent.click(screen.getAllByRole('button', { pressed: false })[0])
    expect(store!.focus()?.killerXuid).toBe('foe')
  })
})

describe('ce que le fil ne change pas', () => {
  it('la ligne garde ses règles : une seule ligne, sans retour', () => {
    renderFeed()
    const [ligne] = screen.getAllByRole('button', { pressed: false })
    expect(ligne.className).toContain('flex-nowrap')
    expect(ligne.className).toContain('overflow-hidden')
  })
})
