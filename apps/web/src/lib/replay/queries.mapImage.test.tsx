/**
 * useReplayMapImage — LA VARIANTE DE L'IMAGE DU FOND : l'adresse du rendu reste nue, celle du plan porte
 * `?variante=plan`, chaque variante a sa clé de cache, et l'image plan qui ne se charge pas retombe sur
 * le rendu sans rien dire.
 */
import type { ReactNode } from 'react'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { createTestQueryClient } from '@/test/render-utils'

import { useReplayMapImage } from './queries'

const getBlob = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, getBlob: (p: string) => getBlob(p) } }
})

const RENDU = '/players/JGtm/matches/m1/replay/background.png'
const PLAN = '/players/JGtm/matches/m1/replay/background.png?variante=plan'

/** Un `Image` qui se « décode » aussitôt, avec l'adresse d'objet qu'on lui a donnée. */
class FausseImage {
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  private source = ''
  get src() {
    return this.source
  }
  set src(valeur: string) {
    this.source = valeur
    queueMicrotask(() => this.onload?.())
  }
}

let client = createTestQueryClient()
const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={client}>{children}</QueryClientProvider>
)

beforeEach(() => {
  client = createTestQueryClient()
  getBlob.mockReset().mockResolvedValue(new Blob(['x']))
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite' })
  vi.stubGlobal('Image', FausseImage)
  let n = 0
  vi.spyOn(URL, 'createObjectURL').mockImplementation(() => `blob:image-${++n}`)
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('useReplayMapImage — variante', () => {
  it('sans variante : l’adresse du rendu, nue', async () => {
    const { result } = renderHook(() => useReplayMapImage('JGtm', 'm1', true), { wrapper })
    await waitFor(() => expect(result.current).not.toBeNull())
    expect(getBlob).toHaveBeenCalledTimes(1)
    expect(getBlob).toHaveBeenCalledWith(RENDU)
  })

  it('variante plan : l’adresse du plan, et le rendu n’est pas demandé', async () => {
    const { result } = renderHook(() => useReplayMapImage('JGtm', 'm1', true, 'plan'), { wrapper })
    await waitFor(() => expect(result.current).not.toBeNull())
    expect(getBlob).toHaveBeenCalledTimes(1)
    expect(getBlob).toHaveBeenCalledWith(PLAN)
  })

  it('une clé de cache par variante : le plan ne sert jamais le rendu, ni l’inverse', async () => {
    const { result, rerender } = renderHook(
      ({ variante }: { variante: 'rendu' | 'plan' }) => useReplayMapImage('JGtm', 'm1', true, variante),
      { wrapper, initialProps: { variante: 'rendu' } },
    )
    await waitFor(() => expect(result.current).not.toBeNull())
    const rendu = result.current
    rerender({ variante: 'plan' })
    await waitFor(() => expect(getBlob).toHaveBeenCalledWith(PLAN))
    await waitFor(() => expect(result.current).not.toBe(rendu))
    expect(client.getQueryData(queryKeys.matchReplayBackgroundImage('JGtm', 'halo_infinite', 'm1', 'rendu'))).toBe(rendu)
    expect(client.getQueryData(queryKeys.matchReplayBackgroundImage('JGtm', 'halo_infinite', 'm1', 'plan'))).toBe(
      result.current,
    )
    expect(queryKeys.matchReplayBackgroundImage('JGtm', 'halo_infinite', 'm1', 'plan')).not.toEqual(
      queryKeys.matchReplayBackgroundImage('JGtm', 'halo_infinite', 'm1', 'rendu'),
    )
  })

  it('l’image plan qui ne se charge pas : le rendu est servi, sans erreur', async () => {
    getBlob.mockImplementation((path: string) =>
      path === PLAN ? Promise.reject(new Error('404')) : Promise.resolve(new Blob(['x'])),
    )
    const { result } = renderHook(() => useReplayMapImage('JGtm', 'm1', true, 'plan'), { wrapper })
    await waitFor(() => expect(result.current).not.toBeNull())
    expect(getBlob).toHaveBeenCalledWith(RENDU)
  })

  it('désactivé (pas de calage) : aucune image demandée, quelle que soit la variante', () => {
    renderHook(() => useReplayMapImage('JGtm', 'm1', false, 'plan'), { wrapper })
    expect(getBlob).not.toHaveBeenCalled()
  })
})
