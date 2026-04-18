/**
 * Tests composant — SetupPage (Slice 1).
 *
 * Smoke test : le composant monte et affiche l'étape initiale.
 * Les mutations API sont mockées via MSW (setup.ts).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
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
})
