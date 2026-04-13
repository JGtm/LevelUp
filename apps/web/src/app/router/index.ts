/**
 * Configuration du routeur TanStack Router.
 *
 * L'arbre de routes est généré automatiquement par le plugin Vite `TanStackRouterVite`
 * depuis les fichiers dans src/routes/ lors du `npm run dev` ou `npm run build`.
 *
 * Le fichier `src/routeTree.gen.ts` est un stub initial remplacé à chaque lancement.
 */

import { createRouter } from '@tanstack/react-router'
import { routeTree } from '@/routeTree.gen'

export const router = createRouter({ routeTree })

// Déclaration des types du router
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
