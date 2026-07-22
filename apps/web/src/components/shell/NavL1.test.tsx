import type { ComponentPropsWithoutRef } from 'react'

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'

import { NavL1 } from './NavL1'

let mockPathname = '/t/halo_infinite/players/test-player/home'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({
      children,
      to,
      params,
      ...props
    }: ComponentPropsWithoutRef<'a'> & {
      to: string
      params?: Record<string, string | undefined>
    }) => (
      <a href={resolveRoutePath(to, params)} {...props}>
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
    mockPathname = '/t/halo_infinite/players/test-player/home'
    // Casts vers `unknown` puis le type cible : évite `as any` (lint strict).
    // Les fixtures volontairement minimales ne représentent pas le full PlayerProfile.
    useAppShellStore.setState({
      currentPlayer: { player_slug: 'test-player', gamertag: 'TestPlayer' } as unknown as ReturnType<typeof useAppShellStore.getState>['currentPlayer'],
      availablePlayers: [{ player_slug: 'test-player', gamertag: 'TestPlayer' }] as unknown as ReturnType<typeof useAppShellStore.getState>['availablePlayers'],
      locale: 'fr',
      authMode: 'none',
      isAdmin: false,
      // Baseline fail-open : titre courant introuvable dans availableTitles → toutes
      // les sections visibles. Les tests de gating posent un titre partiel explicite.
      currentTitleSlug: 'halo_infinite',
      availableTitles: [],
    })
  })

  // Refonte nav L1 (Phase 4 Prestige, commit bde179c8) :
  // - Palmares renomme en "Communaute"
  // - Synthese devenue sous-onglet de Stats (pas L1)
  // - Nouvelle entree L1 "Ascension" (renomme depuis "Objectifs" — V1 commit-8)

  it('affiche Communauté dans la navigation principale', () => {
    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Communauté' })).toBeInTheDocument()
  })

  it('marque Communauté actif sur les sous-routes du hub', () => {
    mockPathname = '/t/halo_infinite/players/test-player/community/relations'

    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Communauté' })).toHaveAttribute('aria-current', 'page')
  })

  it('place Communauté avant Médias et après Ascension dans la L1', () => {
    renderWithProviders(<NavL1 />)

    const objectifsLink = screen.getByRole('link', { name: 'Ascension' })
    const communauteLink = screen.getByRole('link', { name: 'Communauté' })
    const mediaLink = screen.getByRole('link', { name: 'Médias' })

    expect(
      objectifsLink.compareDocumentPosition(communauteLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      communauteLink.compareDocumentPosition(mediaLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it('affiche Solo (parent de Synthèse) dans la navigation principale', () => {
    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Solo' })).toBeInTheDocument()
  })

  it('marque Solo actif sur la route /synthesis (Synthèse est sous-onglet de Solo)', () => {
    mockPathname = '/t/halo_infinite/players/test-player/synthesis'

    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Solo' })).toHaveAttribute('aria-current', 'page')
  })

  it('place Ascension entre Escouade et Communauté', () => {
    renderWithProviders(<NavL1 />)

    const escouadeLink = screen.getByRole('link', { name: 'Escouade' })
    const objectifsLink = screen.getByRole('link', { name: 'Ascension' })
    const communauteLink = screen.getByRole('link', { name: 'Communauté' })

    expect(
      escouadeLink.compareDocumentPosition(objectifsLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      objectifsLink.compareDocumentPosition(communauteLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  it('expose l\'onglet Entraînement dans la dropdown Ascension', () => {
    renderWithProviders(<NavL1 />)

    fireEvent.click(screen.getByRole('button', { name: 'Onglets Ascension' }))

    const coaching = screen.getByRole('menuitem', { name: 'Entraînement' })
    expect(coaching).toBeInTheDocument()
    expect(coaching).toHaveAttribute('href', '/t/halo_infinite/players/test-player/ascension/coaching')
  })

  it('ordonne Profil / Objectifs / Entraînement / Réalisations dans la dropdown Ascension', () => {
    renderWithProviders(<NavL1 />)

    fireEvent.click(screen.getByRole('button', { name: 'Onglets Ascension' }))

    const profile = screen.getByRole('menuitem', { name: 'Profil' })
    const objectives = screen.getByRole('menuitem', { name: 'Objectifs' })
    const coaching = screen.getByRole('menuitem', { name: 'Entraînement' })
    const realisations = screen.getByRole('menuitem', { name: 'Réalisations' })

    expect(objectives).toHaveAttribute('href', '/t/halo_infinite/players/test-player/ascension/objectifs')
    expect(
      profile.compareDocumentPosition(objectives) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      objectives.compareDocumentPosition(coaching) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      coaching.compareDocumentPosition(realisations) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })

  // ─── Gating capabilities multi-titre (Phase 5) ────────────────────────────
  // Le titre courant (halo_infinite par défaut) déclare toutes les capabilities
  // → aucune section masquée. Un titre partiel masque les sections gatées.

  function setPartialTitle(capabilities: string[]) {
    useAppShellStore.setState({
      currentTitleSlug: 'partial',
      availableTitles: [
        { slug: 'partial', name: 'Partial', status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
      ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
    })
  }

  it('masque Médias et Carrière pour un titre sans ces capabilities', () => {
    setPartialTitle(['matchmaking', 'world.leaderboard'])

    renderWithProviders(<NavL1 />)

    expect(screen.queryByRole('link', { name: 'Médias' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Carrière' })).not.toBeInTheDocument()
    // Sections transverses toujours présentes (non gatées).
    expect(screen.getByRole('link', { name: 'Accueil' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Communauté' })).toBeInTheDocument()
  })

  it('masque Ascension pour un titre sans capability lusr', () => {
    setPartialTitle(['matchmaking', 'career', 'media'])

    renderWithProviders(<NavL1 />)

    expect(screen.queryByRole('link', { name: 'Ascension' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Carrière' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Médias' })).toBeInTheDocument()
  })
})
