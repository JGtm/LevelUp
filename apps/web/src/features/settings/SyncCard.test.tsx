/**
 * Tests composant — SyncCard (bouton de synchronisation et état de l'historique du joueur affiché).
 */
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { toast } from 'sonner'
import { renderWithProviders } from '@/test/render-utils'
import { demoCapabilities, testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { PlayerSummary } from '@/lib/api/types'
import { SyncCard } from './SyncCard'

vi.mock('sonner', () => ({ toast: { info: vi.fn(), error: vi.fn(), success: vi.fn() } }))

const API = '/api/v1'
const PLAYER = { player_slug: 'p1', gamertag: 'Pseudo1' } as PlayerSummary

interface ServedStatus {
  syncing?: boolean
  cooldown?: number
  historyComplete?: boolean
  /** Joueur nommé dans la réponse, s'il n'est pas celui demandé. */
  answerFor?: string
  job?: { job_id: string; status: string }
}

/** Sert l'état du sync du joueur ; `urls` garde les joueurs demandés (une entrée par lecture). */
function serveStatus(opts: ServedStatus = {}, urls: string[] = []) {
  server.use(
    http.get(`${API}/players/:playerSlug/sync/status`, ({ params }) => {
      urls.push(String(params.playerSlug))
      return HttpResponse.json({
        player_slug: opts.answerFor ?? params.playerSlug,
        syncing: opts.syncing ?? false,
        cooldown_retry_after_s: opts.cooldown ?? 0,
        history_complete: opts.historyComplete ?? false,
        ...(opts.job ? { job: { job_type: 'delta_sync', player_slug: params.playerSlug, ...opts.job } } : {}),
      })
    }),
  )
}

describe('SyncCard', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      locale: 'fr',
      currentPlayer: PLAYER,
      writablePlayerSlugs: null,
      activeSyncJobId: null,
      capabilities: testCapabilities(),
    })
  })
  afterEach(() => {
    useAppShellStore.setState({
      locale: 'fr',
      currentPlayer: null,
      writablePlayerSlugs: null,
      activeSyncJobId: null,
      capabilities: null,
    })
    vi.clearAllMocks()
  })

  it('propose le bouton avec le droit d’écriture sur le joueur', async () => {
    serveStatus()
    renderWithProviders(<SyncCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    await waitFor(() => expect(bouton).toBeEnabled())
  })

  it('n’affiche aucun bouton sans droit d’écriture, mais dit l’historique', async () => {
    useAppShellStore.setState({ writablePlayerSlugs: ['autre-joueur'] })
    serveStatus({ historyComplete: true })
    renderWithProviders(<SyncCard />)
    expect(await screen.findByTestId('sync-history')).toHaveTextContent('Historique complet')
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('pendant le délai, le bouton est désactivé et le délai restant est dit', async () => {
    serveStatus({ cooldown: 130 })
    renderWithProviders(<SyncCard />)
    expect(await screen.findByTestId('sync-cooldown')).toHaveTextContent('environ 3 min')
    expect(screen.getByRole('button', { name: 'Synchroniser maintenant' })).toBeDisabled()
  })

  it('pendant un job, le bouton dit que la synchronisation est en cours et ne relance rien', async () => {
    let posts = 0
    server.use(
      http.post(`${API}/players/:playerSlug/sync`, () => {
        posts += 1
        return HttpResponse.json({ job_id: 'j' })
      }),
    )
    serveStatus({ syncing: true, cooldown: 120 })
    renderWithProviders(<SyncCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchronisation en cours…' })
    expect(bouton).toBeDisabled()
    expect(screen.queryByTestId('sync-cooldown')).not.toBeInTheDocument()
    fireEvent.click(bouton)
    expect(posts).toBe(0)
  })

  it('titre en pause : pas de bouton, la phrase dit où réactiver, l’historique reste affiché', async () => {
    useAppShellStore.setState({ currentPlayer: { ...PLAYER, sync_enabled: false } })
    serveStatus({ historyComplete: true })
    renderWithProviders(<SyncCard />)
    expect(await screen.findByTestId('sync-paused')).toHaveTextContent(
      'La synchronisation de ce jeu est en pause pour ce joueur : elle se réactive dans Paramètres › Jeux.',
    )
    expect(await screen.findByTestId('sync-history')).toHaveTextContent('Historique complet')
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('titre en pause et historique incomplet : ne dit pas qu’un rattrapage est en cours', async () => {
    useAppShellStore.setState({ currentPlayer: { ...PLAYER, sync_enabled: false } })
    serveStatus({ historyComplete: false })
    renderWithProviders(<SyncCard />)
    const historique = await screen.findByTestId('sync-history')
    expect(historique).toHaveTextContent(
      'Historique incomplet : le rattrapage reprendra quand la synchronisation de ce jeu sera réactivée.',
    )
    expect(historique).not.toHaveTextContent('Rattrapage de l')
  })

  it('en anglais, titre en pause et historique incomplet : même sens', async () => {
    useAppShellStore.setState({ locale: 'en', currentPlayer: { ...PLAYER, sync_enabled: false } })
    serveStatus({ historyComplete: false })
    renderWithProviders(<SyncCard />)
    expect(await screen.findByTestId('sync-history')).toHaveTextContent(
      'History incomplete: the catch-up will resume when sync for this game is turned back on.',
    )
  })

  it('titre actif : aucune phrase de pause', async () => {
    useAppShellStore.setState({ currentPlayer: { ...PLAYER, sync_enabled: true } })
    serveStatus()
    renderWithProviders(<SyncCard />)
    await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    expect(screen.queryByTestId('sync-paused')).not.toBeInTheDocument()
  })

  it('dit « Historique complet » quand la fin de l’historique a été vue', async () => {
    serveStatus({ historyComplete: true })
    renderWithProviders(<SyncCard />)
    expect(await screen.findByTestId('sync-history')).toHaveTextContent('Historique complet')
  })

  it('dit le rattrapage en cours, avec la limite de 200 matchs, tant que la fin n’est pas vue', async () => {
    serveStatus({ historyComplete: false })
    renderWithProviders(<SyncCard />)
    expect(await screen.findByTestId('sync-history')).toHaveTextContent(
      "Rattrapage de l'historique en cours : chaque synchronisation récupère jusqu'à 200 matchs plus anciens",
    )
  })

  it('en anglais, dit les mêmes choses', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveStatus({ historyComplete: false })
    renderWithProviders(<SyncCard />)
    expect(await screen.findByTestId('sync-history')).toHaveTextContent('up to 200 older matches')
    expect(screen.getByRole('button', { name: 'Synchronise now' })).toBeInTheDocument()
  })

  it('en démo, la carte est figée, sans état d’historique et sans lecture du statut', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities(demoCapabilities()) })
    const urls: string[] = []
    serveStatus({ historyComplete: false }, urls)
    renderWithProviders(<SyncCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    expect(bouton).toBeDisabled()
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(screen.queryByTestId('sync-history')).not.toBeInTheDocument()
    expect(urls).toEqual([])
  })

  it('un clic lance POST /players/{slug}/sync, retient le job et relit l’état', async () => {
    const posts: string[] = []
    const urls: string[] = []
    serveStatus({}, urls)
    server.use(
      http.post(`${API}/players/:playerSlug/sync`, ({ params }) => {
        posts.push(String(params.playerSlug))
        return HttpResponse.json({ job_id: 'job-7', status: 'queued' })
      }),
    )
    renderWithProviders(<SyncCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    await waitFor(() => expect(bouton).toBeEnabled())
    const lecturesAvant = urls.length

    fireEvent.click(bouton)
    await waitFor(() => expect(posts).toEqual(['p1']))
    await waitFor(() => expect(useAppShellStore.getState().activeSyncJobId).toBe('job-7'))
    await waitFor(() => expect(urls.length).toBeGreaterThan(lecturesAvant))
  })

  it('à la fin du job lancé d’ici, l’état se relit et le job n’est plus actif', async () => {
    useAppShellStore.setState({ activeSyncJobId: 'job-7' })
    const urls: string[] = []
    serveStatus({ job: { job_id: 'job-7', status: 'succeeded' }, historyComplete: true }, urls)
    renderWithProviders(<SyncCard />)
    await waitFor(() => expect(useAppShellStore.getState().activeSyncJobId).toBeNull())
    await waitFor(() => expect(urls.length).toBeGreaterThanOrEqual(2))
  })

  it.each(['failed', 'cancelled', 'interrupted'])(
    'à la fin d’un job %s lancé d’ici, le job n’est plus actif et l’état se relit',
    async (jobStatus) => {
      useAppShellStore.setState({ activeSyncJobId: 'job-7' })
      const urls: string[] = []
      serveStatus({ job: { job_id: 'job-7', status: jobStatus } }, urls)
      renderWithProviders(<SyncCard />)
      await waitFor(() => expect(useAppShellStore.getState().activeSyncJobId).toBeNull())
      await waitFor(() => expect(urls.length).toBeGreaterThanOrEqual(2))
    },
  )

  it('le job terminé d’un autre lancement ne retire pas le job actif', async () => {
    useAppShellStore.setState({ activeSyncJobId: 'job-7' })
    serveStatus({ job: { job_id: 'job-8', status: 'succeeded' } })
    renderWithProviders(<SyncCard />)
    await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(useAppShellStore.getState().activeSyncJobId).toBe('job-7')
  })

  it('la réponse d’un autre joueur est ignorée : ni historique, ni « en cours »', async () => {
    const urls: string[] = []
    serveStatus({ answerFor: 'autre-joueur', syncing: true, historyComplete: true }, urls)
    renderWithProviders(<SyncCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(bouton).toBeEnabled()
    expect(screen.queryByTestId('sync-history')).not.toBeInTheDocument()
  })

  it('une synchronisation lancée ailleurs désactive le bouton, et sa fin relit l’historique', async () => {
    serveStatus({ syncing: true, historyComplete: false })
    renderWithProviders(<SyncCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchronisation en cours…' })
    expect(bouton).toBeDisabled()
    expect(useAppShellStore.getState().activeSyncJobId).toBeNull()

    // La fin du job vient d'ailleurs : l'état relu par l'interrogation de la carte porte l'historique atteint.
    serveStatus({ syncing: false, historyComplete: true })
    await waitFor(() => expect(screen.getByTestId('sync-history')).toHaveTextContent('Historique complet'), {
      timeout: 6_000,
    })
    expect(screen.getByRole('button', { name: 'Synchroniser maintenant' })).toBeEnabled()
  }, 10_000)

  it('un refus par délai dit pourquoi', async () => {
    serveStatus()
    server.use(
      http.post(`${API}/players/:playerSlug/sync`, () =>
        HttpResponse.json({ status: 429, code: 'sync_cooldown', message: 'x' }, { status: 429 }),
      ),
    )
    renderWithProviders(<SyncCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    await waitFor(() => expect(bouton).toBeEnabled())
    fireEvent.click(bouton)
    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith("Une synchronisation vient d'être lancée : réessaie dans quelques minutes."),
    )
  })
})
