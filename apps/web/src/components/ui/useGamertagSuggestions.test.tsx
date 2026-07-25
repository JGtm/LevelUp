/**
 * Tests unitaires pour useGamertagSuggestions.
 *
 * Couvre :
 *  - Scoring local (exact > prefix > substring, pas de char-by-char)
 *  - Ordre des sources et déduplication entre configured / frequent / remote
 *  - Filtrage via excludeGamertags
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'

import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { PlayerSummary, TeammateOption, GamertagSearchResponse } from '@/lib/api/types'

import { useGamertagSuggestions } from './useGamertagSuggestions'

// ─── Helpers ──────────────────────────────────────────────────────────────────

function makeWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
}

function asPlayer(gamertag: string, slug?: string): PlayerSummary {
  return {
    player_slug: slug ?? gamertag.toLowerCase(),
    gamertag,
    xuid: `xuid-${gamertag}`,
  } as PlayerSummary
}

function asTeammate(gamertag: string, encounter_count = 5): TeammateOption {
  return { gamertag, xuid: `xuid-${gamertag}`, encounter_count, last_seen_at: undefined }
}

// ─── Tests ─────────────────────────────────────────────────────────────────────

describe('useGamertagSuggestions', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      availablePlayers: [
        asPlayer('AlphaPlayer'),
        asPlayer('BravoGamer'),
      ],
    })
  })

  it('retourne tous les configured quand la query est vide', () => {
    const { result } = renderHook(
      () => useGamertagSuggestions({ query: '', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )
    expect(result.current.configured).toHaveLength(2)
    expect(result.current.frequent).toHaveLength(0)
    expect(result.current.remote).toHaveLength(0)
  })

  it('classe exact match en premier devant prefix', () => {
    useAppShellStore.setState({
      availablePlayers: [
        asPlayer('Master'),
        asPlayer('MasterChief'),
        asPlayer('GrandMaster'),
      ],
    })
    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'Master', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )
    expect(result.current.configured.map((c) => c.gamertag)).toEqual([
      'Master',
      'MasterChief',
      'GrandMaster',
    ])
  })

  it('ne matche PAS en char-by-char ambigu (regression)', () => {
    useAppShellStore.setState({
      availablePlayers: [asPlayer('AlphaPlayer'), asPlayer('Zorglub')],
    })
    // Le fuzzy score char-by-char retiré → "az" ne doit plus matcher AlphaPlayer
    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'az', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )
    expect(result.current.configured).toHaveLength(0)
  })

  it('exclut les gamertags via excludeGamertags', () => {
    const { result } = renderHook(
      () =>
        useGamertagSuggestions({
          query: '',
          frequentOptions: [],
          excludeGamertags: ['AlphaPlayer'],
        }),
      { wrapper: makeWrapper() },
    )
    expect(result.current.configured.map((c) => c.gamertag)).toEqual(['BravoGamer'])
  })

  it('déduplique frequent quand un gamertag est déjà dans configured', () => {
    const { result } = renderHook(
      () =>
        useGamertagSuggestions({
          query: '',
          frequentOptions: [
            asTeammate('AlphaPlayer', 10), // déjà dans configured
            asTeammate('CharlieX', 3),
          ],
        }),
      { wrapper: makeWrapper() },
    )
    expect(result.current.configured.map((c) => c.gamertag)).toEqual([
      'AlphaPlayer',
      'BravoGamer',
    ])
    expect(result.current.frequent.map((f) => f.gamertag)).toEqual(['CharlieX'])
  })

  it('déduplique remote contre configured et frequent', async () => {
    server.use(
      http.get('*/directory/gamertags/search', () =>
        HttpResponse.json<GamertagSearchResponse>({
          query: 'al',
          items: [
            { gamertag: 'AlphaPlayer', xuid: 'xuid-AlphaPlayer', score: 1, exact_match: false },
            { gamertag: 'AlbatrosX', xuid: 'xuid-AlbatrosX', score: 1, exact_match: false },
          ],
        }),
      ),
    )

    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'al', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )

    // Wait for debounce + remote fetch
    await waitFor(
      () => {
        expect(result.current.remote.map((r) => r.gamertag)).toEqual(['AlbatrosX'])
      },
      { timeout: 2000 },
    )
    // AlphaPlayer doit toujours être dans configured, pas dupliqué dans remote
    expect(result.current.configured.map((c) => c.gamertag)).toEqual(['AlphaPlayer'])
  })

  it('ne déclenche pas le remote en dessous de 2 caractères', async () => {
    let callCount = 0
    server.use(
      http.get('*/directory/gamertags/search', () => {
        callCount++
        return HttpResponse.json<GamertagSearchResponse>({ query: '', items: [] })
      }),
    )

    renderHook(() => useGamertagSuggestions({ query: 'a', frequentOptions: [] }), {
      wrapper: makeWrapper(),
    })

    // Attente du debounce
    await new Promise((r) => setTimeout(r, 400))
    expect(callCount).toBe(0)
  })

  it('repli live désarmé par défaut : aucun appel live=1 tant que non déclenché', async () => {
    const liveHits: string[] = []
    server.use(
      http.get('*/directory/gamertags/search', ({ request }) => {
        const live = new URL(request.url).searchParams.get('live')
        if (live === '1') liveHits.push('live')
        return HttpResponse.json<GamertagSearchResponse>({ query: 'NeverSeen', items: [] })
      }),
    )
    useAppShellStore.setState({ availablePlayers: [] })

    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'NeverSeen', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )
    await waitFor(() => expect(result.current.remoteAttempted).toBe(true), { timeout: 2000 })
    await new Promise((r) => setTimeout(r, 100))
    // La recherche locale (live=0) a répondu ; le repli live n'a jamais été appelé.
    expect(liveHits).toHaveLength(0)
    expect(result.current.liveAttempted).toBe(false)
    expect(result.current.liveResults).toHaveLength(0)
  })

  it('triggerLiveSearch arme le repli live=1 et remonte le joueur résolu', async () => {
    server.use(
      http.get('*/directory/gamertags/search', ({ request }) => {
        const live = new URL(request.url).searchParams.get('live')
        if (live === '1') {
          return HttpResponse.json<GamertagSearchResponse>({
            query: 'NeverSeen',
            items: [{ gamertag: 'NeverSeen', xuid: 'xuid-live', score: 0, exact_match: true }],
          })
        }
        return HttpResponse.json<GamertagSearchResponse>({ query: 'NeverSeen', items: [] })
      }),
    )
    useAppShellStore.setState({ availablePlayers: [] })

    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'NeverSeen', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )
    await waitFor(() => expect(result.current.remoteAttempted).toBe(true), { timeout: 2000 })
    expect(result.current.liveResults).toHaveLength(0)

    act(() => {
      result.current.triggerLiveSearch()
    })

    await waitFor(
      () => expect(result.current.liveResults.map((r) => r.gamertag)).toEqual(['NeverSeen']),
      { timeout: 2000 },
    )
    expect(result.current.liveAttempted).toBe(true)
  })

  it('ne garde AUCUN résultat live quand la query change (pas de placeholder rémanent)', async () => {
    // Contre-revue V72 : `placeholderData: keepPreviousData` sur la query live
    // (désarmée par défaut) faisait persister les hits Xbox de la recherche
    // PRÉCÉDENTE sous une saisie qui n'a jamais interrogé Xbox.
    server.use(
      http.get('*/directory/gamertags/search', ({ request }) => {
        const url = new URL(request.url)
        if (url.searchParams.get('live') === '1') {
          return HttpResponse.json<GamertagSearchResponse>({
            query: 'NeverSeen',
            items: [{ gamertag: 'NeverSeen', xuid: 'xuid-live', score: 0, exact_match: true }],
          })
        }
        return HttpResponse.json<GamertagSearchResponse>({ query: '', items: [] })
      }),
    )
    useAppShellStore.setState({ availablePlayers: [] })

    const { result, rerender } = renderHook(
      ({ q }: { q: string }) => useGamertagSuggestions({ query: q, frequentOptions: [] }),
      { wrapper: makeWrapper(), initialProps: { q: 'NeverSeen' } },
    )
    await waitFor(() => expect(result.current.remoteAttempted).toBe(true), { timeout: 2000 })
    act(() => result.current.triggerLiveSearch())
    await waitFor(() => expect(result.current.liveResults).toHaveLength(1), { timeout: 2000 })

    // L'utilisateur continue de taper : la nouvelle saisie n'a JAMAIS été résolue
    // en live → aucun résultat Xbox ne doit rester affiché.
    rerender({ q: 'NeverSeenX' })
    await waitFor(() => expect(result.current.liveAttempted).toBe(false), { timeout: 2000 })
    expect(result.current.liveResults).toHaveLength(0)
    expect(result.current.liveEmpty).toBe(false)
  })

  it('liveEmpty=true quand le repli Xbox aboutit sans aucun joueur', async () => {
    server.use(
      http.get('*/directory/gamertags/search', () =>
        HttpResponse.json<GamertagSearchResponse>({ query: 'GhostGT', items: [] }),
      ),
    )
    useAppShellStore.setState({ availablePlayers: [] })

    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'GhostGT', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )
    await waitFor(() => expect(result.current.remoteAttempted).toBe(true), { timeout: 2000 })
    expect(result.current.liveEmpty).toBe(false) // repli non déclenché → pas de verdict

    act(() => result.current.triggerLiveSearch())
    await waitFor(() => expect(result.current.liveEmpty).toBe(true), { timeout: 2000 })
    expect(result.current.isLiveLoading).toBe(false)
  })

  it('liveEmpty=false quand Xbox répond mais que le hit est déjà affiché ailleurs', async () => {
    // Réponse live NON vide dont l'unique hit est déjà dans les joueurs configurés :
    // liveResults est vide après déduplication, mais ce n'est PAS « aucun résultat ».
    server.use(
      http.get('*/directory/gamertags/search', ({ request }) => {
        const url = new URL(request.url)
        if (url.searchParams.get('live') === '1') {
          return HttpResponse.json<GamertagSearchResponse>({
            query: 'AlphaPlayer',
            items: [
              { gamertag: 'AlphaPlayer', xuid: 'xuid-AlphaPlayer', score: 1, exact_match: true },
            ],
          })
        }
        return HttpResponse.json<GamertagSearchResponse>({ query: '', items: [] })
      }),
    )

    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'AlphaPlayer', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )
    await waitFor(() => expect(result.current.remoteAttempted).toBe(true), { timeout: 2000 })
    act(() => result.current.triggerLiveSearch())
    await waitFor(() => expect(result.current.liveAttempted).toBe(true), { timeout: 2000 })
    await waitFor(() => expect(result.current.isLiveLoading).toBe(false), { timeout: 2000 })

    expect(result.current.liveResults).toHaveLength(0) // dédupliqué de « configurés »
    expect(result.current.liveEmpty).toBe(false)
  })

  it('produit hasAnyResult=false et remoteAttempted=true sur 0 résultat', async () => {
    server.use(
      http.get('*/directory/gamertags/search', () =>
        HttpResponse.json<GamertagSearchResponse>({ query: 'xq', items: [] }),
      ),
    )
    useAppShellStore.setState({ availablePlayers: [] })

    const { result } = renderHook(
      () => useGamertagSuggestions({ query: 'xq', frequentOptions: [] }),
      { wrapper: makeWrapper() },
    )

    await waitFor(
      () => {
        expect(result.current.remoteAttempted).toBe(true)
        expect(result.current.isRemoteLoading).toBe(false)
      },
      { timeout: 2000 },
    )
    expect(result.current.hasAnyResult).toBe(false)
  })
})
