/**
 * queries.test.ts — mutation d'exclusion d'un match.
 *
 * Un match exclu sort du graphe de carrière et des composantes LUSR du profil : la mutation
 * doit invalider ces préfixes en plus de l'historique et de la vue du match, sans quoi ces
 * pages restent périmées jusqu'à l'expiration de leur cache.
 */
import { describe, expect, it, vi, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { createElement } from 'react'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useSetMatchExclusion } from './queries'

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
}

function wrapperFor(client: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client, children })
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useSetMatchExclusion', () => {
  it('envoie PATCH avec le drapeau excluded', async () => {
    const patch = vi.spyOn(api, 'patch').mockResolvedValue(undefined)
    const { result } = renderHook(() => useSetMatchExclusion('JGtm'), {
      wrapper: wrapperFor(makeClient()),
    })
    result.current.mutate({ matchId: 'm1', excluded: true })

    await waitFor(() => expect(result.current.isSuccess).toBe(true), { timeout: 2000 })
    expect(patch).toHaveBeenCalledWith('/players/JGtm/matches/m1/exclusion', { excluded: true })
  })

  it('invalide historique, carrière et profil du joueur après un succès', async () => {
    vi.spyOn(api, 'patch').mockResolvedValue(undefined)
    const client = makeClient()
    const invalidate = vi.spyOn(client, 'invalidateQueries')

    const { result } = renderHook(() => useSetMatchExclusion('JGtm'), {
      wrapper: wrapperFor(client),
    })
    result.current.mutate({ matchId: 'm1', excluded: true })

    await waitFor(() => expect(result.current.isSuccess).toBe(true), { timeout: 2000 })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.matchHistoryAll('JGtm') })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.careerAll('JGtm') })
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: queryKeys.playerProfile.profileAll('JGtm'),
    })
  })

  it('le préfixe du profil couvre les clés de profil de toute langue et fenêtre', () => {
    const prefix = queryKeys.playerProfile.profileAll('JGtm')
    const key = queryKeys.playerProfile.profile('JGtm', 'halo_infinite', 'fr', 90)
    expect(key.slice(0, prefix.length)).toEqual([...prefix])
  })

  it("n'invalide rien quand le serveur refuse l'exclusion", async () => {
    vi.spyOn(api, 'patch').mockRejectedValue(new Error('ranked_not_excludable'))
    const client = makeClient()
    const invalidate = vi.spyOn(client, 'invalidateQueries')

    const { result } = renderHook(() => useSetMatchExclusion('JGtm'), {
      wrapper: wrapperFor(client),
    })
    result.current.mutate({ matchId: 'm1', excluded: true })

    await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 2000 })
    expect(invalidate).not.toHaveBeenCalled()
  })
})
