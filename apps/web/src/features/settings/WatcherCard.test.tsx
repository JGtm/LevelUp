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
import { WATCHER_TEST_TEXT as t } from './watcherCardTestText'
import type { WatcherStatusResponse } from '@/lib/api/types'
import { log } from '@/lib/clipboard/_logger'

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
      { player_slug: 'player1', gamertag: 'PlayerOne', xuid: '0001', is_demo: false, waypoint_player: 'PlayerOne', sync_enabled: true },
      { player_slug: 'player2', gamertag: 'PlayerTwo', xuid: '0002', is_demo: false, waypoint_player: 'PlayerTwo', sync_enabled: true },
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

    it('affiche "Jeton expiré" et le bouton Rafraîchir', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText(/Jeton expiré/i)).toBeInTheDocument()
        expect(screen.getByText('Rafraîchir Xbox')).toBeInTheDocument()
      })
    })
  })

  describe('PlayersSelector', () => {
    beforeEach(() => {
      mockStatusData = { ...baseStatusData }
    })

    it('affiche le label "Joueurs surveillés :" et le summary "Tous les joueurs" par défaut', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        // Label inline avec deux-points
        expect(screen.getByText(/Joueurs surveillés :/)).toBeInTheDocument()
        // Summary du <details> affiche "Tous les joueurs" car subscribed = ['all']
        expect(screen.getByText('Tous les joueurs')).toBeInTheDocument()
      })
    })

    it('affiche une checkbox cochée par joueur disponible (mode all)', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        // Les checkboxes sont rendues même quand le <details> est fermé (DOM).
        const cb1 = screen.getByLabelText('PlayerOne') as HTMLInputElement
        const cb2 = screen.getByLabelText('PlayerTwo') as HTMLInputElement
        expect(cb1).toBeChecked()
        expect(cb2).toBeChecked()
      })
    })

    it('décocher 1 joueur sur 2 envoie [PlayerTwo] (sélection explicite)', async () => {
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      const cb1 = await waitFor(() => screen.getByLabelText('PlayerOne') as HTMLInputElement)
      fireEvent.click(cb1)
      expect(mockUpdateSubsMutate).toHaveBeenCalledWith(
        ['PlayerTwo'],
        expect.any(Object),
      )
    })

    it('décocher tous les joueurs envoie ["all"] (équivalent au mode all)', async () => {
      mockStatusData = { ...baseStatusData, subscribed_players: ['PlayerOne'] }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      const cb1 = await waitFor(() => screen.getByLabelText('PlayerOne') as HTMLInputElement)
      fireEvent.click(cb1)
      expect(mockUpdateSubsMutate).toHaveBeenCalledWith(
        ['all'],
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

    it('affiche le label de présence Xbox (Away → Absent) quand FSM=Idle', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          { gamertag: 'P1', xuid: '0001', state: 'Idle', presence_state: 'Away', in_game: false, state_since: '', state_duration: '' },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('Absent')).toBeInTheDocument()
      })
    })

    it('affiche "Hors-ligne" quand presence_state=Offline', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          { gamertag: 'P1', xuid: '0001', state: 'Idle', presence_state: 'Offline', in_game: false, state_since: '', state_duration: '' },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('Hors-ligne')).toBeInTheDocument()
      })
    })

    it('affiche "En ligne" quand presence_state=Online + Idle FSM', async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [
          { gamertag: 'P1', xuid: '0001', state: 'Idle', presence_state: 'Online', in_game: false, state_since: '', state_duration: '' },
        ],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText('En ligne')).toBeInTheDocument()
      })
    })

    it('affiche "—" si presence_state inconnu (pas encore d\'event)', async () => {
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
        expect(screen.getByText('—')).toBeInTheDocument()
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

  describe('last_seen', () => {
    it('affiche le timestamp last_seen si présent (format relatif récent)', async () => {
      const tenMinAgo = new Date(Date.now() - 10 * 60_000).toISOString()
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [{
          gamertag: 'JGtm',
          xuid: '2533274823110022',
          state: 'Idle',
          in_game: false,
          state_since: '',
          state_duration: '',
          last_seen: { timestamp: tenMinAgo, title_name: 'Halo Infinite' },
        }],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.getByText(/Vu il y a 10 min sur Halo Infinite/i)).toBeInTheDocument()
      })
    })

    it("n'affiche pas la ligne last_seen si absente", async () => {
      mockStatusData = {
        ...baseStatusData,
        daemon_running: true,
        rta_connected: true,
        players: [{ gamertag: 'JGtm', xuid: '2533274823110022', state: 'Idle', in_game: false, state_since: '', state_duration: '' }],
      }
      renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
      await waitFor(() => {
        expect(screen.queryByText(/Vu il y a/i)).not.toBeInTheDocument()
      })
    })
  })
})

// ---------------------------------------------------------------------------
// formatLastSeen unit tests
// ---------------------------------------------------------------------------
import { formatLastSeen, resolveTitleDisplayName } from './watcherPresence'

describe('formatLastSeen', () => {
  const baseNow = new Date('2026-05-26T10:00:00Z')

  it('< 1 min → "moins d\'1 min"', () => {
    const ts = new Date(baseNow.getTime() - 30_000).toISOString()
    expect(formatLastSeen(ts, 'Halo Infinite', t, 'fr', baseNow))
      .toBe("Vu il y a moins d'1 min sur Halo Infinite")
  })

  it('< 60 min → "{N} min"', () => {
    const ts = new Date(baseNow.getTime() - 5 * 60_000).toISOString()
    expect(formatLastSeen(ts, 'Halo Infinite', t, 'fr', baseNow))
      .toBe('Vu il y a 5 min sur Halo Infinite')
  })

  it('< 24 h → "{N} h"', () => {
    const ts = new Date(baseNow.getTime() - 3 * 3_600_000).toISOString()
    expect(formatLastSeen(ts, 'Halo Infinite', t, 'fr', baseNow))
      .toBe('Vu il y a 3 h sur Halo Infinite')
  })

  it('< 7 j → "{N} j"', () => {
    const ts = new Date(baseNow.getTime() - 2 * 86_400_000).toISOString()
    expect(formatLastSeen(ts, 'Halo Infinite', t, 'fr', baseNow))
      .toBe('Vu il y a 2 j sur Halo Infinite')
  })

  it('> 7 j → format absolu', () => {
    const ts = '2026-05-10T08:00:00Z'
    const out = formatLastSeen(ts, 'Halo Infinite', t, 'fr', baseNow)
    expect(out).toMatch(/Vu le .* sur Halo Infinite/)
    expect(out).toContain('Halo Infinite')
  })

  it('timestamp invalide → "Jamais vu en jeu"', () => {
    expect(formatLastSeen('not-a-date', 'Halo Infinite', t, 'fr', baseNow))
      .toBe('Jamais vu en jeu')
  })
})

// ---------------------------------------------------------------------------
// resolveTitleDisplayName unit tests
// ---------------------------------------------------------------------------

describe('resolveTitleDisplayName', () => {
  it('mappe "Online" → "l\'accueil Xbox" (Dashboard Xbox)', () => {
    expect(resolveTitleDisplayName('Online', t)).toBe("l'accueil Xbox")
  })

  it('garde "Halo Infinite" tel quel', () => {
    expect(resolveTitleDisplayName('Halo Infinite', t)).toBe('Halo Infinite')
  })

  it('garde "Counter-Strike 2" tel quel', () => {
    expect(resolveTitleDisplayName('Counter-Strike 2', t)).toBe('Counter-Strike 2')
  })

  it('garde une chaîne vide telle quelle', () => {
    expect(resolveTitleDisplayName('', t)).toBe('')
  })
})

// ---------------------------------------------------------------------------
// Bouton « Copier le code » du Device Code Flow — retour visuel (item 7, 2026-09-17)
//
// Le bouton copiait en silence et avalait son échec (`catch(() => {})`) : rien ne disait
// à l'utilisateur que le code était dans le presse-papier. Il passe par le hook partagé
// `lib/clipboard/useCopyToClipboard`, comme `features/auth/CopyCodeButton.tsx`.
// ---------------------------------------------------------------------------

describe('AuthFlow — bouton « Copier le code »', () => {
  /** Ouvre le flux d'auth : `startAuth.mutate` rend la main par son `onSuccess`. */
  function ouvrirLeFlux() {
    mockStatusData = { ...baseStatusData }
    mockStartAuthMutate.mockImplementation(
      (_vars: unknown, opts: { onSuccess: (d: Record<string, unknown>) => void }) => {
        opts.onSuccess({
          attempt_id: 'a1',
          user_code: 'ABCD-1234',
          verification_url: 'https://example.invalid/link',
          expires_in: 900,
        })
      },
    )
    renderWithProviders(<WatcherCard enabled={true} onToggle={vi.fn()} t={t} />)
    fireEvent.click(screen.getByText('Connecter via Xbox'))
  }

  function stubPressePapier(): ReturnType<typeof vi.fn> {
    const fn = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(globalThis.navigator, 'clipboard', {
      value: { writeText: fn },
      configurable: true,
    })
    return fn
  }

  it('écrit le code dans le presse-papier et confirme « copié »', async () => {
    const writeText = stubPressePapier()
    ouvrirLeFlux()

    const bouton = await screen.findByText('Copier le code')
    fireEvent.click(bouton)

    expect(writeText).toHaveBeenCalledWith('ABCD-1234')
    // La confirmation remplace le libellé du bouton le temps de la fenêtre de feedback.
    await waitFor(() => expect(screen.getByText(/Code copié/)).toBeInTheDocument())
    expect(screen.queryByText('Copier le code')).not.toBeInTheDocument()
  })

  it("échec du presse-papier : aucune confirmation, et l'erreur est JOURNALISÉE", async () => {
    log._resetForTests()
    const erreurConsole = vi.spyOn(console, 'error').mockImplementation(() => {})
    const writeText = vi.fn().mockRejectedValue(new Error('presse-papier indisponible'))
    Object.defineProperty(globalThis.navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    })
    ouvrirLeFlux()

    fireEvent.click(await screen.findByText('Copier le code'))

    await waitFor(() => expect(writeText).toHaveBeenCalled())
    // Jamais de fausse confirmation : le code reste sélectionnable à la main.
    expect(screen.queryByText(/Code copié/)).not.toBeInTheDocument()
    expect(screen.getByText('Copier le code')).toBeInTheDocument()
    // Et l'échec ne se perd pas en silence (c'était le `catch(() => {})` d'avant).
    await waitFor(() => expect(erreurConsole).toHaveBeenCalled())
    erreurConsole.mockRestore()
  })
})
