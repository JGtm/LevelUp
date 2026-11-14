/**
 * Tests — accueil d'un joueur sans match : l'état AFFICHÉ est l'état réel du sync, jamais
 * « Synchronisation en cours » par défaut (issue #192).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
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
      authState: 'ready',
      reauthRequired: false,
      currentPlayer: PLAYER,
      writablePlayerSlugs: null,
      activeSyncJobId: null,
      ...patch,
    })
  })
}

function job(status: string, extra: Record<string, unknown> = {}) {
  server.use(
    http.get('/api/v1/jobs/:jobId', () =>
      HttpResponse.json({
        job_id: 'job-9',
        job_type: 'initial_sync',
        status,
        ...extra,
      }),
    ),
  )
}

describe('HomePage — joueur sans match', () => {
  beforeEach(() => {
    navigate.mockClear()
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

  it('rien de lancé : ne dit PAS « en cours », propose de lancer la synchronisation', async () => {
    setStore({})
    renderWithProviders(<HomePage />)
    expect(await screen.findByText("Aucun match synchronisé pour l'instant")).toBeInTheDocument()
    expect(screen.queryByText('Synchronisation en cours')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Lancer la synchronisation' })).toBeInTheDocument()
  })

  it("lancer : le job démarre, puis l'état devient « en cours »", async () => {
    setStore({})
    job('running')
    server.use(
      http.post('/api/v1/players/:playerSlug/sync', () =>
        HttpResponse.json({ job_id: 'job-9', status: 'queued' }, { status: 202 }),
      ),
    )
    renderWithProviders(<HomePage />)
    fireEvent.click(await screen.findByRole('button', { name: 'Lancer la synchronisation' }))
    expect(await screen.findByText('Synchronisation en cours')).toBeInTheDocument()
    expect(useAppShellStore.getState().activeSyncJobId).toBe('job-9')
  })

  it('profil en lecture seule : aucune action impossible', async () => {
    setStore({ writablePlayerSlugs: ['autre-joueur'] })
    renderWithProviders(<HomePage />)
    expect(await screen.findByText("Aucun match synchronisé pour l'instant")).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Lancer la synchronisation' })).not.toBeInTheDocument()
  })

  it('job en cours : « Synchronisation en cours »', async () => {
    setStore({ activeSyncJobId: 'job-9' })
    job('running')
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Synchronisation en cours')).toBeInTheDocument()
  })

  it("compte non connecté : « En attente de connexion » avec l'action de reconnexion", async () => {
    setStore({ reauthRequired: true, activeSyncJobId: 'job-9' })
    job('running')
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('En attente de connexion')).toBeInTheDocument()
    expect(screen.queryByText('Synchronisation en cours')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Se reconnecter' }))
    expect(navigate).toHaveBeenCalledWith({ to: '/login' })
  })

  it('titre en pause : « Synchronisation en pause », sans bouton', async () => {
    setStore({ currentPlayer: { ...PLAYER, sync_enabled: false } })
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Synchronisation en pause')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('job échoué : « Échec de la synchronisation », la cause du code, et « Réessayer »', async () => {
    setStore({ activeSyncJobId: 'job-9' })
    job('failed', {
      error: { code: 'sync_halo_api_error', message: 'x', retryable: true },
    })
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Échec de la synchronisation')).toBeInTheDocument()
    expect(screen.getByText("L'API Halo est temporairement indisponible. Veuillez réessayer.")).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Réessayer' })).toBeInTheDocument()
  })

  it("job terminé sans match : « Aucun match trouvé », et l'accueil est relu", async () => {
    setStore({ activeSyncJobId: 'job-9' })
    job('succeeded')
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
    renderWithProviders(<HomePage />)
    expect(await screen.findByText('Waiting for sign-in')).toBeInTheDocument()
  })
})
