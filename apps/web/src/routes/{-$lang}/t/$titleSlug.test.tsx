/**
 * 2f — Tests du layout titre `t/$titleSlug` (D-6, D-8) : projection déclarative du
 * verdict `resolveTitleGate` + effet de convergence `applyActiveTitle`.
 *
 * Le routeur est mocké (createFileRoute → useParams contrôlé ; Outlet/Link marqueurs).
 * `applyActiveTitle` est mocké (vi.mock du module title-routing) — `resolveTitleGate`
 * reste RÉEL (verdict piloté par l'état du store, posé en setState direct).
 */
import type { ComponentType, ReactNode } from 'react'
import { describe, it, expect, beforeAll, beforeEach, vi } from 'vitest'
import { act, fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TitleSummary } from '@/lib/api/types'

const applyActiveTitleMock = vi.fn<(slug: string) => Promise<void>>(() => Promise.resolve())

// paramsRef : le slug de titre porté par l'« URL » (segment $titleSlug).
const paramsRef: { titleSlug: string; lang?: string } = { titleSlug: 'halo_infinite' }

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    createFileRoute: () => (opts: Record<string, unknown>) => ({
      ...opts,
      useParams: () => paramsRef,
    }),
    Outlet: () => <div data-testid="title-outlet" />,
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

vi.mock('@/lib/title-routing', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/title-routing')>()
  return { ...actual, applyActiveTitle: applyActiveTitleMock }
})

// Import APRÈS les mocks (le composant lit createFileRoute + applyActiveTitle au load).
import { Route } from './$titleSlug'

// `Route.component` n'est pas exposé par le type public de Route (accès interne au
// mock) → cast. Le composant porte aussi `.preload` (wrapper lazy autoCodeSplitting).
const routeComponent = (
  Route as unknown as { component: ComponentType & { preload?: () => Promise<unknown> } }
).component
const TitleLayout: ComponentType = routeComponent

// autoCodeSplitting (vite.config) : le composant de route est chargé en lazy → il
// « suspend » au premier rendu. On le précharge une fois pour rendre les rendus
// synchrones (le mock de createFileRoute conserve le wrapper lazy + son `.preload`).
beforeAll(async () => {
  await routeComponent.preload?.()
})

function title(slug: string, status?: TitleSummary['status']): TitleSummary {
  return {
    slug,
    name: slug,
    status,
    capabilities: [],
    is_default: false,
    effective_hp_to_kill: 225,
  } as unknown as TitleSummary
}

function setStore(partial: Partial<ReturnType<typeof useAppShellStore.getState>>) {
  act(() => useAppShellStore.setState(partial))
}

beforeEach(() => {
  applyActiveTitleMock.mockClear()
  applyActiveTitleMock.mockImplementation(() => Promise.resolve())
  paramsRef.titleSlug = 'halo_infinite'
  paramsRef.lang = undefined
  useAppShellStore.setState({
    isBootstrapped: true,
    isTitleSwitching: false,
    locale: 'fr',
    currentTitleSlug: 'halo_infinite',
    availableTitles: [title('halo_infinite')],
    currentPlayer: null,
  })
})

describe('TitleLayout (2f)', () => {
  it('wait (pré-hydratation) → ne rend rien', () => {
    setStore({ isBootstrapped: false })
    renderWithProviders(<TitleLayout />)
    expect(screen.queryByTestId('title-outlet')).toBeNull()
    expect(screen.queryByText('Titre introuvable')).toBeNull()
  })

  it('valide + convergé → rend l’Outlet', () => {
    renderWithProviders(<TitleLayout />)
    expect(screen.getByTestId('title-outlet')).toBeInTheDocument()
    expect(applyActiveTitleMock).not.toHaveBeenCalled()
  })

  it('divergence segment↔store → applyActiveTitle puis convergence rend l’Outlet', async () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    renderWithProviders(<TitleLayout />)

    // Divergence : effet déclenché, Outlet NON rendu tant que le titre n'a pas convergé.
    expect(applyActiveTitleMock).toHaveBeenCalledWith('halo_5')
    expect(screen.queryByTestId('title-outlet')).toBeNull()

    // Convergence (le store passe au nouveau titre) → Outlet rendu.
    setStore({ currentTitleSlug: 'halo_5' })
    expect(await screen.findByTestId('title-outlet')).toBeInTheDocument()
  })

  it('slug inconnu → écran gate « Titre introuvable » (FR)', () => {
    paramsRef.titleSlug = 'inexistant'
    renderWithProviders(<TitleLayout />)
    expect(screen.getByText('Titre introuvable')).toBeInTheDocument()
    expect(screen.queryByTestId('title-outlet')).toBeNull()
  })

  it('coming_soon → écran gate « Bientôt disponible » (FR)', () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({ availableTitles: [title('halo_infinite'), title('halo_5', 'coming_soon')] })
    renderWithProviders(<TitleLayout />)
    expect(screen.getByText('Bientôt disponible')).toBeInTheDocument()
  })

  it('archived → écran gate « Titre archivé » (FR)', () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({ availableTitles: [title('halo_infinite'), title('halo_5', 'archived')] })
    renderWithProviders(<TitleLayout />)
    expect(screen.getByText('Titre archivé')).toBeInTheDocument()
  })

  it('échec d’apply → écran switch_failed + « Réessayer » re-tente applyActiveTitle', async () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    applyActiveTitleMock.mockImplementationOnce(() => Promise.reject(new Error('boom')))
    renderWithProviders(<TitleLayout />)

    expect(await screen.findByText('Changement de titre impossible')).toBeInTheDocument()
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(1)

    // « Réessayer » réarme l'état d'échec → l'effet re-tente la bascule.
    fireEvent.click(screen.getByText('Réessayer'))
    await act(async () => {
      await Promise.resolve()
    })
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(2)
  })
})
