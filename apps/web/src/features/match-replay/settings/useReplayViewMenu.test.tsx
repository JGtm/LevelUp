/**
 * useReplayViewMenu.test.tsx — LE BOUTON « VUE » branché sur les vrais réglages, sans rendu du panneau : les clés
 * `replay-*` d'avant restent lues telles quelles, la reprise des préférences d'avant passe avant la première lecture,
 * le contexte du match est composé de ce que le canvas publie, et l'ouverture d'un rejeu applique sa vue une fois.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'

import { makeReplayRelief, THREE_FLOORS } from '../test/reliefFixtures'
import { FULL_FILM, SLAYER_FILM } from '../test/settingsContextFixtures'
import { testReplayDoc } from '../test/testDoc'
import type { FilmLayerAvailability } from './filmLayerAvailability'
import { makeSound, NO_SOUND } from '../test/soundFixtures'
import { useReplaySettings } from './useReplaySettings'
import { useReplayViewMenu, type ReplayViewMenuOptions } from './useReplayViewMenu'

const DOC = testReplayDoc()
const ALLEGIANCE = buildFilmAllegiance([], null)

function mount(over: Partial<ReplayViewMenuOptions> = {}, film: FilmLayerAvailability = FULL_FILM) {
  return renderHook(
    (props: { matchKey: string }) => {
      const settings = useReplaySettings()
      const menu = useReplayViewMenu({
        settings,
        orientation: { retained: null, setTurns: vi.fn(), available: true },
        sound: NO_SOUND,
        available: film,
        heat: { mode: 'presence', killsAvailable: true },
        relief: makeReplayRelief(),
        drawn: 'rendu',
        planFeedAvailable: false,
        locale: 'fr',
        feed: { doc: DOC, entries: [], allegiance: ALLEGIANCE },
        matchKey: props.matchKey,
        ...over,
      })
      return { settings, menu }
    },
    { initialProps: { matchKey: 'm1' } },
  )
}

beforeEach(() => window.localStorage.clear())

describe('useReplayViewMenu', () => {
  it('le panneau est fermé par défaut ; le bouton l’ouvre et le referme, la fermeture rend le focus', () => {
    const { result } = mount()
    expect(result.current.menu.open).toBe(false)
    act(() => result.current.menu.toggle())
    expect(result.current.menu.open).toBe(true)
    act(() => result.current.menu.close())
    expect(result.current.menu.open).toBe(false)
  })

  it('la vue d’ouverture par défaut est Complet, sans écart sur un navigateur neuf', () => {
    const { result } = mount()
    expect(result.current.menu.label).toEqual({ name: 'Complet', hint: expect.any(String), modified: false })
  })

  it('les clés d’avant sont relues telles quelles : un calque éteint l’était, la vue le dit modifiée', () => {
    window.localStorage.setItem('replay-show-zones', 'false')
    window.localStorage.setItem('replay-show-vehicles', 'false')
    const { result } = mount()
    expect(result.current.menu.model.values).toMatchObject({ zones: false, vehicles: false, aim: true })
    expect(result.current.menu.label.modified).toBe(true)
    expect(result.current.menu.model.views.changed).toEqual(['zones', 'vehicles'])
  })

  it('un état d’avant les préréglages souples reste sa propre vue : Combat retenu, ses morts et tueurs repris', () => {
    window.localStorage.setItem('replay-view-preset', 'combat')
    for (const k of ['aim', 'trail', 'vehicles', 'mode-objectives', 'flag-carries', 'vip-crown', 'skull-carrier', 'bomb-carrier']) {
      window.localStorage.setItem(`replay-show-${k}`, 'true')
    }
    for (const k of ['zones', 'weapon-pads', 'ground-weapons', 'placements', 'dropped-placements']) {
      window.localStorage.setItem(`replay-show-${k}`, 'false')
    }
    const { result } = mount()
    expect(result.current.settings.showDeaths).toBe(true)
    expect(result.current.settings.showKillers).toBe(true)
    expect(result.current.settings.showSpawns).toBe(false)
    expect(result.current.menu.model.views.state.activeId).toBe('combat')
  })

  it('un réglage écrit par le modèle garde sa clé d’avant', () => {
    const { result } = mount()
    act(() => result.current.menu.model.set('aim', false))
    expect(result.current.settings.showAim).toBe(false)
    expect(window.localStorage.getItem('replay-show-aim')).toBe('false')
    act(() => result.current.menu.model.set('heatSpan', 'live'))
    expect(window.localStorage.getItem('replay-heatmap-span')).toBe('live')
  })

  it('compose le contexte du match : fond posé, étage, fil, filtres, son, mode', () => {
    const relief = makeReplayRelief({
      options: ['plan', 'rendu'],
      floor: { ...makeReplayRelief().floor, state: { state: 'ready', chips: THREE_FLOORS, min: 0, max: 4.4 } },
    })
    const { result } = mount({ relief, drawn: 'plan', planFeedAvailable: true, thinBarAvailable: true, sound: makeSound(), heat: { mode: 'presence', killsAvailable: false } })
    const { ctx } = result.current.menu.model
    expect(ctx).toMatchObject({
      drawn: 'plan', backgroundOptions: ['plan', 'rendu'], floor: 'ready', planFeedAvailable: true, thinBarAvailable: true,
      soundAvailable: true, heatKillsAvailable: false, orientationAvailable: true,
    })
    expect(ctx.feedFilters).toEqual(['tout', 'kills', 'pickups', 'vehicles', 'medals', 'assists'])
  })

  it('la lecture de la chaleur est celle qui est servie', () => {
    const { result } = mount({ heat: { mode: 'kills', killsAvailable: true } })
    expect(result.current.menu.model.values.heatMode).toBe('kills')
  })

  it('applique la vue d’ouverture au montage d’un rejeu, et une seule fois par rejeu', () => {
    window.localStorage.setItem('replay-view-default', 'objectif')
    const { result, rerender } = mount()
    expect(result.current.menu.model.views.state.activeId).toBe('objectif')
    expect(result.current.settings.showAim).toBe(false)
    act(() => result.current.menu.model.set('aim', true))
    rerender({ matchKey: 'm1' })
    expect(result.current.settings.showAim).toBe(true)
    rerender({ matchKey: 'm2' })
    expect(result.current.settings.showAim).toBe(false)
  })

  it('sur un Assassin, la vue d’ouverture n’écrit que ce que le match propose', () => {
    window.localStorage.setItem('replay-view-default', 'combat')
    const { result } = mount({}, SLAYER_FILM)
    expect(result.current.menu.model.views.state.activeId).toBe('combat')
    expect(window.localStorage.getItem('replay-show-zones')).toBeNull()
    expect(window.localStorage.getItem('replay-show-deaths')).toBeNull()
  })
})
