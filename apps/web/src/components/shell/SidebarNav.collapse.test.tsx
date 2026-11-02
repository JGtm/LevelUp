/**
 * Tests — barre latérale repliable en icônes (L21.0) : trois états (auto, replié manuel, déplié manuel),
 * mémorisation, repli automatique d'une fenêtre étroite sans écraser un choix manuel, raccourci Ctrl+B,
 * infobulles au survol et au focus, menu volant des onglets (clavier compris), sigle et initiale.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { act, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAssetDrawerStore } from '@/features/asset-drawer'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer'

import { SidebarNav } from './SidebarNav'
import { SIDEBAR_AUTO_COLLAPSE_QUERY, SIDEBAR_PREFERENCE_STORAGE_KEY } from './sidebarCollapse'

const pathname = '/t/halo_infinite/players/test-player/stats/sessions'

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
    useRouterState: () => ({ location: { pathname } }),
  }
})

vi.mock('./ThemeToggle', () => ({ ThemeToggle: () => <div data-testid="theme-toggle" /> }))

type State = ReturnType<typeof useAppShellStore.getState>

/** matchMedia pilotable : la fenêtre est étroite ou non, et un changement de largeur se joue à la main. */
function installMatchMedia(initialNarrow: boolean) {
  let narrow = initialNarrow
  const listeners = new Set<(e: MediaQueryListEvent) => void>()
  window.matchMedia = ((query: string) =>
    ({
      get matches() {
        return query === SIDEBAR_AUTO_COLLAPSE_QUERY ? narrow : false
      },
      media: query,
      onchange: null,
      addEventListener: (_type: string, cb: (e: MediaQueryListEvent) => void) => listeners.add(cb),
      removeEventListener: (_type: string, cb: (e: MediaQueryListEvent) => void) => listeners.delete(cb),
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }) as unknown as MediaQueryList) as typeof window.matchMedia
  return {
    resize(next: boolean) {
      narrow = next
      act(() => listeners.forEach((cb) => cb({ matches: next } as MediaQueryListEvent)))
    },
  }
}

const sidebar = () => screen.getByRole('complementary', { name: 'Navigation principale' })
const state = () => sidebar().getAttribute('data-sidebar')

describe('SidebarNav — repliement en icônes', () => {
  const originalMatchMedia = window.matchMedia
  beforeEach(() => {
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
  afterEach(() => {
    window.matchMedia = originalMatchMedia
  })

  it('largeur confortable, rien de mémorisé : déplié ; le bouton du pied replie et mémorise', () => {
    installMatchMedia(false)
    renderWithProviders(<SidebarNav />)
    expect(state()).toBe('expanded')
    const collapse = screen.getByRole('button', { name: 'Replier la barre latérale' })
    expect(collapse).toHaveAttribute('title', 'Replier la barre latérale (Ctrl+B)')

    fireEvent.click(collapse)

    expect(state()).toBe('collapsed')
    expect(localStorage.getItem(SIDEBAR_PREFERENCE_STORAGE_KEY)).toBe('collapsed')
    expect(screen.getByRole('button', { name: 'Déplier la barre latérale' })).toBeInTheDocument()
  })

  it('un choix manuel survit au rechargement', () => {
    installMatchMedia(false)
    localStorage.setItem(SIDEBAR_PREFERENCE_STORAGE_KEY, 'collapsed')
    renderWithProviders(<SidebarNav />)
    expect(state()).toBe('collapsed')
  })

  it('Ctrl+B bascule ; Ctrl+Maj+B ne fait rien', () => {
    installMatchMedia(false)
    renderWithProviders(<SidebarNav />)
    fireEvent.keyDown(document, { key: 'b', ctrlKey: true, shiftKey: true })
    expect(state()).toBe('expanded')
    fireEvent.keyDown(document, { key: 'b', ctrlKey: true })
    expect(state()).toBe('collapsed')
    fireEvent.keyDown(document, { key: 'B', ctrlKey: true })
    expect(state()).toBe('expanded')
  })

  it('fenêtre étroite : replié d’office ; il se redéplie quand la fenêtre s’élargit', () => {
    const media = installMatchMedia(true)
    renderWithProviders(<SidebarNav />)
    expect(state()).toBe('collapsed')
    expect(localStorage.getItem(SIDEBAR_PREFERENCE_STORAGE_KEY)).toBeNull()

    media.resize(false)
    expect(state()).toBe('expanded')
    media.resize(true)
    expect(state()).toBe('collapsed')
  })

  it('un choix manuel n’est jamais écrasé par la largeur de la fenêtre', () => {
    const media = installMatchMedia(true)
    renderWithProviders(<SidebarNav />)
    // Déplié à la main sur une fenêtre étroite : il le reste quand la fenêtre change.
    fireEvent.click(screen.getByRole('button', { name: 'Déplier la barre latérale' }))
    expect(state()).toBe('expanded')
    media.resize(false)
    media.resize(true)
    expect(state()).toBe('expanded')

    // Replié à la main sur une fenêtre large : il le reste.
    fireEvent.click(screen.getByRole('button', { name: 'Replier la barre latérale' }))
    media.resize(false)
    expect(state()).toBe('collapsed')
  })

  it('ne se replie jamais au défilement', () => {
    installMatchMedia(false)
    renderWithProviders(<SidebarNav />)
    fireEvent.scroll(window, { target: { scrollY: 800 } })
    fireEvent.scroll(screen.getByRole('navigation', { name: 'Sections' }), { target: { scrollTop: 400 } })
    expect(state()).toBe('expanded')
  })

  describe('barre d’icônes', () => {
    beforeEach(() => {
      installMatchMedia(false)
      localStorage.setItem(SIDEBAR_PREFERENCE_STORAGE_KEY, 'collapsed')
    })

    it('une entrée par section, sans texte visible, et l’initiale du joueur', () => {
      renderWithProviders(<SidebarNav />)
      const nav = screen.getByRole('navigation', { name: 'Sections' })
      expect(within(nav).getByRole('link', { name: 'Accueil' })).toHaveAttribute('href', '/t/halo_infinite/players/test-player/home')
      for (const label of ['Solo', 'Escouade', 'Carrière', 'Ascension', 'Communauté']) {
        expect(within(nav).getByRole('button', { name: label })).toHaveAttribute('aria-haspopup', 'menu')
      }
      expect(within(nav).getByRole('link', { name: 'Médias' })).toBeInTheDocument()
      expect(within(nav).getByRole('link', { name: 'Explorer' })).toBeInTheDocument()
      // Aucune étiquette visible : seules les icônes (aria-hidden) sont rendues.
      expect(within(nav).queryByText('Accueil')).not.toBeInTheDocument()
      expect(screen.getByRole('img', { name: 'TestPlayer' })).toHaveTextContent('T')
      expect(screen.queryByTestId('sidebar-version')).not.toBeInTheDocument()
    })

    it('infobulle au survol et au focus, retirée quand le pointeur ou le focus part', () => {
      renderWithProviders(<SidebarNav />)
      const home = screen.getByRole('link', { name: 'Accueil' })
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()

      fireEvent.mouseEnter(home.parentElement as HTMLElement)
      expect(screen.getByRole('tooltip')).toHaveTextContent('Accueil')
      fireEvent.mouseLeave(home.parentElement as HTMLElement)
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()

      fireEvent.focus(home)
      expect(screen.getByRole('tooltip')).toHaveTextContent('Accueil')
      fireEvent.blur(home)
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    })

    it('le survol d’une section à onglets ouvre un menu volant qui les liste', () => {
      renderWithProviders(<SidebarNav />)
      const solo = screen.getByRole('button', { name: 'Solo' })
      fireEvent.mouseEnter(solo.parentElement as HTMLElement)

      const menu = screen.getByRole('menu', { name: 'Solo' })
      expect(solo).toHaveAttribute('aria-expanded', 'true')
      const sessions = within(menu).getByRole('menuitem', { name: 'Sessions' })
      expect(sessions).toHaveAttribute('href', '/t/halo_infinite/players/test-player/stats/sessions')
      expect(sessions).toHaveAttribute('aria-current', 'page')
      expect(within(menu).getByRole('menuitem', { name: 'Synthèse' })).toBeInTheDocument()

      fireEvent.mouseLeave(solo.parentElement as HTMLElement)
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    })

    it('un clic épingle le menu (le pointeur peut partir) ; un clic ailleurs le ferme', () => {
      renderWithProviders(<SidebarNav />)
      const solo = screen.getByRole('button', { name: 'Solo' })
      fireEvent.mouseEnter(solo.parentElement as HTMLElement)
      fireEvent.click(solo, { detail: 1 })
      fireEvent.mouseLeave(solo.parentElement as HTMLElement)
      expect(screen.getByRole('menu', { name: 'Solo' })).toBeInTheDocument()

      fireEvent.mouseDown(document.body)
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    })

    it('au clavier : flèche basse ouvre et donne le focus, flèches parcourent, Échap ferme et rend le focus', () => {
      renderWithProviders(<SidebarNav />)
      const solo = screen.getByRole('button', { name: 'Solo' })
      solo.focus()
      fireEvent.focus(solo)
      expect(screen.getByRole('tooltip')).toHaveTextContent('Solo')

      fireEvent.keyDown(solo, { key: 'ArrowDown' })
      const menu = screen.getByRole('menu', { name: 'Solo' })
      const items = within(menu).getAllByRole('menuitem')
      expect(items.length).toBeGreaterThan(1)
      expect(items[0]).toHaveFocus()

      fireEvent.keyDown(menu, { key: 'ArrowDown' })
      expect(items[1]).toHaveFocus()
      fireEvent.keyDown(menu, { key: 'ArrowUp' })
      fireEvent.keyDown(menu, { key: 'ArrowUp' })
      expect(items[items.length - 1]).toHaveFocus()

      fireEvent.keyDown(menu, { key: 'Escape' })
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()
      expect(solo).toHaveFocus()
    })

    it('Entrée (clic clavier) ouvre le menu, un second clic clavier le ferme', () => {
      renderWithProviders(<SidebarNav />)
      const community = screen.getByRole('button', { name: 'Communauté' })
      fireEvent.click(community, { detail: 0 })
      expect(screen.getByRole('menu', { name: 'Communauté' })).toBeInTheDocument()
      fireEvent.click(community, { detail: 0 })
      expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    })

    it('le pied de barre garde les mêmes actions, en icônes, et le bouton qui déplie', () => {
      renderWithProviders(<SidebarNav />)
      const foot = screen.getByRole('group', { name: /Aide, paramètres/ })
      expect(within(foot).getByRole('link', { name: 'Aide' })).toBeInTheDocument()
      expect(within(foot).getByRole('link', { name: 'Paramètres' })).toBeInTheDocument()
      expect(within(foot).getByRole('switch', { name: 'Thème' })).toBeInTheDocument()

      fireEvent.click(within(foot).getByRole('button', { name: 'Référentiels' }))
      expect(useAssetDrawerStore.getState().isOpen).toBe(true)
      fireEvent.click(within(foot).getByRole('button', { name: 'Signaler un problème' }))
      expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
      fireEvent.click(within(foot).getByRole('button', { name: 'À propos' }))
      expect(screen.getByRole('dialog', { name: 'LevelUp' })).toBeInTheDocument()
      fireEvent.click(screen.getByRole('button', { name: 'Fermer' }))

      fireEvent.click(within(foot).getByRole('button', { name: 'Déplier la barre latérale' }))
      expect(state()).toBe('expanded')
    })

    it('plusieurs jeux : le sigle du jeu courant ouvre le menu des jeux', () => {
      useAppShellStore.setState({
        availableTitles: [
          { slug: 'halo_infinite', name: 'Halo Infinite', status: 'active', capabilities: [], is_default: true, effective_hp_to_kill: 225 },
          { slug: 'halo_5', name: 'Halo 5', status: 'active', capabilities: [], is_default: false, effective_hp_to_kill: 225 },
        ] as unknown as State['availableTitles'],
      })
      renderWithProviders(<SidebarNav />)
      const game = screen.getByRole('button', { name: 'Jeu' })
      expect(game).toHaveTextContent('HI')
      fireEvent.mouseEnter(game.parentElement as HTMLElement)
      const menu = screen.getByRole('menu', { name: 'Jeu' })
      expect(within(menu).getByRole('menuitemradio', { name: 'Halo Infinite' })).toHaveAttribute('aria-checked', 'true')
      expect(within(menu).getByRole('menuitemradio', { name: 'Halo 5' })).toHaveAttribute('aria-checked', 'false')
    })

    it('suit la langue de l’interface', () => {
      useAppShellStore.setState({ locale: 'en' })
      renderWithProviders(<SidebarNav />)
      expect(screen.getByRole('button', { name: 'Expand the sidebar' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Career' })).toBeInTheDocument()
    })
  })
})
