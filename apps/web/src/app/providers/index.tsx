/**
 * Providers racine de l'application — TanStack Query + Router.
 *
 * Ce composant enveloppe l'arbre complet et est monté dans main.tsx.
 *
 * P8.6 (revue 2026-04-29) : queryClient extrait dans `app/queryClient.ts`
 * pour permettre aux route loaders TanStack Router d'y accéder.
 */

import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { router } from '@/app/router'
import { queryClient } from '@/app/queryClient'
import { ThemeProvider } from './theme-provider'

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <RouterProvider router={router} />
      </ThemeProvider>
    </QueryClientProvider>
  )
}
