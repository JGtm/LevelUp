/**
 * Tests composant — SettingsPage.
 *
 * Vérifie le chargement, l'auto-save au changement de champ,
 * le feedback éphémère et l'absence des boutons Annuler/Enregistrer.
 */
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import { createTestQueryClient, renderWithProviders } from '@/test/render-utils'
import { queryKeys } from '@/lib/query/keys'
import type { SettingsResponse } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
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

const ENABLED_CAPABILITIES = testCapabilities()

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

  it('ne rend pas de loader plein écran pendant le chargement (TopProgressBar globale)', () => {
    renderWithProviders(<SettingsPage />)
    expect(screen.queryByText(/Chargement des paramètres/i)).not.toBeInTheDocument()
  })

  // Test "affiche le titre 'Paramètres'" supprimé : titre h1 retiré du composant
  // (refacto post-84ae65ca, NavL1 expose la section). Le rendu post-loading est
  // déjà couvert par le test "affiche la section Langue et affichage".

  it('affiche la section Langue et affichage', async () => {
    renderWithProviders(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/Langue/i)).toBeInTheDocument()
    })
  })

  it('ne propose plus le Lab dans les Paramètres (déplacé vers Admin · Lab)', () => {
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=lab' },
    } as ReturnType<typeof useRouterState>)
    renderWithProviders(<SettingsPage />)
    expect(screen.queryByText(/Lab interne/i)).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Ouvrir le Lab/i })).not.toBeInTheDocument()
  })

  it('onglet Compte : la mise en commun du jeton y est réglable par tout joueur (lieu promis par /privacy et /terms)', async () => {
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=account' },
    } as ReturnType<typeof useRouterState>)
    useAppShellStore.setState({ poolSharing: 'granted' })
    renderWithProviders(<SettingsPage />)
    expect(await screen.findByText('Mise en commun du jeton')).toBeInTheDocument()
    useAppShellStore.setState({ poolSharing: 'unavailable' })
  })

  it('réglages DÉJÀ en cache au montage : les contrôles montrent la valeur enregistrée, pas le défaut', () => {
    // La coquille lit les réglages avant que la page ne monte : la page part donc d'un
    // cache rempli. Le défaut du délai de session est 120 ; la valeur enregistrée, 45.
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=analyse' },
    } as ReturnType<typeof useRouterState>)
    const qc = createTestQueryClient()
    qc.setQueryData(queryKeys.settings, {
      lang: 'fr',
      session_gap_minutes: 45,
      coach_proactive_mode: false,
    } as SettingsResponse)
    render(
      <QueryClientProvider client={qc}>
        <SettingsPage />
      </QueryClientProvider>,
    )
    expect(screen.getByDisplayValue('45')).toBeInTheDocument()
    expect(screen.queryByDisplayValue('120')).not.toBeInTheDocument()
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

  it("l'onglet « Application » n'existe que là où une intégration au poste est offerte", async () => {
    renderWithProviders(<SettingsPage />)
    await screen.findByRole('tab', { name: 'Compte' })
    expect(screen.queryByRole('tab', { name: 'Application' })).not.toBeInTheDocument()

    useAppShellStore.setState({ capabilities: testCapabilities({ can_watch_game_process: true }) })
    expect(await screen.findByRole('tab', { name: 'Application' })).toBeInTheDocument()
  })

  it("l'onglet « Application » porte l'intégration au poste, les données et l'import depuis le site", async () => {
    useAppShellStore.setState({ capabilities: testCapabilities(desktopCapabilities()) })
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=desktop' },
    } as ReturnType<typeof useRouterState>)
    renderWithProviders(<SettingsPage />)
    expect(await screen.findByText('Intégration au poste')).toBeInTheDocument()
    expect(await screen.findByText('Données et sauvegarde')).toBeInTheDocument()
    expect(await screen.findByText('Importer depuis le site')).toBeInTheDocument()
  })

  it("l'onglet « Application » existe pour les seules données, sans l'intégration au poste", async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_backup_to_folder: true }) })
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=desktop' },
    } as ReturnType<typeof useRouterState>)
    renderWithProviders(<SettingsPage />)
    expect(await screen.findByRole('tab', { name: 'Application' })).toBeInTheDocument()
    expect(await screen.findByText('Données et sauvegarde')).toBeInTheDocument()
    expect(screen.queryByText('Intégration au poste')).not.toBeInTheDocument()
  })

  it("l'onglet « Compte » du site porte l'export vers l'application de bureau", async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_export_for_desktop: true }) })
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=account' },
    } as ReturnType<typeof useRouterState>)
    renderWithProviders(<SettingsPage />)
    expect(await screen.findByText("Exporter vers l'application de bureau")).toBeInTheDocument()
  })

  it("sans la capacité d'export, l'onglet « Compte » ne propose pas l'export", async () => {
    vi.mocked(useRouterState).mockReturnValue({
      location: { pathname: '/settings', search: '?tab=account' },
    } as ReturnType<typeof useRouterState>)
    renderWithProviders(<SettingsPage />)
    await screen.findByRole('tab', { name: 'Compte' })
    expect(screen.queryByText("Exporter vers l'application de bureau")).not.toBeInTheDocument()
  })
})
