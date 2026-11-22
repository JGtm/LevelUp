/**
 * Tests composant — SyncNowCard : le bouton « Synchroniser maintenant » lance le cycle, reste « en cours »
 * tant que le serveur dit qu'un cycle tourne, et dit le refus d'un lancement dans la langue de l'interface.
 */
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { act, screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { toast } from 'sonner'
import { renderWithProviders } from '@/test/render-utils'
import { testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { SyncNowCard } from './SyncNowCard'

vi.mock('sonner', () => ({ toast: { info: vi.fn(), error: vi.fn(), success: vi.fn() } }))

const API = '/api/v1'

let launches = 0

function serveur(opts: { running?: boolean; refus?: { status: number; code: string } }) {
  server.use(
    http.get(`${API}/sync/cycle`, () => HttpResponse.json({ running: opts.running ?? false })),
    http.post(`${API}/sync/now`, () => {
      launches += 1
      if (opts.refus) {
        return HttpResponse.json(
          { status: opts.refus.status, code: opts.refus.code, message: 'MESSAGE DU SERVEUR', retryable: false },
          { status: opts.refus.status },
        )
      }
      return HttpResponse.json({ status: 'started' }, { status: 202 })
    }),
  )
}

describe('SyncNowCard', () => {
  beforeEach(() => {
    launches = 0
    vi.clearAllMocks()
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_run_sync: true }) })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('propose « Synchroniser maintenant » et lance le cycle au clic', async () => {
    serveur({})
    renderWithProviders(<SyncNowCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    expect(bouton).toBeEnabled()
    fireEvent.click(bouton)
    await waitFor(() => expect(launches).toBe(1))
    await waitFor(() => expect(toast.info).toHaveBeenCalledWith('Synchronisation lancée.'))
  })

  it('en anglais, le bouton se nomme « Sync now »', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveur({})
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByRole('button', { name: 'Sync now' })).toBeInTheDocument()
  })

  it('reste « en cours » et désactivé tant que le serveur dit qu’un cycle tourne', async () => {
    serveur({ running: true })
    renderWithProviders(<SyncNowCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchronisation en cours…' })
    expect(bouton).toBeDisabled()
    fireEvent.click(bouton)
    expect(launches).toBe(0)
  })

  it('un cycle déjà en cours côté serveur (409) se dit par le texte de l’interface, pas celui du serveur', async () => {
    serveur({ refus: { status: 409, code: 'sync_cycle_busy' } })
    renderWithProviders(<SyncNowCard />)
    fireEvent.click(await screen.findByRole('button', { name: 'Synchroniser maintenant' }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith('Une synchronisation est déjà en cours.'))
    expect(toast.error).not.toHaveBeenCalledWith('MESSAGE DU SERVEUR')
  })

  it('tout autre refus dit que la synchronisation n’a pas pu être lancée', async () => {
    serveur({ refus: { status: 503, code: 'scheduler_unavailable' } })
    renderWithProviders(<SyncNowCard />)
    fireEvent.click(await screen.findByRole('button', { name: 'Synchroniser maintenant' }))
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("La synchronisation n'a pas pu être lancée."))
  })

  it('sans la capacité can_run_sync (démo), l’encadré n’existe pas', () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_run_sync: false }) })
    serveur({})
    const { container } = renderWithProviders(<SyncNowCard />)
    expect(container).toBeEmptyDOMElement()
  })
})


// ─── État de l'historique ────────────────────────────────────────────────────────────────────────────────

const JOUEUR = { player_slug: 'JGtm', gamertag: 'JGtm' } as unknown as ReturnType<typeof useAppShellStore.getState>['currentPlayer']

const TITRE_RATTRAPAGE = {
  title_slug: 'halo_infinite',
  title_name: 'Halo Infinite',
  state: 'catching_up',
  state_label: 'Rattrapage en cours',
  match_count: 6812,
  films_pending: false,
  oldest_match_at: '2019-09-03T18:20:00Z',
  last_pass: { at: '2026-11-22T14:00:00Z', matches_added: 200, status: 'ok', status_label: 'Réussie' },
}

let historyReads = 0

function serveurHistorique(body: unknown | { status: number }, opts: { running?: boolean } = {}) {
  server.use(
    http.get(`${API}/sync/cycle`, () => HttpResponse.json({ running: opts.running ?? false })),
    http.get(`${API}/players/JGtm/sync/history`, () => {
      historyReads += 1
      if (body && typeof body === 'object' && 'status' in body && Object.keys(body).length === 1) {
        return HttpResponse.json({ code: 'history_sync_failed', message: 'x', retryable: false }, { status: (body as { status: number }).status })
      }
      return HttpResponse.json(body)
    }),
  )
}

describe('SyncNowCard — état de l’historique', () => {
  beforeEach(() => {
    historyReads = 0
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_run_sync: true }), currentPlayer: JOUEUR })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null, currentPlayer: null })
  })

  it('rattrapage en cours : état, matchs enregistrés, plus ancien, dernière et prochaine passe', async () => {
    serveurHistorique({
      auto_sync_enabled: true,
      next_cycle_at: '2026-11-22T14:10:00Z',
      titles: [TITRE_RATTRAPAGE],
    })
    renderWithProviders(<SyncNowCard />)
    const bloc = await screen.findByTestId('history-title-halo_infinite')
    expect(bloc).toHaveTextContent('Halo Infinite')
    expect(screen.getByTestId('history-state')).toHaveTextContent('Rattrapage en cours')
    expect(screen.getByTestId('history-matches')).toHaveTextContent(/6\s?812/)
    expect(screen.getByTestId('history-oldest')).toHaveTextContent('2019')
    expect(screen.getByTestId('history-last-pass')).toHaveTextContent(/Réussie, le .*2026/)
    expect(screen.getByTestId('history-last-pass')).toHaveTextContent('Matchs ajoutés : 200')
    expect(screen.getByTestId('history-next-pass')).toHaveTextContent(/Vers .*2026/)
  })

  it('historique complet : le libellé du serveur, et une seule lecture au repos', async () => {
    serveurHistorique({
      auto_sync_enabled: true,
      titles: [{ ...TITRE_RATTRAPAGE, state: 'complete', state_label: 'Complet', last_pass: undefined }],
    })
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByTestId('history-state')).toHaveTextContent('Complet')
    expect(screen.getByTestId('history-last-pass')).toHaveTextContent('Aucune passe depuis le démarrage')
    expect(screen.getByTestId('history-next-pass')).toHaveTextContent('Pas encore planifiée')
    await new Promise((r) => setTimeout(r, 150))
    expect(historyReads).toBe(1)
  })

  it('historique complet mais films en retard : l’indication d’analyse des tuiles, pas « Complet »', async () => {
    serveurHistorique({
      auto_sync_enabled: true,
      titles: [{ ...TITRE_RATTRAPAGE, state: 'complete', state_label: 'Complet', films_pending: true, last_pass: undefined }],
    })
    renderWithProviders(<SyncNowCard />)
    const etat = await screen.findByTestId('history-state')
    expect(etat).toHaveTextContent('Analyse en attente')
    expect(etat).not.toHaveTextContent('Complet')
  })

  it('en anglais, le même texte que la pastille d’une tuile de match', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveurHistorique({
      auto_sync_enabled: true,
      titles: [{ ...TITRE_RATTRAPAGE, state: 'complete', state_label: 'Complete', films_pending: true, last_pass: undefined }],
    })
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByTestId('history-state')).toHaveTextContent('Analysis pending')
  })

  it('une passe en échec se signale, sans compte de matchs ajoutés', async () => {
    serveurHistorique({
      auto_sync_enabled: true,
      titles: [
        { ...TITRE_RATTRAPAGE, last_pass: { at: '2026-11-22T14:00:00Z', matches_added: 0, status: 'failed', status_label: 'Échec' } },
      ],
    })
    renderWithProviders(<SyncNowCard />)
    const passe = await screen.findByTestId('history-last-pass')
    expect(passe).toHaveTextContent(/Échec, le/)
    expect(passe).not.toHaveTextContent('Matchs ajoutés')
    expect(passe.querySelector('.text-destructive')).not.toBeNull()
  })

  it('un cycle en cours se lit dans la prochaine passe, la synchronisation automatique coupée aussi', async () => {
    serveurHistorique({ auto_sync_enabled: false, titles: [TITRE_RATTRAPAGE] }, { running: true })
    renderWithProviders(<SyncNowCard />)
    await screen.findByTestId('history-title-halo_infinite')
    await waitFor(() => expect(screen.getByTestId('history-next-pass')).toHaveTextContent('Synchronisation en cours'))
  })

  it('synchronisation automatique désactivée hors cycle', async () => {
    serveurHistorique({ auto_sync_enabled: false, titles: [TITRE_RATTRAPAGE] })
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByTestId('history-next-pass')).toHaveTextContent('Synchronisation automatique désactivée')
  })

  it('en anglais, les rangées et les dates suivent la langue de l’interface', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveurHistorique({
      auto_sync_enabled: true,
      titles: [{ ...TITRE_RATTRAPAGE, state_label: 'Catching up', last_pass: { at: '2026-11-22T14:00:00Z', matches_added: 200, status: 'ok', status_label: 'Succeeded' } }],
    })
    renderWithProviders(<SyncNowCard />)
    const bloc = await screen.findByTestId('history-title-halo_infinite')
    expect(bloc).toHaveTextContent('Recorded matches')
    expect(bloc).toHaveTextContent('Oldest match')
    expect(screen.getByTestId('history-state')).toHaveTextContent('Catching up')
    expect(screen.getByTestId('history-matches')).toHaveTextContent('6,812')
    expect(screen.getByTestId('history-last-pass')).toHaveTextContent(/Succeeded, on .*2026/)
  })

  it('aucun titre suivi : un bloc d’état vide, sans action', async () => {
    serveurHistorique({ auto_sync_enabled: true, titles: [] })
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByText('Aucun titre suivi')).toBeInTheDocument()
    expect(screen.queryByTestId('history-state')).toBeNull()
  })

  it('lecture en échec : le dit, le bouton reste utilisable', async () => {
    serveurHistorique({ status: 500 })
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByText("L'état de l'historique n'a pas pu être lu.")).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Synchroniser maintenant' })).toBeEnabled()
  })

  it('pendant la lecture : un texte de chargement', async () => {
    server.use(
      http.get(`${API}/sync/cycle`, () => HttpResponse.json({ running: false })),
      http.get(`${API}/players/JGtm/sync/history`, () => new Promise(() => {})),
    )
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByText("Lecture de l'état de l'historique…")).toBeInTheDocument()
  })
})

// ─── Les trois états du bouton : disponible, en cours, délai minimal ─────────────────────────────────────────

/** Un instant décalé de `ms` par rapport à maintenant, en ISO. */
const IN = (ms: number) => new Date(Date.now() + ms).toISOString()

function serveurCycle(state: () => Record<string, unknown>, opts: { refus?: { status: number; code: string } } = {}) {
  server.use(
    http.get(`${API}/sync/cycle`, () => HttpResponse.json(state())),
    http.post(`${API}/sync/now`, () => {
      launches += 1
      if (opts.refus) {
        return HttpResponse.json(
          { status: opts.refus.status, code: opts.refus.code, message: 'MESSAGE DU SERVEUR', retryable: false },
          { status: opts.refus.status },
        )
      }
      return HttpResponse.json({ status: 'started' }, { status: 202 })
    }),
  )
}

describe('SyncNowCard — état du bouton', () => {
  beforeEach(() => {
    launches = 0
    vi.clearAllMocks()
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_run_sync: true }) })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('disponible : le bouton propose l’action, aucune ligne de statut', async () => {
    serveurCycle(() => ({ running: false, last_cycle_at: IN(-600_000) }))
    renderWithProviders(<SyncNowCard />)
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' })
    expect(bouton).toBeEnabled()
    expect(screen.queryByTestId('sync-now-status')).toBeNull()
  })

  it('en cours : libellé et ligne dans les deux langues, bouton désactivé', async () => {
    serveurCycle(() => ({ running: true }))
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByRole('button', { name: 'Synchronisation en cours…' })).toBeDisabled()
    expect(screen.getByTestId('sync-now-status')).toHaveTextContent('Synchronisation en cours.')

    act(() => useAppShellStore.setState({ locale: 'en' }))
    expect(await screen.findByRole('button', { name: 'Sync in progress…' })).toBeDisabled()
    expect(screen.getByTestId('sync-now-status')).toHaveTextContent('Sync in progress.')
  })

  it('délai minimal : « Disponible dans 1 min 40 », fin du dernier cycle et prochain lancement, sans clic possible', async () => {
    serveurCycle(() => ({ running: false, last_cycle_at: IN(-20_000), manual_available_at: IN(100_500) }))
    renderWithProviders(<SyncNowCard />)
    const bouton = await screen.findByRole('button', { name: /^Disponible dans 1 min \d\d$/ })
    expect(bouton).toBeDisabled()
    const ligne = screen.getByTestId('sync-now-status')
    expect(ligne).toHaveTextContent(/^Dernière synchronisation terminée à \d{2}:\d{2}\. Prochain lancement possible à \d{2}:\d{2}\.$/)
    expect(ligne.className).not.toMatch(/destructive|warning/)
    fireEvent.click(bouton)
    expect(launches).toBe(0)
  })

  it('délai minimal en anglais', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serveurCycle(() => ({ running: false, last_cycle_at: IN(-20_000), manual_available_at: IN(100_500) }))
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByRole('button', { name: /^Available in 1 min \d\d$/ })).toBeDisabled()
    expect(screen.getByTestId('sync-now-status')).toHaveTextContent(/^Last sync finished at .+\. Next run available at .+\.$/)
  })

  it('compte à rebours à la seconde, puis retour seul à « Synchroniser maintenant »', async () => {
    serveurCycle(() => ({ running: false, last_cycle_at: IN(-119_000), manual_available_at: IN(1_500) }))
    renderWithProviders(<SyncNowCard />)
    expect(await screen.findByRole('button', { name: /^Disponible dans [12] s$/ })).toBeDisabled()
    const bouton = await screen.findByRole('button', { name: 'Synchroniser maintenant' }, { timeout: 4000 })
    expect(bouton).toBeEnabled()
    expect(screen.queryByTestId('sync-now-status')).toBeNull()
  })

  it('un 429 malgré tout (deux onglets) : la ligne du délai, pas de toast d’erreur', async () => {
    let apresRefus = false
    serveurCycle(() =>
      apresRefus
        ? { running: false, last_cycle_at: IN(-10_000), manual_available_at: IN(110_000) }
        : { running: false },
    )
    server.use(
      http.post(`${API}/sync/now`, () => {
        launches += 1
        apresRefus = true
        return HttpResponse.json({ code: 'sync_cycle_cooldown', message: 'MESSAGE DU SERVEUR', retryable: false }, { status: 429 })
      }),
    )
    renderWithProviders(<SyncNowCard />)
    fireEvent.click(await screen.findByRole('button', { name: 'Synchroniser maintenant' }))
    expect(await screen.findByRole('button', { name: /^Disponible dans 1 min \d\d$/ })).toBeDisabled()
    expect(screen.getByTestId('sync-now-status')).toHaveTextContent(/Prochain lancement possible à/)
    expect(launches).toBe(1)
    expect(toast.error).not.toHaveBeenCalled()
  })
})
