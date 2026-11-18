/**
 * useMapViewPreferences.test.ts — les autres vues 2D lisent LES MÊMES préférences que le rejeu :
 * l'orientation par carte, l'interrupteur du plan des murs et le style du fond, sans clé qui leur soit propre.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { useBackgroundStyle } from './useBackgroundStyle'
import { useMapViewPreferences } from './useMapViewPreferences'

beforeEach(() => window.localStorage.clear())

describe('useMapViewPreferences', () => {
  it('par défaut : carte non tournée, plan des murs allumé, fond plan', () => {
    const { result } = renderHook(() => useMapViewPreferences('ridgeline'))
    expect(result.current).toEqual({
      turns: 0,
      showWallPlan: true,
      backgroundStyle: 'plan',
      setBackgroundStyle: expect.any(Function),
    })
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

  it('lit le style du fond du rejeu (replay-background-style) : rendu, ou plan (le défaut) pour toute autre valeur', () => {
    window.localStorage.setItem('replay-background-style', 'rendu')
    expect(renderHook(() => useMapViewPreferences('ridgeline')).result.current.backgroundStyle).toBe('rendu')
    window.localStorage.setItem('replay-background-style', 'inconnu')
    expect(renderHook(() => useMapViewPreferences('ridgeline')).result.current.backgroundStyle).toBe('plan')
  })

  it('sans carte connue : pas de rotation', () => {
    window.localStorage.setItem('replay-orientation.', '2')
    expect(renderHook(() => useMapViewPreferences(undefined)).result.current.turns).toBe(0)
  })

  it('le style du fond s’écrit sous la clé du rejeu, et le rejeu (autre lecteur de la clé) le suit', () => {
    const tactique = renderHook(() => useMapViewPreferences('ridgeline'))
    const rejeu = renderHook(() => useBackgroundStyle())
    act(() => tactique.result.current.setBackgroundStyle('rendu'))
    expect(window.localStorage.getItem('replay-background-style')).toBe('rendu')
    expect(rejeu.result.current[0]).toBe('rendu')
    act(() => rejeu.result.current[1]('plan'))
    expect(tactique.result.current.backgroundStyle).toBe('plan')
  })
})

describe('useMapViewPreferences — l’Auto sur la zone de la vue', () => {
  const HAUTE = { largeur: 659, hauteur: 1288 }
  const CARREE = { largeur: 100, hauteur: 100 }
  const LARGE = { largeur: 1000, hauteur: 500 }
  const HAUTE_ZONE = { largeur: 400, hauteur: 900 }

  it('sans zone : le plan reste droit, même sans choix retenu', () => {
    expect(renderHook(() => useMapViewPreferences('ronin')).result.current.turns).toBe(0)
  })

  it('sans choix retenu, une carte haute dans une zone large est tournée', () => {
    const { result } = renderHook(() => useMapViewPreferences('ronin', { carte: HAUTE, boite: LARGE }))
    expect(result.current.turns).toBe(1)
  })

  it('la même carte dans une zone haute reste droite', () => {
    const { result } = renderHook(() => useMapViewPreferences('ronin', { carte: HAUTE, boite: HAUTE_ZONE }))
    expect(result.current.turns).toBe(0)
  })

  it('une carte carrée reste droite, une carte inconnue aussi, une zone pas mesurée aussi', () => {
    expect(renderHook(() => useMapViewPreferences('ronin', { carte: CARREE, boite: LARGE })).result.current.turns).toBe(0)
    expect(renderHook(() => useMapViewPreferences('ronin', { carte: null, boite: LARGE })).result.current.turns).toBe(0)
    expect(
      renderHook(() => useMapViewPreferences('ronin', { carte: HAUTE, boite: { largeur: 0, hauteur: 0 } })).result.current.turns,
    ).toBe(0)
  })

  it('zone large puis zone haute : le plan tourne, puis revient droit', () => {
    const { result, rerender } = renderHook(
      ({ boite }: { boite: { largeur: number; hauteur: number } }) => useMapViewPreferences('ronin', { carte: HAUTE, boite }),
      { initialProps: { boite: LARGE } },
    )
    expect(result.current.turns).toBe(1)
    rerender({ boite: HAUTE_ZONE })
    expect(result.current.turns).toBe(0)
  })

  it('un choix retenu pour la carte prime sur l’Auto, dans les deux sens', () => {
    window.localStorage.setItem('replay-orientation.ronin', '0')
    expect(renderHook(() => useMapViewPreferences('ronin', { carte: HAUTE, boite: LARGE })).result.current.turns).toBe(0)
    window.localStorage.setItem('replay-orientation.ronin', '2')
    expect(renderHook(() => useMapViewPreferences('ronin', { carte: HAUTE, boite: HAUTE_ZONE })).result.current.turns).toBe(2)
  })

  it('un autre choix par carte : l’Auto d’une carte ne déteint pas sur l’autre', () => {
    const { result, rerender } = renderHook(
      ({ mapId }: { mapId: string }) => useMapViewPreferences(mapId, { carte: HAUTE, boite: LARGE }),
      { initialProps: { mapId: 'ronin' } },
    )
    expect(result.current.turns).toBe(1)
    window.localStorage.setItem('replay-orientation.streets', '0')
    rerender({ mapId: 'streets' })
    expect(result.current.turns).toBe(0)
  })

  it('sans carte connue (aucun identifiant) : droit, même avec une zone', () => {
    expect(renderHook(() => useMapViewPreferences(undefined, { carte: HAUTE, boite: LARGE })).result.current.turns).toBe(0)
  })
})
