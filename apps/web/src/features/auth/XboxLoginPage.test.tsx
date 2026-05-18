/**
 * Tests composant — XboxLoginPage.
 *
 * Smoke + flow Device Code (MSW mocke /auth/device-flow/start et /:attemptId).
 * Couvre aussi le toggle "Connexion admin (mot de passe)".
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { XboxLoginPage } from './XboxLoginPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  }
})

describe('XboxLoginPage', () => {
  const resetStores = () => {
    useAppShellStore.setState({
      authMode: 'xbox',
      currentUsername: null,
      firstLaunch: false,
      isBootstrapped: true,
    })
  }

  beforeEach(resetStores)
  afterEach(resetStores)

  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<XboxLoginPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le user_code retourné par le Device Code Flow', async () => {
    renderWithProviders(<XboxLoginPage />)
    await waitFor(() => {
      expect(screen.getByText(/ABCD-1234/i)).toBeInTheDocument()
    })
  })

  it('affiche le lien microsoft.com/link', async () => {
    renderWithProviders(<XboxLoginPage />)
    await waitFor(() => {
      const links = screen.getAllByRole('link')
      const msLink = links.find((a) => a.getAttribute('href')?.includes('microsoft.com/link'))
      expect(msLink).toBeDefined()
    })
  })

  it('affiche le disclaimer anti-phishing', async () => {
    renderWithProviders(<XboxLoginPage />)
    await waitFor(() => {
      // Texte du disclaimer
      expect(screen.getByText(/ne saisis ce code que si tu/i)).toBeInTheDocument()
    })
  })

  it('toggle "Connexion admin" affiche le form password', async () => {
    renderWithProviders(<XboxLoginPage />)

    // Le bouton "Connexion admin" est visible après le rendu du panel Xbox.
    await waitFor(() => {
      expect(screen.getByText(/Connexion admin/i)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText(/Connexion admin/i))

    // Le panel admin password apparaît avec son avertissement.
    await waitFor(() => {
      expect(screen.getByText(/connexion admin uniquement/i)).toBeInTheDocument()
    })
    // Champs de form présents.
    expect(screen.getByLabelText(/Identifiant/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Mot de passe/i)).toBeInTheDocument()
  })

  it('toggle "Retour à la connexion Xbox" depuis le panel admin', async () => {
    renderWithProviders(<XboxLoginPage />)
    await waitFor(() => {
      expect(screen.getByText(/Connexion admin/i)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText(/Connexion admin/i))
    await waitFor(() => {
      expect(screen.getByText(/Retour à la connexion Xbox/i)).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText(/Retour à la connexion Xbox/i))
    await waitFor(() => {
      // De retour : le user_code est de nouveau visible.
      expect(screen.getByText(/ABCD-1234/i)).toBeInTheDocument()
    })
  })
})
