/**
 * Tests composant — SetupPage (Slice 1).
 *
 * Smoke test : le composant monte et affiche l'étape initiale.
 * Les mutations API sont mockées via MSW (setup.ts).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { SetupPage } from './SetupPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

describe('SetupPage', () => {
  const resetStores = () => {
    useSetupFlowStore.getState().reset()
    useAppShellStore.setState({
      currentPlayer: null,
      availablePlayers: [],
      setupRequired: false,
      setupState: 'no_halo_link',
      isBootstrapped: false,
      linkedHaloIdentity: null,
      activeSyncJobId: null,
    })
  }

  beforeEach(() => {
    resetStores()
  })

  afterEach(() => {
    resetStores()
  })

  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SetupPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner tant que le bootstrap n’est pas terminé', () => {
    renderWithProviders(<SetupPage />)
    expect(screen.getByText(/Vérification de la configuration/i)).toBeInTheDocument()
  })

  it('affiche l’étape de connexion Microsoft quand setupState=no_halo_link', async () => {
    useAppShellStore.setState({
      isBootstrapped: true,
      setupRequired: true,
      setupState: 'no_halo_link',
    })

    renderWithProviders(<SetupPage />)
    await waitFor(() => {
      expect(screen.getByText(/Connexion Microsoft/i)).toBeInTheDocument()
    })
  })

  it('affiche le code renvoyé par le Device Code Flow', async () => {
    useAppShellStore.setState({
      isBootstrapped: true,
      setupRequired: true,
      setupState: 'no_halo_link',
    })

    renderWithProviders(<SetupPage />)
    await waitFor(() => {
      expect(screen.getByText(/ABCD-1234/i)).toBeInTheDocument()
    })
  })

  // Garde-rail anti-régression du « spinner infini » (Lot A) : quand
  // POST /device-flow/start renvoie 500, l'UI doit basculer sur un message
  // d'erreur + bouton « Réessayer », jamais rester bloquée sur le spinner.
  it('affiche une erreur + bouton Réessayer quand le start échoue (500)', async () => {
    useAppShellStore.setState({
      isBootstrapped: true,
      setupRequired: true,
      setupState: 'no_halo_link',
    })
    server.use(
      http.post('/api/v1/auth/device-flow/start', () =>
        HttpResponse.json(
          { code: 'msal_init_error', message: 'impossible de démarrer le Device Code Flow', retryable: false },
          { status: 500 },
        ),
      ),
    )

    renderWithProviders(<SetupPage />)

    await waitFor(() => {
      expect(screen.getByText(/Impossible de démarrer la connexion Microsoft/i)).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /Réessayer/i })).toBeInTheDocument()
    // Le spinner de démarrage ne doit plus être présent.
    expect(screen.queryByText(/Démarrage du Device Code Flow/i)).not.toBeInTheDocument()
  })
})
