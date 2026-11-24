/**
 * sharedSettings.guard.test.tsx — GARDE-RAIL : un préréglage ou une vue enregistrée n'écrit JAMAIS un réglage partagé avec une
 * autre page (la Tactique, « Occupation du terrain »). Appliquer « Combat » ne change pas ce qu'elles montrent.
 *
 * La liste des réglages partagés vit à UN endroit (`SHARED_SETTING_IDS`, `settingsTable.ts`) ; ce test la lit, vérifie que
 * chacun est hors de la couverture des préréglages, hors de l'état « modifié », et qu'aucun geste d'application
 * (quatre préréglages, une vue enregistrée qui en porte les valeurs) ne touche ni leur clé ni leur geste.
 */
import { fireEvent, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { makeReplayRelief } from '../test/reliefFixtures'
import { makeSettingsContext } from '../test/settingsContextFixtures'
import { renderOpenPanel } from './replayViewPanelHarness'
import { defOf, factoryValues, PRESET_COVERED, SHARED_SETTING_IDS, type SettingId } from './settingsTable'
import { changedFrom, PRESET_IDS, presetValues } from './viewPresets'

/** La clé de stockage (ou son préfixe) de chaque réglage partagé : le test exige qu'aucun ne manque. */
const STORAGE_PREFIX: Readonly<Partial<Record<SettingId, string>>> = {
  bg: 'replay-background-style',
  reliefLines: 'replay-relief-lines.',
  wallPlan: 'replay-show-wall-plan',
  orientation: 'replay-orientation.',
}

const LABEL = { complet: 'Complet', combat: 'Combat', deplacements: 'Déplacements', objectif: 'Objectif' } as const

const shared = () =>
  Object.entries(window.localStorage as unknown as Record<string, string>)
    .filter(([k]) => Object.values(STORAGE_PREFIX).some((p) => k.startsWith(p as string)))
    .sort(([a], [b]) => a.localeCompare(b))

beforeEach(() => window.localStorage.clear())

describe('les réglages partagés avec une autre page', () => {
  it('sont listés à un endroit, et chacun a sa clé dans ce test', () => {
    expect([...SHARED_SETTING_IDS].sort()).toEqual(['bg', 'orientation', 'reliefLines', 'wallPlan'])
    for (const id of SHARED_SETTING_IDS) expect(STORAGE_PREFIX[id]).toBeTruthy()
  })

  it('ne sont couverts par aucun préréglage, et la table le dit', () => {
    for (const id of SHARED_SETTING_IDS) {
      expect(defOf(id).preset).toBe(false)
      expect(PRESET_COVERED).not.toContain(id)
      for (const p of PRESET_IDS) expect(presetValues(p)).not.toHaveProperty(id)
    }
    expect(PRESET_COVERED).toHaveLength(29)
  })

  it('ne comptent pas dans l’état « modifié »', () => {
    const values = { ...factoryValues(), bg: 'relief', reliefLines: false, wallPlan: false, orientation: '2' }
    expect(changedFrom(presetValues('complet'), values, makeSettingsContext())).toEqual([])
  })
})

describe('appliquer une vue ne touche à aucun réglage partagé', () => {
  function seed() {
    window.localStorage.setItem('replay-background-style', 'rendu')
    window.localStorage.setItem('replay-show-wall-plan', 'false')
    window.localStorage.setItem('replay-relief-lines.plan', 'true')
    window.localStorage.setItem('replay-orientation.carte-a', '2')
  }

  it.each(PRESET_IDS)('le préréglage %s : ni clé partagée écrite, ni geste du relief ni de l’orientation', (id) => {
    seed()
    const before = shared()
    const lines = vi.fn()
    const setTurns = vi.fn()
    const relief = makeReplayRelief({ options: ['plan', 'rendu', 'relief'], selected: 'rendu', lines: { available: true, show: false, onToggle: lines } })
    renderOpenPanel({ relief, drawn: 'rendu', orientation: { retained: 2, setTurns, available: true } })
    fireEvent.click(screen.getByRole('button', { name: LABEL[id] }))
    expect(shared()).toEqual(before)
    expect(lines).not.toHaveBeenCalled()
    expect(setTurns).not.toHaveBeenCalled()
  })

  it('une vue enregistrée, même si son contenu porte des valeurs partagées, ne les écrit pas', () => {
    seed()
    const before = shared()
    window.localStorage.setItem(
      'replay-views',
      JSON.stringify([{ id: 'v1', name: 'Ma vue', mode: null, values: { aim: false, bg: 'plan', reliefLines: true, wallPlan: true, orientation: '0' } }]),
    )
    const lines = vi.fn()
    const setTurns = vi.fn()
    const relief = makeReplayRelief({ options: ['plan', 'rendu', 'relief'], selected: 'rendu', lines: { available: true, show: false, onToggle: lines } })
    renderOpenPanel({ relief, drawn: 'rendu', orientation: { retained: 2, setTurns, available: true } })
    fireEvent.click(screen.getByRole('button', { name: 'Ma vue' }))
    expect(screen.getByRole('switch', { name: 'Visée' })).toHaveAttribute('aria-checked', 'false')
    expect(shared()).toEqual(before)
    expect(lines).not.toHaveBeenCalled()
    expect(setTurns).not.toHaveBeenCalled()
  })

  it('« Enregistrer comme ma vue » ne retient aucun réglage partagé', () => {
    renderOpenPanel()
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer comme ma vue' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Nom de la vue' }), { target: { value: 'Ma vue' } })
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer' }))
    const [view] = JSON.parse(window.localStorage.getItem('replay-views') ?? '[]')
    for (const id of SHARED_SETTING_IDS) expect(view.values).not.toHaveProperty(id)
  })
})
