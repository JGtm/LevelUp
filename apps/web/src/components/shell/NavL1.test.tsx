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
    useAppShellStore.setState({
      currentPlayer: { player_slug: 'test-player', gamertag: 'TestPlayer' } as any,
      availablePlayers: [{ player_slug: 'test-player', gamertag: 'TestPlayer' }] as any,
      locale: 'fr',
      authMode: 'none',
      isAdmin: false,
    })
  })

  it('affiche Palmarès dans la navigation principale', () => {
    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Palmarès' })).toBeInTheDocument()
  })

  it('marque Palmarès actif sur les sous-routes du hub', () => {
    mockPathname = '/players/test-player/palmares/relations'

    renderWithProviders(<NavL1 />)

    expect(screen.getByRole('link', { name: 'Palmarès' })).toHaveAttribute('aria-current', 'page')
  })

  it('place Palmarès en avant-dernier dans la L1', () => {
    renderWithProviders(<NavL1 />)

    const mediaLink = screen.getByRole('link', { name: 'Médias' })
    const palmaresLink = screen.getByRole('link', { name: 'Palmarès' })
    const careerLink = screen.getByRole('link', { name: 'Carrière' })

    expect(
      mediaLink.compareDocumentPosition(palmaresLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      palmaresLink.compareDocumentPosition(careerLink) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  })
})
