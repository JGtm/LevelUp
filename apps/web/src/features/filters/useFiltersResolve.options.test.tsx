/**
 * Tests — options de `useFiltersResolve` / `useFiltersPreview` (lot perf L4a,
 * 2026-09-23, D4.3 et D4.4) : `matchContext` entre dans le corps ET dans la clé
 * (un résolu escouade n'est jamais servi au solo, ni l'inverse) ; `enabled: false`
 * n'envoie rien.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { server } from '@/test/setup'
import { queryKeys } from '@/lib/query/keys'
import { createFilterStore } from '@/stores/createFilterStore'
import type { FilterContextInput } from '@/lib/api/types'
import { useFiltersPreview, useFiltersResolve } from './queries'

const corps: FilterContextInput[] = []
let n = 0

beforeEach(() => {
  corps.length = 0
  server.use(
    http.post('/api/v1/players/:playerSlug/filters/resolve', async ({ request }) => {
      corps.push((await request.json()) as FilterContextInput)
      return HttpResponse.json({ available_options: { experience_types: [], playlists: [], modes: [], maps: [] } })
    }),
  )
})

function client() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
  return { qc, wrapper }
}

async function laisserRetomber() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 80))
  })
}

describe('useFiltersResolve — matchContext et enabled', () => {
  it('matchContext squad : dans le corps ET dans la clé ; le filterContext du store est inchangé', async () => {
    const store = createFilterStore({ name: `levelup-test-resolve-${++n}` })
    const { qc, wrapper } = client()
    renderHook(() => useFiltersResolve('p', store, { matchContext: 'squad' }), { wrapper })
    await waitFor(() => expect(corps).toHaveLength(1))

    expect(corps[0].match_context).toBe('squad')
    expect(store.getState().filterContext.match_context).toBeUndefined()
    const hash = store.getState().filterContextHash
    expect(qc.getQueryData(queryKeys.filtersResolve('p', 'halo_infinite', hash, 'squad'))).toBeTruthy()
    expect(qc.getQueryData(queryKeys.filtersResolve('p', 'halo_infinite', hash, 'all'))).toBeUndefined()
  })

  it('sans matchContext (solo) : corps tel quel, clé « all » — distincte de celle de l escouade', async () => {
    const store = createFilterStore({ name: `levelup-test-resolve-${++n}` })
    const { qc, wrapper } = client()
    renderHook(() => useFiltersResolve('p', store), { wrapper })
    await waitFor(() => expect(corps).toHaveLength(1))

    expect('match_context' in corps[0]).toBe(false)
    const hash = store.getState().filterContextHash
    expect(qc.getQueryData(queryKeys.filtersResolve('p', 'halo_infinite', hash, 'all'))).toBeTruthy()
    expect(store.getState().resolvedContext).toBeTruthy()
  })

  it('enabled false : aucune requête, rien dans le store', async () => {
    const store = createFilterStore({ name: `levelup-test-resolve-${++n}` })
    const { wrapper } = client()
    renderHook(() => useFiltersResolve('p', store, { enabled: false }), { wrapper })
    await laisserRetomber()
    expect(corps).toHaveLength(0)
    expect(store.getState().resolvedContext).toBeNull()
  })
})

describe('useFiltersPreview — enabled', () => {
  const input: FilterContextInput = { filter_mode: 'period', match_context: 'squad' }

  it('enabled false : aucune requête', async () => {
    const { wrapper } = client()
    const { result } = renderHook(() => useFiltersPreview('p', input, { enabled: false }), { wrapper })
    await laisserRetomber()
    expect(corps).toHaveLength(0)
    expect(result.current.fetchStatus).toBe('idle')
  })

  it('par défaut : la requête part (comportement du solo inchangé)', async () => {
    const { wrapper } = client()
    renderHook(() => useFiltersPreview('p', input), { wrapper })
    await waitFor(() => expect(corps).toHaveLength(1))
  })
})
