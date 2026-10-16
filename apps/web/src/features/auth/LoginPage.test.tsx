/**
 * Tests composant — LoginPage (mode mot de passe) : message affiché quand la
 * connexion est refusée.
 *
 * Un 429 vient soit du limiteur d'échecs de connexion de l'API (corps JSON
 * `too_many_login_attempts`), soit de la limite de débit de nginx (corps HTML) :
 * dans les deux cas l'écran affiche le même message localisé, jamais le texte brut
 * du serveur ni « Erreur HTTP 429 ».
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import type { ComponentPropsWithoutRef } from 'react'
import { screen, fireEvent } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { LoginPage } from './LoginPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useSearch: () => mockSearch,
    Link: ({ children, to, ...props }: ComponentPropsWithoutRef<'a'> & { to: string }) => (
      <a href={to} {...props}>
        {children}
      </a>
    ),
  }
})

let mockSearch: Record<string, unknown> = {}

const TOO_MANY_FR = 'Trop de tentatives de connexion. Réessayez dans 15 minutes.'
const TOO_MANY_EN = 'Too many sign-in attempts. Try again in 15 minutes.'

const apiLimiter429 = () =>
  HttpResponse.json(
    { code: 'too_many_login_attempts', message: 'too many failed sign-in attempts, try again later', retryable: false },
    { status: 429, headers: { 'Retry-After': '900' } },
  )

const nginx429 = () =>
  new HttpResponse('<html><body><h1>429 Too Many Requests</h1></body></html>', {
    status: 429,
    headers: { 'Content-Type': 'text/html' },
  })

function submitLogin(usernameLabel: RegExp, passwordLabel: RegExp, submitLabel: RegExp) {
  fireEvent.change(screen.getByLabelText(usernameLabel), { target: { value: 'joueur' } })
  fireEvent.change(screen.getByLabelText(passwordLabel), { target: { value: 'mauvais-mot-de-passe' } })
  fireEvent.click(screen.getByRole('button', { name: submitLabel }))
}

describe('LoginPage — refus de connexion', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      authMode: 'password',
      isBootstrapped: true,
      firstLaunch: false,
      registrationMode: 'closed',
      locale: 'fr',
    })
  })

  it.each([
    ['limiteur de l’API (JSON)', apiLimiter429],
    ['limite de débit nginx (HTML)', nginx429],
  ])('429 %s : message clair en français', async (_label, respond) => {
    server.use(http.post('/api/v1/auth/login', respond))
    renderWithProviders(<LoginPage />)
    submitLogin(/Identifiant/i, /Mot de passe/i, /Se connecter/i)
    expect(await screen.findByText(TOO_MANY_FR)).toBeInTheDocument()
  })

  it('429 : message clair en anglais', async () => {
    useAppShellStore.setState({ locale: 'en' })
    server.use(http.post('/api/v1/auth/login', apiLimiter429))
    renderWithProviders(<LoginPage />)
    submitLogin(/Username/i, /Password/i, /Sign in/i)
    expect(await screen.findByText(TOO_MANY_EN)).toBeInTheDocument()
  })

  it('401 : le message « identifiants incorrects » reste inchangé', async () => {
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json(
          { code: 'invalid_credentials', message: 'identifiants incorrects', retryable: false },
          { status: 401 },
        ),
      ),
    )
    renderWithProviders(<LoginPage />)
    submitLogin(/Identifiant/i, /Mot de passe/i, /Se connecter/i)
    expect(await screen.findByText('Identifiants incorrects.')).toBeInTheDocument()
  })
})

describe('LoginPage — refus du callback de connexion Xbox (?oauth_error=)', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      authMode: 'xbox',
      oauthCodeFlowEnabled: true,
      isBootstrapped: true,
      firstLaunch: false,
      registrationMode: 'closed',
      locale: 'fr',
    })
  })
  afterEach(() => {
    mockSearch = {}
  })

  it.each([
    ['instance_locked', 'fr', 'Cette instance est fermée aux nouvelles inscriptions.'],
    ['instance_locked', 'en', 'This instance is closed to new registrations.'],
    ['invite_invalid', 'fr', "Cette invitation n'est plus valable. Demande un nouveau lien à la personne qui t'a invité."],
    ['invite_invalid', 'en', 'This invitation is no longer valid. Ask the person who invited you for a new link.'],
    ['oauth_denied', 'fr', 'Connexion Microsoft annulée.'],
    ['oauth_denied', 'en', 'Microsoft sign-in was cancelled.'],
    ['state_mismatch', 'fr', 'La connexion a expiré ou a été interrompue. Réessaie.'],
    ['state_mismatch', 'en', 'The sign-in expired or was interrupted. Please try again.'],
    ['code_exchange_failed', 'fr', "Microsoft ou Xbox n'a pas répondu. Réessaie dans un instant."],
    ['code_exchange_failed', 'en', 'Microsoft or Xbox did not respond. Please try again in a moment.'],
    ['link_failed', 'fr', "La connexion n'a pas pu aboutir. Réessaie."],
    ['link_failed', 'en', 'Sign-in could not be completed. Please try again.'],
  ])('%s (%s) : message localisé', (code, locale, message) => {
    useAppShellStore.setState({ locale: locale as 'fr' | 'en' })
    mockSearch = { oauth_error: code }
    renderWithProviders(<LoginPage />)
    expect(screen.getByRole('alert')).toHaveTextContent(message)
  })

  it('sans refus : aucun message', () => {
    renderWithProviders(<LoginPage />)
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('mode mot de passe : le refus est aussi affiché', () => {
    useAppShellStore.setState({ authMode: 'password' })
    mockSearch = { oauth_error: 'link_failed' }
    renderWithProviders(<LoginPage />)
    expect(screen.getByRole('alert')).toHaveTextContent("La connexion n'a pas pu aboutir. Réessaie.")
  })
})
