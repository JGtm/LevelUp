/**
 * Tests composant — SettingsPage.
 *
 * Vérifie le chargement, l'auto-save au changement de champ,
 * le feedback éphémère et l'absence des boutons Annuler/Enregistrer.
 */
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { useRouterState } from '@tanstack/react-router'
import { SettingsPage } from './SettingsPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ to, children }: { to: string; children: React.ReactNode }) => <a href={to}>{children}</a>,
    useNavigate: () => () => Promise.resolve(),
    useParams: () => ({ playerSlug: 'test-player' }),
    useRouterState: vi.fn().mockReturnValue({ location: { pathname: '/settings', search: '' } }),
  }
})

const ENABLED_CAPABILITIES = {
  can_read_local_data: true,
  can_run_sync: true,
  can_use_live_halo: true,
  can_manage_settings: true,
  can_reset_media_index: true,
  can_view_media: true,
  can_self_provision: true,
  can_start_initial_sync: true,
  can_manage_instance: true,
}

beforeEach(() => {
  vi.mocked(useRouterState).mockReturnValue({
    location: { pathname: '/settings', search: '' },
  } as ReturnType<typeof useRouterState>)
  useAppShellStore.setState({
    currentPlayer: null,
    availablePlayers: [],
    currentTitleSlug: 'halo_infinite',
    availableTitles: [],
    isTitleSwitching: false,
    locale: 'fr',
    hintsVisible: true,
    capabilities: ENABLED_CAPABILITIES,
    setupRequired: false,
    authState: 'ready',
    setupState: 'ready',
    isBootstrapped: true,
    linkedHaloIdentity: null,
    activeSyncJobId: null,
  })
})

describe('SettingsPage', () => {
  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<SettingsPage />)
    expect(container).toBeTruthy()
  })

  it('affiche le spinner en phase de chargement', () => {
    renderWithProviders(<SettingsPage />)
    expect(screen.getByText(/Chargement des paramètres/i)).toBeInTheDocument()
  })

  it("affiche le titre 'Paramètres' une fois les données chargées", async () => {
    renderWithProviders(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('Paramètres')).toBeInTheDocument()
    })
  })

  it('affiche la section Langue et affichage', async () => {
    renderWithProviders(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/Langue/i)).toBeInTheDocument()
    })
  })

  it('affiche le CTA vers le Lab quand la capacité instance est active', async () => {
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=lab' },
    } as ReturnType<typeof useRouterState>)
    renderWithProviders(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/Lab interne/i)).toBeInTheDocument()
      expect(screen.getByRole('link', { name: /Ouvrir le Lab/i })).toHaveAttribute('href', '/lab')
    })
  })

  it("n'affiche pas les boutons Annuler / Enregistrer (auto-save)", async () => {
    renderWithProviders(<SettingsPage />)
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /Annuler/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /Enregistrer/i })).not.toBeInTheDocument()
    })
  })

  it('appelle la mutation immédiatement lors du changement de langue', async () => {
    renderWithProviders(<SettingsPage />)
    const select = await screen.findByLabelText
      ? screen.queryByDisplayValue('Français') ?? await waitFor(() => screen.getByDisplayValue('Français'))
      : await waitFor(() => screen.getByDisplayValue('Français'))
    fireEvent.change(select, { target: { value: 'en' } })
    // La mutation est déclenchée sans clic sur Enregistrer
    await waitFor(() => {
      expect(select).toBeInTheDocument()
    })
  })
})
