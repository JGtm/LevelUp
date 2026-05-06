/**
 * Route racine — layout partagé par toutes les pages.
 *
 * Au montage, déclenche le bootstrap et hydrate l'AppShellStore.
 * Bloque le rendu tant que bootstrap n'a pas répondu.
 * En mode setup_required, redirige vers /setup.
 */

import { createRootRouteWithContext, Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import type { RouterContext } from '@/app/router'
import { useEffect } from 'react'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { resolvePageTitle } from '@/lib/pageTitle'
import { useAppShellStore } from '@/stores/appShellStore'
import { AppShell } from '@/components/shell/AppShell'
import type { BootstrapResponse } from '@/lib/api/types'

function RootLayout() {
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const hydrateFromBootstrap = useAppShellStore((s) => s.hydrateFromBootstrap)
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const setupRequired = useAppShellStore((s) => s.setupRequired)
  const authMode = useAppShellStore((s) => s.authMode)
  const currentUsername = useAppShellStore((s) => s.currentUsername)
  const firstLaunch = useAppShellStore((s) => s.firstLaunch)

  const { data, isLoading, isError, failureCount } = useQuery({
    queryKey: queryKeys.bootstrap,
    queryFn: () => api.get<BootstrapResponse>('/bootstrap'),
    staleTime: 2 * 60 * 1000,
    refetchOnWindowFocus: false,
    // Le serveur Go peut mettre 5–15 s à démarrer (CGO + DuckDB) en dev
    // (`air`) ou sur VPS (cold start, redéploiement). On retry en backoff
    // exponentiel pour absorber la fenêtre de démarrage avant d'afficher
    // l'écran "API injoignable" : 0.5 → 1 → 2 → 4 → 4 → 4 s ≈ 15 s total.
    retry: 6,
    retryDelay: (n) => Math.min(500 * 2 ** n, 4000),
  })

  useEffect(() => {
    document.title = resolvePageTitle(pathname)
  }, [pathname])

  useEffect(() => {
    if (!data) return
    hydrateFromBootstrap(data)

    // Auth locale : rediriger si pas connecté
    if (data.auth_mode === 'password') {
      const path = window.location.pathname
      if (data.first_launch && path !== '/register') {
        navigate({ to: '/register' })
        return
      }
      if (!data.current_username && path !== '/login' && path !== '/register') {
        navigate({ to: '/login' })
        return
      }
    }

    if (data.setup_required) {
      navigate({ to: '/setup' })
    }
  }, [data, hydrateFromBootstrap, navigate])

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <span className="text-sm text-muted-foreground animate-pulse">
          {failureCount > 0
            ? `Connexion à l'API… (tentative ${failureCount + 1}/7)`
            : 'Chargement LevelUp…'}
        </span>
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-center space-y-2">
          <p className="font-semibold text-destructive">Impossible de contacter l'API.</p>
          <p className="text-sm text-muted-foreground">
            Vérifiez que le serveur Go est démarré (<code>make go-api-run</code>).
          </p>
          <button
            className="text-sm underline text-primary"
            onClick={() => window.location.reload()}
          >
            Réessayer
          </button>
        </div>
      </div>
    )
  }

  // Setup en cours → pas de shell
  if (!isBootstrapped || setupRequired) {
    return <Outlet />
  }

  // Auth locale non connectée → pages login/register sans shell
  if (authMode === 'password' && !currentUsername) {
    return <Outlet />
  }

  // Premier lancement → page register sans shell
  if (authMode === 'password' && firstLaunch) {
    return <Outlet />
  }

  return <AppShell />
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
})
