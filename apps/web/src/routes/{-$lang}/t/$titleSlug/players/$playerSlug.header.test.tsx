/**
 * Tests — layout joueur (PlayerLayout) : il pose l'en-tête de page des sections (`SectionHeader`)
 * AVANT le contenu. Contrairement à `$playerSlug.test.tsx`, qui neutralise `SectionHeader`, celui-ci
 * le rend pour de vrai : retirer `<SectionHeader />` du layout fait échouer le test, et une section
 * du site se retrouverait sans titre.
 */
import type { ComponentPropsWithoutRef, ComponentType } from 'react'
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'

const pathRef = { current: '/t/un_titre/players/p/career' }

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    createFileRoute: () => (opts: Record<string, unknown>) => ({
      ...opts,
      useParams: () => ({ titleSlug: 'un_titre', playerSlug: 'p' }),
    }),
    Link: ({
      children,
      to,
      params,
      ...props
    }: ComponentPropsWithoutRef<'a'> & { to: string; params?: Record<string, string | undefined> }) => (
      <a href={resolveRoutePath(to, params)} {...props}>
        {children}
      </a>
    ),
    useParams: () => ({ playerSlug: 'p' }),
    useRouterState: (opts?: { select?: (s: { location: { pathname: string } }) => unknown }) => {
      const state = { location: { pathname: pathRef.current } }
      return opts?.select ? opts.select(state) : state
    },
    Navigate: () => <div data-testid="player-navigate" />,
    Outlet: () => <div data-testid="player-outlet" />,
  }
})

vi.mock('@/components/shell/SoloFiltersBar', () => ({ SoloFiltersBar: () => null }))
vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { show_progression: true } }),
}))

import { Route } from './$playerSlug'

const routeComponent = (
  Route as unknown as { component: ComponentType & { preload?: () => Promise<unknown> } }
).component

beforeAll(async () => {
  await routeComponent.preload?.()
})

beforeEach(() => {
  pathRef.current = '/t/un_titre/players/p/career'
  useAppShellStore.setState({
    isBootstrapped: true,
    locale: 'fr',
    currentTitleSlug: 'un_titre',
    availablePlayers: [
      { player_slug: 'p', gamertag: 'P', xuid: '1', waypoint_player: 'P', is_demo: false },
    ] as ReturnType<typeof useAppShellStore.getState>['availablePlayers'],
    availableTitles: [
      {
        slug: 'un_titre',
        name: 'T',
        status: 'active',
        capabilities: ['career', 'lusr', 'media', 'replay'],
        is_default: false,
        effective_hp_to_kill: 225,
      },
    ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
    activeSyncJobId: null,
  })
})

describe('PlayerLayout : en-tête de section', () => {
  it('pose le titre (h1) et les onglets de la section AVANT le contenu', () => {
    const PlayerLayout = routeComponent
    renderWithProviders(<PlayerLayout />)
    const titre = screen.getByRole('heading', { level: 1 })
    expect(titre).toHaveTextContent('Carrière')
    const contenu = screen.getByTestId('player-outlet')
    expect(titre.compareDocumentPosition(contenu) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Progression' })).toBeInTheDocument()
  })

  it('l’accueil n’a pas d’en-tête de section : sa bannière en tient lieu', () => {
    pathRef.current = '/t/un_titre/players/p/home'
    const PlayerLayout = routeComponent
    renderWithProviders(<PlayerLayout />)
    expect(screen.queryByRole('heading', { level: 1 })).toBeNull()
  })
})
