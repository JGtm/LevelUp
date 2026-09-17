/**
 * Queries admin — Annuaire des joueurs (ADR 0035 D7).
 *
 * GET /admin/identities : les quatre registres d'identité (compte, profils de
 * suivi, identifiants, suivi live) lus ENSEMBLE par xuid, plus le témoin disque.
 * Route admin-gated (RequireAuth + RequireAdmin), servie sans cache côté serveur.
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { AdminIdentitiesResponse } from '@/lib/api/types'

export function useAdminIdentities() {
  return useQuery({
    queryKey: queryKeys.adminIdentities,
    queryFn: () => api.get<AdminIdentitiesResponse>('/admin/identities'),
    // La lecture balaie quatre fichiers et les dossiers joueur : pas de refetch
    // agressif, mais assez court pour qu'une correction se voie au retour.
    staleTime: 30_000,
    retry: false,
  })
}
