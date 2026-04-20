/**
 * Route racine — layout partagé par toutes les pages.
 *
 * Au montage, déclenche le bootstrap et hydrate l'AppShellStore.
 * Bloque le rendu tant que bootstrap n'a pas répondu.
 * En mode setup_required, redirige vers /setup.
 */

import { createRootRoute, Outlet, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { AppShell } from '@/components/shell/AppShell'
import type { BootstrapResponse } from '@/lib/api/types'

function RootLayout() {
  const navigate = useNavigate()
  const hydrateFromBootstrap = useAppShellStore((s) => s.hydrateFromBootstrap)
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const setupRequired = useAppShellStore((s) => s.setupRequired)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.bootstrap,
    queryFn: () => api.get<BootstrapResponse>('/bootstrap'),
    staleTime: 2 * 60 * 1000,
    refetchOnWindowFocus: false,
  })

  useEffect(() => {
    if (!data) return
    hydrateFromBootstrap(data)
    if (data.setup_required) {
      navigate({ to: '/setup' })
    }
  }, [data, hydrateFromBootstrap, navigate])

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <span className="text-sm text-muted-foreground animate-pulse">
          Chargement LevelUp…
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

  return <AppShell />
}

export const Route = createRootRoute({
  component: RootLayout,
})
