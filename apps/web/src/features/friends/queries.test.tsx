/**
 * Tests — hooks de la liste d'amis d'un joueur.
 *
 * Deux propriétés qui comptent : la requête est scopée PAR JOUEUR (deux slugs =
 * deux entrées de cache, jamais une liste d'instance partagée), et elle ne part
 * pas tant qu'aucun joueur n'est désigné.
 */
import { describe, it, expect } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { usePlayerFriends, useFriendGamertags, useFriendGamertagsState } from './queries'
import { server } from '@/test/setup'
import { queryKeys } from '@/lib/query/keys'

function wrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  }
}

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

describe('usePlayerFriends', () => {
  it('lit la liste du joueur demandé', async () => {
    server.use(
      http.get('/api/v1/players/alice/friends', () =>
        HttpResponse.json({ xuid: 'xuid-alice', gamertags: ['Charlie'], can_edit: true }),
      ),
    )
    const qc = newClient()
    const { result } = renderHook(() => usePlayerFriends('alice'), { wrapper: wrapper(qc) })
    await waitFor(() => expect(result.current.data).toBeTruthy())
    expect(result.current.data?.gamertags).toEqual(['Charlie'])
    expect(result.current.data?.can_edit).toBe(true)
  })

  it('deux joueurs = deux entrées de cache distinctes', async () => {
    server.use(
      http.get('/api/v1/players/alice/friends', () =>
        HttpResponse.json({ xuid: 'xuid-alice', gamertags: ['Charlie'], can_edit: true }),
      ),
      http.get('/api/v1/players/bob/friends', () =>
        HttpResponse.json({ xuid: 'xuid-bob', gamertags: ['Delta', 'Echo'], can_edit: false }),
      ),
    )
    const qc = newClient()
    const alice = renderHook(() => useFriendGamertags('alice'), { wrapper: wrapper(qc) })
    const bob = renderHook(() => useFriendGamertags('bob'), { wrapper: wrapper(qc) })
    await waitFor(() => {
      expect(alice.result.current).toEqual(['Charlie'])
      expect(bob.result.current).toEqual(['Delta', 'Echo'])
    })
    expect(qc.getQueryData(queryKeys.playerFriends('alice'))).toBeTruthy()
    expect(qc.getQueryData(queryKeys.playerFriends('bob'))).toBeTruthy()
  })

  it('slug vide : aucune requête, liste vide', async () => {
    const qc = newClient()
    const { result } = renderHook(() => usePlayerFriends(undefined), { wrapper: wrapper(qc) })
    expect(result.current.fetchStatus).toBe('idle')
    expect(result.current.data).toBeUndefined()
  })

  it('useFriendGamertags rend une liste vide avant la réponse', () => {
    const qc = newClient()
    const { result } = renderHook(() => useFriendGamertags('alice'), { wrapper: wrapper(qc) })
    expect(result.current).toEqual([])
  })
})

// Lot perf L4a (D4.2, 2026-09-23) : l'Escouade doit distinguer « liste vide » (une
// réponse, donc une composition) de « pas encore là » (aucune requête ne doit partir).
describe('useFriendGamertagsState', () => {
  it('pas encore là : ni succès ni échec ; arrivée VIDE : succès, liste vide', async () => {
    server.use(
      http.get('/api/v1/players/alice/friends', () =>
        HttpResponse.json({ xuid: 'xuid-alice', gamertags: [], can_edit: true }),
      ),
    )
    const qc = newClient()
    const { result } = renderHook(() => useFriendGamertagsState('alice'), { wrapper: wrapper(qc) })
    expect(result.current).toEqual({ gamertags: [], isSuccess: false, isError: false })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.gamertags).toEqual([])
    expect(result.current.isError).toBe(false)
  })

  it('échec : isError, liste vide (même valeur que useFriendGamertags)', async () => {
    server.use(
      http.get('/api/v1/players/alice/friends', () => HttpResponse.json({ error: 'x' }, { status: 500 })),
    )
    const qc = newClient()
    const { result } = renderHook(() => useFriendGamertagsState('alice'), { wrapper: wrapper(qc) })
    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.gamertags).toEqual([])
    expect(result.current.isSuccess).toBe(false)
  })
})
