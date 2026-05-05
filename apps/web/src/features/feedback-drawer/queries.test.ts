import { describe, expect, it, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { createElement } from 'react'
import { useSimilarIssues } from './queries'
import { log } from './_logger'

let originalFetch: typeof window.fetch

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client, children })
}

beforeEach(() => {
  originalFetch = window.fetch
  log._resetForTests()
})

afterEach(() => {
  window.fetch = originalFetch
  vi.restoreAllMocks()
})

describe('useSimilarIssues — sécurité', () => {
  it('appelle fetch avec credentials: omit (pas de cookie LevelUp envoyé à GitHub)', async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [], total_count: 0 }), { status: 200 }),
    )
    window.fetch = fetchSpy as typeof window.fetch
    const { result } = renderHook(() => useSimilarIssues('crash bug', true), {
      wrapper: makeWrapper(),
    })
    await waitFor(() => expect(fetchSpy).toHaveBeenCalled(), { timeout: 1500 })
    const init = fetchSpy.mock.calls[0]?.[1] as RequestInit
    expect(init.credentials).toBe('omit')
    expect(result.current).toBeDefined()
  })

  it("n'envoie aucun header X-LevelUp-Title", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [], total_count: 0 }), { status: 200 }),
    )
    window.fetch = fetchSpy as typeof window.fetch
    renderHook(() => useSimilarIssues('feature', true), { wrapper: makeWrapper() })
    await waitFor(() => expect(fetchSpy).toHaveBeenCalled(), { timeout: 1500 })
    const init = fetchSpy.mock.calls[0]?.[1] as RequestInit
    const headers = init.headers as Record<string, string>
    expect(headers['X-LevelUp-Title']).toBeUndefined()
    // Cas-insensitif
    const keys = Object.keys(headers).map((k) => k.toLowerCase())
    expect(keys).not.toContain('x-levelup-title')
  })

  it("le titre avec opérateurs réservés n'apparait pas non-escaped dans l'URL", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [], total_count: 0 }), { status: 200 }),
    )
    window.fetch = fetchSpy as typeof window.fetch
    renderHook(() => useSimilarIssues('crash: TypeError "boom"', true), {
      wrapper: makeWrapper(),
    })
    await waitFor(() => expect(fetchSpy).toHaveBeenCalled(), { timeout: 1500 })
    const url = fetchSpy.mock.calls[0]?.[0] as string
    const decoded = decodeURIComponent(new URL(url).searchParams.get('q') ?? '')
    expect(decoded).not.toContain('"')
    // is:issue et repo: sont ajoutés APRÈS escape, donc autorisés
    expect(decoded).toContain('is:issue')
    expect(decoded).toContain('repo:JGtm/LevelUp')
  })
})

describe('useSimilarIssues — gating', () => {
  it("ne fire pas si enabled=false", async () => {
    const fetchSpy = vi.fn()
    window.fetch = fetchSpy as typeof window.fetch
    renderHook(() => useSimilarIssues('crash', false), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 700))
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it("ne fire pas si titre < 3 chars", async () => {
    const fetchSpy = vi.fn()
    window.fetch = fetchSpy as typeof window.fetch
    renderHook(() => useSimilarIssues('ab', true), { wrapper: makeWrapper() })
    await new Promise((r) => setTimeout(r, 700))
    expect(fetchSpy).not.toHaveBeenCalled()
  })
})

describe('useSimilarIssues — error path', () => {
  it("retourne [] si fetch throw", async () => {
    window.fetch = vi.fn().mockRejectedValue(new Error('network'))
    const { result } = renderHook(() => useSimilarIssues('crash', true), {
      wrapper: makeWrapper(),
    })
    await waitFor(() => expect(result.current.data).toEqual([]), { timeout: 1500 })
  })

  it("retourne [] si HTTP 403 (rate-limited)", async () => {
    window.fetch = vi.fn().mockResolvedValue(new Response('', { status: 403 }))
    const { result } = renderHook(() => useSimilarIssues('crash', true), {
      wrapper: makeWrapper(),
    })
    await waitFor(() => expect(result.current.data).toEqual([]), { timeout: 1500 })
  })

  it("mappe les items vers SimilarIssueRef", async () => {
    window.fetch = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          items: [
            { number: 1, title: 'A', html_url: 'https://x/1', state: 'open' },
            { number: 2, title: 'B', html_url: 'https://x/2', state: 'open' },
          ],
          total_count: 2,
        }),
        { status: 200 },
      ),
    )
    const { result } = renderHook(() => useSimilarIssues('crash', true), {
      wrapper: makeWrapper(),
    })
    await waitFor(() => expect(result.current.data).toHaveLength(2), { timeout: 1500 })
    expect(result.current.data?.[0]).toEqual({
      number: 1,
      title: 'A',
      url: 'https://x/1',
    })
  })
})
