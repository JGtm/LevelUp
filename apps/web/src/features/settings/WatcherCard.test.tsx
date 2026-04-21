/**
 * Tests composant — WatcherCard.
 *
 * Couvre :
 * - Rendu quand le watcher est désactivé (toggle off)
 * - Rendu quand le watcher est activé avec token absent/valide/expiré
 * - Bouton d'auth → appelle startAuth
 * - PlayersSelector → affiche liste et "Tous les joueurs"
 * - RTAStatus → visible uniquement si daemon_running
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { WatcherCard } from './WatcherCard'
import type { SettingsText } from './i18n'
import type { WatcherStatusResponse } from '@/lib/api/types'

// ---------------------------------------------------------------------------
// Mock des hooks watcher-queries
// ---------------------------------------------------------------------------
const mockStartAuthMutate = vi.fn()
const mockUpdateSubsMutate = vi.fn()

let mockStatusData: WatcherStatusResponse | undefined = undefined
let mockPollData: { status: string } | undefined = undefined

vi.mock('./watcher-queries', () => ({
  useWatcherStatus: () => ({ data: mockStatusData }),
  useStartWatcherAuth: () => ({ mutate: mockStartAuthMutate, isPending: false }),
  useWatcherAuthPoll: () => ({ data: mockPollData }),
  useUpdateWatcherSubscriptions: () => ({ mutate: mockUpdateSubsMutate, isPending: false }),
}))

// ---------------------------------------------------------------------------
// Fixture i18n (subset des clés utilisées par WatcherCard)
// ---------------------------------------------------------------------------
const t: SettingsText = {
  pageTitle: 'Paramètres',
  pageSubtitle: "Configuration de l'application",
  savedStatus: '✓ Enregistré',
  errorStatus: '✗ Erreur',
  loading: 'Chargement…',
  tabGeneral: 'Général',
  tabSync: 'Sync',
  tabLab: 'Lab',
  tabUsers: 'Utilisateurs',
  manualSyncTitle: 'Sync manuelle',
  manualSyncButton: 'Synchroniser',
  manualSyncRunning: 'En cours…',
  manualSyncDescription: '',
  instanceLabel: 'Instance',
  instanceTitle: 'Lab',
  instanceDescription: '',
  openLabButton: 'Ouvrir',
  usersTitle: 'Utilisateurs',
  usersDescription: '',
  openUsersButton: 'Ouvrir',
  interfaceTitle: 'Interface',
  langLabel: 'Langue',
  langFr: 'FR',
  langEn: 'EN',
  timezoneLabel: 'Fuseau',
  showRecords: 'Records',
  normalizeModeLabels: 'Normaliser modes',
  excludeBTB: 'Exclure BTB',
  refreshClearsCaches: 'Vider caches',
  discordTitle: 'Discord',
  discordEnabled: 'Activé',
  discordNotifySync: 'Notifier sync',
  discordNotifyBackfill: 'Notifier backfill',
  discordNotifyNewMedia: 'Notifier médias',
  discordNoWebhook: 'Webhook absent',
  mediaTitle: 'Médias',
  mediaWatcherEnabled: 'Surveillance médias',
  mediaToleranceLabel: 'Tolérance',
  mediaNoBaseDir: 'Aucun dossier',
  spnkrTitle: 'SPNKr',
  spnkrAutoSync: 'Auto-sync',
  spnkrAutoSyncInterval: 'Intervalle',
  spnkrAutoSyncIntervalUnit: 'h',
  spnkrAutoSyncIntervalMinutes: 'Intervalle (min)',
  spnkrAutoSyncIntervalMinutesUnit: 'min',
  spnkrRefreshWithBackfill: 'Backfill post-sync',
  watcherTitle: 'Détection de présence',
  watcherPresenceEnabled: 'Détection automatique',
  watcherPresenceDescription: 'Description',
  watcherAuthButton: 'Connecter via Xbox',
  watcherAuthReconnect: 'Reconnecter Xbox',
  watcherAuthInstructions: 'Rendez-vous sur {url}',
  watcherAuthCopyCode: 'Copier le code',
  watcherAuthOpenLink: 'Ouvrir le lien',
  watcherAuthPending: 'En attente…',
  watcherAuthSuccess: 'Connexion réussie !',
  watcherAuthFailed: 'Échec de la connexion.',
  watcherTokenValid: 'Jeton valide jusqu\'au {date} ({gamertag})',
  watcherTokenExpired: 'Jeton expiré',
  watcherTokenMissing: 'Aucun jeton Xbox',
  watcherPlayersLabel: 'Joueurs surveillés',
  watcherPlayersAll: 'Tous les joueurs',
  watcherSubscriptionsUpdated: 'Mis à jour',
  watcherRtaConnected: 'RTA connecté',
  watcherRtaDisconnected: 'RTA déconnecté',
  watcherSubscribeError: 'Échec surveillance',
  watcherStateIdle: 'En attente',
  watcherStateWatching: 'En surveillance',
  watcherStateSyncing: 'Synchronisation',
  watcherStateCooling: 'Cooldown',
  watcherInGame: 'En jeu',
  backfillTitle: 'Backfill',
  backfillMedals: 'Médailles',
  backfillSkill: 'CSR/MMR',
  backfillAliases: 'Alias',
  backfillPersonalScores: 'Scores',
  backfillPerfScores: 'Perf',
  backfillLUSR: 'LUSR',
  backfillEvents: 'Événements',
  backfillWeapons: 'Armes',
}

const baseStatusData: WatcherStatusResponse = {
  daemon_running: false,
  rta_connected: false,
  token_valid: false,
  token_expires_at: undefined,
  token_gamertag: undefined,
  subscribed_players: ['all'],
  players: [],
}

beforeEach(() => {
  vi.clearAllMocks()
  mockStatusData = undefined
  mockPollData = undefined
  useAppShellStore.setState({
    currentPlayer: null,
    availablePlayers: [
      { player_slug: 'player1', gamertag: 'PlayerOne', xuid: '0001', is_demo: false },
      { player_slug: 'player2', gamertag: 'PlayerTwo', xuid: '0002', is_demo: false },
    ],
    currentTitleSlug: 'halo_infinite',
    availableTitles: [],
    isTitleSwitching: false,
    locale: 'fr',
    hintsVisible: true,
    capabilities: {
      can_read_local_data: true,
      can_run_sync: true,
      can_use_live_halo: true,
      can_manage_settings: true,
      can_reset_media_index: true,
      can_view_media: true,
      can_self_provision: true,
      can_start_initial_sync: true,
      can_manage_instance: true,
    },
    setupRequired: false,
    authState: 'ready',
    setupState: 'ready',
    isBootstrapped: true,
    linkedHaloIdentity: null,
    activeSyncJobId: null,
  })
})

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('WatcherCard', () => {
  it('affiche le titre "Détection de présence"', () => {
    renderWithProviders(<WatcherCard enabled={false} onToggle={vi.fn()} t={t} />)
    expect(screen.getByText('Détection de présence')).toBeInTheDocument()
  })

  it('masque le contenu quand désactivé', () => {
    renderWithProviders(<WatcherCard enabled={false} onToggle={vi.fn()} t={t} />)
    expect(screen.queryByText('Aucun jeton Xbox')).not.toBeInTheDocument()
    expect(screen.queryByText('Connecter via Xbox')).not.toBeInTheDocument()
  })

  it('appelle onToggle quand on clique le toggle', () => {
    const onToggle = vi.fn()
    renderWithProviders(<WatcherCard enabled={false} onToggle={onToggle} t={t} />)
    const toggle = screen.getByRole('button')
    fireEvent.click(toggle)
    expect(onToggle).toHaveBeenCalledWith(true)
  })

  describe('quand activé — token absent', () => {
    beforeEach(() => {
      mockStatusData = { ...baseStatusData }
    })

    it('affiche "Aucun jeton Xbox" et le bouton Connecter', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText(/Aucun jeton Xbox/i)).toBeInTheDocument()
        expect(screen.getByText('Connecter via Xbox')).toBeInTheDocument()
      })
    })

    it('appelle startAuth.mutate quand on clique Connecter', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => screen.getByText('Connecter via Xbox'))
      fireEvent.click(screen.getByText('Connecter via Xbox'))
      expect(mockStartAuthMutate).toHaveBeenCalled()
    })
  })

  describe('quand activé — token valide', () => {
    beforeEach(() => {
      mockStatusData = {
        ...baseStatusData,
        token_valid: true,
        token_expires_at: '2099-12-31T00:00:00Z',
        token_gamertag: 'TestPlayer',
      }
    })

    it('affiche le message de validité', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText(/Jeton valide/i)).toBeInTheDocument()
        expect(screen.getByText(/TestPlayer/i)).toBeInTheDocument()
      })
    })

    it("n'affiche pas le bouton Connecter", async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => screen.getByText(/Jeton valide/i))
      expect(screen.queryByText('Connecter via Xbox')).not.toBeInTheDocument()
    })
  })

  describe('quand activé — token expiré', () => {
    beforeEach(() => {
      mockStatusData = {
        ...baseStatusData,
        token_valid: false,
        token_expires_at: '2020-01-01T00:00:00Z',
        token_gamertag: 'OldPlayer',
      }
    })

    it('affiche "Jeton expiré" et le bouton Reconnecter', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText(/Jeton expiré/i)).toBeInTheDocument()
        expect(screen.getByText('Reconnecter Xbox')).toBeInTheDocument()
      })
    })
  })

  describe('PlayersSelector', () => {
    beforeEach(() => {
      mockStatusData = { ...baseStatusData }
    })

    it('affiche l\'option "Tous les joueurs"', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('Joueurs surveillés')).toBeInTheDocument()
        expect(screen.getByRole('option', { name: 'Tous les joueurs' })).toBeInTheDocument()
      })
    })

    it('affiche les joueurs disponibles dans la liste', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByRole('option', { name: 'PlayerOne' })).toBeInTheDocument()
        expect(screen.getByRole('option', { name: 'PlayerTwo' })).toBeInTheDocument()
      })
    })

    it('appelle updateSubs.mutate lors du changement de sélection', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => screen.getByRole('option', { name: 'PlayerOne' }))
      const select = screen.getByRole('combobox')
      fireEvent.change(select, { target: { value: 'PlayerOne' } })
      expect(mockUpdateSubsMutate).toHaveBeenCalledWith(
        ['PlayerOne'],
        expect.any(Object),
      )
    })
  })

  describe('RTAStatus', () => {
    it("n'affiche pas le statut RTA si daemon_running=false", async () => {
      mockStatusData = { ...baseStatusData, daemon_running: false }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => screen.getByText(/Aucun jeton Xbox/i))
      expect(screen.queryByText('RTA connecté')).not.toBeInTheDocument()
      expect(screen.queryByText('RTA déconnecté')).not.toBeInTheDocument()
    })

    it('affiche "RTA connecté" si daemon_running=true et rta_connected=true', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('RTA connecté')).toBeInTheDocument()
      })
    })

    it('affiche "RTA déconnecté" si daemon_running=true et rta_connected=false', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: false,
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('RTA déconnecté')).toBeInTheDocument()
      })
    })

    it('liste les joueurs en ligne si players est non-vide', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          { gamertag: 'PlayerOne', xuid: '0001', state: 'Watching', in_game: false, state_since: '', state_duration: '' },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        // PlayerOne apparaît à la fois dans le dropdown et dans la liste RTA → getAllByText
        const matches = screen.getAllByText('PlayerOne')
        expect(matches.length).toBeGreaterThanOrEqual(2) // option + span RTAStatus
        // L'état est traduit via resolveStateLabel
        expect(screen.getByText('En surveillance')).toBeInTheDocument()
      })
    })

    it('traduit les états FSM (Idle → En attente)', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          { gamertag: 'P1', xuid: '0001', state: 'Idle', in_game: false, state_since: '', state_duration: '' },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('En attente')).toBeInTheDocument()
      })
    })

    it('affiche "En jeu" quand in_game=true', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          { gamertag: 'P1', xuid: '0001', state: 'Watching', in_game: true, state_since: '', state_duration: '' },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('En jeu')).toBeInTheDocument()
      })
    })

    it('affiche le badge Échec surveillance si subscribe_error est défini', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          {
            gamertag: 'P1',
            xuid: '0001',
            state: 'Idle',
            in_game: false,
            state_since: '',
            state_duration: '',
            subscribe_error: 'rta: timeout',
          },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        const badge = screen.getByText(/Échec surveillance/i)
        expect(badge).toBeInTheDocument()
        // Le tooltip contient le message d'erreur brut
        expect(badge.closest('[title]')?.getAttribute('title')).toBe('rta: timeout')
      })
    })

    it("n'affiche pas de badge d'erreur si subscribe_error est absent", async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          { gamertag: 'P1', xuid: '0001', state: 'Watching', in_game: false, state_since: '', state_duration: '' },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.queryByText(/Échec surveillance/i)).not.toBeInTheDocument()
      })
    })
  })
})
