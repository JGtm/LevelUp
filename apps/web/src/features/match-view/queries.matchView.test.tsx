/**
 * useMatchView — une vue construite avec une source en panne n'est pas réutilisée par le
 * cache de requêtes (E4, D-9) : au retour sur la page, l'API est interrogée de nouveau et
 * la vue complète remplace la vue dégradée. Une vue complète garde sa fraîcheur (pas de
 * second appel).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'

import { api } from '@/lib/api/client'
import type { MatchViewResponse } from '@/lib/api/types'

import { useMatchView } from './queries'

vi.mock('@/lib/api/client', () => ({
  api: { get: vi.fn(), post: vi.fn() },
  getApiTitleSlug: () => 'titre_test',
  setApiTitleSlug: vi.fn(),
  setApiLocale: vi.fn(),
}))

const apiGet = vi.mocked(api.get)

const enPanne = { is_partial: true, partial_reasons: ['scoreboard_unavailable'] } as MatchViewResponse
const complete = { is_partial: false } as MatchViewResponse

/** Monte la page, la démonte, la remonte sur le MÊME cache : un « retour sur la page ». */
async function ouvrirDeuxFois(client: QueryClient) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  const premiere = renderHook(() => useMatchView('joueur', 'm1'), { wrapper })
  await waitFor(() => expect(premiere.result.current.data).toBeDefined())
  premiere.unmount()
  return renderHook(() => useMatchView('joueur', 'm1'), { wrapper })
}

beforeEach(() => {
  apiGet.mockReset()
})

describe('useMatchView — fraîcheur du cache', () => {
  it("vue en panne : le retour sur la page interroge de nouveau l'API et sert la vue complète", async () => {
    apiGet.mockResolvedValueOnce(enPanne).mockResolvedValueOnce(complete)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    const seconde = await ouvrirDeuxFois(client)

    await waitFor(() => expect(seconde.result.current.data).toEqual(complete))
    expect(apiGet).toHaveBeenCalledTimes(2)
  })

  it('vue complète : le retour sur la page réutilise le cache (un seul appel)', async () => {
    apiGet.mockResolvedValue(complete)
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })

    const seconde = await ouvrirDeuxFois(client)

    await waitFor(() => expect(seconde.result.current.data).toEqual(complete))
    expect(apiGet).toHaveBeenCalledTimes(1)
  })
})
