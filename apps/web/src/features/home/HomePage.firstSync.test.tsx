/**
 * Tests — accueil d'un joueur sans match : l'état AFFICHÉ est l'état réel du sync lu auprès du serveur
 * (`GET /players/{slug}/sync/status`), jamais « Synchronisation en cours » par défaut (issue #192).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { toast } from 'sonner'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { PlayerSummary } from '@/lib/api/types'
import { HomePage } from './HomePage'

const navigate = vi.fn()
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigate,
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() } }))

const PLAYER = {
  player_slug: 'test-player',
  gamertag: 'TestPlayer',
  xuid: '1',
  is_demo: false,
  sync_enabled: true,
  waypoint_player: 'TestPlayer',
} as PlayerSummary

const EMPTY_HOME = {
  hero: { player_name: 'TestPlayer', kpis: { total_matches: 0 }, trend: null },
  highlights: [],
  recent_matches: [],
  recent_matches_total: 0,
  recent_matches_has_more: false,
  last_session_outcomes: [],
  solo_session: null,
  squad_session: null,
}

function setStore(patch: Partial<ReturnType<typeof useAppShellStore.getState>>) {
  act(() => {
    useAppShellStore.setState({
      locale: 'fr',
      authState: 'missing', // session ouverte par mot de passe : jamais AuthReady, et pourtant synchronisable
      reauthRequired: false,
      currentPlayer: PLAYER,
      writablePlayerSlugs: null,
      activeSyncJobId: null,
      ...patch,
    })
  })
}

interface ServedJob {
  status: string
  error?: { code: string; message: string; retryable: boolean }
  player_slug?: string
  job_id?: string
}

/** Sert l'état du sync du joueur ; `urls` garde les joueurs demandés. */
function serveStatus(opts: { syncing?: boolean; job?: ServedJob; cooldown?: number } = {}, urls: string[] = []) {
  server.use(
    http.get('/api/v1/players/:playerSlug/sync/status', ({ params }) => {
      urls.push(String(params.playerSlug))
      return HttpResponse.json({
        player_slug: params.playerSlug,
        syncing: opts.syncing ?? false,
        cooldown_retry_after_s: opts.cooldown ?? 0,
        ...(opts.job
          ? { job: { job_id: 'job-9', job_type: 'initial_sync', player_slug: params.playerSlug, ...opts.job } }
          : {}),
      })
    }),
  )
}

describe('HomePage — joueur sans match', () => {
  beforeEach(() => {
    navigate.mockClear()
    vi.mocked(toast.error).mockClear()
    server.use(http.get('/api/v1/players/:playerSlug/pages/home', () => HttpResponse.json(EMPTY_HOME)))
  })
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({
        authState: 'missing',
        reauthRequired: false,
        currentPlayer: null,
        writablePlayerSlugs: null,
        activeSyncJobId: null,
      })
    })
  })

  it('session par mot de passe (auth_state « missing ») : le vrai état et « Lancer », jamais « en attente de connexion »', async () => {
    setStore({ authState: 'missing' })
    serveStatus()
    renderWithProviders(<HomePage />)
    expect(await screen.findByText("Aucun match synchronisé pour l'instant")).toBeInTheDocument()
    expect(screen.queryByText('En attente de connexion')).not.toBeInTheDocument()
    expect(screen.queryByText('Synchronisation en cours')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lancer la synchronisation' })).toBeInTheDocument()
  })

  it('jeton mort sur son profil : « En attente de connexion » avec l’action de reconnexion', async () => {
    setStore({ reauthRequired: true })
    serveStatus({ syncing: true, job: { status: 'running' } })
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('En attente de connexion')).toBeInTheDocument()
    expect(screen.queryByText('Synchronisation en cours')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Se reconnecter' }))
    expect(navigate).toHaveBeenCalledWith({ to: '/login' })
  })

  it("visiteur en lecture seule avec SON jeton mort : aucune invite à reconnecter sur le profil d'un autre", async () => {
    setStore({ reauthRequired: true, writablePlayerSlugs: ['autre-joueur'] })
    serveStatus()
    renderWithProviders(<HomePage />)
    expect(await screen.findByText("Aucun match synchronisé pour l'instant")).toBeInTheDocument()
    expect(screen.queryByText('En attente de connexion')).not.toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it("rechargement pendant un job : le serveur dit « en cours », l'identifiant gardé en mémoire n'est pas nécessaire", async () => {
    setStore({ activeSyncJobId: null })
    serveStatus({ syncing: true, job: { status: 'running' } })
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Synchronisation en cours')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Lancer la synchronisation' })).not.toBeInTheDocument()
  })

  it('synchronisation automatique en vol, sans job : « en cours »', async () => {
    setStore({})
    serveStatus({ syncing: true })
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Synchronisation en cours')).toBeInTheDocument()
  })

  it("l'état est lu pour le joueur AFFICHÉ, et le job d'un autre joueur est ignoré", async () => {
    setStore({})
    const urls: string[] = []
    serveStatus({ job: { status: 'running', player_slug: 'autre-joueur' } }, urls)
    renderWithProviders(<HomePage />)
    expect(await screen.findByText("Aucun match synchronisé pour l'instant")).toBeInTheDocument()
    expect(urls).toEqual(['test-player'])
    expect(screen.queryByText('Synchronisation en cours')).not.toBeInTheDocument()
  })

  it('lancer : le serveur est relu et dit « en cours »', async () => {
    setStore({})
    let started = false
    server.use(
      http.get('/api/v1/players/:playerSlug/sync/status', () =>
        HttpResponse.json(
          started
            ? {
                player_slug: 'test-player',
                syncing: true,
                cooldown_retry_after_s: 300,
                job: { job_id: 'job-9', job_type: 'initial_sync', status: 'running', player_slug: 'test-player' },
              }
            : { player_slug: 'test-player', syncing: false, cooldown_retry_after_s: 0 },
        ),
      ),
      http.post('/api/v1/players/:playerSlug/sync', () => {
        started = true
        return HttpResponse.json({ job_id: 'job-9', status: 'queued' }, { status: 202 })
      }),
    )
    renderWithProviders(<HomePage />)
    fireEvent.click(await screen.findByRole('button', { name: 'Lancer la synchronisation' }))
    expect(await screen.findByText('Synchronisation en cours')).toBeInTheDocument()
    expect(useAppShellStore.getState().activeSyncJobId).toBe('job-9')
  })

  it('profil en lecture seule : aucune action impossible', async () => {
    setStore({ writablePlayerSlugs: ['autre-joueur'] })
    serveStatus()
    renderWithProviders(<HomePage />)
    expect(await screen.findByText("Aucun match synchronisé pour l'instant")).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Lancer la synchronisation' })).not.toBeInTheDocument()
  })

  it('délai du sync manuel : pas de « Lancer », et le délai restant est dit', async () => {
    setStore({})
    serveStatus({ cooldown: 241 })
    renderWithProviders(<HomePage />)
    expect(await screen.findByText(/Une nouvelle synchronisation sera possible dans environ 5 min/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Lancer la synchronisation' })).not.toBeInTheDocument()
  })

  it('titre en pause : « Synchronisation en pause », sans bouton', async () => {
    setStore({ currentPlayer: { ...PLAYER, sync_enabled: false } })
    serveStatus()
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Synchronisation en pause')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  describe('job échoué', () => {
    const echec = { status: 'failed', error: { code: 'sync_halo_api_error', message: 'x', retryable: true } }

    it('la cause du code et « Réessayer »', async () => {
      setStore({})
      serveStatus({ job: echec })
      renderWithProviders(<HomePage />)
      expect(await screen.findByText('Échec de la synchronisation')).toBeInTheDocument()
      expect(screen.getByText("L'API Halo est temporairement indisponible. Veuillez réessayer.")).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Réessayer' })).toBeInTheDocument()
    })

    it('profil en lecture seule : la cause, aucun « Réessayer »', async () => {
      setStore({ writablePlayerSlugs: ['autre-joueur'] })
      serveStatus({ job: echec })
      renderWithProviders(<HomePage />)
      expect(await screen.findByText('Échec de la synchronisation')).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Réessayer' })).not.toBeInTheDocument()
    })

    it('échec non réessayable (retryable = false) : aucun « Réessayer »', async () => {
      setStore({})
      serveStatus({ job: { ...echec, error: { ...echec.error, retryable: false } } })
      renderWithProviders(<HomePage />)
      expect(await screen.findByText('Échec de la synchronisation')).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Réessayer' })).not.toBeInTheDocument()
    })

    it('pendant le délai du sync manuel : aucun « Réessayer », le délai restant est dit', async () => {
      setStore({})
      serveStatus({ job: echec, cooldown: 120 })
      renderWithProviders(<HomePage />)
      expect(await screen.findByText('Échec de la synchronisation')).toBeInTheDocument()
      expect(screen.getByText(/dans environ 2 min/)).toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Réessayer' })).not.toBeInTheDocument()
    })

    it('un refus 429 sync_cooldown du serveur : toast précis, pas le message générique', async () => {
      setStore({})
      serveStatus({ job: echec })
      server.use(
        http.post('/api/v1/players/:playerSlug/sync', () =>
          HttpResponse.json({ code: 'sync_cooldown', message: 'x' }, { status: 429 }),
        ),
      )
      renderWithProviders(<HomePage />)
      fireEvent.click(await screen.findByRole('button', { name: 'Réessayer' }))
      await waitFor(() =>
        expect(toast.error).toHaveBeenCalledWith(
          "Une synchronisation vient d'être lancée : réessaie dans quelques minutes.",
        ),
      )
    })

    it('un refus 409 sync_already_active du serveur : toast précis', async () => {
      setStore({})
      serveStatus({ job: echec })
      server.use(
        http.post('/api/v1/players/:playerSlug/sync', () =>
          HttpResponse.json({ code: 'sync_already_active', message: 'x' }, { status: 409 }),
        ),
      )
      renderWithProviders(<HomePage />)
      fireEvent.click(await screen.findByRole('button', { name: 'Réessayer' }))
      await waitFor(() =>
        expect(toast.error).toHaveBeenCalledWith('Une synchronisation de ce joueur est déjà en cours.'),
      )
    })
  })

  it("job terminé sans match : « Aucun match trouvé », et l'accueil est relu", async () => {
    setStore({})
    serveStatus({ job: { status: 'succeeded' } })
    let homeReads = 0
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/home', () => {
        homeReads += 1
        return HttpResponse.json(EMPTY_HOME)
      }),
    )
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Aucun match trouvé')).toBeInTheDocument()
    await waitFor(() => expect(homeReads).toBeGreaterThanOrEqual(2))
  })

  it('en anglais : mêmes états, texte anglais', async () => {
    setStore({ locale: 'en', reauthRequired: true })
    serveStatus()
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Waiting for sign-in')).toBeInTheDocument()
  })
})
