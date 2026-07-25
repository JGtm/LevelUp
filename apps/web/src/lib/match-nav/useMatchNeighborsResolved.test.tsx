/**
 * useMatchNeighborsResolved — test de la cascade (state → session → API).
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { useMatchNeighborsResolved } from './useMatchNeighborsResolved'
import { persistNavContext, type MatchNavContext } from './navContext'

const stateCtxRef: { current: MatchNavContext | undefined } = { current: undefined }

vi.mock('@tanstack/react-router', () => ({
  useRouterState: ({
    select,
  }: {
    select: (s: { location: { state: { matchNavContext?: MatchNavContext } } }) => unknown
  }) =>
    select({ location: { state: { matchNavContext: stateCtxRef.current } } }),
}))

vi.mock('@/lib/api/client', () => ({
  api: {
    get: vi.fn(async () => ({
      previous_match_id: 'api-prev',
      next_match_id: 'api-next',
      current_index: 7,
      total_matches: 50,
    })),
  },
  // Le hook lit désormais le titre courant (useAppShellStore -> soloFilterStore ->
  // getApiTitleSlug) : le mock doit exposer ces exports sinon l'import du store échoue.
  getApiTitleSlug: () => 'halo_infinite',
  setApiTitleSlug: vi.fn(),
  setApiLocale: vi.fn(),
}))

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

const histCtx: MatchNavContext = {
  source: 'history',
  matchIds: ['m1', 'm2', 'm3', 'm4'],
  filtersLabel: 'Classée · 7 derniers jours',
}

beforeEach(() => {
  sessionStorage.clear()
  stateCtxRef.current = undefined
})

describe('useMatchNeighborsResolved', () => {
  it('priorité 1 — router state : retourne neighbors locaux + source=router-state', () => {
    stateCtxRef.current = histCtx
    const { result } = renderHook(() => useMatchNeighborsResolved('me', 'm2'), { wrapper })

    expect(result.current.source).toBe('router-state')
    expect(result.current.data).toEqual({
      previous_match_id: 'm3',
      next_match_id: 'm1',
      current_index: 1,
      total_matches: 4,
    })
    expect(result.current.contextLabel).toBe('Classée · 7 derniers jours')
    expect(result.current.navContext).toBe(histCtx)
  })

  it('priorité 2 — sessionStorage : si state vide, lit le storage + source=session-storage', () => {
    persistNavContext('m2', histCtx)
    stateCtxRef.current = undefined
    const { result } = renderHook(() => useMatchNeighborsResolved('me', 'm2'), { wrapper })

    expect(result.current.source).toBe('session-storage')
    expect(result.current.data?.previous_match_id).toBe('m3')
    expect(result.current.data?.next_match_id).toBe('m1')
    expect(result.current.contextLabel).toBe('Classée · 7 derniers jours')
  })

  it('priorité 3 — fallback API : ni state ni session → source=api', async () => {
    stateCtxRef.current = undefined
    const { result, rerender } = renderHook(
      () => useMatchNeighborsResolved('me', 'm2'),
      { wrapper },
    )

    // Premier render : isPending = true, data = undefined
    expect(result.current.source).toBe('api')

    // Attendre la résolution de la query (microtask + re-render)
    await new Promise((r) => setTimeout(r, 0))
    rerender()

    expect(result.current.source).toBe('api')
    // contextLabel non défini en mode API
    expect(result.current.contextLabel).toBeUndefined()
  })

  it('matchId hors liste : ignore le ctx, warn dev + tape l\'API', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    stateCtxRef.current = histCtx
    const { result } = renderHook(() => useMatchNeighborsResolved('me', 'inconnu'), {
      wrapper,
    })

    // Le matchId 'inconnu' n'est pas dans matchIds → fallback API
    expect(result.current.source).toBe('api')
    // Observabilité (Phase 3) : le fallback est signalé en dev (avant : silencieux).
    expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('absent du contexte'))
    warnSpy.mockRestore()
  })

  it('expose contextDescriptor depuis le contexte router-state', () => {
    stateCtxRef.current = {
      ...histCtx,
      contextDescriptor: { kind: 'with_player', gamertag: 'CoolMate' },
    }
    const { result } = renderHook(() => useMatchNeighborsResolved('me', 'm2'), { wrapper })

    expect(result.current.contextDescriptor).toEqual({
      kind: 'with_player',
      gamertag: 'CoolMate',
    })
  })
})
