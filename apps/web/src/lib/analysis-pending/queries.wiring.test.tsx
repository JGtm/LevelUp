/**
 * Câblage — les requêtes de l'accueil et de la vue match se rechargent toutes les 60 s tant
 * qu'une analyse de film est en cours, puis s'arrêtent. La règle vit dans
 * `analysisPending.logic.ts` (testée à part) ; ici on prouve que les VRAIS hooks l'appliquent
 * à leurs options de requête, avec la fraîcheur raccourcie.
 */
import type { ReactNode } from 'react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider, type Query } from '@tanstack/react-query'

const hoisted = vi.hoisted(() => ({ payload: undefined as unknown }))

vi.mock('@/lib/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/client')>()),
  api: { get: vi.fn(async () => hoisted.payload) },
}))

import { useHomePage } from '@/features/home/queries'
import { useMatchView } from '@/features/match-view/queries'
import { ANALYSIS_PENDING_REFETCH_MS } from './analysisPending.logic'

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, wrapper }
}

type Opts = {
  refetchInterval: (q: Query) => number | false
  staleTime: (q: Query) => number
}

function optionsDe(client: QueryClient): { query: Query; options: Opts } {
  const query = client.getQueryCache().getAll()[0] as Query
  return { query, options: query.options as unknown as Opts }
}

const home = (pending: boolean) => ({
  hero: { kpis: { total_matches: 12 } },
  recent_matches: [{ match_id: 'm1', analysis_pending: pending }],
  favorite_matches: [],
})

describe('useHomePage — rechargement tant qu\'une analyse est en cours', () => {
  beforeEach(() => {
    hoisted.payload = undefined
  })

  it('60 s et fraîcheur courte tant qu\'une tuile est en cours', async () => {
    hoisted.payload = home(true)
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useHomePage('chief'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const { query, options } = optionsDe(client)
    expect(options.refetchInterval(query)).toBe(ANALYSIS_PENDING_REFETCH_MS)
    expect(options.staleTime(query)).toBe(ANALYSIS_PENDING_REFETCH_MS)
  })

  it('s\'arrête quand plus rien n\'est en cours, fraîcheur normale (5 min)', async () => {
    hoisted.payload = home(false)
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useHomePage('chief'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const { query, options } = optionsDe(client)
    expect(options.refetchInterval(query)).toBe(false)
    expect(options.staleTime(query)).toBe(5 * 60 * 1000)
  })
})

describe('useMatchView — rechargement tant que l\'analyse est en cours', () => {
  const vue = (pending: boolean) => ({ header: { analysis_pending: pending }, is_partial: false })

  it('60 s et vue jamais figée par les 10 min tant que l\'analyse est en cours', async () => {
    hoisted.payload = vue(true)
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useMatchView('chief', 'm1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const { query, options } = optionsDe(client)
    expect(options.refetchInterval(query)).toBe(ANALYSIS_PENDING_REFETCH_MS)
    expect(options.staleTime(query)).toBe(0)
  })

  it('30 s tant qu\'une source est en panne (« Nouvel essai automatique » est vrai), vue jamais conservée', async () => {
    hoisted.payload = { header: {}, is_partial: true, partial_reasons: ['scoreboard_unavailable'] }
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useMatchView('chief', 'm1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const { query, options } = optionsDe(client)
    expect(options.refetchInterval(query)).toBe(30_000)
    expect(options.staleTime(query)).toBe(0)
  })

  it('puis s\'arrête, fraîcheur normale (10 min)', async () => {
    hoisted.payload = vue(false)
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useMatchView('chief', 'm1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const { query, options } = optionsDe(client)
    expect(options.refetchInterval(query)).toBe(false)
    expect(options.staleTime(query)).toBe(10 * 60 * 1000)
  })
})
