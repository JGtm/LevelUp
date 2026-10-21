/**
 * Câblage — les tuiles de l'Accueil : la tête arrive avec la page, les pages suivantes se
 * demandent une à une (curseur de la page précédente), chacune est SA requête (rechargement de 60 s
 * propre à la page qui porte une analyse en cours), les favoris ont leur lecture.
 */
import type { ReactNode } from 'react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider, type Query } from '@tanstack/react-query'
import type { HomeMatchesPage, RecentMatchItem } from '@/lib/api/types'

const hoisted = vi.hoisted(() => ({
  pages: {} as Record<string, unknown>,
  calls: [] as string[],
}))

vi.mock('@/lib/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/client')>()),
  api: {
    get: vi.fn(async (url: string) => {
      hoisted.calls.push(url)
      const key = url.includes('kind=favorites')
        ? 'favorites'
        : (new URL(url, 'http://x').searchParams.get('before') ?? '')
      return hoisted.pages[key]
    }),
  },
}))

import { api } from '@/lib/api/client'
import { ANALYSIS_PENDING_REFETCH_MS } from '@/lib/analysis-pending/analysisPending.logic'
import { useHomeFavorites, useHomeTiles } from './queries'

const tile = (id: string, pending = false): RecentMatchItem =>
  ({ match_id: id, analysis_pending: pending }) as RecentMatchItem
const page = (ids: string[], next?: string, pending = false): HomeMatchesPage => ({
  items: ids.map((id) => tile(id, pending)),
  has_more: !!next,
  next_before: next,
})

type Opts = { refetchInterval: (q: Query) => number | false }

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, wrapper }
}

const head = (next?: string) => ({
  recent_matches: [tile('h1'), tile('h2')],
  recent_matches_has_more: !!next,
  recent_matches_next_before: next,
})

beforeEach(() => {
  hoisted.pages = {}
  hoisted.calls = []
  vi.mocked(api.get).mockClear()
})

describe('useHomeTiles', () => {
  it('au montage : la tête seule, aucune requête de page', () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useHomeTiles('chief', head('c0')), { wrapper })
    expect(result.current.tiles.map((t) => t.match_id)).toEqual(['h1', 'h2'])
    expect(result.current.hasMore).toBe(true)
    expect(hoisted.calls).toEqual([])
  })

  it("fetchMore : demande la page du curseur de la tête, puis celle du curseur de la page", async () => {
    hoisted.pages = { c0: page(['p1a', 'p1b'], 'c1'), c1: page(['p2a']) }
    const { wrapper } = setup()
    const { result } = renderHook(() => useHomeTiles('chief', head('c0')), { wrapper })

    act(() => result.current.fetchMore())
    await waitFor(() => expect(result.current.tiles).toHaveLength(4))
    expect(hoisted.calls).toHaveLength(1)
    expect(hoisted.calls[0]).toContain('before=c0')
    expect(result.current.hasMore).toBe(true)

    act(() => result.current.fetchMore())
    await waitFor(() =>
      expect(result.current.tiles.map((t) => t.match_id)).toEqual(['h1', 'h2', 'p1a', 'p1b', 'p2a']),
    )
    expect(hoisted.calls[1]).toContain('before=c1')
    expect(result.current.hasMore).toBe(false)

    act(() => result.current.fetchMore())
    expect(hoisted.calls).toHaveLength(2)
  })

  it("fetchMore est idempotent : appelé plusieurs fois avant l'arrivée, une seule requête", async () => {
    hoisted.pages = { c0: page(['p1'], 'c1') }
    const { wrapper } = setup()
    const { result } = renderHook(() => useHomeTiles('chief', head('c0')), { wrapper })
    act(() => {
      result.current.fetchMore()
      result.current.fetchMore()
      result.current.fetchMore()
    })
    await waitFor(() => expect(result.current.tiles).toHaveLength(3))
    expect(hoisted.calls).toHaveLength(1)
  })

  it('sans suite annoncée : fetchMore ne demande rien', () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useHomeTiles('chief', head(undefined)), { wrapper })
    act(() => result.current.fetchMore())
    expect(hoisted.calls).toEqual([])
    expect(result.current.hasMore).toBe(false)
  })

  it('une tête qui change de curseur (synchro) repart de la tête : les anciennes pages ne se recollent pas', async () => {
    hoisted.pages = { c0: page(['p1'], 'c1') }
    const { wrapper } = setup()
    const { result, rerender } = renderHook(({ h }) => useHomeTiles('chief', h), {
      wrapper,
      initialProps: { h: head('c0') },
    })
    act(() => result.current.fetchMore())
    await waitFor(() => expect(result.current.tiles).toHaveLength(3))

    rerender({ h: head('c0-nouveau') })
    expect(result.current.tiles.map((t) => t.match_id)).toEqual(['h1', 'h2'])
    expect(result.current.nextBefore).toBe('c0-nouveau')
  })

  it("chaque page a SON rechargement : 60 s seulement pour celle qui porte une analyse en cours", async () => {
    hoisted.pages = { c0: page(['p1'], 'c1', false), c1: page(['p2'], undefined, true) }
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useHomeTiles('chief', head('c0')), { wrapper })
    act(() => result.current.fetchMore())
    await waitFor(() => expect(result.current.tiles).toHaveLength(3))
    act(() => result.current.fetchMore())
    await waitFor(() => expect(result.current.tiles).toHaveLength(4))

    const queries = client.getQueryCache().getAll()
    expect(queries).toHaveLength(2)
    const intervals = Object.fromEntries(
      queries.map((q) => [String(q.queryKey[5]), (q.options as unknown as Opts).refetchInterval(q)]),
    )
    expect(intervals).toEqual({ c0: false, c1: ANALYSIS_PENDING_REFETCH_MS })
  })

  it("une page en échec : loadError, pas de nouvelle demande tant qu'on n'a pas réessayé", async () => {
    const { wrapper } = setup()
    vi.mocked(api.get).mockRejectedValueOnce(new Error('boom'))
    const { result } = renderHook(() => useHomeTiles('chief', head('c0')), { wrapper })
    act(() => result.current.fetchMore())
    await waitFor(() => expect(result.current.loadError).toBe(true))
    act(() => result.current.fetchMore())
    expect(vi.mocked(api.get)).toHaveBeenCalledTimes(1)
    hoisted.pages = { c0: page(['p1']) }
    act(() => result.current.retry())
    await waitFor(() => expect(result.current.tiles).toHaveLength(3))
    expect(result.current.loadError).toBe(false)
  })
})

describe('useHomeFavorites', () => {
  it("lit kind=favorites et se recharge toutes les 60 s tant qu'un favori est en cours d'analyse", async () => {
    hoisted.pages = { favorites: page(['f1'], undefined, true) }
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useHomeFavorites('chief'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(hoisted.calls[0]).toContain('kind=favorites')
    const q = client.getQueryCache().getAll()[0] as Query
    expect((q.options as unknown as Opts).refetchInterval(q)).toBe(ANALYSIS_PENDING_REFETCH_MS)
  })
})
