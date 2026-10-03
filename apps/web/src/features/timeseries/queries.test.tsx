/**
 * Annulation d'une page lourde (plan perf 2026-09-23, D3.4) : le hook transmet à fetch le
 * signal de TanStack Query ; quitter la page pendant le calcul abandonne la requête
 * (connexion fermée → contexte Go annulé → le serveur arrête de calculer, 499).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { createTestQueryClient } from '@/test/render-utils'
import type { TimeseriesQueryRequest } from '@/lib/api/types'
import { useTimeseriesPage } from './queries'

describe('useTimeseriesPage — annulation', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('démonter la page pendant le calcul abandonne la requête', async () => {
    let signal: AbortSignal | null | undefined
    vi.spyOn(globalThis, 'fetch').mockImplementation((_url: unknown, init?: RequestInit) => {
      signal = init?.signal
      return new Promise<Response>(() => {}) // calcul serveur encore en cours
    })
    const client = createTestQueryClient()
    function Wrapper({ children }: { children: ReactNode }) {
      return <QueryClientProvider client={client}>{children}</QueryClientProvider>
    }
    const request = {} as TimeseriesQueryRequest
    const { unmount } = renderHook(() => useTimeseriesPage('p', request, 'hash'), {
      wrapper: Wrapper,
    })

    await waitFor(() => expect(signal).toBeInstanceOf(AbortSignal))
    expect(signal?.aborted).toBe(false)
    unmount()
    expect(signal?.aborted).toBe(true)
  })
})
