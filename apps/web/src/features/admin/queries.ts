/**
 * Queries admin — Intégrité des données (invariants sync).
 *
 * GET /admin/invariants : exécute les invariants déclarés
 * (internal/sync/invariants côté Go) pour chaque joueur suivi.
 * Route admin-gated (RequireAuth + RequireAdmin).
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AdminInvariantsResponse } from '@/lib/api/types'

export function useAdminInvariants() {
  return useQuery({
    queryKey: queryKeys.adminInvariants,
    queryFn: () => api.get<AdminInvariantsResponse>('/admin/invariants'),
    // Le check ouvre les DBs de tous les joueurs : pas de refetch agressif.
    staleTime: 60_000,
    retry: false,
  })
}
