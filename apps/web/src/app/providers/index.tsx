/**
 * Providers racine de l'application — TanStack Query + Router.
 *
 * Ce composant enveloppe l'arbre complet et est monté dans main.tsx.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { router } from '@/app/router'
import { ThemeProvider } from './theme-provider'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        const apiError = error as { status?: number; retryable?: boolean }
        // Pas de retry sur les erreurs 4xx sauf si retryable explicite
        if (apiError?.status != null && apiError.status < 500) return false
        return failureCount < 2
      },
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10_000),
      staleTime: 5 * 60 * 1000, // 5 min par défaut
      gcTime: 10 * 60 * 1000,
    },
  },
})

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <RouterProvider router={router} />
      </ThemeProvider>
    </QueryClientProvider>
  )
}
