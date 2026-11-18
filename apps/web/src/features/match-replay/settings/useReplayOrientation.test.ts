/**
 * useReplayOrientation.test.ts — l'orientation de la carte est un réglage PAR CARTE à trois états :
 * aucun choix retenu (Auto), ou 0 à 3 quarts de tour. Retenue sous `replay-orientation.<mapId>`, jamais
 * partagée entre deux cartes, validée à la lecture, sans commande tant que la carte n'est pas connue.
 * Une valeur retenue avant l'Auto (« 0 » à « 3 ») est lue telle quelle.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { orientationKey, readOrientation } from './replayPreferences'
import { useReplayOrientation } from './useReplayOrientation'

beforeEach(() => {
  window.localStorage.clear()
})

describe('useReplayOrientation', () => {
  it('sans choix retenu : aucun choix (Auto), commande disponible', () => {
    const { result } = renderHook(() => useReplayOrientation('ridgeline'))
    expect(result.current.retained).toBeNull()
    expect(result.current.available).toBe(true)
  })

  it('retient le choix sous la clé de CETTE carte, et seulement elle', () => {
    const { result } = renderHook(() => useReplayOrientation('ridgeline'))
    act(() => result.current.setTurns(3))
    expect(result.current.retained).toBe(3)
    expect(window.localStorage.getItem('replay-orientation.ridgeline')).toBe('3')
    expect(window.localStorage.getItem(orientationKey('streets'))).toBeNull()
    expect(readOrientation('streets')).toBeNull()
  })

  it('un 0 retenu est un choix : il s’écrit, se relit comme 0 et n’est pas Auto', () => {
    const { result } = renderHook(() => useReplayOrientation('ridgeline'))
    act(() => result.current.setTurns(0))
    expect(result.current.retained).toBe(0)
    expect(window.localStorage.getItem('replay-orientation.ridgeline')).toBe('0')
    expect(readOrientation('ridgeline')).toBe(0)
  })

  it('Auto (null) EFFACE le choix retenu de la carte, et seulement d’elle', () => {
    window.localStorage.setItem('replay-orientation.streets', '2')
    const { result } = renderHook(() => useReplayOrientation('ridgeline'))
    act(() => result.current.setTurns(1))
    act(() => result.current.setTurns(null))
    expect(result.current.retained).toBeNull()
    expect(window.localStorage.getItem('replay-orientation.ridgeline')).toBeNull()
    expect(window.localStorage.getItem('replay-orientation.streets')).toBe('2')
  })

  it('survit au remontage, y compris le retour en Auto', () => {
    const premier = renderHook(() => useReplayOrientation('ridgeline'))
    act(() => premier.result.current.setTurns(1))
    premier.unmount()
    const second = renderHook(() => useReplayOrientation('ridgeline'))
    expect(second.result.current.retained).toBe(1)
    act(() => second.result.current.setTurns(null))
    second.unmount()
    expect(renderHook(() => useReplayOrientation('ridgeline')).result.current.retained).toBeNull()
  })

  it('une valeur retenue à l’ancien format (« 0 » à « 3 ») est gardée telle quelle, jamais réinterprétée', () => {
    for (const turns of [0, 1, 2, 3] as const) {
      window.localStorage.setItem('replay-orientation.ridgeline', String(turns))
      const { result } = renderHook(() => useReplayOrientation('ridgeline'))
      expect(result.current.retained).toBe(turns)
      expect(window.localStorage.getItem('replay-orientation.ridgeline')).toBe(String(turns))
    }
  })

  it('relit le stockage quand la carte change, sans fuite de l’une à l’autre', () => {
    window.localStorage.setItem('replay-orientation.streets', '2')
    const { result, rerender } = renderHook(({ id }) => useReplayOrientation(id), {
      initialProps: { id: 'ridgeline' },
    })
    act(() => result.current.setTurns(1))
    rerender({ id: 'streets' })
    expect(result.current.retained).toBe(2)
    rerender({ id: 'ridgeline' })
    expect(result.current.retained).toBe(1)
  })

  it('ignore une valeur stockée hors des quatre positions : aucun choix', () => {
    for (const bad of ['4', '-1', '1.5', 'abc', '']) {
      window.localStorage.setItem('replay-orientation.ridgeline', bad)
      const { result } = renderHook(() => useReplayOrientation('ridgeline'))
      expect(result.current.retained).toBeNull()
    }
  })

  it('sans carte connue : plan droit (0), commande indisponible, rien d’écrit', () => {
    for (const id of [undefined, null, '']) {
      const { result } = renderHook(() => useReplayOrientation(id))
      expect(result.current.available).toBe(false)
      expect(result.current.retained).toBe(0)
      act(() => result.current.setTurns(2))
      expect(result.current.retained).toBe(0)
      act(() => result.current.setTurns(null))
      expect(result.current.retained).toBe(0)
    }
    expect(window.localStorage.length).toBe(0)
  })
})
