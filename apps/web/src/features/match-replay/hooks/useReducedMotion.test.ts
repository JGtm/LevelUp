/**
 * Tests — la préférence de mouvement réduit lue par le canvas : vraie quand le navigateur la
 * déclare, fausse sinon, interrogée avec la bonne requête.
 */
import { renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { useReducedMotion } from './useReducedMotion'

function mockMedia(matches: boolean) {
  const matchMedia = vi.fn().mockReturnValue({ matches })
  vi.stubGlobal('matchMedia', matchMedia)
  Object.defineProperty(window, 'matchMedia', { value: matchMedia, configurable: true, writable: true })
  return matchMedia
}

afterEach(() => vi.unstubAllGlobals())

describe('useReducedMotion', () => {
  it('vrai quand le navigateur déclare la préférence, avec la bonne requête', () => {
    const mm = mockMedia(true)
    expect(renderHook(() => useReducedMotion()).result.current).toBe(true)
    expect(mm).toHaveBeenCalledWith('(prefers-reduced-motion: reduce)')
  })

  it('faux sinon', () => {
    mockMedia(false)
    expect(renderHook(() => useReducedMotion()).result.current).toBe(false)
  })
})
