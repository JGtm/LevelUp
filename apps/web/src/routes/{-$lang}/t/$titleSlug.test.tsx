/**
 * 2f — Tests du layout titre `t/$titleSlug` (D-6, D-8) : projection déclarative du
 * verdict `resolveTitleGate` + effet de convergence `applyActiveTitle`.
 *
 * Le routeur est mocké (createFileRoute → useParams contrôlé ; Outlet/Link marqueurs).
 * `applyActiveTitle` est mocké (vi.mock du module title-routing) — `resolveTitleGate`
 * reste RÉEL (verdict piloté par l'état du store, posé en setState direct).
 */
import type { ComponentType, ReactNode } from 'react'
import { describe, it, expect, beforeAll, beforeEach, vi } from 'vitest'
import { act, fireEvent, screen, waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TitleSummary } from '@/lib/api/types'

const applyActiveTitleMock = vi.fn<(slug: string) => Promise<void>>(() => Promise.resolve())
// navigate + toast : chemin d'erreur D-6 complet (4b) — la bascule échouée toaste
// et renvoie (replace) vers le titre courant.
const navigateMock = vi.fn()
const toastErrorMock = vi.fn()
// history.replace : backstop d'émission du segment lang par défaut (I10).
const historyReplaceMock = vi.fn()

// paramsRef : le slug de titre porté par l'« URL » (segment $titleSlug).
const paramsRef: { titleSlug: string; lang?: string } = { titleSlug: 'halo_infinite' }
// locationRef : l'« URL » brute lue par le backstop (pathname + searchStr avec `?` +
// hash SANS `#`, fidèle au champ TanStack useLocation()).
const locationRef: { pathname: string; searchStr: string; hash: string } = {
  pathname: '/t/halo_infinite/players/jgtm/home',
  searchStr: '',
  hash: '',
}

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    createFileRoute: () => (opts: Record<string, unknown>) => ({
      ...opts,
      useParams: () => paramsRef,
    }),
    useNavigate: () => navigateMock,
    useLocation: () => locationRef,
    useRouter: () => ({ history: { replace: historyReplaceMock } }),
    Outlet: () => <div data-testid="title-outlet" />,
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

vi.mock('sonner', () => ({ toast: { error: toastErrorMock } }))

vi.mock('@/lib/title-routing', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/title-routing')>()
  return { ...actual, applyActiveTitle: applyActiveTitleMock }
})

// Import APRÈS les mocks (le composant lit createFileRoute + applyActiveTitle au load).
import { Route } from './$titleSlug'

// `Route.component` n'est pas exposé par le type public de Route (accès interne au
// mock) → cast. Le composant porte aussi `.preload` (wrapper lazy autoCodeSplitting).
const routeComponent = (
  Route as unknown as { component: ComponentType & { preload?: () => Promise<unknown> } }
).component
const TitleLayout: ComponentType = routeComponent

// autoCodeSplitting (vite.config) : le composant de route est chargé en lazy → il
// « suspend » au premier rendu. On le précharge une fois pour rendre les rendus
// synchrones (le mock de createFileRoute conserve le wrapper lazy + son `.preload`).
beforeAll(async () => {
  await routeComponent.preload?.()
})

function title(slug: string, status?: TitleSummary['status']): TitleSummary {
  return {
    slug,
    name: slug,
    status,
    capabilities: [],
    is_default: false,
    effective_hp_to_kill: 225,
  } as unknown as TitleSummary
}

const PLAYER = {
  player_slug: 'jgtm', gamertag: 'JG', xuid: '1', waypoint_player: 'JG', is_demo: false, sync_enabled: true,
}

function setStore(partial: Partial<ReturnType<typeof useAppShellStore.getState>>) {
  act(() => useAppShellStore.setState(partial))
}

beforeEach(() => {
  applyActiveTitleMock.mockClear()
  applyActiveTitleMock.mockImplementation(() => Promise.resolve())
  navigateMock.mockClear()
  toastErrorMock.mockClear()
  historyReplaceMock.mockClear()
  paramsRef.titleSlug = 'halo_infinite'
  paramsRef.lang = undefined
  locationRef.pathname = '/t/halo_infinite/players/jgtm/home'
  locationRef.searchStr = ''
  locationRef.hash = ''
  useAppShellStore.setState({
    isBootstrapped: true,
    isTitleSwitching: false,
    locale: 'fr',
    currentTitleSlug: 'halo_infinite',
    availableTitles: [title('halo_infinite')],
    currentPlayer: null,
  })
})

describe('TitleLayout (2f)', () => {
  it('wait (pré-hydratation) → ne rend rien', () => {
    setStore({ isBootstrapped: false })
    renderWithProviders(<TitleLayout />)
    expect(screen.queryByTestId('title-outlet')).toBeNull()
    expect(screen.queryByText('Titre introuvable')).toBeNull()
  })

  it('valide + convergé → rend l’Outlet', () => {
    renderWithProviders(<TitleLayout />)
    expect(screen.getByTestId('title-outlet')).toBeInTheDocument()
    expect(applyActiveTitleMock).not.toHaveBeenCalled()
  })

  it('divergence segment↔store → applyActiveTitle puis convergence rend l’Outlet', async () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    renderWithProviders(<TitleLayout />)

    // Divergence : effet déclenché, Outlet NON rendu tant que le titre n'a pas convergé.
    expect(applyActiveTitleMock).toHaveBeenCalledWith('halo_5')
    expect(screen.queryByTestId('title-outlet')).toBeNull()

    // Convergence (le store passe au nouveau titre) → Outlet rendu.
    setStore({ currentTitleSlug: 'halo_5' })
    expect(await screen.findByTestId('title-outlet')).toBeInTheDocument()
  })

  it('slug inconnu → écran gate « Titre introuvable » (FR)', () => {
    paramsRef.titleSlug = 'inexistant'
    renderWithProviders(<TitleLayout />)
    expect(screen.getByText('Titre introuvable')).toBeInTheDocument()
    expect(screen.queryByTestId('title-outlet')).toBeNull()
  })

  it('coming_soon → écran gate « Bientôt disponible » (FR)', () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({ availableTitles: [title('halo_infinite'), title('halo_5', 'coming_soon')] })
    renderWithProviders(<TitleLayout />)
    expect(screen.getByText('Bientôt disponible')).toBeInTheDocument()
  })

  it('archived → écran gate « Titre archivé » (FR)', () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({ availableTitles: [title('halo_infinite'), title('halo_5', 'archived')] })
    renderWithProviders(<TitleLayout />)
    expect(screen.getByText('Titre archivé')).toBeInTheDocument()
  })

  it('échec d’apply → écran switch_failed + « Réessayer » re-tente applyActiveTitle', async () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    applyActiveTitleMock.mockImplementationOnce(() => Promise.reject(new Error('boom')))
    renderWithProviders(<TitleLayout />)

    expect(await screen.findByText('Changement de titre impossible')).toBeInTheDocument()
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(1)

    // « Réessayer » réarme l'état d'échec → l'effet re-tente la bascule.
    fireEvent.click(screen.getByText('Réessayer'))
    await act(async () => {
      await Promise.resolve()
    })
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(2)
  })
})

describe('TitleLayout — chemin d’erreur D-6 complet (4b)', () => {
  it('échec AVEC joueur courant → toast + navigate replace titre courant, sans boucle ni blocage', async () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      currentPlayer: PLAYER,
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    applyActiveTitleMock.mockImplementationOnce(() => Promise.reject(new Error('boom')))
    const { rerender } = renderWithProviders(<TitleLayout />)

    // Divergence → 1er appel ; l'échec (joueur présent) toaste + navigue REPLACE vers
    // le SEGMENT du titre courant (non basculé), PAS d'écran switch_failed.
    await waitFor(() => expect(toastErrorMock).toHaveBeenCalledTimes(1))
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(1)
    expect(navigateMock).toHaveBeenCalledWith({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
      params: { titleSlug: 'halo_infinite', playerSlug: 'jgtm' },
      replace: true,
    })
    expect(screen.queryByText('Changement de titre impossible')).toBeNull()

    // Simuler l'arrivée sur le segment du titre courant (le navigate replace, que le
    // mock n'exécute pas) : le layout reste MONTÉ, param → halo_infinite.
    paramsRef.titleSlug = 'halo_infinite'
    rerender(<TitleLayout />)

    // Convergence : Outlet rendu (applyFailed jamais posé → aucun blocage), AUCUN
    // nouvel appel (anti-boucle : segment == store → plus de divergence).
    expect(screen.getByTestId('title-outlet')).toBeInTheDocument()
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(1)
  })

  it('échec SANS joueur courant → écran switch_failed, ni toast ni navigate (fallback inchangé)', async () => {
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      currentPlayer: null,
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    applyActiveTitleMock.mockImplementationOnce(() => Promise.reject(new Error('boom')))
    renderWithProviders(<TitleLayout />)

    expect(await screen.findByText('Changement de titre impossible')).toBeInTheDocument()
    expect(toastErrorMock).not.toHaveBeenCalled()
    expect(navigateMock).not.toHaveBeenCalled()
  })
})

describe('TitleLayout — course back/forward + refocus (4c)', () => {
  it('popstate pendant une bascule en vol → convergence, exactement 2 appels [B, A]', async () => {
    const resolvers: Array<() => void> = []
    // Mock PENDING contrôlable, fidèle aux effets store d'applyActiveTitle : pose
    // isTitleSwitching + currentTitleSlug=slug à l'appel (comme le vrai, tôt), reste
    // en vol jusqu'à resolvers[i](), remet isTitleSwitching=false à la résolution.
    applyActiveTitleMock.mockImplementation((slug: string) => {
      useAppShellStore.setState({ isTitleSwitching: true, currentTitleSlug: slug })
      return new Promise<void>((resolve) => {
        resolvers.push(() => {
          useAppShellStore.setState({ isTitleSwitching: false })
          resolve()
        })
      })
    })

    // (1) segment B / store A → applyActiveTitle(B).
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    const { rerender } = renderWithProviders(<TitleLayout />)
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(1)
    expect(applyActiveTitleMock).toHaveBeenNthCalledWith(1, 'halo_5')

    // (2) popstate back → segment A PENDANT le vol (isTitleSwitching=true, store=B) :
    // AUCUN 2e appel (gardes isTitleSwitching + applyingRef).
    paramsRef.titleSlug = 'halo_infinite'
    rerender(<TitleLayout />)
    expect(applyActiveTitleMock).toHaveBeenCalledTimes(1)

    // (3) résolution du vol B → isTitleSwitching=false, store=B ; divergence A↔B
    // re-détectée → 2e appel applyActiveTitle(A).
    await act(async () => {
      resolvers[0]()
      await Promise.resolve()
    })
    await waitFor(() => expect(applyActiveTitleMock).toHaveBeenCalledTimes(2))
    expect(applyActiveTitleMock).toHaveBeenNthCalledWith(2, 'halo_infinite')

    // Convergence finale : résoudre le vol A → store=A=segment → Outlet rendu.
    await act(async () => {
      resolvers[1]()
      await Promise.resolve()
    })
    await waitFor(() => expect(screen.getByTestId('title-outlet')).toBeInTheDocument())
  })

  it('refocus bootstrap ré-écrivant le store pendant la bascule → convergence ré-absorbe', async () => {
    // §7 du plan : refetchOnWindowFocus peut ré-hydrater le store (currentTitleSlug ←
    // session) PENDANT une bascule. La convergence par re-comparaison doit l'absorber.
    // Mock qui converge immédiatement (pose currentTitleSlug=slug, résout).
    applyActiveTitleMock.mockImplementation((slug: string) => {
      useAppShellStore.setState({ currentTitleSlug: slug })
      return Promise.resolve()
    })

    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    const { rerender } = renderWithProviders(<TitleLayout />)
    await waitFor(() => expect(applyActiveTitleMock).toHaveBeenCalledTimes(1))
    expect(screen.getByTestId('title-outlet')).toBeInTheDocument()

    // Refocus : hydrateFromBootstrap ré-écrit currentTitleSlug avec la SESSION (ancien
    // titre A) alors que le segment est encore B → re-divergence.
    setStore({ currentTitleSlug: 'halo_infinite' })
    rerender(<TitleLayout />)

    // La convergence par re-comparaison ré-absorbe : 2e applyActiveTitle(B), pas de
    // patch ad hoc nécessaire (le layout re-run à chaque rendu).
    await waitFor(() => expect(applyActiveTitleMock).toHaveBeenCalledTimes(2))
    expect(applyActiveTitleMock).toHaveBeenNthCalledWith(2, 'halo_5')
    expect(screen.getByTestId('title-outlet')).toBeInTheDocument()
  })
})

// setLocale RÉEL capturé avant tout override (restauré en afterEach du bloc 5a).
const realSetLocale = useAppShellStore.getState().setLocale

describe('TitleLayout — réconciliation locale←segment (5a, D-12)', () => {
  const setLocaleSpy = vi.fn()

  beforeEach(() => {
    setLocaleSpy.mockClear()
    // On remplace setLocale du store par un spy pour observer l'appel sans effet de
    // bord (le spy ne met PAS à jour la locale → l'effet ne re-run pas). Le beforeEach
    // GLOBAL ne réinitialise pas setLocale (merge Zustand) → restauration explicite.
    useAppShellStore.setState({ setLocale: setLocaleSpy })
  })
  afterEach(() => {
    useAppShellStore.setState({ setLocale: realSetLocale })
  })

  it('segment lang=en + store fr → setLocale(en) appelé', () => {
    paramsRef.lang = 'en' // store.locale = 'fr' (beforeEach global)
    renderWithProviders(<TitleLayout />)
    expect(setLocaleSpy).toHaveBeenCalledWith('en')
  })

  it('segment absent (lang undefined) → setLocale JAMAIS appelé (no-op strict)', () => {
    paramsRef.lang = undefined
    renderWithProviders(<TitleLayout />)
    expect(setLocaleSpy).not.toHaveBeenCalled()
  })

  it('segment lang == locale (fr) → no-op, setLocale JAMAIS appelé', () => {
    paramsRef.lang = 'fr'
    renderWithProviders(<TitleLayout />)
    expect(setLocaleSpy).not.toHaveBeenCalled()
  })

  it('segment lang inconnu (bruit) → ignoré, setLocale JAMAIS appelé', () => {
    paramsRef.lang = 'xyz'
    renderWithProviders(<TitleLayout />)
    expect(setLocaleSpy).not.toHaveBeenCalled()
  })
})

describe('TitleLayout — émission du segment lang par défaut (backstop I10)', () => {
  it('segment lang ABSENT + titre valide/convergé → history.replace vers /{locale}/t/…', () => {
    paramsRef.lang = undefined // store.locale = 'fr' (beforeEach global)
    renderWithProviders(<TitleLayout />)
    expect(historyReplaceMock).toHaveBeenCalledWith('/fr/t/halo_infinite/players/jgtm/home')
  })

  it('préserve ?search + #hash byte-exact (enveloppe ?f= share-link)', () => {
    paramsRef.lang = undefined
    locationRef.pathname = '/t/halo_infinite/players/jgtm/stats/timeseries'
    locationRef.searchStr = '?f=abc123'
    locationRef.hash = 'top' // TanStack : hash SANS '#'
    renderWithProviders(<TitleLayout />)
    expect(historyReplaceMock).toHaveBeenCalledWith(
      '/fr/t/halo_infinite/players/jgtm/stats/timeseries?f=abc123#top',
    )
  })

  it('locale en → history.replace vers /en/t/…', () => {
    paramsRef.lang = undefined
    setStore({ locale: 'en' })
    renderWithProviders(<TitleLayout />)
    expect(historyReplaceMock).toHaveBeenCalledWith('/en/t/halo_infinite/players/jgtm/home')
  })

  it('segment lang DÉJÀ présent → aucun replace (idempotent)', () => {
    paramsRef.lang = 'fr'
    renderWithProviders(<TitleLayout />)
    expect(historyReplaceMock).not.toHaveBeenCalled()
  })

  it('titre INVALIDE (gate unknown) → aucun replace', () => {
    paramsRef.lang = undefined
    paramsRef.titleSlug = 'inexistant'
    renderWithProviders(<TitleLayout />)
    expect(historyReplaceMock).not.toHaveBeenCalled()
  })

  it('divergence segment↔store en cours → aucun replace (attend la convergence)', () => {
    paramsRef.lang = undefined
    paramsRef.titleSlug = 'halo_5'
    setStore({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [title('halo_infinite'), title('halo_5')],
    })
    renderWithProviders(<TitleLayout />)
    expect(historyReplaceMock).not.toHaveBeenCalled()
  })
})
