/**
 * Tests useUpdateSettings — réalignement défensif du segment lang de l'URL après un
 * changement de langue (Phase 5b, D-12).
 *
 * La page Settings étant AGNOSTIQUE (hors segment, D-3), ce chemin est THÉORIQUE
 * aujourd'hui (parseRouteSegments('/settings') === {}) ; ces tests verrouillent le
 * comportement pour toute future surface de changement de langue title-scoped.
 *
 * demoMode : la mutation de langue s'applique client-side (pas de réseau à mocker) ;
 * useNavigate est mocké pour observer le navigate REPLACE sans RouterProvider.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'

import { useUpdateSettings } from './queries'
import { useAppShellStore } from '@/stores/appShellStore'

const navigateMock = vi.fn()

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => navigateMock }
})

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return createElement(QueryClientProvider, { client }, children)
}

function setPathname(pathname: string) {
  window.history.replaceState({}, '', pathname)
}

describe('useUpdateSettings — réalignement segment lang (5b, D-12)', () => {
  beforeEach(() => {
    navigateMock.mockClear()
    useAppShellStore.setState({ locale: 'fr', demoMode: true })
    setPathname('/')
  })

  it('URL AVEC segment lang ≠ nouvelle locale → navigate REPLACE, même route, nouveau lang', async () => {
    setPathname('/fr/t/halo_infinite/players/jgtm/home')
    const { result } = renderHook(() => useUpdateSettings(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ lang: 'en' })
    })

    expect(navigateMock).toHaveBeenCalledTimes(1)
    const arg = navigateMock.mock.calls[0][0] as {
      to: string
      replace?: boolean
      params: (prev: Record<string, unknown>) => Record<string, unknown>
    }
    expect(arg.to).toBe('.')
    expect(arg.replace).toBe(true)
    // Le params updater réécrit UNIQUEMENT lang, préserve titleSlug/playerSlug.
    expect(arg.params({ lang: 'fr', titleSlug: 'halo_infinite', playerSlug: 'jgtm' })).toEqual({
      lang: 'en',
      titleSlug: 'halo_infinite',
      playerSlug: 'jgtm',
    })
  })

  it('URL SANS segment (page agnostique /settings) → aucun navigate (comportement actuel)', async () => {
    setPathname('/settings')
    const { result } = renderHook(() => useUpdateSettings(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ lang: 'en' })
    })

    expect(navigateMock).not.toHaveBeenCalled()
  })

  it('même langue que la locale courante → aucun navigate (pas de changement)', async () => {
    setPathname('/fr/t/halo_infinite/players/jgtm/home')
    const { result } = renderHook(() => useUpdateSettings(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ lang: 'fr' })
    })

    expect(navigateMock).not.toHaveBeenCalled()
  })
})
