/**
 * NavL1MobileActions.test.tsx — menu mobile « compte & outils » (kebab droite).
 */
import type { ComponentPropsWithoutRef } from 'react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAssetDrawerStore } from '@/features/asset-drawer/assetDrawer.store'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer/feedbackDrawer.store'

import { NavL1MobileActions, type SettingsTabItem } from './NavL1MobileActions'

const mockMutate = vi.fn()

vi.mock('@/features/auth/queries', () => ({
  useLogout: () => ({ mutate: mockMutate, isPending: false }),
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

const SETTINGS_TABS: SettingsTabItem[] = [
  { key: 'appearance', label: 'Apparence', tab: 'appearance' },
  { key: 'analyse', label: 'Analyse', tab: 'analyse' },
]

function setup(pathname = '/t/halo_infinite/players/test-player/home') {
  return renderWithProviders(<NavL1MobileActions settingsTabs={SETTINGS_TABS} pathname={pathname} isAdmin={false} />)
}

function openMenu() {
  fireEvent.click(screen.getByRole('button', { name: 'Ouvrir le menu compte et outils' }))
}

describe('NavL1MobileActions', () => {
  beforeEach(() => {
    mockMutate.mockClear()
    useAppShellStore.setState({ locale: 'fr', currentUsername: 'tester' })
    useAssetDrawerStore.setState({ isOpen: false })
    useFeedbackDrawerStore.setState({ isOpen: false })
  })

  it('affiche le bouton kebab', () => {
    setup()
    expect(
      screen.getByRole('button', { name: 'Ouvrir le menu compte et outils' }),
    ).toBeInTheDocument()
  })

  it('ouvre le panneau au clic', () => {
    setup()
    openMenu()
    expect(screen.getByRole('menu')).toHaveAttribute('aria-hidden', 'false')
  })

  it('rend le toggle thème et les onglets Paramètres', () => {
    setup()
    openMenu()
    expect(screen.getByTestId('theme-toggle')).toBeInTheDocument()
    const appearance = screen.getByRole('menuitem', { name: 'Apparence' })
    expect(appearance).toHaveAttribute('href', '/settings')
  })

  it('rend les outils latéraux (Référentiels, Feedback)', () => {
    setup()
    openMenu()
    expect(screen.getByRole('menuitem', { name: 'Référentiels' })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: 'Envoyer un retour' })).toBeInTheDocument()
  })

  it('ouvre le drawer Référentiels via son store et ferme le menu', () => {
    setup()
    openMenu()
    fireEvent.click(screen.getByRole('menuitem', { name: 'Référentiels' }))
    expect(useAssetDrawerStore.getState().isOpen).toBe(true)
    expect(screen.getByRole('menu', { hidden: true })).toHaveAttribute('aria-hidden', 'true')
  })

  it('ouvre le drawer Feedback via son store', () => {
    setup()
    openMenu()
    fireEvent.click(screen.getByRole('menuitem', { name: 'Envoyer un retour' }))
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
  })

  it('déclenche la déconnexion quand une session est ouverte', () => {
    setup()
    openMenu()
    fireEvent.click(screen.getByRole('menuitem', { name: 'Se déconnecter' }))
    expect(mockMutate).toHaveBeenCalledTimes(1)
  })

  it('masque la déconnexion sans session', () => {
    useAppShellStore.setState({ currentUsername: null })
    setup()
    openMenu()
    expect(screen.queryByRole('menuitem', { name: 'Se déconnecter' })).not.toBeInTheDocument()
  })
})
