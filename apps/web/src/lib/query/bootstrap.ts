/**
 * bootstrap — la lecture de `GET /bootstrap`, écrite UNE fois : le chemin, la clé de cache et le hook.
 *
 * Quatre sites la faisaient chacun à sa façon (le layout racine, l'arrivée après la connexion Xbox, la
 * vérification d'une session expirée, le re-bootstrap d'une bascule de titre) : même endpoint, même clé,
 * options différentes. Ils partagent maintenant ce module et ne gardent que leurs options.
 * `bootstrap.guard.test.ts` interdit d'écrire le chemin ailleurs.
 */
import { useQuery, type UseQueryOptions } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { BootstrapResponse } from '@/lib/api/types'
import { queryKeys } from './keys'

/** Lit le bootstrap sans passer par le cache (re-bootstrap d'une bascule, vérification de session). */
export function fetchBootstrap(): Promise<BootstrapResponse> {
  return api.get<BootstrapResponse>('/bootstrap')
}

/** Clé et lecture du bootstrap, pour `useQuery`, `fetchQuery` ou `prefetchQuery`. */
export function bootstrapQueryOptions() {
  return { queryKey: queryKeys.bootstrap, queryFn: () => fetchBootstrap() }
}

type BootstrapQueryExtras = Omit<UseQueryOptions<BootstrapResponse>, 'queryKey' | 'queryFn'>

/** Le bootstrap en cache TanStack ; `options` règle fraîcheur, rejeu et rechargement au retour. */
export function useBootstrap(options: BootstrapQueryExtras = {}) {
  return useQuery({ ...bootstrapQueryOptions(), ...options })
}
