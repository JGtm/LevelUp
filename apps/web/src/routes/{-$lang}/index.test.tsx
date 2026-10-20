/**
 * Index sous le segment de langue optionnel : `/`, `/en/` et `/fr/` (avec ou sans barre
 * finale) sont la MÊME route d'index. Avant le déplacement, seul `/` existait : `/en/` et
 * `/fr/` tombaient sur la page « introuvable » du routeur pour un utilisateur connecté.
 *
 * Deux niveaux :
 *  - résolution sur l'arbre de routes RÉEL (`routeTree.gen`) : qui gagne pour chaque URL ;
 *  - comportement de l'index (anonyme → /login, connecté → accueil du joueur) sur un arbre
 *    minimal qui réutilise les options RÉELLES de la route (composant + garde de langue),
 *    sans le `__root` réel (bootstrap réseau + coquille).
 */
import { describe, it, expect, beforeAll, beforeEach } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import {
  createMemoryHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
  type RouteComponent,
  useParams,
} from '@tanstack/react-router'
import type { ComponentType } from 'react'

import type { RouterContext } from '@/app/router'
import { routeTree } from '@/routeTree.gen'
import { queryClient } from '@/app/queryClient'
import { useAppShellStore } from '@/stores/appShellStore'
import { Route as IndexRoute } from './index'

const INDEX_ID = '/{-$lang}/'

function leafRouteId(router: { state: { matches: { routeId: string }[] } }): string {
  const matches = router.state.matches
  return matches[matches.length - 1].routeId
}

describe('résolution de l’index sous {-$lang} (arbre réel)', () => {
  async function loadAt(pathname: string) {
    const router = createRouter({
      routeTree,
      context: { queryClient },
      history: createMemoryHistory({ initialEntries: [pathname] }),
    })
    await router.load()
    return router
  }

  it.each(['/', '/en', '/en/', '/fr', '/fr/'])('%s résout la route d’index', async (pathname) => {
    const router = await loadAt(pathname)
    expect(leafRouteId(router)).toBe(INDEX_ID)
    expect(router.state.matches.some((m) => m.status === 'notFound')).toBe(false)
  })

  it('porte la langue de l’URL dans les params (et rien à la racine)', async () => {
    const en = await loadAt('/en/')
    expect(en.state.matches[en.state.matches.length - 1].params).toEqual({ lang: 'en' })
    const root = await loadAt('/')
    expect(root.state.matches[root.state.matches.length - 1].params).toEqual({})
  })

  it('une langue inconnue (/xx/) reste « introuvable », comme avant le déplacement', async () => {
    for (const pathname of ['/xx/', '/xx']) {
      const router = await loadAt(pathname)
      expect(router.state.matches.some((m) => m.status === 'notFound')).toBe(true)
    }
  })

  it('les pages statiques et le splat legacy gardent la priorité sur le segment optionnel', async () => {
    expect(leafRouteId(await loadAt('/login'))).toBe('/login')
    expect(leafRouteId(await loadAt('/help'))).toBe('/help')
    expect(leafRouteId(await loadAt('/players'))).toBe('/players/$')
    expect(leafRouteId(await loadAt('/players/jgtm/home'))).toBe('/players/$')
  })

  it('le sous-arbre title-scoped reste atteignable avec et sans langue', async () => {
    const plain = await loadAt('/t/halo_infinite/players/jgtm/home')
    expect(plain.state.matches.some((m) => m.routeId === '/{-$lang}/t/$titleSlug')).toBe(true)
    const fr = await loadAt('/fr/t/halo_infinite/players/jgtm/home')
    expect(fr.state.matches.some((m) => m.routeId === '/{-$lang}/t/$titleSlug')).toBe(true)
  })
})

describe('comportement de l’index (options réelles de la route)', () => {
  const PLAYER = {
    player_slug: 'jgtm',
    gamertag: 'JG',
    xuid: '1',
    waypoint_player: 'JG',
    is_demo: false,
    sync_enabled: true,
  }

  // Le composant est chargé en lazy (autoCodeSplitting) : préchargé une fois pour que le
  // rendu soit synchrone.
  const indexComponent = IndexRoute.options.component as ComponentType & {
    preload?: () => Promise<unknown>
  }
  beforeAll(async () => {
    await indexComponent.preload?.()
  })

  function HomeMarker() {
    const p = useParams({ strict: false }) as Record<string, string | undefined>
    return (
      <div data-testid="home" data-lang={p.lang} data-title={p.titleSlug} data-player={p.playerSlug} />
    )
  }

  function renderAt(pathname: string) {
    const root = createRootRouteWithContext<RouterContext>()({ component: Outlet })
    const index = createRoute({
      getParentRoute: () => root,
      path: '/{-$lang}/',
      beforeLoad: IndexRoute.options.beforeLoad,
      component: indexComponent as RouteComponent,
    })
    const login = createRoute({
      getParentRoute: () => root,
      path: '/login',
      component: () => <div data-testid="login" />,
    })
    const home = createRoute({
      getParentRoute: () => root,
      path: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
      component: HomeMarker,
    })
    const router = createRouter({
      routeTree: root.addChildren([index, login, home]),
      context: { queryClient },
      history: createMemoryHistory({ initialEntries: [pathname] }),
    })
    render(<RouterProvider router={router} />)
    return router
  }

  beforeEach(() => {
    act(() => {
      useAppShellStore.setState({
        isBootstrapped: true,
        authMode: 'xbox',
        currentUsername: 'alice',
        currentPlayer: PLAYER,
        availablePlayers: [PLAYER],
        currentTitleSlug: 'halo_infinite',
        locale: 'fr',
      } as never)
    })
  })

  it.each(['/', '/fr/', '/en/'])('connecté : %s redirige vers l’accueil du joueur', async (pathname) => {
    renderAt(pathname)
    const home = await screen.findByTestId('home')
    expect(home.dataset.title).toBe('halo_infinite')
    expect(home.dataset.player).toBe('jgtm')
  })

  it('connecté : la langue connue de l’URL est conservée dans la redirection', async () => {
    renderAt('/en/')
    const home = await screen.findByTestId('home')
    expect(home.dataset.lang).toBe('en')
  })

  it('connecté : sans langue dans l’URL, le segment émis est la locale de session', async () => {
    renderAt('/')
    const home = await screen.findByTestId('home')
    expect(home.dataset.lang).toBe('fr')
  })

  it.each(['/', '/fr/', '/en/'])('anonyme : %s redirige vers /login', async (pathname) => {
    act(() => {
      useAppShellStore.setState({ currentUsername: null, currentPlayer: null, availablePlayers: [] } as never)
    })
    renderAt(pathname)
    await screen.findByTestId('login')
    expect(screen.queryByTestId('home')).toBeNull()
  })

  it('langue inconnue (/xx/) : ni accueil ni /login — page introuvable', async () => {
    const router = renderAt('/xx/')
    await waitFor(() => expect(router.state.matches.some((m) => m.status === 'notFound')).toBe(true))
    expect(screen.getByText('Not Found')).toBeTruthy()
    expect(screen.queryByTestId('home')).toBeNull()
    expect(screen.queryByTestId('login')).toBeNull()
  })
})
