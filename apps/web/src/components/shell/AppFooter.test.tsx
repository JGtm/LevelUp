/**
 * Tests composant — AppFooter (pied de page projet + soutien).
 */
import { describe, it, expect, afterEach, vi } from 'vitest'
import type { ComponentPropsWithoutRef } from 'react'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer/feedbackDrawer.store'
import { AppFooter } from './AppFooter'
import { GITHUB_URL, GITHUB_PROFILE_URL, SPONSORS_URL, PAYPAL_URL } from '@/lib/appLinks'

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

function hrefOf(name: RegExp): string | null {
  return screen.getByRole('link', { name }).getAttribute('href')
}

describe('AppFooter', () => {
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
    useFeedbackDrawerStore.setState({ isOpen: false })
  })

  it('variante complète : liens projet + soutien, cibles externes correctes', () => {
    renderWithProviders(<AppFooter />)
    expect(hrefOf(/Code source/i)).toBe(GITHUB_URL)
    expect(hrefOf(/GitHub Sponsors/i)).toBe(SPONSORS_URL)
    expect(hrefOf(/PayPal/i)).toBe(PAYPAL_URL)
    expect(hrefOf(/Nouveautés/i)).toBe('/changelog')
    expect(hrefOf(/Confidentialité/i)).toBe('/privacy')
    expect(hrefOf(/Le développeur/i)).toBe(GITHUB_PROFILE_URL)
    expect(screen.getByText(/sans lien avec Microsoft/i)).toBeInTheDocument()
  })

  it('le lien CSinsight suit la locale de l’app', () => {
    const { unmount } = renderWithProviders(<AppFooter />)
    expect(hrefOf(/CSinsight/i)).toBe('https://csinsight.eu/fr')
    unmount()

    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<AppFooter />)
    expect(hrefOf(/CSinsight/i)).toBe('https://csinsight.eu/en')
  })

  it('les liens externes portent rel="noopener noreferrer" et target="_blank"', () => {
    renderWithProviders(<AppFooter />)
    const sponsors = screen.getByRole('link', { name: /GitHub Sponsors/i })
    expect(sponsors).toHaveAttribute('target', '_blank')
    expect(sponsors).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('« Signaler un problème » ouvre le tiroir de feedback', () => {
    renderWithProviders(<AppFooter />)
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(false)
    fireEvent.click(screen.getByRole('button', { name: /Signaler un problème/i }))
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
  })

  it('variante minimale : confidentialité + soutien, pas de lien vers le tiroir de feedback', () => {
    renderWithProviders(<AppFooter variant="minimal" />)
    // L'écran de connexion est le seul que voit un visiteur anonyme : le lien
    // confidentialité DOIT y être, sinon la page est inatteignable sans compte.
    expect(hrefOf(/Confidentialité/i)).toBe('/privacy')
    expect(hrefOf(/GitHub Sponsors/i)).toBe(SPONSORS_URL)
    expect(hrefOf(/PayPal/i)).toBe(PAYPAL_URL)
    expect(hrefOf(/CSinsight/i)).toBe('https://csinsight.eu/fr')
    expect(screen.queryByRole('button', { name: /Signaler un problème/i })).toBeNull()
  })

  it('rend les libellés en anglais quand locale=en', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<AppFooter />)
    expect(screen.getByText(/not affiliated with Microsoft/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Source code/i })).toBeInTheDocument()
  })
})
