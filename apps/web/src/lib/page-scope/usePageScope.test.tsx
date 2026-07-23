/**
 * usePageScope — test du primitif hybride URL + localStorage.
 *
 * Le routeur TanStack est mocké : `useSearch` lit un `searchRef` mutable et
 * `useNavigate` applique l'updater fonctionnel à ce ref (simulation d'une
 * mise à jour d'URL). On vérifie la décode, le merge de setScope, la purge de
 * reset et la restauration cold-start.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'

import { usePageScope } from './usePageScope'
import { csvToSet, setToCsv } from './serialize'

interface TestApp {
  q: string
  tags: Set<string>
}
interface TestUrl {
  q?: string
  tags?: string
}

const searchRef: { current: Record<string, unknown> } = { current: {} }

const navigateMock = vi.fn((opts: { search?: unknown; replace?: boolean }) => {
  if (typeof opts.search === 'function') {
    searchRef.current = (opts.search as (p: Record<string, unknown>) => Record<string, unknown>)(
      searchRef.current,
    )
  } else if (opts.search && typeof opts.search === 'object') {
    searchRef.current = opts.search as Record<string, unknown>
  }
})

vi.mock('@tanstack/react-router', () => ({
  useSearch: () => searchRef.current,
  useNavigate: () => navigateMock,
}))

const STORAGE_KEY = 'test-page-scope'
const URL_KEYS = ['q', 'tags'] as const

function useTestScope() {
  return usePageScope<TestApp, TestUrl>({
    to: '/settings',
    params: {},
    storageKey: STORAGE_KEY,
    encode: (a) => ({ q: a.q || undefined, tags: setToCsv(a.tags) }),
    decode: (u) => ({ q: u.q ?? '', tags: csvToSet(u.tags) }),
    urlKeys: URL_KEYS,
  })
}

beforeEach(() => {
  searchRef.current = {}
  navigateMock.mockClear()
  localStorage.clear()
})

describe('decode', () => {
  it('expose le scope décodé depuis le search courant', () => {
    searchRef.current = { q: 'halo', tags: 'a,b' }
    const { result } = renderHook(() => useTestScope())
    expect(result.current.scope).toEqual({ q: 'halo', tags: new Set(['a', 'b']) })
  })
})

describe('setScope', () => {
  it('navigate en replace + écrit le miroir localStorage', () => {
    const { result, rerender } = renderHook(() => useTestScope())
    act(() => result.current.setScope({ q: 'x' }))

    const call = navigateMock.mock.calls.at(-1)?.[0]
    expect(call?.replace).toBe(true)
    expect(searchRef.current).toEqual({ q: 'x', tags: undefined })
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}')).toEqual({ q: 'x' })

    rerender()
    expect(result.current.scope.q).toBe('x')
  })

  it('une valeur vidée retire le param de l’URL', () => {
    searchRef.current = { q: 'x' }
    const { result } = renderHook(() => useTestScope())
    act(() => result.current.setScope({ q: '' }))
    expect(searchRef.current.q).toBeUndefined()
  })

  it('merge un patch partiel sans écraser les autres clés', () => {
    searchRef.current = { q: 'x', tags: 'a' }
    const { result } = renderHook(() => useTestScope())
    act(() => result.current.setScope({ tags: new Set(['a', 'b']) }))
    expect(searchRef.current).toEqual({ q: 'x', tags: 'a,b' })
  })
})

describe('reset', () => {
  it('supprime toutes les clés de scope et purge le miroir', () => {
    searchRef.current = { q: 'x', tags: 'a,b' }
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ q: 'x' }))
    const { result } = renderHook(() => useTestScope())
    act(() => result.current.reset())
    expect(searchRef.current).toEqual({})
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })
})

describe('cold-start', () => {
  it('restaure depuis localStorage quand l’URL est vierge', () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ q: 'saved', tags: 'a' }))
    renderHook(() => useTestScope())
    // L'effet de montage a navigué pour réinjecter le scope sauvegardé.
    expect(navigateMock).toHaveBeenCalledTimes(1)
    expect(searchRef.current).toEqual({ q: 'saved', tags: 'a' })
  })

  it('ne restaure PAS si l’URL porte déjà un scope (URL gagne)', () => {
    searchRef.current = { q: 'fromurl' }
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ q: 'saved' }))
    renderHook(() => useTestScope())
    expect(navigateMock).not.toHaveBeenCalled()
    expect(searchRef.current).toEqual({ q: 'fromurl' })
  })

  it('ne restaure PAS si le miroir est vide', () => {
    renderHook(() => useTestScope())
    expect(navigateMock).not.toHaveBeenCalled()
  })
})
