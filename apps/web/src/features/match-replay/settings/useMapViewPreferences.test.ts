/**
 * useMapViewPreferences.test.ts — les autres vues 2D lisent LES MÊMES préférences que le rejeu :
 * l'orientation par carte, l'interrupteur du plan des murs et le style du fond, sans clé qui leur soit propre.
 */
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { useMapViewPreferences } from './useMapViewPreferences'

beforeEach(() => window.localStorage.clear())

describe('useMapViewPreferences', () => {
  it('par défaut : carte non tournée, plan des murs allumé, fond rendu', () => {
    const { result } = renderHook(() => useMapViewPreferences('ridgeline'))
    expect(result.current).toEqual({ turns: 0, showWallPlan: true, backgroundStyle: 'rendu' })
  })

  it('lit l’orientation écrite par le rejeu sous replay-orientation.<carte>, carte par carte', () => {
    window.localStorage.setItem('replay-orientation.ridgeline', '3')
    expect(renderHook(() => useMapViewPreferences('ridgeline')).result.current.turns).toBe(3)
    expect(renderHook(() => useMapViewPreferences('streets')).result.current.turns).toBe(0)
  })

  it('lit l’interrupteur du plan des murs du rejeu (replay-show-wall-plan)', () => {
    window.localStorage.setItem('replay-show-wall-plan', 'false')
    expect(renderHook(() => useMapViewPreferences('ridgeline')).result.current.showWallPlan).toBe(false)
  })

  it('lit le style du fond du rejeu (replay-background-style) : plan, ou rendu pour toute autre valeur', () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    expect(renderHook(() => useMapViewPreferences('ridgeline')).result.current.backgroundStyle).toBe('plan')
    window.localStorage.setItem('replay-background-style', 'inconnu')
    expect(renderHook(() => useMapViewPreferences('ridgeline')).result.current.backgroundStyle).toBe('rendu')
  })

  it('sans carte connue : pas de rotation', () => {
    window.localStorage.setItem('replay-orientation.', '2')
    expect(renderHook(() => useMapViewPreferences(undefined)).result.current.turns).toBe(0)
  })
})
