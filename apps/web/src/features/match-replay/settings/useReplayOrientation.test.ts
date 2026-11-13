/**
 * useReplayOrientation.test.ts — l'orientation de la carte est un réglage PAR CARTE : défaut 0,
 * retenue sous `replay-orientation.<mapId>`, jamais partagée entre deux cartes, validée à la
 * lecture, et sans commande tant que la carte n'est pas connue.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { orientationKey, readOrientation } from './replayPreferences'
import { useReplayOrientation } from './useReplayOrientation'

beforeEach(() => {
  window.localStorage.clear()
})

describe('useReplayOrientation', () => {
  it('démarre à 0 quart de tour, commande disponible', () => {
    const { result } = renderHook(() => useReplayOrientation('ridgeline'))
    expect(result.current.turns).toBe(0)
    expect(result.current.available).toBe(true)
  })

  it('retient le choix sous la clé de CETTE carte, et seulement elle', () => {
    const { result } = renderHook(() => useReplayOrientation('ridgeline'))
    act(() => result.current.setTurns(3))
    expect(result.current.turns).toBe(3)
    expect(window.localStorage.getItem('replay-orientation.ridgeline')).toBe('3')
    expect(window.localStorage.getItem(orientationKey('streets'))).toBeNull()
    expect(readOrientation('streets')).toBe(0)
  })

  it('survit au remontage', () => {
    const premier = renderHook(() => useReplayOrientation('ridgeline'))
    act(() => premier.result.current.setTurns(1))
    premier.unmount()
    const second = renderHook(() => useReplayOrientation('ridgeline'))
    expect(second.result.current.turns).toBe(1)
  })

  it('relit le stockage quand la carte change, sans fuite de l’une à l’autre', () => {
    window.localStorage.setItem('replay-orientation.streets', '2')
    const { result, rerender } = renderHook(({ id }) => useReplayOrientation(id), {
      initialProps: { id: 'ridgeline' },
    })
    act(() => result.current.setTurns(1))
    rerender({ id: 'streets' })
    expect(result.current.turns).toBe(2)
    rerender({ id: 'ridgeline' })
    expect(result.current.turns).toBe(1)
  })

  it('ignore une valeur stockée hors des quatre positions', () => {
    for (const bad of ['4', '-1', '1.5', 'abc', '']) {
      window.localStorage.setItem('replay-orientation.ridgeline', bad)
      const { result } = renderHook(() => useReplayOrientation('ridgeline'))
      expect(result.current.turns).toBe(0)
    }
  })

  it('sans carte connue : 0 quart de tour, commande indisponible, rien d’écrit', () => {
    for (const id of [undefined, null, '']) {
      const { result } = renderHook(() => useReplayOrientation(id))
      expect(result.current.available).toBe(false)
      expect(result.current.turns).toBe(0)
      act(() => result.current.setTurns(2))
      expect(result.current.turns).toBe(0)
    }
    expect(window.localStorage.length).toBe(0)
  })
})
