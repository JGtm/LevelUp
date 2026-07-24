/**
 * Tests du store d'apparence Spartan (V72-14) : isolation par joueur ET par titre,
 * couleurs emblème/bannière indépendantes, migration v2→v3 (reset propre).
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'

import {
  useSpartanAppearanceStore,
  useSpartanAppearance,
  appearanceKey,
  migrateSpartanAppearance,
  DEFAULT_SPARTAN_APPEARANCE,
  type SpartanAppearance,
} from './store'

const appA: SpartanAppearance = {
  emblemId: '200',
  nameplateId: '201',
  emblemColors: { primary: '#111111', secondary: '#222222', tertiary: '#333333' },
  nameplateColors: { primary: '#444444', secondary: '#555555', tertiary: '#666666' },
}

const appB: SpartanAppearance = {
  emblemId: '300',
  nameplateId: '301',
  emblemColors: { primary: '#aaaaaa', secondary: '#bbbbbb', tertiary: '#cccccc' },
  nameplateColors: { primary: '#dddddd', secondary: '#eeeeee', tertiary: '#ffffff' },
}

beforeEach(() => {
  useSpartanAppearanceStore.setState({ byKey: {} })
})

describe('SpartanAppearance store — clé composite (titre, joueur)', () => {
  it('isole les apparences par JOUEUR (même titre)', () => {
    useSpartanAppearanceStore.getState().setAppearance('halo_5', 'alice', appA)

    // Le joueur enregistré porte son apparence…
    expect(useSpartanAppearanceStore.getState().byKey[appearanceKey('halo_5', 'alice')]).toEqual(appA)
    // …un autre joueur du même titre reste au défaut (pas de fuite partagée).
    const { result } = renderHook(() => useSpartanAppearance('halo_5', 'bob'))
    expect(result.current).toEqual(DEFAULT_SPARTAN_APPEARANCE)
  })

  it('isole les apparences par TITRE (même joueur)', () => {
    useSpartanAppearanceStore.getState().setAppearance('halo_5', 'alice', appA)
    useSpartanAppearanceStore.getState().setAppearance('halo_infinite', 'alice', appB)

    expect(useSpartanAppearanceStore.getState().byKey[appearanceKey('halo_5', 'alice')]).toEqual(appA)
    expect(useSpartanAppearanceStore.getState().byKey[appearanceKey('halo_infinite', 'alice')]).toEqual(appB)
  })

  it('retourne le défaut (jamais vide) quand rien n\'est enregistré', () => {
    const { result } = renderHook(() => useSpartanAppearance('halo_5', 'unknown'))
    expect(result.current).toEqual(DEFAULT_SPARTAN_APPEARANCE)
    // Les deux jeux de couleurs par défaut existent.
    expect(result.current.emblemColors).toBeDefined()
    expect(result.current.nameplateColors).toBeDefined()
  })

  it('couleurs emblème et bannière indépendantes', () => {
    useSpartanAppearanceStore.getState().setAppearance('halo_5', 'alice', appA)
    const saved = useSpartanAppearanceStore.getState().byKey[appearanceKey('halo_5', 'alice')]
    expect(saved.emblemColors).not.toEqual(saved.nameplateColors)
    expect(saved.emblemColors.primary).toBe('#111111')
    expect(saved.nameplateColors.primary).toBe('#444444')
  })
})

describe('migration v2 -> v3', () => {
  it('RÉINITIALISE proprement (pas de fallback partagé qui recréerait la fuite)', () => {
    // migrate ignore l'état persisté (v1 global OU v2 byTitle partagé entre joueurs) et
    // repart d'un byKey vide. Un fallback qui rattacherait ces données recréerait
    // exactement le bug V72-14 (composition partagée entre tous les joueurs).
    expect(migrateSpartanAppearance().byKey).toEqual({})
  })
})
