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
import type {
  AdminInvariantsResponse,
  DBContentionResponse,
  TokenHealthResponse,
} from '@/lib/api/types'

export function useAdminInvariants() {
  return useQuery({
    queryKey: queryKeys.adminInvariants,
    queryFn: () => api.get<AdminInvariantsResponse>('/admin/invariants'),
    // Le check ouvre les DBs de tous les joueurs : pas de refetch agressif.
    staleTime: 60_000,
    retry: false,
  })
}

/**
 * GET /admin/db-contention : compteurs du sharedprovider B-swap (swaps RO↔RW,
 * durées, lectures rejetées en 503). Lecture seule des métriques expvar.
 */
export function useAdminDBContention() {
  return useQuery({
    queryKey: queryKeys.adminDbContention,
    queryFn: () => api.get<DBContentionResponse>('/admin/db-contention'),
    staleTime: 10_000,
    retry: false,
  })
}

/**
 * GET /admin/token-health : santé des tokens auth (Accès / XSTS / Refresh) par
 * joueur, lue depuis le MultiUserTokenStore (ADR 0023) sans refresh réseau.
 */
export function useAdminTokenHealth() {
  return useQuery({
    queryKey: queryKeys.adminTokenHealth,
    queryFn: () => api.get<TokenHealthResponse>('/admin/token-health'),
    staleTime: 30_000,
    retry: false,
  })
}
