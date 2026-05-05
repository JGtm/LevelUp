import { renderHook, act } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useMediaQuery } from './useMediaQuery'

type MockMql = {
  matches: boolean
  addEventListener: ReturnType<typeof vi.fn>
  removeEventListener: ReturnType<typeof vi.fn>
}

describe('useMediaQuery', () => {
  let mockMql: MockMql

  beforeEach(() => {
    mockMql = {
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue(mockMql))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('retourne true si la query matche au démarrage', () => {
    mockMql.matches = true
    const { result } = renderHook(() => useMediaQuery('(max-width: 768px)'))
    expect(result.current).toBe(true)
  })

  it('retourne false si la query ne matche pas au démarrage', () => {
    mockMql.matches = false
    const { result } = renderHook(() => useMediaQuery('(max-width: 768px)'))
    expect(result.current).toBe(false)
  })

  it('se met à jour quand le media query change', () => {
    mockMql.matches = false
    const { result } = renderHook(() => useMediaQuery('(max-width: 768px)'))
    expect(result.current).toBe(false)

    const handler = mockMql.addEventListener.mock.calls[0][1] as (e: MediaQueryListEvent) => void
    act(() => {
      handler({ matches: true } as MediaQueryListEvent)
    })
    expect(result.current).toBe(true)
  })

  it('supprime le listener au unmount', () => {
    const { unmount } = renderHook(() => useMediaQuery('(max-width: 768px)'))
    unmount()
    expect(mockMql.removeEventListener).toHaveBeenCalledWith('change', expect.any(Function))
  })

  it('ajoute un listener change au montage', () => {
    renderHook(() => useMediaQuery('(max-width: 768px)'))
    expect(mockMql.addEventListener).toHaveBeenCalledWith('change', expect.any(Function))
  })
})
