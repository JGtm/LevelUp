/**
 * Tests composant — SyncNowCard : le bouton « Synchroniser maintenant » lance le cycle, reste « en cours »
 * tant que le serveur dit qu'un cycle tourne, et dit le refus d'un lancement dans la langue de l'interface.
 */
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
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
