/**
 * NavL1 — fermeture des deux menus à onglets (section et Paramètres) : Échap ferme le menu ouvert, un appui
 * hors du menu aussi ; deux menus ouverts se ferment un par un, le dernier ouvert d'abord.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'

import { NavL1 } from './NavL1'

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
    useRouterState: () => ({ location: { pathname: '/t/halo_infinite/players/test-player/home' } }),
  }
})

vi.mock('./ThemeToggle', () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}))

const PARAMETRES = 'Onglets Paramètres'

/** Le déclencheur du menu des onglets d'une section (ni Paramètres ni Aide). */
function declencheurSection(): HTMLElement {
  const [premier] = screen
    .getAllByRole('button', { name: /^Onglets / })
    .filter((b) => b.getAttribute('aria-label') !== PARAMETRES && b.getAttribute('aria-label') !== 'Onglets Aide')
  return premier
}

describe('NavL1 — fermeture des menus à onglets', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      currentPlayer: { player_slug: 'test-player', gamertag: 'TestPlayer' } as unknown as ReturnType<typeof useAppShellStore.getState>['currentPlayer'],
      availablePlayers: [{ player_slug: 'test-player', gamertag: 'TestPlayer' }] as unknown as ReturnType<typeof useAppShellStore.getState>['availablePlayers'],
      locale: 'fr',
      isAdmin: false,
      currentTitleSlug: 'halo_infinite',
      availableTitles: [],
    })
  })

  it('menu d’une section : Échap ferme, un appui dehors ferme, un appui dedans non', () => {
    renderWithProviders(<NavL1 />)
    const section = declencheurSection()
    fireEvent.click(section)
    expect(section).toHaveAttribute('aria-expanded', 'true')
    fireEvent.mouseDown(screen.getAllByRole('menu')[0])
    expect(section).toHaveAttribute('aria-expanded', 'true')
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(section).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(section)
    fireEvent.mouseDown(document.body)
    expect(section).toHaveAttribute('aria-expanded', 'false')
  })

  it('menu Paramètres : Échap ferme, un appui dehors ferme', () => {
    renderWithProviders(<NavL1 />)
    const parametres = screen.getByRole('button', { name: PARAMETRES })
    fireEvent.click(parametres)
    expect(parametres).toHaveAttribute('aria-expanded', 'true')
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(parametres).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(parametres)
    fireEvent.mouseDown(document.body)
    expect(parametres).toHaveAttribute('aria-expanded', 'false')
  })

  it('deux menus ouverts : la première frappe d’Échap ferme le dernier ouvert seul, la suivante l’autre', () => {
    renderWithProviders(<NavL1 />)
    const section = declencheurSection()
    const parametres = screen.getByRole('button', { name: PARAMETRES })
    fireEvent.click(section)
    fireEvent.click(parametres)
    expect(section).toHaveAttribute('aria-expanded', 'true')
    expect(parametres).toHaveAttribute('aria-expanded', 'true')

    fireEvent.keyDown(window, { key: 'Escape' })
    expect(parametres).toHaveAttribute('aria-expanded', 'false')
    expect(section).toHaveAttribute('aria-expanded', 'true')

    fireEvent.keyDown(window, { key: 'Escape' })
    expect(section).toHaveAttribute('aria-expanded', 'false')
  })
})
