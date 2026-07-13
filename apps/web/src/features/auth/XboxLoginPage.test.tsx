/**
 * Tests composant — XboxLoginPage.
 *
 * Smoke + flow Device Code (MSW mocke /auth/device-flow/start et /:attemptId).
 * Couvre aussi le toggle "Connexion admin (mot de passe)".
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
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

  it('affiche une erreur + bouton Réessayer quand le start échoue (500)', async () => {
    // Garde-rail anti-régression (Lot A) : un start en 500 ne doit jamais
    // laisser le spinner « préparation » tourner indéfiniment.
    server.use(
      http.post('/api/v1/auth/device-flow/start', () =>
        HttpResponse.json(
          { code: 'msal_init_error', message: 'impossible de démarrer le Device Code Flow', retryable: false },
          { status: 500 },
        ),
      ),
    )

    renderWithProviders(<XboxLoginPage />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Réessayer/i })).toBeInTheDocument()
    })
    // Pas de code affiché : le flow n'a pas démarré.
    expect(screen.queryByText(/ABCD-1234/i)).not.toBeInTheDocument()
  })

  it('relance automatiquement le flow sur attempt_not_found (récupération gracieuse)', async () => {
    // 1er start → attempt-1 (que le polling déclarera introuvable) ;
    // 2e start (relance auto) → attempt-2 avec un nouveau code.
    let startCount = 0
    server.use(
      http.post('/api/v1/auth/device-flow/start', () => {
        startCount += 1
        return startCount === 1
          ? HttpResponse.json({ attempt_id: 'attempt-1', user_code: 'ABCD-1234', verification_uri: 'https://microsoft.com/link', expires_in: 900, poll_interval_sec: 5 })
          : HttpResponse.json({ attempt_id: 'attempt-2', user_code: 'WXYZ-9999', verification_uri: 'https://microsoft.com/link', expires_in: 900, poll_interval_sec: 5 })
      }),
      http.get('/api/v1/auth/device-flow/:attemptId', ({ params }) => {
        if (params.attemptId === 'attempt-1') {
          return HttpResponse.json({ code: 'attempt_not_found', message: 'tentative introuvable', retryable: false }, { status: 404 })
        }
        return HttpResponse.json({ attempt_id: 'attempt-2', status: 'pending', gamertag: null, xuid: null, error: null })
      }),
    )

    renderWithProviders(<XboxLoginPage />)

    // Le flow doit avoir été relancé et afficher le code de la NOUVELLE tentative.
    await waitFor(() => {
      expect(screen.getByText(/WXYZ-9999/i)).toBeInTheDocument()
    })
    expect(startCount).toBeGreaterThanOrEqual(2)
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
