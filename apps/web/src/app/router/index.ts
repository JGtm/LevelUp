/**
 * Configuration du routeur TanStack Router.
 *
 * L'arbre de routes est généré automatiquement par le plugin Vite `TanStackRouterVite`
 * depuis les fichiers dans src/routes/ lors du `npm run dev` ou `npm run build`.
 *
 * Le fichier `src/routeTree.gen.ts` est un stub initial remplacé à chaque lancement.
 *
 * P8.6 (revue 2026-04-29) : `queryClient` injecté dans `context` pour
 * permettre aux loaders de route de prefetcher via `queryClient.ensureQueryData`.
 */

import { createRouter } from '@tanstack/react-router'
import type { QueryClient } from '@tanstack/react-query'
import { routeTree } from '@/routeTree.gen'
import { queryClient } from '@/app/queryClient'

export interface RouterContext {
  queryClient: QueryClient
}

export const router = createRouter({
  routeTree,
  context: { queryClient } satisfies RouterContext,
})

// Déclaration des types du router
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
