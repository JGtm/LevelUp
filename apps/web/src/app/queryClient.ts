/**
 * Instance singleton du QueryClient TanStack Query.
 *
 * Extrait dans son propre module pour permettre aux route loaders TanStack
 * Router d'y accéder (P8.6 — revue 2026-04-29). Le client est consommé par :
 *   - app/providers : enveloppe `<QueryClientProvider>` autour de l'arbre
 *   - app/router    : injecté dans `context: { queryClient }` du routeur
 *   - lib/query/prefetch : utilisé via useQueryClient() (hooks React)
 */
import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
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
