/**
 * Requêtes de l'Escouade (lot perf L4b, 2026-09-23) :
 *  - ANNULATION (découverte des lots L3 et L4a) : les deux hooks transmettent à fetch le
 *    signal de TanStack Query ; quitter la page, ou passer à une autre clé (clic du rail),
 *    abandonne la requête devenue inutile — connexion fermée, contexte Go annulé, le
 *    serveur s'arrête entre deux sections (499) ;
 *  - la lecture légère construit l'URL que le serveur attend ;
 *  - REJEU de la lecture légère (lot perf L9-web, 2026-09-23, revue C) : un 503 « base
 *    occupée » est rejoué UNE fois, après un court délai, avant le repli sur la lourde ;
 *    aucune autre erreur ne l'est.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { server } from '@/test/setup'
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

describe('useCompositionSessions — rejeu (L9-web)', () => {
  const SESSIONS_URL = '/api/v1/players/:playerSlug/pages/teammates/sessions'
  const REPONSE = { composition_sessions: [], latest_composition_session: '' }

  /** Sert `echecs` réponses (statut, ou erreur réseau si null) avant un 200 ; compte les appels. */
  function servir(echecs: (number | null)[]) {
    const compteur = { appels: 0 }
    server.use(
      http.get(SESSIONS_URL, () => {
        const echec = echecs[compteur.appels]
        compteur.appels += 1
        if (echec === undefined) return HttpResponse.json(REPONSE)
        if (echec === null) return HttpResponse.error()
        return HttpResponse.json({ code: 'x', retryable: true }, { status: echec })
      }),
    )
    return compteur
  }

  it('503 (base occupée) puis 200 : rejouée UNE fois, la donnée arrive', async () => {
    const compteur = servir([503])
    const { result } = renderHook(() => useCompositionSessions('p', ['Alice'], true, true), { wrapper: avecClient() })
    await waitFor(() => expect(result.current.isSuccess).toBe(true), { timeout: 3000 })
    expect(compteur.appels).toBe(2)
  })

  it('503 deux fois : pas de second rejeu, échec (repli sur la lourde)', async () => {
    const compteur = servir([503, 503])
    const { result } = renderHook(() => useCompositionSessions('p', ['Alice'], true, true), { wrapper: avecClient() })
    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 3000 })
    expect(compteur.appels).toBe(2)
  })

  it.each([
    ['500', 500],
    ['502', 502],
    ['erreur réseau', null],
  ])('%s : jamais rejouée (échec aussitôt)', async (_nom, echec) => {
    const compteur = servir([echec])
    const { result } = renderHook(() => useCompositionSessions('p', ['Alice'], true, true), { wrapper: avecClient() })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(compteur.appels).toBe(1)
  })
})
