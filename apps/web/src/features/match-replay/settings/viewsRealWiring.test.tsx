/**
 * viewsRealWiring.test.tsx — LES VUES SUR LE VRAI CÂBLAGE : le vrai `useReplaySettings`, le vrai `useReplaySound`,
 * `bindSettings` et `useViewsController` ensemble, sans faux. C'est ce que les tests au son simulé ne pouvaient pas voir :
 * plusieurs catégories de son écrites dans le même geste (un préréglage, le maître d'une rubrique, « Réinitialiser »)
 * doivent TOUTES prendre, pas seulement la dernière.
 */
import { act, renderHook } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { effectiveFeedFilter, type FeedFilterId } from '../model/feedFilterLogic'
import { useReplaySound } from '../sound/useReplaySound'
import { installFakeAudio } from '../test/fakeAudio'
import { CHATTER_FILM, makeSettingsContext } from '../test/settingsContextFixtures'
import { testReplayDoc } from '../test/testDoc'
import { isProposed, masterWrites } from './settingsAvailability'
import { bindSettings, type BindingInput } from './settingsBinding'
import { PRESET_COVERED, type SettingId } from './settingsTable'
import { useFeedFilter } from './useFeedFilter'
import { useReplaySettings } from './useReplaySettings'
import { useViewsController } from './useViewsController'
import { PRESET_IDS, presetValues, type PresetId } from './viewPresets'

const CTX = makeSettingsContext({ film: CHATTER_FILM })
const DOC = testReplayDoc({ frameIntervalMs: 100 })
const SOUND_IDS: readonly SettingId[] = ['catWeapon', 'catGrenade', 'catMelee', 'catEquip', 'chatterVoice']

const RELIEF = {
  selected: 'plan',
  lines: { available: true, show: true, onToggle: vi.fn() },
  floor: { cut: null, follow: { available: false, on: false, onToggle: vi.fn() }, selectAll: vi.fn(), setHeight: vi.fn() },
} as unknown as BindingInput['relief']
const ORIENTATION = { retained: null, setTurns: vi.fn(), available: true } as unknown as BindingInput['orientation']

function mount() {
  return renderHook(() => {
    const settings = useReplaySettings()
    const sound = useReplaySound(DOC, [], 1, { allegiance: NO_ALLEGIANCE, endMatch: null, locale: undefined, chatterVoices: [{ ms: 1, stem: 'repliques/fr/a', chatter: true }] })
    const [feed, setFeed] = useState<FeedFilterId>('tout')
    const binding = bindSettings({
      settings, relief: RELIEF, orientation: ORIENTATION, sound,
      feed: { value: feed, choose: setFeed }, heatMode: settings.heatmapMode, followedKnown: true,
    })
    const controller = useViewsController({ ctx: CTX, binding, mode: 'flag', matchKey: 'm1' })
    return { controller, values: binding.values, sound, binding }
  })
}

/** Ce que le son porte, par réglage : l'état RÉEL, relu après le geste. */
const soundState = (v: Record<string, unknown>) => Object.fromEntries(SOUND_IDS.map((id) => [id, v[id]]))

beforeEach(() => {
  window.localStorage.clear()
  installFakeAudio() // le maître Son allume aussi « Son » : le lecteur s'ouvre sur le double Web Audio
})

afterEach(() => vi.unstubAllGlobals())

describe('les quatre préréglages, de chacun vers chacun, sur le vrai son', () => {
  for (const from of PRESET_IDS) {
    for (const to of PRESET_IDS) {
      if (from === to) continue
      it(`${from} puis ${to} : exactement la table de ${to}, vue non modifiée, « Revenir à » sans effet`, () => {
        const { result } = mount()
        act(() => result.current.controller.apply(from))
        act(() => result.current.controller.apply(to))
        const table = presetValues(to as PresetId)
        for (const id of PRESET_COVERED) {
          if (table[id] !== undefined && isProposed(id, CTX)) expect(result.current.values[id], `${id} après ${from} puis ${to}`).toBe(table[id])
        }
        expect(result.current.controller.changed).toEqual([])
        expect(result.current.controller.state.activeId).toBe(to)
      })
    }
  }

  it('« Revenir à » ramène en UN geste une vue dont plusieurs catégories de son ont été changées à la main', () => {
    const { result } = mount()
    act(() => result.current.controller.apply('complet'))
    act(() => {
      result.current.binding.set('catWeapon', false)
      result.current.binding.set('catGrenade', false)
      result.current.binding.set('chatterVoice', false)
    })
    expect(result.current.controller.changed).toEqual(['catWeapon', 'catGrenade', 'chatterVoice'])
    act(() => result.current.controller.revert())
    expect(result.current.controller.changed).toEqual([])
    expect(soundState(result.current.values)).toEqual({ catWeapon: true, catGrenade: true, catMelee: true, catEquip: true, chatterVoice: true })
  })
})

describe('le maître de la rubrique Son et « Réinitialiser »', () => {
  it('le maître Son éteint tout en un geste, puis rallume tout en un geste', () => {
    const { result } = mount()
    act(() => { for (const [id, v] of masterWrites('son', CTX, false)) result.current.binding.set(id, v) })
    expect(soundState(result.current.values)).toEqual({ catWeapon: false, catGrenade: false, catMelee: false, catEquip: false, chatterVoice: false })
    expect(result.current.sound.categories).toMatchObject({ weapon: false, grenade: false, melee: false, equipment: false, voice: false })
    act(() => { for (const [id, v] of masterWrites('son', CTX, true)) result.current.binding.set(id, v) })
    expect(soundState(result.current.values)).toEqual({ catWeapon: true, catGrenade: true, catMelee: true, catEquip: true, chatterVoice: true })
  })

  it('« Réinitialiser » rétablit en un geste toutes les catégories coupées', () => {
    const { result } = mount()
    act(() => { for (const [id, v] of masterWrites('son', CTX, false)) result.current.binding.set(id, v) })
    act(() => result.current.controller.reset())
    expect(soundState(result.current.values)).toEqual({ catWeapon: true, catGrenade: true, catMelee: true, catEquip: true, chatterVoice: true })
  })

  it('les catégories coupées survivent au rechargement (une seule écriture, la bonne)', () => {
    const first = mount()
    act(() => { for (const [id, v] of masterWrites('son', CTX, false)) first.result.current.binding.set(id, v) })
    first.unmount()
    const second = mount()
    expect(second.result.current.sound.categories).toMatchObject({ weapon: false, grenade: false, melee: false, equipment: false, voice: false })
  })
})

describe('« Réinitialiser » et les préférences que ce match ne propose pas', () => {
  /** Un match sans joueur suivi (« Qui = Point de vue » n'y est pas proposé) et sans ligne d'objectif (filtre « Objectifs » absent). */
  function mountSansSuiviNiObjectif() {
    return renderHook(() => {
      const settings = useReplaySettings()
      const sound = useReplaySound(DOC, [], 1, { allegiance: NO_ALLEGIANCE, endMatch: null, locale: undefined })
      const [stored, choose] = useFeedFilter()
      const binding = bindSettings({
        settings, relief: RELIEF, orientation: ORIENTATION, sound,
        feed: { value: effectiveFeedFilter(stored, [{ id: 'tout', count: 1 }, { id: 'kills', count: 1 }]), retained: stored, choose },
        heatMode: settings.heatmapMode, followedKnown: false,
      })
      const controller = useViewsController({ ctx: CTX, binding, mode: 'flag', matchKey: 'm1' })
      return { controller, settings, binding }
    })
  }

  it('remet à l’usine « Qui = Point de vue » et le filtre « Objectifs » retenus, même quand ils ne s’appliquent pas ici', () => {
    window.localStorage.setItem('replay-lives-who', 'followed')
    window.localStorage.setItem('replay-feed-filter', 'objectives')
    const { result } = mountSansSuiviNiObjectif()
    expect(result.current.binding.values.livesWho).toBe('all') // la valeur appliquée : le repli
    expect(result.current.settings.livesWho).toBe('followed') // la valeur retenue : celle du lecteur
    act(() => result.current.controller.reset())
    expect(result.current.settings.livesWho).toBe('all')
    expect(window.localStorage.getItem('replay-lives-who')).toBe('all')
    expect(window.localStorage.getItem('replay-feed-filter')).toBe('tout')
  })
})
