/**
 * useMatchNeighborsResolved — test de la cascade (state → session → API).
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
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

describe('useMatchNeighborsResolved — contexte sans liste (Accueil : matchs récents)', () => {
  const recentCtx: MatchNavContext = {
    source: 'home_recent',
    matchIds: [],
    contextDescriptor: { kind: 'recent' },
  }

  it('voisins et total viennent de l\'API (tout l\'historique), le libellé vient du descripteur', async () => {
    stateCtxRef.current = recentCtx
    const { result } = renderHook(() => useMatchNeighborsResolved('me', 'm2'), { wrapper })

    expect(result.current.source).toBe('api')
    expect(result.current.contextDescriptor).toEqual({ kind: 'recent' })
    // Rendu pour que prev/next propagent le contexte à la tuile suivante.
    expect(result.current.navContext).toBe(recentCtx)

    await waitFor(() => expect(result.current.data).toBeDefined())
    // Compteur = vrai total de l'API (50), pas la longueur d'une fenêtre.
    expect(result.current.data?.total_matches).toBe(50)
    expect(result.current.data?.current_index).toBe(7)
  })

  it('aucun avertissement de développement (ce n\'est pas un match absent du contexte)', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    stateCtxRef.current = recentCtx
    renderHook(() => useMatchNeighborsResolved('me', 'm2'), { wrapper })
    expect(warnSpy).not.toHaveBeenCalled()
    warnSpy.mockRestore()
  })

  it('survit à un rechargement : le contexte persisté (sessionStorage) est relu', () => {
    persistNavContext('m2', recentCtx)
    stateCtxRef.current = undefined
    const { result } = renderHook(() => useMatchNeighborsResolved('me', 'm2'), { wrapper })
    expect(result.current.contextDescriptor).toEqual({ kind: 'recent' })
    expect(result.current.source).toBe('api')
  })
})

describe('useMatchNeighborsResolved — pas d\'un match au suivant', () => {
  it('garde les voisins du match précédent (placeholder) pendant le chargement du suivant', async () => {
    const { api } = await import('@/lib/api/client')
    let resolveNext: (v: never) => void = () => {}
    stateCtxRef.current = undefined
    const { result, rerender } = renderHook(({ id }) => useMatchNeighborsResolved('me', id), {
      wrapper,
      initialProps: { id: 'm2' },
    })
    await waitFor(() => expect(result.current.data?.current_index).toBe(7))
    expect(result.current.isPlaceholder).toBe(false)

    vi.mocked(api.get).mockImplementationOnce(
      () => new Promise((resolve) => { resolveNext = resolve }) as Promise<never>,
    )
    rerender({ id: 'm3' })
    // En attente du match suivant : les voisins précédents restent (la barre ne disparaît pas),
    // marqués comme tels pour qu'aucun compteur ni cible de pas n'en soit déduit.
    expect(result.current.data?.current_index).toBe(7)
    expect(result.current.isPlaceholder).toBe(true)

    resolveNext({ previous_match_id: 'p9', next_match_id: 'n9', current_index: 8, total_matches: 51 } as never)
    await waitFor(() => expect(result.current.isPlaceholder).toBe(false))
    expect(result.current.data?.current_index).toBe(8)
    expect(result.current.data?.total_matches).toBe(51)
  })
})
