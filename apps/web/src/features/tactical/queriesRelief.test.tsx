/**
 * Tests — queriesRelief : où se lit le relief du fond d'une carte de l'onglet Tactique, sous quelles clés, et ce
 * que le sidecar du fond en annonce.
 */
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { createTestQueryClient } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { tacticalReliefSource, useTacticalMapOffers } from './queriesRelief'

const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, get: (path: string) => get(path) } }
})

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={createTestQueryClient()}>{children}</QueryClientProvider>
}

beforeEach(() => {
  get.mockReset()
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite' })
})

describe('tacticalReliefSource', () => {
  const source = tacticalReliefSource('halo_infinite', 'JGtm', 'map/rtx 1')

  it('lit les données et l’image sur les routes de la carte, la carte encodée', () => {
    expect(source.dataPath).toBe('/players/JGtm/tactical/map%2Frtx%201/background.relief.json')
    expect(source.imagePath).toBe('/players/JGtm/tactical/map%2Frtx%201/background.png?variante=relief')
  })

  it('range les deux fichiers sous des clés de la CARTE : ni joueur, ni langue', () => {
    expect(source.dataKey).toEqual(['tactical-map-relief-data', 'halo_infinite', 'map/rtx 1'])
    expect(source.imageKey).toEqual(['tactical-map-relief-image', 'halo_infinite', 'map/rtx 1'])
    expect(JSON.stringify([source.dataKey, source.imageKey])).not.toContain('JGtm')
  })
})

describe('useTacticalMapOffers', () => {
  it('annonce le plan et le relief que le sidecar porte', async () => {
    get.mockResolvedValue({ calibration: {}, planImage: 'plan.webp', reliefImage: 'r.webp', reliefData: 'r.json' })
    const { result } = renderHook(() => useTacticalMapOffers('JGtm', 'streets'), { wrapper })
    await waitFor(() => expect(result.current.reliefOffered).toBe(true))
    expect(result.current.planImage).toBe('plan.webp')
  })

  it('un relief sans ses deux fichiers n’est pas offert', async () => {
    get.mockResolvedValue({ calibration: {}, reliefImage: 'r.webp' })
    const { result } = renderHook(() => useTacticalMapOffers('JGtm', 'streets'), { wrapper })
    await waitFor(() => expect(get).toHaveBeenCalled())
    expect(result.current.reliefOffered).toBe(false)
  })

  it('une carte sans fond figé (404) n’offre rien', async () => {
    get.mockRejectedValue(new Error('404'))
    const { result } = renderHook(() => useTacticalMapOffers('JGtm', 'streets'), { wrapper })
    await waitFor(() => expect(get).toHaveBeenCalled())
    expect(result.current).toEqual({ planImage: undefined, reliefOffered: false })
  })
})
