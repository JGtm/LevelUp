/**
 * settingsBinding.test.tsx — LE LIEN TABLE <-> ÉTATS, sur les vrais réglages : chaque identifiant de la table lit
 * le bon état et l'écrit par le bon geste, un oui/non ne déclenche sa bascule que si la valeur diffère, et les
 * gestes sans état propre (son, étage, orientation, filtre du fil) sont relayés tels quels.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { makeReplayRelief } from '../test/reliefFixtures'
import type { ReplaySound } from '../sound/useReplaySound'
import { bindSettings, type BindingInput } from './settingsBinding'
import { SETTING_IDS } from './settingsTable'
import { useReplaySettings } from './useReplaySettings'

function makeSound(over: Partial<ReplaySound> = {}): ReplaySound {
  return {
    available: true, on: false, toggle: vi.fn(), volume: 0.7, setVolume: vi.fn(),
    categories: { weapon: true, grenade: true, melee: false, equipment: true, objective: true },
    toggleCategory: vi.fn(),
    ...over,
  } as unknown as ReplaySound
}

function mount(over: Partial<BindingInput> = {}) {
  const extra = {
    relief: makeReplayRelief(),
    orientation: { retained: null, setTurns: vi.fn(), available: true },
    sound: makeSound(),
    feed: { value: 'tout' as const, choose: vi.fn() },
    heatMode: 'presence' as const,
    followedKnown: true,
    ...over,
  }
  const hook = renderHook(() => {
    const settings = useReplaySettings()
    return { settings, binding: bindSettings({ settings, ...extra }) }
  })
  return { hook, extra }
}

beforeEach(() => window.localStorage.clear())

describe('bindSettings — lire', () => {
  it('rend une valeur pour chaque réglage de la table, aux défauts d’usine', () => {
    const { hook } = mount()
    const { values } = hook.result.current.binding
    expect(Object.keys(values).sort()).toEqual([...SETTING_IDS].sort())
    expect(values).toMatchObject({
      aim: true, spawns: false, forecast: true, reliefLines: false, wallPlan: true, zones: true, heatmap: false,
      colors: 'team', livesWho: 'all', orientation: 'auto', floor: null, feedFilter: 'tout', sound: false, volume: 0.7,
      catWeapon: true, catMelee: false, thinBar: true, planFeed: true, groundSpecial: false,
    })
  })

  it('lit le fond retenu après repli, la chaleur servie et l’orientation retenue', () => {
    const { hook } = mount({
      relief: makeReplayRelief({ selected: 'relief' }),
      heatMode: 'kills',
      orientation: { retained: 3, setTurns: vi.fn(), available: true },
    })
    expect(hook.result.current.binding.values).toMatchObject({ bg: 'relief', heatMode: 'kills', orientation: '3' })
  })

  it('« point de vue » sans joueur suivi se lit « tous », sans effacer la préférence', () => {
    const { hook } = mount({ followedKnown: false })
    act(() => hook.result.current.settings.setLivesWho('followed'))
    expect(hook.result.current.binding.values.livesWho).toBe('all')
    expect(hook.result.current.settings.livesWho).toBe('followed')
  })
})

describe('bindSettings — écrire un oui/non', () => {
  it('bascule le calque quand la valeur diffère, et pas sinon', () => {
    const { hook } = mount()
    act(() => hook.result.current.binding.set('aim', true))
    expect(hook.result.current.settings.showAim).toBe(true)
    act(() => hook.result.current.binding.set('aim', false))
    expect(hook.result.current.settings.showAim).toBe(false)
    expect(window.localStorage.getItem('replay-show-aim')).toBe('false')
  })

  it('plusieurs réglages distincts dans un même geste s’écrivent tous', () => {
    const { hook } = mount()
    act(() => {
      const { set } = hook.result.current.binding
      set('deaths', true)
      set('killers', true)
      set('zones', false)
      set('placeDropped', false)
    })
    const s = hook.result.current.settings
    expect([s.showDeaths, s.showKillers, s.showZones, s.showDroppedPlacements]).toEqual([true, true, false, false])
  })

  it('relaie les bascules du relief et de l’étage tenues ailleurs', () => {
    const relief = makeReplayRelief({ lines: { available: true, show: false, onToggle: vi.fn() } })
    const { hook } = mount({ relief })
    act(() => hook.result.current.binding.set('reliefLines', true))
    expect(relief.lines.onToggle).toHaveBeenCalledTimes(1)
    act(() => hook.result.current.binding.set('reliefLines', false))
    expect(relief.lines.onToggle).toHaveBeenCalledTimes(1)
  })
})

describe('bindSettings — écrire un choix ou un niveau', () => {
  it('écrit la couleur, la chaleur, les vies, le dessin et le fond par leurs gestes', () => {
    const { hook } = mount()
    act(() => {
      const { set } = hook.result.current.binding
      set('colors', 'player')
      set('heatMode', 'kills')
      set('heatSpan', 'live')
      set('livesWho', 'team')
      set('livesWhen', 'sofar')
      set('forecastDraw', 'zones')
      set('bg', 'rendu')
    })
    const s = hook.result.current.settings
    expect([s.markerColors, s.heatmapMode, s.heatmapSpan, s.livesWho, s.livesWhen, s.respawnForecastDraw, s.backgroundStyle]).toEqual([
      'player', 'kills', 'live', 'team', 'sofar', 'zones', 'rendu',
    ])
  })

  it('l’orientation : « auto » efface le choix, un chiffre retient le quart de tour', () => {
    const setTurns = vi.fn()
    const { hook } = mount({ orientation: { retained: 1, setTurns, available: true } })
    act(() => hook.result.current.binding.set('orientation', 'auto'))
    expect(setTurns).toHaveBeenLastCalledWith(null)
    act(() => hook.result.current.binding.set('orientation', '2'))
    expect(setTurns).toHaveBeenLastCalledWith(2)
  })

  it('le filtre du fil, le volume et la coupe d’étage', () => {
    const relief = makeReplayRelief()
    const sound = makeSound()
    const feed = { value: 'tout' as const, choose: vi.fn() }
    const { hook } = mount({ relief, sound, feed })
    act(() => {
      const { set } = hook.result.current.binding
      set('feedFilter', 'kills')
      set('volume', 0.4)
      set('floor', null)
      set('floor', 2.5)
    })
    expect(feed.choose).toHaveBeenCalledWith('kills')
    expect(sound.setVolume).toHaveBeenCalledWith(0.4)
    expect(relief.floor.selectAll).toHaveBeenCalledTimes(1)
    expect(relief.floor.setHeight).toHaveBeenCalledWith(2.5)
  })
})

describe('bindSettings — le son', () => {
  it('l’interrupteur et chaque catégorie passent par le son de la page, sans bascule inutile', () => {
    const sound = makeSound({ on: false })
    const { hook } = mount({ sound })
    act(() => {
      const { set } = hook.result.current.binding
      set('sound', true)
      set('sound', false)
      set('catMelee', true)
      set('catWeapon', true)
    })
    expect(sound.toggle).toHaveBeenCalledTimes(1)
    expect(sound.toggleCategory).toHaveBeenCalledTimes(1)
    expect(sound.toggleCategory).toHaveBeenCalledWith('melee')
  })
})
