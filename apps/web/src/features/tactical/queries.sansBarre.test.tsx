/**
 * Un clic de zone (détail d'une cellule) et le préchargement des lectures voisines ne sont pas un
 * changement de page : ils ne doivent pas allumer la barre de chargement du haut d'écran.
 */
import { describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { compteDansLaBarre } from '@/lib/query/sansBarre'
import { usePrechargementDesLectures, useTacticalCellule, type ParamsDuRaster } from './queries'

const post = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, post: (path: string, body: unknown) => post(path, body) } }
})

function monter<T>(hook: () => T) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, ...renderHook(hook, { wrapper }) }
}

describe('requêtes de la Tactique sans barre de chargement', () => {
  it('le détail d une cellule en cours ne compte pas dans la barre', async () => {
    let fin!: (v: unknown) => void
    post.mockReset().mockImplementation(() => new Promise((r) => (fin = r)))
    const { client } = monter(() =>
      useTacticalCellule('JGtm', 'streets', { col: 1, lig: 2, pas_m: 2 }, { match_ids: ['m1'] }),
    )
    await waitFor(() => expect(client.isFetching()).toBe(1))
    expect(client.isFetching({ predicate: compteDansLaBarre })).toBe(0)
    await act(async () => fin({}))
  })

  it('le préchargement d une lecture voisine ne compte pas dans la barre', async () => {
    let fin!: (v: unknown) => void
    post.mockReset().mockImplementation(() => new Promise((r) => (fin = r)))
    const params: ParamsDuRaster = { match_ids: ['m1'], coequipiers: [], question: 'morts', qui: 'moi', spawn: undefined }
    const { client, result } = monter(() => usePrechargementDesLectures('JGtm', 'streets', params, ['temps']))
    act(() => result.current())
    await waitFor(() => expect(client.isFetching()).toBe(1))
    expect(client.isFetching({ predicate: compteDansLaBarre })).toBe(0)
    await act(async () => fin({ voisines: [] }))
  })
})
