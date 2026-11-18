/**
 * useReplayDimension.test.ts — l'emplacement du commutateur 2D / 3D n'existe que par la capacité de rejeu 3D du
 * serveur : lue en UN endroit (`replay3dCapabilityOf`), défensive, fausse tant que le contrat ne la porte pas.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { useAppShellStore } from '@/stores/appShellStore'
import { testCapabilities } from '../../../test/capabilities'
import { REPLAY_3D_CAPABILITY_KEY, replay3dCapabilityOf, useReplayDimension } from './useReplayDimension'

afterEach(() => {
  act(() => useAppShellStore.setState({ capabilities: null }))
})

describe('replay3dCapabilityOf', () => {
  it('fausse sans bootstrap, sans le champ, ou avec autre chose qu’un booléen vrai', () => {
    expect(replay3dCapabilityOf(null)).toBe(false)
    expect(replay3dCapabilityOf(undefined)).toBe(false)
    expect(replay3dCapabilityOf({})).toBe(false)
    expect(replay3dCapabilityOf(testCapabilities())).toBe(false)
    for (const v of [false, 'true', 1, null, {}]) expect(replay3dCapabilityOf({ [REPLAY_3D_CAPABILITY_KEY]: v })).toBe(false)
  })

  it('vraie pour le champ supposé à true', () => {
    expect(REPLAY_3D_CAPABILITY_KEY).toBe('can_view_replay_3d')
    expect(replay3dCapabilityOf({ [REPLAY_3D_CAPABILITY_KEY]: true })).toBe(true)
  })
})

describe('useReplayDimension', () => {
  it('capacité absente : indisponible, 2D', () => {
    act(() => useAppShellStore.setState({ capabilities: testCapabilities() }))
    const { result } = renderHook(() => useReplayDimension())
    expect(result.current.available).toBe(false)
    expect(result.current.value).toBe('2d')
  })

  it('capacité vraie : disponible, la bascule change la vue, et la capacité suit le store', () => {
    act(() => useAppShellStore.setState({ capabilities: Object.assign(testCapabilities(), { [REPLAY_3D_CAPABILITY_KEY]: true }) }))
    const { result } = renderHook(() => useReplayDimension())
    expect(result.current.available).toBe(true)
    act(() => result.current.onChange('3d'))
    expect(result.current.value).toBe('3d')
    act(() => useAppShellStore.setState({ capabilities: testCapabilities() }))
    expect(result.current.available).toBe(false)
  })
})
