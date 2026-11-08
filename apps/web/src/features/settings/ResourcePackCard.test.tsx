/**
 * Tests — ResourcePackCard (paquet de ressources de l'application de bureau, lot L24) et sa logique pure.
 */
import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { ResourcePackCard } from './ResourcePackCard'
import { canRetryResourcePack, resourcePackPercent, resourcePackView } from './resourcePackLogic'
import type { ResourcePackStatus } from './resourcePackQueries'
import { RESOURCE_PACK_TEXT, resourcePackErrorKey } from './resourcePackText'

const API = '/api/v1'

function status(over: Partial<ResourcePackStatus>): ResourcePackStatus {
  return { phase: 'idle', percent: 0, restart_needed: false, ...over }
}

function serve(body: ResourcePackStatus) {
  server.use(http.get(`${API}/desktop/resources`, () => HttpResponse.json(body)))
}

describe('resourcePackView', () => {
  it('chaque phase mène à une vue, le bouton n’existe que pour un échec ou un paquet différent', () => {
    expect(resourcePackView(status({ phase: 'current' }))).toBe('upToDate')
    expect(resourcePackView(status({ phase: 'idle', active: 'a', expected: 'a' }))).toBe('upToDate')
    expect(resourcePackView(status({ phase: 'idle', active: 'a', expected: 'b' }))).toBe('mismatch')
    expect(resourcePackView(status({ phase: 'waiting_game' }))).toBe('waitingGame')
    expect(resourcePackView(status({ phase: 'downloading', percent: 40 }))).toBe('downloading')
    expect(resourcePackView(status({ phase: 'verifying' }))).toBe('verifying')
    expect(resourcePackView(status({ phase: 'ready', restart_needed: true }))).toBe('ready')
    expect(resourcePackView(status({ phase: 'ready', restart_needed: false }))).toBe('upToDate')
    expect(resourcePackView(status({ phase: 'failed', code: 'resources_download_failed' }))).toBe('failed')
    expect(canRetryResourcePack('failed')).toBe(true)
    expect(canRetryResourcePack('mismatch')).toBe(true)
    expect(canRetryResourcePack('downloading')).toBe(false)
    expect(canRetryResourcePack('upToDate')).toBe(false)
  })

  it('le pourcentage est borné', () => {
    expect(resourcePackPercent(status({ percent: 140 }))).toBe(100)
    expect(resourcePackPercent(status({ percent: -3 }))).toBe(0)
    expect(resourcePackPercent(status({ percent: 41.6 }))).toBe(42)
  })

  it('chaque code d’erreur du serveur a un texte dans chaque langue', () => {
    const codes = [
      'resources_download_failed',
      'resources_signature_invalid',
      'resources_checksum_mismatch',
      'resources_unpack_failed',
      'resources_no_key',
      'resources_refused',
      'code_futur',
    ]
    const keys = new Set(codes.map(resourcePackErrorKey))
    expect(keys.size).toBe(7)
    for (const text of Object.values(RESOURCE_PACK_TEXT)) {
      for (const key of keys) expect(text.errors[key]).not.toBe('')
    }
  })
})

describe('ResourcePackCard', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities(desktopCapabilities()) })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('dit que les ressources sont à jour', async () => {
    renderWithProviders(<ResourcePackCard />)
    expect(await screen.findByText('Les ressources sont à jour.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Réessayer' })).toBeNull()
  })

  it('montre la progression du téléchargement', async () => {
    serve(status({ phase: 'downloading', percent: 42 }))
    renderWithProviders(<ResourcePackCard />)
    expect(await screen.findByText('Téléchargement des ressources : 42 %.')).toBeInTheDocument()
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '42')
  })

  it('dit l’échec en français et propose de réessayer ; le clic relance, la réponse porte déjà la progression', async () => {
    // Le mock répond ce que répond le vrai serveur : la phase est posée AVANT la réponse de Retry (jamais l'échec
    // qu'on réessaie), et une seconde demande pendant la préparation reçoit 409.
    let current = status({ phase: 'failed', code: 'resources_signature_invalid', fingerprint: 'f', expected: 'f' })
    let retried = 0
    let refused = 0
    server.use(
      http.get(`${API}/desktop/resources`, () => HttpResponse.json(current)),
      http.post(`${API}/desktop/resources/retry`, () => {
        retried += 1
        if (current.phase === 'downloading') {
          refused += 1
          return HttpResponse.json({ status: 409, code: 'resources_busy', message: 'x' }, { status: 409 })
        }
        current = status({ phase: 'downloading', percent: 0, fingerprint: 'f', expected: 'f' })
        return HttpResponse.json(current, { status: 202 })
      }),
    )
    renderWithProviders(<ResourcePackCard />)
    expect(await screen.findByText(/La signature des ressources téléchargées est invalide/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Réessayer' }))
    await waitFor(() => expect(retried).toBe(1))
    expect(await screen.findByText('Téléchargement des ressources : 0 %.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Réessayer' })).toBeNull()
    expect(refused).toBe(0)
  })

  it('parle anglais quand la langue est l’anglais', async () => {
    useAppShellStore.setState({ locale: 'en' })
    serve(status({ phase: 'ready', restart_needed: true }))
    renderWithProviders(<ResourcePackCard />)
    expect(await screen.findByText(/Resources are ready/)).toBeInTheDocument()
  })

  it('s’efface quand le serveur ne gère aucun paquet (503) et hors de la capacité', async () => {
    server.use(
      http.get(`${API}/desktop/resources`, () =>
        HttpResponse.json({ status: 503, code: 'resources_unavailable', message: 'x' }, { status: 503 }),
      ),
    )
    const { container } = renderWithProviders(<ResourcePackCard />)
    await waitFor(() => expect(container.querySelector('[data-testid="resource-pack-card"]')).toBeNull())
    useAppShellStore.setState({ capabilities: testCapabilities({}) })
    const second = renderWithProviders(<ResourcePackCard />)
    expect(second.container.querySelector('[data-testid="resource-pack-card"]')).toBeNull()
  })
})
