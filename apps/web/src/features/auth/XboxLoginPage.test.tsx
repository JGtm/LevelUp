/**
 * Tests composant — XboxLoginPage.
 *
 * Smoke + flow Device Code (MSW mocke /auth/device-flow/start et /:attemptId).
 * Couvre aussi le toggle "Connexion admin (mot de passe)".
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, fireEvent, render } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { queryKeys } from '@/lib/query/keys'
import type { BootstrapResponse } from '@/lib/api/types'
import { XboxLoginPage } from './XboxLoginPage'

// Spy navigate STABLE (vi.hoisted → référençable dans la factory vi.mock) pour
// asserter la destination post-login (A3).
const { mockNavigate } = vi.hoisted(() => ({ mockNavigate: vi.fn() }))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

describe('XboxLoginPage', () => {
  const resetStores = () => {
    useAppShellStore.setState({
      authMode: 'xbox',
      currentUsername: null,
      firstLaunch: false,
      isBootstrapped: true,
      oauthCodeFlowEnabled: false, // device-code panel (XboxFlowPanel) → onAuthorized testable
    })
    mockNavigate.mockClear()
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
          { code: 'device_flow_init_error', message: 'impossible de démarrer le Device Code Flow', retryable: false },
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

  // A3 (revue 2026-07) — après un device-flow abouti, un joueur DÉJÀ établi doit
  // atterrir sur le DASHBOARD, jamais sur l'onboarding, même si le cache bootstrap
  // ANONYME (préchargé avant le login) est encore frais. fetchQuery({staleTime:0})
  // force la lecture réseau autoritative. Sans staleTime:0, fetchQuery renverrait le
  // bootstrap anonyme caché (défaut global 5 min) → onboarding forcé (régression).
  it('post-login: joueur établi → dashboard malgré un cache bootstrap anonyme frais', async () => {
    // Device-flow abouti immédiatement (status authorized).
    server.use(
      http.get('/api/v1/auth/device-flow/:attemptId', () =>
        HttpResponse.json({
          attempt_id: 'attempt-1',
          status: 'authorized',
          gamertag: 'TestPlayer',
          xuid: '0000000000000001',
          error: null,
        }),
      ),
    )

    // Client avec le staleTime de PROD (5 min) — le client de test par défaut a
    // staleTime 0 et ne reproduirait pas le bug.
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 5 * 60 * 1000 } },
    })
    // Cache bootstrap ANONYME préchargé (avant le login), frais (<5 min) : setup non
    // 'ready' + aucun current_player → isEstablishedPlayer=false → onboarding SI servi.
    qc.setQueryData(queryKeys.bootstrap, {
      setup_required: false,
      setup_state: 'auth_only',
      current_player: null,
      auth_mode: 'xbox',
    } as unknown as BootstrapResponse)

    render(
      <QueryClientProvider client={qc}>
        <XboxLoginPage />
      </QueryClientProvider>,
    )

    // Le /bootstrap réseau (fixture MSW) renvoie un joueur établi → destination '/'.
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith({ to: '/' })
    })
    expect(mockNavigate).not.toHaveBeenCalledWith({ to: '/onboarding/openspartan' })
  })
})
