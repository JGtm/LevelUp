/**
 * Tests — NavSectionList, rendu des sections partagé par le tiroir mobile et la barre latérale.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ComponentPropsWithoutRef } from 'react'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'
import { NavSectionList } from './NavSectionList'
import type { L1Section } from './navL1Sections'

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
  }
})

const SECTIONS: L1Section[] = [
  {
    key: 'home',
    labelKey: 'common.nav.section_home',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
    matchPathname: (p) => p.includes('/home'),
  },
  {
    key: 'stats',
    labelKey: 'common.nav.section_solo',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries',
    matchPathname: (p) => p.includes('/stats'),
    tabs: [
      {
        key: 'sessions',
        labelKey: 'common.nav.tab_sessions',
        path: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/sessions',
      },
    ],
  },
  {
    key: 'squad',
    labelKey: 'common.nav.section_squad',
    defaultPath: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies',
    matchPathname: (p) => p.includes('/squad'),
    tabs: [
      {
        key: 'synergies',
        labelKey: 'common.nav.tab_synergies',
        path: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad/synergies',
      },
    ],
  },
]

function setup(variant: 'drawer' | 'sidebar', pathname: string, onNavigate = vi.fn()) {
  renderWithProviders(
    <NavSectionList
      variant={variant}
      sections={SECTIONS}
      pathname={pathname}
      titleSlug="halo_infinite"
      playerSlug="test-player"
      onNavigate={onNavigate}
    />,
  )
  return onNavigate
}

describe('NavSectionList', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('tiroir : menu ARIA, tous les onglets visibles sous chaque section, aucun chevron', () => {
    setup('drawer', '/t/halo_infinite/players/test-player/home')
    expect(screen.getAllByRole('menuitem').length).toBe(5)
    expect(screen.getByRole('menuitem', { name: 'Sessions' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'Synergies' })).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('barre latérale : liens, onglets seulement sous la section active', () => {
    setup('sidebar', '/t/halo_infinite/players/test-player/stats/sessions')
    expect(screen.queryByRole('menuitem')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Accueil' })).toHaveAttribute(
      'href',
      '/t/halo_infinite/players/test-player/home',
    )
    expect(screen.getByRole('link', { name: 'Sessions' })).toHaveAttribute('aria-current', 'page')
    expect(screen.queryByRole('link', { name: 'Synergies' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Solo' })).toHaveAttribute('aria-current', 'page')
  })

  it('barre latérale : le chevron déplie et replie une section sans naviguer', () => {
    const onNavigate = setup('sidebar', '/t/halo_infinite/players/test-player/home')
    const chevron = screen.getByRole('button', { name: 'Onglets Escouade' })
    expect(chevron).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(chevron)
    expect(chevron).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: 'Synergies' })).toBeInTheDocument()

    fireEvent.click(chevron)
    expect(screen.queryByRole('link', { name: 'Synergies' })).not.toBeInTheDocument()
    expect(onNavigate).not.toHaveBeenCalled()
  })

  it('un lien prévient son parent (fermeture du tiroir)', () => {
    const onNavigate = setup('drawer', '/t/halo_infinite/players/test-player/home')
    fireEvent.click(screen.getByRole('menuitem', { name: 'Sessions' }))
    expect(onNavigate).toHaveBeenCalledTimes(1)
  })
})
