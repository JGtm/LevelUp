import type { ComponentPropsWithoutRef } from 'react'

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { NavL1 } from './NavL1'

let mockPathname = '/players/test-player/home'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ children, to, ...props }: ComponentPropsWithoutRef<'a'> & { to: string }) => (
      <a href={to} {...props}>
        {children}
      </a>
    ),
    useNavigate: () => vi.fn(),
    useRouterState: () => ({ location: { pathname: mockPathname } }),
  }
})

vi.mock('./ThemeToggle', () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}))

describe('NavL1', () => {
  beforeEach(() => {
    mockPathname = '/players/test-player/home'
    // Casts vers `unknown` puis le type cible : évite `as any` (lint strict).
    // Les fixtures volontairement minimales ne représentent pas le full PlayerProfile.
    useAppShellStore.setState({
      currentPlayer: { player_slug: 'test-player', gamertag: 'TestPlayer' } as unknown as ReturnType<typeof useAppShellStore.getState>['currentPlayer'],
      availablePlayers: [{ player_slug: 'test-player', gamertag: 'TestPlayer' }] as unknown as ReturnType<typeof useAppShellStore.getState>['availablePlayers'],
      locale: 'fr',
      authMode: 'none',
      isAdmin: false,
    })
  })

  // Refonte nav L1 (Phase 4 Prestige, commit bde179c8) :
  // - Palmares renomme en "Communaute"
  // - Synthese devenue sous-onglet de Stats (pas L1)
  // - Nouvelle entree L1 "Objectifs"

  it('affiche Communauté dans la navigation principale', () => {
    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Communauté' })).toBeInTheDocument()
  })

  it('marque Communauté actif sur les sous-routes du hub', () => {
    mockPathname = '/players/test-player/palmares/relations'

    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Communauté' })).toHaveAttribute('aria-current', 'page')
  })

  it('place Communauté avant Médias et après Objectifs dans la L1', () => {
    renderWithProviders(<NavL1 />)

    const objectifsLink = screen.getByRole('link', { name: 'Objectifs' })
    const communauteLink = screen.getByRole('link', { name: 'Communauté' })
    const mediaLink = screen.getByRole('link', { name: 'Médias' })

    expect(
      objectifsLink.compareDocumentPosition(communauteLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      communauteLink.compareDocumentPosition(mediaLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it('affiche Stats (parent de Synthèse) dans la navigation principale', () => {
    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Stats' })).toBeInTheDocument()
  })

  it('marque Stats actif sur la route /synthesis (Synthèse est sous-onglet de Stats)', () => {
    mockPathname = '/players/test-player/synthesis'

    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Stats' })).toHaveAttribute('aria-current', 'page')
  })

  it('place Objectifs entre Escouade et Communauté', () => {
    renderWithProviders(<NavL1 />)

    const escouadeLink = screen.getByRole('link', { name: 'Escouade' })
    const objectifsLink = screen.getByRole('link', { name: 'Objectifs' })
    const communauteLink = screen.getByRole('link', { name: 'Communauté' })

    expect(
      escouadeLink.compareDocumentPosition(objectifsLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      objectifsLink.compareDocumentPosition(communauteLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })
})
