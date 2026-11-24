/**
 * useViewsController.test.tsx — LES VUES EN ACTION sur un jeu de réglages en mémoire : appliquer, « modifié »
 * déduit, revenir, enregistrer, mettre à jour, vue d'ouverture et vue de mode, épingles, réinitialiser. La mémoire
 * est le vrai stockage du navigateur (jsdom) : on remonte le hook pour vérifier ce qui survit.
 */
import { act, renderHook } from '@testing-library/react'
import { useState } from 'react'
import { beforeEach, describe, expect, it } from 'vitest'

import { makeSettingsContext, SLAYER_FILM } from '../test/settingsContextFixtures'
import type { MatchModeKey, SettingsContext } from './settingsAvailability'
import type { SettingsBinding } from './settingsBinding'
import { factoryValues, type SettingValues } from './settingsTable'
import { useViewsController } from './useViewsController'

interface Opts {
  ctx?: SettingsContext
  mode?: MatchModeKey
  matchKey?: string
  initial?: Partial<Record<string, unknown>>
}

/** Les réglages en mémoire : `set` les change et rend le rendu suivant. */
function mount({ ctx = makeSettingsContext(), mode = 'flag', matchKey = 'm1', initial = {} }: Opts = {}) {
  let counter = 0
  return renderHook(
    (props: { matchKey: string }) => {
      const [values, setValues] = useState<SettingValues>({ ...factoryValues(), ...initial } as SettingValues)
      const binding: SettingsBinding = { values, set: (id, value) => setValues((v) => ({ ...v, [id]: value })) }
      const controller = useViewsController({ ctx, binding, mode, matchKey: props.matchKey, newId: () => `v${++counter}` })
      return { controller, values }
    },
    { initialProps: { matchKey } },
  )
}

beforeEach(() => window.localStorage.clear())

describe('appliquer une vue', () => {
  it('Combat écrit ses réglages, la vue active est Combat, et rien n’est modifié', () => {
    const { result } = mount()
    act(() => result.current.controller.apply('combat'))
    expect(result.current.values).toMatchObject({ deaths: true, killers: true, zones: false, feedFilter: 'kills', aim: true })
    expect(result.current.controller.state.activeId).toBe('combat')
    expect(result.current.controller.changed).toEqual([])
  })

  it('un réglage changé à la main rend la vue « modifiée » ; le rétablir la rend telle quelle', () => {
    const { result: r2 } = mount({ initial: { aim: false } })
    expect(r2.current.controller.changed).toEqual(['aim'])
    act(() => r2.current.controller.revert())
    expect(r2.current.values.aim).toBe(true)
    expect(r2.current.controller.changed).toEqual([])
  })

  it('un réglage sans objet sur ce match ne rend pas la vue modifiée et reste intact à l’application', () => {
    const { result } = mount({ ctx: makeSettingsContext({ film: SLAYER_FILM }), initial: { zones: true, objectives: true } })
    act(() => result.current.controller.apply('combat'))
    expect(result.current.values.zones).toBe(true)
    expect(result.current.controller.changed).toEqual([])
  })

  it('retient la vue active pour le prochain rejeu', () => {
    const first = mount()
    act(() => first.result.current.controller.apply('deplacements'))
    first.unmount()
    const second = mount()
    expect(second.result.current.controller.state.activeId).toBe('deplacements')
  })
})

describe('Mes vues', () => {
  it('enregistre l’état courant sous un nom, devient la vue active, et se retrouve au prochain montage', () => {
    const { result } = mount({ initial: { aim: false, heatmap: true } })
    let res: string = ''
    act(() => { res = result.current.controller.saveAs('Ma vue') })
    expect(res).toBe('ok')
    expect(result.current.controller.state.views.map((v) => v.name)).toEqual(['Ma vue'])
    expect(result.current.controller.state.activeId).toBe('v1')
    expect(result.current.controller.activeUser?.values).toMatchObject({ aim: false, heatmap: true })
    expect(result.current.controller.changed).toEqual([])
    expect(mount().result.current.controller.state.views).toHaveLength(1)
  })

  it('refuse un nom vide et une quatrième vue', () => {
    const { result } = mount()
    let res: string = ''
    act(() => { res = result.current.controller.saveAs('  ') })
    expect(res).toBe('name')
    for (const n of ['A', 'B', 'C']) act(() => { result.current.controller.saveAs(n) })
    act(() => { res = result.current.controller.saveAs('D') })
    expect(res).toBe('limit')
    expect(result.current.controller.state.views).toHaveLength(3)
  })

  it('met à jour la vue active quand on l’a modifiée', () => {
    const { result } = mount()
    act(() => { result.current.controller.saveAs('Ma vue') })
    act(() => result.current.controller.updateActive())
    expect(result.current.controller.activeUser?.name).toBe('Ma vue')
  })

  it('appliquer une vue enregistrée rétablit ses réglages', () => {
    const { result } = mount({ initial: { heatmap: true } })
    act(() => { result.current.controller.saveAs('Chaleur') })
    act(() => result.current.controller.apply('combat'))
    expect(result.current.values.heatmap).toBe(false)
    act(() => result.current.controller.apply('v1'))
    expect(result.current.values.heatmap).toBe(true)
  })

  it('renomme, supprime (la vue active revient à Complet), choisit la vue d’ouverture et un mode', () => {
    const { result } = mount()
    act(() => { result.current.controller.saveAs('A') })
    act(() => result.current.controller.rename('v1', 'Autre'))
    act(() => result.current.controller.setDefault('v1'))
    act(() => result.current.controller.attach('v1', 'flag'))
    expect(result.current.controller.state.views[0]).toMatchObject({ name: 'Autre', mode: 'flag' })
    expect(result.current.controller.state.defaultId).toBe('v1')
    act(() => result.current.controller.remove('v1'))
    expect(result.current.controller.state.views).toEqual([])
    expect(result.current.controller.state.activeId).toBe('complet')
    expect(result.current.controller.state.defaultId).toBeNull()
  })
})

describe('l’ouverture d’un rejeu', () => {
  it('sans vue d’ouverture, les réglages restent comme laissés', () => {
    const { result } = mount({ initial: { aim: false } })
    expect(result.current.values.aim).toBe(false)
  })

  it('applique la vue d’ouverture une seule fois par rejeu ouvert', () => {
    const first = mount()
    act(() => first.result.current.controller.setDefault('combat'))
    first.unmount()
    const { result, rerender } = mount()
    expect(result.current.values.killers).toBe(true)
    act(() => result.current.controller.apply('objectif'))
    rerender({ matchKey: 'm1' })
    expect(result.current.controller.state.activeId).toBe('objectif')
    rerender({ matchKey: 'm2' })
    expect(result.current.controller.state.activeId).toBe('combat')
  })

  it('la vue rattachée au mode du match passe avant la vue d’ouverture', () => {
    const first = mount()
    act(() => { first.result.current.controller.saveAs('Drapeau') })
    act(() => first.result.current.controller.attach('v1', 'flag'))
    act(() => first.result.current.controller.setDefault('combat'))
    first.unmount()
    expect(mount({ mode: 'flag' }).result.current.controller.state.activeId).toBe('v1')
    expect(mount({ mode: 'open', ctx: makeSettingsContext({ film: SLAYER_FILM }) }).result.current.controller.state.activeId).toBe('combat')
  })
})

describe('les réglages épinglés', () => {
  it('épingle jusqu’à cinq, dit quand c’est plein, et se retrouve au prochain montage', () => {
    const { result } = mount()
    expect(result.current.controller.pins).toEqual(['aim', 'trail', 'heatmap', 'planFeed'])
    let res: string = ''
    act(() => { res = result.current.controller.pin('zones') })
    expect(res).toBe('ok')
    act(() => { res = result.current.controller.pin('vehicles') })
    expect(res).toBe('full')
    expect(result.current.controller.pins).toHaveLength(5)
    act(() => { result.current.controller.pin('aim') })
    expect(mount().result.current.controller.pins).toEqual(['trail', 'heatmap', 'planFeed', 'zones'])
  })
})

describe('réinitialiser', () => {
  it('remet les réglages d’usine et Complet, garde Mes vues, la vue d’ouverture et les épingles', () => {
    const { result } = mount({ initial: { aim: false, bg: 'relief', volume: 0.2, sound: true, floor: 2, reliefLines: false } })
    act(() => { result.current.controller.saveAs('Ma vue') })
    act(() => result.current.controller.setDefault('v1'))
    act(() => { result.current.controller.pin('zones') })
    act(() => result.current.controller.reset())
    // Les traits du relief n'ont pas de valeur d'usine unique (elle dépend du fond) : la remise à zéro n'y touche pas.
    expect(result.current.values).toEqual({ ...factoryValues(), reliefLines: false })
    expect(result.current.controller.state.activeId).toBe('complet')
    expect(result.current.controller.state.views).toHaveLength(1)
    expect(result.current.controller.state.defaultId).toBe('v1')
    expect(result.current.controller.pins).toContain('zones')
  })
})
