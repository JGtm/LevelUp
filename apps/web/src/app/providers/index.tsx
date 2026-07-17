/**
 * Providers racine de l'application — TanStack Query + Router.
 *
 * Ce composant enveloppe l'arbre complet et est monté dans main.tsx.
 *
 * P8.6 (revue 2026-04-29) : queryClient extrait dans `app/queryClient.ts`
 * pour permettre aux route loaders TanStack Router d'y accéder.
 */

import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { RouterProvider } from '@tanstack/react-router'
import { router } from '@/app/router'
import { queryClient } from '@/app/queryClient'
import { ThemeProvider } from './theme-provider'
import { DocumentLangProvider } from './document-lang-provider'

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <DocumentLangProvider>
          <RouterProvider router={router} />
        </DocumentLangProvider>
      </ThemeProvider>
      {/* Devtools React Query — dev uniquement (tree-shaké en prod via DEV),
          OFF par défaut, activable à la demande via VITE_SHOW_QUERY_DEVTOOLS=true */}
      {import.meta.env.DEV &&
        import.meta.env.VITE_SHOW_QUERY_DEVTOOLS === 'true' && (
          <ReactQueryDevtools initialIsOpen={false} />
        )}
    </QueryClientProvider>
  )
}
