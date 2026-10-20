/**
 * Tests — sous-onglets de la section Carrière : « Citations » (moteur calculé) ou
 * « Commendations » (natives) selon la capability de titre `citations`, jamais le slug.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
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
    useParams: () => ({ playerSlug: 'test-player' }),
    useRouterState: () => ({ location: { pathname: '/t/partial/players/test-player/career' } }),
  }
})
vi.mock('./FilterOmnibar', () => ({ FilterOmnibar: () => null }))
vi.mock('./PeriodSessionRail', () => ({ PeriodSessionRail: () => null }))

import { NavL2 } from './NavL2'

type Titles = ReturnType<typeof useAppShellStore.getState>['availableTitles']

function setTitle(slug: string, capabilities: string[]) {
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: slug,
    availableTitles: [
      { slug, name: slug, status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
    ] as unknown as Titles,
  })
}

describe('NavL2 : onglet Citations / Commendations', () => {
  beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))

  it('pointe vers la page des citations calculées quand le titre déclare citations', () => {
    setTitle('partial', ['career', 'citations'])
    renderWithProviders(<NavL2 />)

    expect(screen.getByRole('link', { name: 'Citations' })).toHaveAttribute(
      'href',
      '/t/partial/players/test-player/career/citations',
    )
  })

  it('pointe vers les commendations natives quand le titre ne déclare pas citations', () => {
    setTitle('partial', ['career'])
    renderWithProviders(<NavL2 />)

    expect(screen.getByRole('link', { name: 'Citations' })).toHaveAttribute(
      'href',
      '/t/partial/players/test-player/career/commendations',
    )
  })

  it('ne se fie pas au slug : halo_5 qui déclare citations garde la page calculée', () => {
    setTitle('halo_5', ['career', 'citations'])
    renderWithProviders(<NavL2 />)

    expect(screen.getByRole('link', { name: 'Citations' })).toHaveAttribute(
      'href',
      '/t/halo_5/players/test-player/career/citations',
    )
  })
})
