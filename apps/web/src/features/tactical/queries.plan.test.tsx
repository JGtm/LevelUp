/**
 * Le plan des murs du fond d'une carte vient du MÊME sidecar que son calage (`/tactical/{map}/background`) :
 * une seule requête, le plan normalisé à l'arrivée, `null` quand la carte n'en a pas.
 */
import { describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { useTacticalMapBackgroundFrame, useTacticalMapBackgroundPlan } from './queries'

const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, get: (path: string) => get(path) } }
})

const CALAGE = { metersPerPixel: 0.5, originX: -10, originY: 30, widthPx: 80, heightPx: 40, convention: 'c' }

function monter<T>(hook: () => T) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return renderHook(hook, { wrapper })
}

describe('useTacticalMapBackgroundPlan', () => {
  it('rend le plan normalisé du sidecar, et partage la requête avec le calage', async () => {
    get.mockReset().mockResolvedValue({
      calibration: CALAGE,
      plan: { edges: [[0, 1, 2, 3], null], faces: 1, source: 'navmesh', vertices: 4 },
    })
    const { result } = monter(() => ({
      plan: useTacticalMapBackgroundPlan('JGtm', 'streets'),
      frame: useTacticalMapBackgroundFrame('JGtm', 'streets'),
    }))
    await waitFor(() => expect(result.current.plan).not.toBeNull())
    expect(result.current.plan?.edges).toEqual([[0, 1, 2, 3]])
    expect(result.current.frame?.widthM).toBeGreaterThan(0)
    expect(get).toHaveBeenCalledTimes(1)
    expect(get.mock.calls[0][0]).toContain('/tactical/streets/background')
  })

  it('carte sans plan (Live Fire, cartes Forge) : null', async () => {
    get.mockReset().mockResolvedValue({ calibration: CALAGE })
    const { result } = monter(() => useTacticalMapBackgroundPlan('JGtm', 'streets'))
    await waitFor(() => expect(get).toHaveBeenCalled())
    expect(result.current).toBeNull()
  })
})
