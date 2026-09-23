/**
 * Requêtes de l'Escouade (lot perf L4b, 2026-09-23) :
 *  - ANNULATION (découverte des lots L3 et L4a) : les deux hooks transmettent à fetch le
 *    signal de TanStack Query ; quitter la page, ou passer à une autre clé (clic du rail),
 *    abandonne la requête devenue inutile — connexion fermée, contexte Go annulé, le
 *    serveur s'arrête entre deux sections (499) ;
 *  - la lecture légère construit l'URL que le serveur attend.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { createTestQueryClient } from '@/test/render-utils'
import type { TeammatesQueryRequest } from '@/lib/api/types'
import { useCompositionSessions, useTeammates } from './queries'

interface Appel {
  url: string
  signal?: AbortSignal | null
}

/** fetch qui ne répond jamais (calcul serveur en cours) et retient chaque appel. */
function fetchEnCours(): Appel[] {
  const appels: Appel[] = []
  vi.spyOn(globalThis, 'fetch').mockImplementation((url: unknown, init?: RequestInit) => {
    appels.push({ url: String(url), signal: init?.signal })
    return new Promise<Response>(() => {})
  })
  return appels
}

function avecClient() {
  const client = createTestQueryClient()
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

const REQUETE = {} as TeammatesQueryRequest

describe('useTeammates — annulation de la requête lourde', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('démonter la page pendant le calcul abandonne la requête', async () => {
    const appels = fetchEnCours()
    const { unmount } = renderHook(() => useTeammates('p', REQUETE, 'h', ['Alice'], true), {
      wrapper: avecClient(),
    })
    await waitFor(() => expect(appels).toHaveLength(1))
    expect(appels[0].signal).toBeInstanceOf(AbortSignal)
    expect(appels[0].signal?.aborted).toBe(false)
    unmount()
    expect(appels[0].signal?.aborted).toBe(true)
  })

  it('une nouvelle clé (clic du rail) abandonne la requête devenue inutile', async () => {
    const appels = fetchEnCours()
    const { rerender } = renderHook(
      ({ hash }: { hash: string }) => useTeammates('p', REQUETE, hash, ['Alice'], true),
      { wrapper: avecClient(), initialProps: { hash: 'session-S2' } },
    )
    await waitFor(() => expect(appels).toHaveLength(1))
    rerender({ hash: 'session-S1' })
    await waitFor(() => expect(appels).toHaveLength(2))
    expect(appels[0].signal?.aborted).toBe(true)
    expect(appels[1].signal?.aborted).toBe(false)
  })
})

describe('useCompositionSessions — la lecture légère', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('URL ?teammates=a,b&exact=… et signal transmis (démontage = abandon)', async () => {
    const appels = fetchEnCours()
    const { unmount } = renderHook(() => useCompositionSessions('p', ['Alice', 'Bob'], true, true), {
      wrapper: avecClient(),
    })
    await waitFor(() => expect(appels).toHaveLength(1))
    const url = new URL(appels[0].url, 'http://localhost')
    expect(url.pathname).toBe('/api/v1/players/p/pages/teammates/sessions')
    expect(url.searchParams.get('teammates')).toBe('Alice,Bob')
    expect(url.searchParams.get('exact')).toBe('true')
    unmount()
    expect(appels[0].signal?.aborted).toBe(true)
  })

  it('sans coéquipier : aucun paramètre teammates, option transmise telle quelle', async () => {
    const appels = fetchEnCours()
    renderHook(() => useCompositionSessions('p', [], false, true), { wrapper: avecClient() })
    await waitFor(() => expect(appels).toHaveLength(1))
    const url = new URL(appels[0].url, 'http://localhost')
    expect(url.searchParams.has('teammates')).toBe(false)
    expect(url.searchParams.get('exact')).toBe('false')
  })

  it('désactivée (composition initiale inconnue) : aucune requête', async () => {
    const appels = fetchEnCours()
    renderHook(() => useCompositionSessions('p', ['Alice'], true, false), { wrapper: avecClient() })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(appels).toHaveLength(0)
  })
})
