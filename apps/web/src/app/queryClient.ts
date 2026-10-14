/**
 * Instance singleton du QueryClient TanStack Query.
 *
 * Extrait dans son propre module pour permettre aux route loaders TanStack
 * Router d'y accéder (P8.6 — revue 2026-04-29). Le client est consommé par :
 *   - app/providers : enveloppe `<QueryClientProvider>` autour de l'arbre
 *   - app/router    : injecté dans `context: { queryClient }` du routeur
 *   - lib/query/prefetch : utilisé via useQueryClient() (hooks React)
 */
import { MutationCache, QueryClient } from '@tanstack/react-query'
import { handleDemoRefusal } from '@/lib/api/demoReadOnly'
import { useAppShellStore } from '@/stores/appShellStore'

/**
 * Statuts de passerelle jamais rejoués (plan perf 2026-09-23, D3.3) : 502 (réponse
 * coupée ou proxy sans réponse) et 504 (délai du proxy dépassé). Le 2026-09-23, une page
 * Escouade tronquée par le serveur remontait en 502 et le rejeu relançait deux fois le
 * même calcul lourd. 500 et 503 (base occupée, transitoire) restent rejoués.
 */
const GATEWAY_STATUSES_NOT_RETRIED = new Set([502, 504])

export const queryClient = new QueryClient({
  // Refus « démo en lecture seule » (403 demo_mode_forbidden) : message dans la langue de
  // l'interface, et un toast quand la mutation ne gère pas son erreur (cf. demoReadOnly.ts).
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      handleDemoRefusal(error, mutation.options.onError != null, useAppShellStore.getState().locale)
    },
  }),
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        const status = (error as { status?: number } | null)?.status
        // 4xx : la requête elle-même est en cause, la rejouer ne changerait rien.
        if (status != null && status < 500) return false
        if (status != null && GATEWAY_STATUSES_NOT_RETRIED.has(status)) return false
        // 500, 503 et erreurs réseau sans statut : deux nouvelles tentatives.
        return failureCount < 2
      },
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10_000),
      staleTime: 5 * 60 * 1000, // 5 min par défaut
      gcTime: 10 * 60 * 1000,
    },
  },
})
