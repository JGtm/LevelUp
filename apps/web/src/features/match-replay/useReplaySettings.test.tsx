/**
 * useReplaySettings.test.tsx — les calques et la vitesse survivent à la page (même règle
 * de persistance que le son), et démarrent sur les valeurs par défaut d'aujourd'hui quand
 * rien n'a encore été choisi.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SPEED_MULTIPLIERS, useReplaySettings } from './useReplaySettings'

describe('useReplaySettings — valeurs par défaut', () => {
  it("visée, zones et noms allumés, vitesse à 1x — comportement inchangé sans préférence stockée", () => {
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(true)
    expect(result.current.showZones).toBe(true)
    expect(result.current.showNames).toBe(true)
    expect(result.current.speed).toBe(1)
  })
})

describe('useReplaySettings — bascules', () => {
  it('toggleAim et toggleZones inversent leur propre calque, jamais l autre', () => {
    const { result } = renderHook(() => useReplaySettings())
    act(() => result.current.toggleAim())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(true)
    act(() => result.current.toggleZones())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(false)
  })

  it('setSpeed accepte un multiplicateur de la liste proposée', () => {
    const { result } = renderHook(() => useReplaySettings())
    act(() => result.current.setSpeed(4))
    expect(result.current.speed).toBe(4)
    expect(SPEED_MULTIPLIERS).toContain(4)
  })
})

describe('useReplaySettings — préférences persistées (localStorage, comme le son)', () => {
  it('survivent au remontage', () => {
    const first = renderHook(() => useReplaySettings())
    act(() => first.result.current.toggleAim())
    act(() => first.result.current.toggleZones())
    act(() => first.result.current.toggleNames())
    act(() => first.result.current.setSpeed(2))
    first.unmount()

    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.showAim).toBe(false)
    expect(result.current.showZones).toBe(false)
    expect(result.current.showNames).toBe(false)
    expect(result.current.speed).toBe(2)
  })

  it('une vitesse hors liste stockée par un autre moyen retombe sur 1x, jamais une valeur inventée', () => {
    localStorage.setItem('replay-speed', '3')
    const { result } = renderHook(() => useReplaySettings())
    expect(result.current.speed).toBe(1)
  })
})
