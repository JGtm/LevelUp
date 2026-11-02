/**
 * Tests composant — connexion Microsoft par le navigateur (L21.00) dans l'étape de connexion de
 * l'assistant : le bouton n'existe que sous la capacité `can_login_via_browser`, ouvre l'adresse par un
 * lien `target="_blank"` (jamais dans la fenêtre), sonde la tentative et fait avancer l'assistant ; le
 * code d'appareil reste le secours.
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { queryKeys } from '@/lib/query/keys'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { StepDeviceCode } from './StepDeviceCode'

const AUTHORIZE_URL = 'https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize?state=abc'

function renderStep() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries')
  render(
    <QueryClientProvider client={queryClient}>
      <StepDeviceCode />
    </QueryClientProvider>,
  )
  return { invalidate }
}

function mockBrowserLogin(status: Record<string, unknown>) {
  const started = vi.fn()
  server.use(
    http.get('/api/v1/auth/xbox/login', () => {
      started()
      return HttpResponse.json({
        attempt_id: 'att-browser',
        authorize_url: AUTHORIZE_URL,
        expires_in: 600,
        poll_interval_sec: 5,
      })
    }),
    http.get('/api/v1/auth/device-flow/:attemptId', () =>
      HttpResponse.json({ attempt_id: 'att-browser', ...status }),
    ),
  )
  return started
}

describe('StepDeviceCode — connexion par le navigateur', () => {
  let clicked: HTMLAnchorElement[]
  beforeEach(() => {
    useSetupFlowStore.getState().reset()
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities(desktopCapabilities()) })
    clicked = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      clicked.push(this)
    })
  })
  afterEach(() => {
    vi.restoreAllMocks()
    useSetupFlowStore.getState().reset()
    useAppShellStore.setState({ capabilities: null })
  })

  it('propose le bouton, ouvre l’adresse Microsoft dans un nouvel onglet et attend le retour', async () => {
    const started = mockBrowserLogin({ status: 'pending' })
    renderStep()
    expect(screen.getByText(/Une fenêtre de votre navigateur s'ouvre/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Se connecter avec Microsoft' }))

    expect(await screen.findByText('En attente de la connexion dans votre navigateur…')).toBeInTheDocument()
    expect(started).toHaveBeenCalledTimes(1)
    // L'ouverture automatique passe par un lien target=_blank que la coquille envoie au navigateur du système.
    expect(clicked).toHaveLength(1)
    expect(clicked[0].href).toBe(AUTHORIZE_URL)
    expect(clicked[0].target).toBe('_blank')
    // Le même lien reste à l'écran en repli, et la fenêtre ne navigue jamais vers Microsoft.
    const fallback = screen.getByRole('link', { name: 'Ouvrir la page Microsoft' })
    expect(fallback).toHaveAttribute('href', AUTHORIZE_URL)
    expect(fallback).toHaveAttribute('target', '_blank')
  })

  it('passe seule à la suite quand la tentative a abouti (bootstrap relu)', async () => {
    mockBrowserLogin({ status: 'provisioned', gamertag: 'PosteSeul' })
    const { invalidate } = renderStep()
    await userEvent.click(screen.getByRole('button', { name: 'Se connecter avec Microsoft' }))

    expect(await screen.findByText('PosteSeul')).toBeInTheDocument()
    await waitFor(() => expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.bootstrap }))
    expect(invalidate.mock.calls.filter(([f]) => JSON.stringify(f) === JSON.stringify({ queryKey: queryKeys.bootstrap }))).toHaveLength(1)
  })

  it('dit pourquoi la connexion a échoué et laisse recommencer ou prendre un code', async () => {
    mockBrowserLogin({ status: 'failed', error_code: 'oauth_denied' })
    renderStep()
    await userEvent.click(screen.getByRole('button', { name: 'Se connecter avec Microsoft' }))

    expect(await screen.findByText('Connexion Microsoft annulée.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Réessayer' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Utiliser un code/ })).toBeInTheDocument()
  })

  it('en anglais', async () => {
    useAppShellStore.setState({ locale: 'en' })
    mockBrowserLogin({ status: 'pending' })
    renderStep()
    expect(screen.getByRole('button', { name: 'Sign in with Microsoft' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: "Can't open the browser? Use a code instead" })).toBeInTheDocument()
  })

  it('un refus du démarrage est dit, sans texte du serveur', async () => {
    server.use(
      http.get('/api/v1/auth/xbox/login', () =>
        HttpResponse.json({ code: 'session_save_failed', message: 'internal error', retryable: true }, { status: 500 }),
      ),
    )
    renderStep()
    await userEvent.click(screen.getByRole('button', { name: 'Se connecter avec Microsoft' }))
    expect(await screen.findByText('Impossible de démarrer la connexion Microsoft. Veuillez réessayer.')).toBeInTheDocument()
    expect(screen.queryByText('internal error')).not.toBeInTheDocument()
  })

  it('le code d’appareil reste disponible en secours', async () => {
    server.use(
      http.post('/api/v1/auth/device-flow/start', () =>
        HttpResponse.json({
          attempt_id: 'att-code',
          user_code: 'ABCD1234',
          verification_uri: 'https://microsoft.com/devicelogin',
          expires_in: 900,
          poll_interval_sec: 5,
        }),
      ),
      http.get('/api/v1/auth/device-flow/:attemptId', () => HttpResponse.json({ attempt_id: 'att-code', status: 'pending' })),
    )
    renderStep()
    await userEvent.click(screen.getByRole('button', { name: /Utiliser un code/ }))
    expect(await screen.findByText('ABCD1234')).toBeInTheDocument()
  })

  it('sans la capacité, le code d’appareil est le seul chemin', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({}) })
    server.use(
      http.post('/api/v1/auth/device-flow/start', () =>
        HttpResponse.json({
          attempt_id: 'att-code',
          user_code: 'WXYZ5678',
          verification_uri: 'https://microsoft.com/devicelogin',
          expires_in: 900,
          poll_interval_sec: 5,
        }),
      ),
      http.get('/api/v1/auth/device-flow/:attemptId', () => HttpResponse.json({ attempt_id: 'att-code', status: 'pending' })),
    )
    renderStep()
    expect(await screen.findByText('WXYZ5678')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Se connecter avec Microsoft' })).not.toBeInTheDocument()
  })
})
