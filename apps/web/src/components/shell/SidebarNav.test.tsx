/**
 * Tests — SidebarNav, barre latérale de l'application de bureau.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAssetDrawerStore } from '@/features/asset-drawer'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer'

import { SidebarNav } from './SidebarNav'

let mockPathname = '/t/halo_infinite/players/test-player/stats/sessions'

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
    useNavigate: () => vi.fn(),
    useRouterState: () => ({ location: { pathname: mockPathname } }),
  }
})

vi.mock('./ThemeToggle', () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}))

type State = ReturnType<typeof useAppShellStore.getState>

describe('SidebarNav', () => {
  beforeEach(() => {
    mockPathname = '/t/halo_infinite/players/test-player/stats/sessions'
    useAssetDrawerStore.setState({ isOpen: false, search: '', focusToken: 0 })
    useFeedbackDrawerStore.setState({ isOpen: false })
    useAppShellStore.setState({
      currentPlayer: { player_slug: 'test-player', gamertag: 'TestPlayer' } as unknown as State['currentPlayer'],
      availablePlayers: [{ player_slug: 'test-player', gamertag: 'TestPlayer' }] as unknown as State['availablePlayers'],
      locale: 'fr',
      currentTitleSlug: 'halo_infinite',
      availableTitles: [],
      currentUsername: null,
      appInfo: { version: '8.1.0', data_root: 'C:\\Donnees' },
    })
  })

  it('est la navigation principale et porte les sections du tiroir mobile', () => {
    renderWithProviders(<SidebarNav />)
    expect(screen.getByRole('complementary', { name: 'Navigation principale' })).toBeInTheDocument()
    const sections = screen.getByRole('navigation', { name: 'Sections' })
    for (const label of ['Accueil', 'Solo', 'Escouade', 'Carrière', 'Ascension', 'Communauté', 'Médias', 'Explorer']) {
      expect(within(sections).getByRole('link', { name: label })).toBeInTheDocument()
    }
  })

  it('déplie les onglets de la section active et marque l’onglet courant', () => {
    renderWithProviders(<SidebarNav />)
    expect(screen.getByRole('link', { name: 'Sessions' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Synthèse' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Synergies' })).not.toBeInTheDocument()
  })

  it('titre sans citations calculées : l’onglet Commendations est actif sur la page des commendations', () => {
    mockPathname = '/t/halo_5/players/test-player/career/commendations'
    useAppShellStore.setState({
      currentTitleSlug: 'halo_5',
      availableTitles: [
        { slug: 'halo_5', name: 'Halo 5', status: 'active', capabilities: ['career'], is_default: false, effective_hp_to_kill: 225 },
      ] as unknown as State['availableTitles'],
    })
    renderWithProviders(<SidebarNav />)
    const tab = screen.getByRole('link', { name: 'Citations' })
    expect(tab).toHaveAttribute('href', '/t/halo_5/players/test-player/career/commendations')
    expect(tab).toHaveAttribute('aria-current', 'page')
  })

  it('le pied de barre porte Aide, Paramètres, la bascule de thème, À propos et la version', () => {
    renderWithProviders(<SidebarNav />)
    const foot = screen.getByRole('group', { name: /Aide, paramètres/ })
    expect(within(foot).getByRole('link', { name: 'Aide' })).toBeInTheDocument()
    expect(within(foot).getByRole('link', { name: 'Paramètres' })).toBeInTheDocument()
    expect(within(foot).getByTestId('theme-toggle')).toBeInTheDocument()
    expect(within(foot).getByRole('button', { name: 'À propos' })).toBeInTheDocument()
    expect(screen.getByTestId('sidebar-version')).toHaveTextContent('Version 8.1.0')
  })

  it('« Référentiels » bascule le volet ancré, annonce son état et porte le raccourci en infobulle', () => {
    renderWithProviders(<SidebarNav />)
    const trigger = within(screen.getByRole('group', { name: /Aide, paramètres/ })).getByRole('button', {
      name: 'Référentiels',
    })
    expect(trigger).toHaveAttribute('title', 'Référentiels (Ctrl+Maj+L)')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(trigger)
    expect(useAssetDrawerStore.getState().isOpen).toBe(true)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    expect(trigger).toHaveAttribute('aria-controls', 'asset-docked-panel')
    fireEvent.click(trigger)
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
  })

  it('« Signaler un problème » ouvre le formulaire de retour', () => {
    renderWithProviders(<SidebarNav />)
    const foot = screen.getByRole('group', { name: /Aide, paramètres/ })
    fireEvent.click(within(foot).getByRole('button', { name: 'Signaler un problème' }))
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
  })

  it('ne porte rien de ce qui n’a de sens que sur un site', () => {
    useAppShellStore.setState({ currentUsername: 'quelquun' })
    renderWithProviders(<SidebarNav />)
    expect(screen.queryByRole('button', { name: /Se déconnecter/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Application Windows/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /CSinsight/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Développeur/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('contentinfo')).not.toBeInTheDocument()
  })

  it('« À propos » ouvre la fenêtre modale, qui se referme', () => {
    renderWithProviders(<SidebarNav />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'À propos' }))
    expect(screen.getByRole('dialog', { name: 'LevelUp' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Fermer' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('sans joueur actif, pas de sections ni de ligne joueur, mais le pied de barre reste', () => {
    useAppShellStore.setState({ currentPlayer: null, availablePlayers: [] })
    renderWithProviders(<SidebarNav />)
    expect(screen.queryByRole('link', { name: 'Accueil' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'À propos' })).toBeInTheDocument()
  })

  it('suit la langue de l’interface', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SidebarNav />)
    expect(screen.getByRole('button', { name: 'About' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Help' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'References' })).toHaveAttribute('title', 'References (Ctrl+Shift+L)')
    expect(screen.getByRole('button', { name: 'Report a problem' })).toBeInTheDocument()
    expect(screen.getByTestId('sidebar-version')).toHaveTextContent('Version 8.1.0')
  })
})
