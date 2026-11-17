/**
 * Tests — hooks de la génération d'un rejeu : requête seulement quand elle est permise, relecture toutes les 3 s en
 * file et en cours seulement, invalidations au passage à `ready`, état posé au cache par la demande, 409 et 429
 * qui relisent l'état.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  REPLAY_GENERATION_POLL_MS,
  useReplayGenerationStatus,
  useRequestReplayGeneration,
  type ReplayGenerationOutcome,
  type ReplayGenerationStatus,
} from './queries'

vi.mock('@/lib/api/client', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...original, api: { ...original.api, get: vi.fn(), post: vi.fn() } }
})

const get = vi.mocked(api.get)
const post = vi.mocked(api.post)

const PLAYER = 'joueur'
const MATCH = 'match-1'
const TITLE = 'halo_infinite'
const GEN_PATH = `/players/${PLAYER}/matches/${MATCH}/replay/generation`
const POST_PATH = `/players/${PLAYER}/matches/${MATCH}/replay/generate`

let qc: QueryClient

function wrapper({ children }: { children: ReactNode }) {
  return createElement(QueryClientProvider, { client: qc }, children)
}

const status = (rest: Partial<ReplayGenerationStatus> & Pick<ReplayGenerationStatus, 'state'>) => rest

beforeEach(() => {
  qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  useAppShellStore.setState({ currentTitleSlug: TITLE })
  get.mockReset()
  post.mockReset()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useReplayGenerationStatus', () => {
  it('ne lit rien tant que la lecture n’est pas permise', () => {
    renderHook(() => useReplayGenerationStatus(PLAYER, MATCH, false), { wrapper })
    expect(get).not.toHaveBeenCalled()
  })

  it('lit l’état du match et le pose sous la clé du titre', async () => {
    get.mockResolvedValue(status({ state: 'generatable' }))
    const { result } = renderHook(() => useReplayGenerationStatus(PLAYER, MATCH, true), { wrapper })
    await waitFor(() => expect(result.current.data).toEqual({ state: 'generatable' }))
    expect(get).toHaveBeenCalledWith(GEN_PATH)
    expect(qc.getQueryData(queryKeys.matchReplayGeneration(PLAYER, TITLE, MATCH))).toEqual({ state: 'generatable' })
  })

  it('relit toutes les 3 s en file et en cours, et plus dès que l’état est terminal', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    get
      .mockResolvedValueOnce(status({ state: 'queued', position: 1 }))
      .mockResolvedValueOnce(status({ state: 'running', phase: 'downloading' }))
      .mockResolvedValue(status({ state: 'failed', reason: 'fetch_failed', retryable: true }))
    const { result } = renderHook(() => useReplayGenerationStatus(PLAYER, MATCH, true), { wrapper })
    await waitFor(() => expect(result.current.data?.state).toBe('queued'))
    expect(get).toHaveBeenCalledTimes(1)

    await act(() => vi.advanceTimersByTimeAsync(REPLAY_GENERATION_POLL_MS + 50))
    await waitFor(() => expect(result.current.data?.state).toBe('running'))
    expect(get).toHaveBeenCalledTimes(2)

    await act(() => vi.advanceTimersByTimeAsync(REPLAY_GENERATION_POLL_MS + 50))
    await waitFor(() => expect(result.current.data?.state).toBe('failed'))
    expect(get).toHaveBeenCalledTimes(3)

    await act(() => vi.advanceTimersByTimeAsync(REPLAY_GENERATION_POLL_MS * 4))
    expect(get).toHaveBeenCalledTimes(3)
  })

  it('ne relit jamais un état qui n’attend rien', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    get.mockResolvedValue(status({ state: 'generatable' }))
    const { result } = renderHook(() => useReplayGenerationStatus(PLAYER, MATCH, true), { wrapper })
    await waitFor(() => expect(result.current.data?.state).toBe('generatable'))
    await act(() => vi.advanceTimersByTimeAsync(REPLAY_GENERATION_POLL_MS * 4))
    expect(get).toHaveBeenCalledTimes(1)
  })

  it('au passage à ready : le rejeu, la vue du match (toutes langues) et l’accueil se relisent', async () => {
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    get.mockResolvedValue(status({ state: 'ready' }))
    renderHook(() => useReplayGenerationStatus(PLAYER, MATCH, true), { wrapper })
    await waitFor(() => expect(invalidate).toHaveBeenCalledTimes(3))
    const keys = invalidate.mock.calls.map((c) => c[0]?.queryKey)
    expect(keys).toContainEqual(queryKeys.matchReplay(PLAYER, TITLE, MATCH))
    expect(keys).toContainEqual(queryKeys.matchViewAllLocales(PLAYER, TITLE, MATCH))
    expect(keys).toContainEqual(queryKeys.homeAll(PLAYER))
  })

  it('aucune invalidation tant que l’état n’est pas ready', async () => {
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    get.mockResolvedValue(status({ state: 'queued', position: 1 }))
    const { result } = renderHook(() => useReplayGenerationStatus(PLAYER, MATCH, true), { wrapper })
    await waitFor(() => expect(result.current.data?.state).toBe('queued'))
    expect(invalidate).not.toHaveBeenCalled()
  })
})

describe('useRequestReplayGeneration', () => {
  const key = () => queryKeys.matchReplayGeneration(PLAYER, TITLE, MATCH)

  it('envoie la demande et pose l’état rendu au cache', async () => {
    const outcome: ReplayGenerationOutcome = { created: true, status: { state: 'queued', position: 1 } }
    post.mockResolvedValue(outcome)
    const { result } = renderHook(() => useRequestReplayGeneration(PLAYER, MATCH), { wrapper })
    await act(() => result.current.mutateAsync())
    expect(post).toHaveBeenCalledWith(POST_PATH, {})
    expect(qc.getQueryData(key())).toEqual({ state: 'queued', position: 1 })
  })

  it.each([409, 429])('un %i relit l’état au lieu de garder l’état supposé', async (code) => {
    qc.setQueryData(key(), { state: 'generatable' })
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    post.mockRejectedValue({ code: 'x', message: 'refus', retryable: false, status: code })
    const { result } = renderHook(() => useRequestReplayGeneration(PLAYER, MATCH), { wrapper })
    await act(() => result.current.mutateAsync().catch(() => undefined))
    expect(invalidate).toHaveBeenCalledWith({ queryKey: key() })
  })

  it('une autre erreur ne relit rien', async () => {
    const invalidate = vi.spyOn(qc, 'invalidateQueries')
    post.mockRejectedValue({ code: 'x', message: 'panne', retryable: false, status: 500 })
    const { result } = renderHook(() => useRequestReplayGeneration(PLAYER, MATCH), { wrapper })
    await act(() => result.current.mutateAsync().catch(() => undefined))
    expect(invalidate).not.toHaveBeenCalled()
  })
})
