/**
 * useCoequipierOptions — la liste des coéquipiers proposés est MÉMOÏSÉE sur la réponse
 * (retours rejeu L2, 2026-09-23).
 *
 * Un tableau neuf à chaque rendu recalculait, à chaque rendu de la page, la composition,
 * les paramètres du raster et leur empreinte (`hashFiltre` sur toute la liste de
 * `match_id`). Ce test cadenasse l'identité : deux rendus sur la même réponse rendent LE
 * MÊME tableau.
 */
import { describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { createTestQueryClient } from '@/test/render-utils'

import { useCoequipierOptions } from './queries'

const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, get: (path: string) => get(path) } }
})

describe('useCoequipierOptions', () => {
  it('rend le MÊME tableau d’un rendu à l’autre sur la même réponse', async () => {
    get.mockResolvedValue({
      teammates: [{ gamertag: 'Ami', xuid: 'xuid(42)', match_count: 30, as_teammate: 30, as_enemy: 0, avg_kda: null }],
      enemies: [],
      total: 1,
    })
    const client = createTestQueryClient()
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    )
    const { result, rerender } = renderHook(() => useCoequipierOptions('JGtm'), { wrapper })
    await waitFor(() => expect(result.current.chargees).toBe(true))
    const avant = result.current.options
    expect(avant).toEqual([{ gamertag: 'Ami', xuid: 'xuid(42)', encounter_count: 30 }])

    rerender()
    expect(result.current.options).toBe(avant)
  })
})
