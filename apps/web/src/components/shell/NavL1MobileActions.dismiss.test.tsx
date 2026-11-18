/**
 * NavL1MobileActions — fermeture : Échap ferme le panneau ouvert, et ne fait rien panneau fermé.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { NavL1MobileActions } from './NavL1MobileActions'

vi.mock('@/features/auth/queries', () => ({
  useLogout: () => ({ mutate: vi.fn(), isPending: false }),
}))

vi.mock('./ThemeToggle', () => ({
  ThemeToggle: () => <div data-testid="theme-toggle" />,
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ children, to, ...props }: ComponentPropsWithoutRef<'a'> & { to: string }) => (
      <a href={to} {...props}>
        {children}
      </a>
    ),
  }
})

const NOM_DECLENCHEUR = 'Ouvrir le menu compte et outils'

describe('NavL1MobileActions — fermeture', () => {
  beforeEach(() => useAppShellStore.setState({ locale: 'fr', currentUsername: 'tester' }))

  it('Échap ferme le panneau ouvert', () => {
    renderWithProviders(<NavL1MobileActions settingsTabs={[]} pathname="/home" isAdmin={false} />)
    const declencheur = screen.getByRole('button', { name: NOM_DECLENCHEUR })
    fireEvent.click(declencheur)
    expect(declencheur).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('menu')).toHaveAttribute('aria-hidden', 'false')
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(declencheur).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByRole('menu', { hidden: true })).toHaveAttribute('aria-hidden', 'true')
  })

  it('panneau fermé : Échap ne fait rien', () => {
    renderWithProviders(<NavL1MobileActions settingsTabs={[]} pathname="/home" isAdmin={false} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.getByRole('button', { name: NOM_DECLENCHEUR })).toHaveAttribute('aria-expanded', 'false')
  })
})
