/**
 * Tests — RootLayout : garde anti-éjection sur bootstrap anonyme transitoire.
 *
 * Étape 2 du fix « boucle /login ». Vérifie que :
 *  - un refetch /bootstrap ANONYME alors que le store porte déjà un utilisateur
 *    authentifié n'éjecte PAS vers /login et préserve currentUsername ;
 *  - un montage frais (store vide) + bootstrap anonyme redirige bien vers /login
 *    (le chemin logout — reload plein sur '/' — reste intact).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render } from '@testing-library/react'
import { useAppShellStore } from '@/stores/appShellStore'
import { log } from '@/components/shell/_logger'
import type { BootstrapResponse } from '@/lib/api/types'

const navigateMock = vi.fn()

// Données pilotées par test, lues par le mock useQuery.
let queryData: BootstrapResponse | undefined

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>()
  return {
    ...actual,
    useQuery: () => ({ data: queryData, isLoading: false, isError: false, failureCount: 0 }),
  }
})

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => navigateMock,
    useRouterState: () => '/',
    Outlet: () => null,
    // Évite d'exiger un vrai contexte routeur au chargement du module.
    createRootRouteWithContext: () => () => ({ component: null }),
  }
})

// Stub AppShell : le rendu authentifié ne doit pas tirer tout le shell.
vi.mock('@/components/shell/AppShell', () => ({
  AppShell: () => <div data-testid="app-shell" />,
}))

// Import APRÈS les mocks (le module route lit createRootRouteWithContext au load).
const { RootLayout } = await import('./__root')

/** Payload /bootstrap anonyme (mode xbox, aucun utilisateur connecté). */
function anonBootstrap(): BootstrapResponse {
  return {
    current_username: null,
    auth_mode: 'xbox',
    current_player: null,
    available_players: [],
    hints_visible_default: true,
    setup_required: false,
    auth_state: 'missing',
    first_launch: false,
  } as unknown as BootstrapResponse
}

describe('RootLayout — garde anti-éjection transitoire', () => {
  beforeEach(() => {
    navigateMock.mockReset()
    log._resetForTests()
    // Réinitialise l'état pertinent du store à ses défauts anonymes.
    useAppShellStore.setState({
      currentUsername: null,
      isBootstrapped: false,
      authMode: 'none',
      firstLaunch: false,
      setupRequired: false,
    })
  })

  afterEach(() => {
    queryData = undefined
  })

  it('refetch anonyme + utilisateur authentifié → pas de /login, currentUsername préservé', () => {
    // Store déjà authentifié (bootstrap initial réussi + AppShell monté).
    useAppShellStore.setState({
      currentUsername: 'alice',
      isBootstrapped: true,
      authMode: 'xbox',
    })
    queryData = anonBootstrap()

    render(<RootLayout />)

    expect(navigateMock).not.toHaveBeenCalled()
    expect(useAppShellStore.getState().currentUsername).toBe('alice')
  })

  it('montage frais (store vide) + bootstrap anonyme → redirige vers /login', () => {
    queryData = anonBootstrap()

    render(<RootLayout />)

    expect(navigateMock).toHaveBeenCalledWith({ to: '/login' })
    expect(useAppShellStore.getState().currentUsername).toBeNull()
  })
})

/**
 * Fix A — listener `levelup:auth-required` : sortir du shell mort sur une vraie
 * expiration de session. Le client HTTP dispatche l'événement sur un 401
 * `auth_required` hors /bootstrap ; RootLayout recharge la page en plein SI le
 * store se croyait authentifié, et ne fait RIEN sinon (anti-boucle sur /login).
 */
describe('RootLayout — listener levelup:auth-required (Fix A)', () => {
  const assignMock = vi.fn()
  let originalLocation: Location

  beforeEach(() => {
    assignMock.mockReset()
    // jsdom : window.location.assign lève « Not implemented ». On remplace
    // window.location par un stub minimal (assign espionné) le temps du test —
    // même pattern que le mock matchMedia du setup Vitest. queryData reste
    // undefined → l'effet bootstrap sort tôt et ne lit pas window.location.
    originalLocation = window.location
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: { pathname: '/', href: 'http://localhost/', search: '', hash: '', assign: assignMock },
    })
    log._resetForTests()
    useAppShellStore.setState({
      currentUsername: null,
      isBootstrapped: false,
      authMode: 'none',
      firstLaunch: false,
      setupRequired: false,
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
    queryData = undefined
  })

  it('store authentifié + dispatch levelup:auth-required → reload plein vers /', () => {
    useAppShellStore.setState({ currentUsername: 'alice', isBootstrapped: true, authMode: 'xbox' })

    render(<RootLayout />)
    window.dispatchEvent(new CustomEvent('levelup:auth-required'))

    expect(assignMock).toHaveBeenCalledWith('/')
  })

  it('store anonyme + dispatch levelup:auth-required → aucun reload (anti-boucle)', () => {
    render(<RootLayout />)
    window.dispatchEvent(new CustomEvent('levelup:auth-required'))

    expect(assignMock).not.toHaveBeenCalled()
  })
})
